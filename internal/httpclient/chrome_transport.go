package httpclient

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// chromeTransport is an http.RoundTripper that sends each request over a real
// Chrome TLS + HTTP/2 fingerprint via bogdanfinn/tls-client (utls fork + fhttp).
//
// Why this exists: Go's standard net/http (even with utls for the ClientHello)
// cannot match Chrome's HTTP/2 layer — pseudo-header order, SETTINGS and
// WINDOW_UPDATE values are Go's, not Chrome's. tls-client bundles a coherent
// Chrome profile (JA3/JA4 ClientHello + the matching h2 fingerprint), so the
// whole handshake and h2 frames look like Chrome.
//
// Design:
//   - One chromeTransport per worker (per account). It owns its own tls-client
//     instances and connection pools — no state is shared between workers, same
//     invariant as the old transport.
//   - The Chrome *version* is chosen per request from the request's User-Agent,
//     so an account whose UA claims Chrome 131 also gets the Chrome 131 TLS/h2
//     fingerprint (UA↔JA3 coherence). Each distinct major is built once and
//     cached; a worker uses one UA, so the cache normally holds a single client.
//   - Redirects and the cookie jar are intentionally left to the wrapping
//     stdlib http.Client (WithNotFollowRedirects + no jar here): the stdlib
//     client adds the Cookie header before RoundTrip and reads Set-Cookie after,
//     and it follows 3xx itself — so resp.Request.URL still tracks the final URL.
//   - IPv4 is forced (WithDisableIPV6) to match the project-wide tcp4 policy.
//
// This client is used solely for authorized Facebook/Meta API calls.
type chromeTransport struct {
	proxy   string
	timeout time.Duration

	mu      sync.Mutex
	clients map[int]tls_client.HttpClient // keyed by Chrome major version
}

func newChromeTransport(proxy string, timeout time.Duration) *chromeTransport {
	return &chromeTransport{
		proxy:   proxy,
		timeout: timeout,
		clients: make(map[int]tls_client.HttpClient),
	}
}

var reChromeMajor = regexp.MustCompile(`Chrome/(\d+)`)

// chromeMajor extracts the Chrome major version from a User-Agent, or 0 if
// absent. Edge UAs also carry a "Chrome/<n>" token (Edge is Chromium), so they
// map to the matching Chromium profile — Edge's TLS/h2 is ~identical to Chrome.
func chromeMajor(ua string) int {
	m := reChromeMajor.FindStringSubmatch(ua)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// profileForMajor maps a Chrome major version to the nearest tls-client profile
// that does not exceed it (tls-client ships a subset of versions). Unknown or
// older UAs fall back to a stable mid profile.
func profileForMajor(major int) profiles.ClientProfile {
	switch {
	case major >= 152:
		return profiles.Chrome_152
	case major >= 150:
		return profiles.Chrome_150
	case major >= 146:
		return profiles.Chrome_146
	case major >= 144:
		return profiles.Chrome_144
	case major >= 133:
		return profiles.Chrome_133
	case major >= 131:
		return profiles.Chrome_131
	case major >= 124: // pool's Chrome/Edge 124–130 (nearest non-PSK is 124)
		return profiles.Chrome_124
	case major >= 1: // any older Chrome/Edge
		return profiles.Chrome_120
	default: // no Chrome token in UA — should not happen in normal flows
		return profiles.Chrome_133
	}
}

// chromeHeaderOrder is the order Chrome emits common request headers. tls-client
// reorders the request headers to this sequence (pseudo-header order comes from
// the profile itself). Headers not listed keep their relative order after these.
// note: lower-case per fhttp requirement; this is a secondary fingerprint signal.
var chromeHeaderOrder = []string{
	"host",
	"connection",
	"cache-control",
	"sec-ch-ua",
	"x-asbd-id",          // Facebook telemetry — appears after sec-ch-ua in Chrome captures
	"sec-ch-ua-mobile",
	"x-fb-lsd",
	"sec-ch-ua-platform",
	"sec-ch-ua-full-version-list",
	"sec-ch-ua-platform-version",
	"sec-ch-ua-model",
	"sec-ch-prefers-color-scheme",
	"content-type",
	"x-fb-friendly-name", // GraphQL mutation name — appears before User-Agent in Chrome captures
	"upgrade-insecure-requests",
	"user-agent",
	"accept",
	"origin",
	"sec-fetch-site",
	"sec-fetch-mode",
	"sec-fetch-user",
	"sec-fetch-dest",
	"referer",
	"accept-encoding",
	"accept-language",
	"priority",
	"cookie",
	"dpr",
	"viewport-width",
	"content-length",
}

func (t *chromeTransport) clientFor(ua string) (tls_client.HttpClient, error) {
	key := chromeMajor(ua)
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.clients[key]; ok {
		return c, nil
	}
	secs := int(t.timeout / time.Second)
	if secs <= 0 {
		secs = 30
	}
	opts := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profileForMajor(key)),
		tls_client.WithNotFollowRedirects(), // stdlib http.Client owns redirects
		tls_client.WithDisableIPV6(),        // force IPv4 (project-wide tcp4 policy)
		tls_client.WithTimeoutSeconds(secs),
	}
	if t.proxy != "" {
		opts = append(opts, tls_client.WithProxyUrl(normalizeProxyURL(t.proxy)))
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, err
	}
	t.clients[key] = c
	return c, nil
}

// normalizeProxyURL prepends http:// when the scheme is omitted, matching the
// lenient format the rest of the app accepts (e.g. "127.0.0.1:8001").
func normalizeProxyURL(proxy string) string {
	if !strings.Contains(proxy, "://") {
		return "http://" + proxy
	}
	return proxy
}

func (t *chromeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	client, err := t.clientFor(req.Header.Get("User-Agent"))
	if err != nil {
		return nil, err
	}

	// Single-shot send: the wrapping stdlib http.Client handles retries and
	// redirects, so we pass the body straight through (no GetBody needed here).
	fReq, err := fhttp.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), req.Body)
	if err != nil {
		return nil, err
	}
	fReq.Host = req.Host
	fReq.ContentLength = req.ContentLength

	// Deep-copy headers so adding the order key never mutates the caller's map.
	h := make(fhttp.Header, len(req.Header)+1)
	for k, vv := range req.Header {
		cp := make([]string, len(vv))
		copy(cp, vv)
		h[k] = cp
	}
	// Clone: tls-client lower-cases this slice in place (allToLower), and the
	// package-level chromeHeaderOrder is shared across concurrent workers.
	h[fhttp.HeaderOrderKey] = slices.Clone(chromeHeaderOrder)
	fReq.Header = h

	fResp, err := client.Do(fReq)
	if err != nil {
		return nil, err
	}

	resp := &http.Response{
		Status:        fResp.Status,
		StatusCode:    fResp.StatusCode,
		Proto:         fResp.Proto,
		ProtoMajor:    fResp.ProtoMajor,
		ProtoMinor:    fResp.ProtoMinor,
		Header:        make(http.Header, len(fResp.Header)),
		Body:          fResp.Body,
		ContentLength: fResp.ContentLength,
		Request:       req, // stdlib client reads resp.Request.URL to track final URL
	}
	for k, vv := range fResp.Header {
		resp.Header[k] = vv
	}
	return resp, nil
}

// CloseIdleConnections is called by net/http.Client.CloseIdleConnections (the
// worker defers it on stop) — release every cached tls-client's pool.
func (t *chromeTransport) CloseIdleConnections() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, c := range t.clients {
		c.CloseIdleConnections()
	}
}
