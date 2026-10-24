package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	"facebook-inbox/internal/httpclient"
)

// seedJar loads a browser credential cookie string into the http.CookieJar for
// facebook.com. Call once per session (in messengerSession.init) so all
// subsequent requests through that client send cookies via the jar and capture
// any Set-Cookie updates automatically — without re-setting the Cookie header
// on every individual request.
func seedJar(jar http.CookieJar, credential string) {
	if jar == nil || credential == "" {
		return
	}
	fbURL, _ := url.Parse("https://www.facebook.com")
	// Collect names already in the jar (from prior Set-Cookie responses) so we
	// don't overwrite a fresher value that Facebook already rotated.
	existing := make(map[string]bool)
	for _, c := range jar.Cookies(fbURL) {
		existing[c.Name] = true
	}
	var toAdd []*http.Cookie
	for _, part := range strings.Split(credential, ";") {
		part = strings.TrimSpace(part)
		eq := strings.IndexByte(part, '=')
		if eq <= 0 {
			continue
		}
		name := strings.TrimSpace(part[:eq])
		value := strings.TrimSpace(part[eq+1:])
		if name == "" || existing[name] {
			continue
		}
		toAdd = append(toAdd, &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: ".facebook.com",
			Path:   "/",
			Secure: true,
		})
	}
	if len(toAdd) > 0 {
		jar.SetCookies(fbURL, toAdd)
	}
}

// jarCookieHeader returns the current Cookie header value from the jar for
// rawURL. Used by WebSocket dial paths that bypass the http.Client's jar.
// Falls back to fallback when the jar is nil or has no cookies for the URL.
func jarCookieHeader(jar http.CookieJar, rawURL, fallback string) string {
	if jar == nil {
		return fallback
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fallback
	}
	cookies := jar.Cookies(u)
	if len(cookies) == 0 {
		return fallback
	}
	parts := make([]string, len(cookies))
	for i, c := range cookies {
		parts[i] = c.Name + "=" + c.Value
	}
	return strings.Join(parts, "; ")
}

// fbGetWithFinalURL performs a GET with full browser headers, follows redirects,
// and returns (body, finalURL, err). finalURL is the URL after all redirects.
// The extra Sec-Fetch-* headers are required — without them Facebook returns a
// login page and LSD/DTSG extraction fails (confirmed in debug session).
// fetchSite must be "none" for cold navigations (address-bar direct) or
// "same-origin" for in-session navigations (clicks within facebook.com).
// Mirrors Python HttpAdapter.get() + tracking the final redirect URL.
func fbGetWithFinalURL(ctx context.Context, client *http.Client, cookie, rawURL string, p browserProfile, fetchSite string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", rawURL, err
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	p.setHeaders(req)      // UA + Sec-Ch-Ua family
	p.setDeviceHints(req)  // Dpr + Viewport-Width (accepted via Accept-CH)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Accept-Encoding: not set explicitly — Go transport adds gzip and decompresses automatically,
	// which looks more natural (identity is never sent by real browsers on navigation requests).
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", fetchSite)
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	// same-origin navigations always carry a Referer (address-bar navigations do not)
	if fetchSite == "same-origin" {
		req.Header.Set("Referer", "https://www.facebook.com/")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", rawURL, err
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MB max
	return string(raw), finalURL, nil
}

// parseFBJSON strips the "for (;;);" anti-hijacking prefix Facebook prepends to
// JSON responses, then unmarshals into a map.
// Mirrors Python _parse_facebook_json().
func parseFBJSON(raw string) (map[string]interface{}, error) {
	body := strings.TrimPrefix(raw, "for (;;);")
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		log.Printf("messenger: parseFBJSON failed. Raw (first 300 chars): %.300s", raw)
		return nil, err
	}
	return out, nil
}

// detectCheckpoint returns the checkpoint URL if the FB JSON response is a
// security redirect, or "" if not. Matches the ServerRedirect pattern:
//
//	"jsmods":{"require":[["ServerRedirect","redirectPageTo",[],["/checkpoint/..."]]}
func detectCheckpoint(data map[string]interface{}) string {
	jsmods, _ := data["jsmods"].(map[string]interface{})
	if jsmods == nil {
		return ""
	}
	require, _ := jsmods["require"].([]interface{})
	for _, item := range require {
		arr, _ := item.([]interface{})
		if len(arr) < 4 {
			continue
		}
		if name, _ := arr[0].(string); name != "ServerRedirect" {
			continue
		}
		args, _ := arr[3].([]interface{})
		if len(args) == 0 {
			continue
		}
		if u, _ := args[0].(string); strings.Contains(u, "checkpoint") {
			if strings.HasPrefix(u, "http") {
				return u
			}
			return "https://www.facebook.com" + u
		}
	}
	return ""
}

// wsHeaders returns the HTTP headers needed to upgrade a connection to a
// Facebook WebSocket. Mirrors Python _ws_headers().
// Sec-Fetch-* headers match what Chrome sends on a WebSocket upgrade from a
// same-site page (facebook.com → gateway.facebook.com / edge-chat.facebook.com).
// Note: no Dpr/Viewport-Width — those are HTTP-only client hints, not part of
// the WS upgrade handshake.
func wsHeaders(cookie string, p browserProfile) http.Header {
	h := http.Header{
		"Cookie":          {cookie},
		"Origin":          {"https://www.facebook.com"},
		"Accept-Language": {"en-US,en;q=0.9"},
		"Accept-Encoding": {"gzip, deflate, br, zstd"},
		"Cache-Control":   {"no-cache"},
		"Pragma":          {"no-cache"},
		"Sec-Fetch-Site":  {"same-site"},
		"Sec-Fetch-Mode":  {"websocket"},
		"Sec-Fetch-Dest":  {"websocket"},
	}
	// Apply UA + client hints via a scratch request then copy — setHeaders needs *http.Request.
	tmp, _ := http.NewRequest(http.MethodGet, "https://www.facebook.com", nil)
	p.setHeaders(tmp)
	for k, v := range tmp.Header {
		h[k] = v
	}
	return h
}

// newWSDialer returns a gorilla/websocket Dialer whose TLS handshake carries a
// real Chrome fingerprint (utls, profile picked from ua) — the same impersonation
// as the HTTP path. proxy may be "" for a direct connection; when set, the
// handshake tunnels through an HTTP CONNECT proxy. See httpclient.NewChromeWSDialTLS
// for why ALPN is forced to http/1.1 and the resulting JA4 caveat.
func newWSDialer(proxy, ua string) *websocket.Dialer {
	return &websocket.Dialer{
		NetDialTLSContext: httpclient.NewChromeWSDialTLS(ua, proxy),
		HandshakeTimeout:  messengerWSTimeout,
	}
}

// safeCloseWS sends a close frame and closes the underlying TCP connection.
// Errors are ignored — the connection may already be closing.
func safeCloseWS(conn *websocket.Conn) {
	if conn == nil {
		return
	}
	_ = conn.Close()
}

// mapKeys returns the keys of m as a slice — used only for debug logging.
func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// parseInt64 parses a decimal string to int64.
func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// checkpointFromPayload wraps detectCheckpoint and returns a formatted error.
// Unused by current callers (upload.go calls detectCheckpoint directly) but
// kept as a convenience for future callers.
func checkpointFromPayload(data map[string]interface{}) error {
	cpURL := detectCheckpoint(data)
	if cpURL == "" {
		return nil
	}
	return fmt.Errorf("messenger: tài khoản bị checkpoint — cần xác minh tại %s", cpURL)
}
