package httpclient

import (
	"net/http"
	"net/http/cookiejar"
	"time"
)

// New builds an http.Client dedicated to a single worker. Each call returns a
// fresh client with its OWN Transport (chromeTransport), its OWN connection
// pools, its OWN cookie jar and (optionally) its OWN proxy, so no mutable
// network state is ever shared between workers.
//
// The transport sends every request with a real Chrome TLS + HTTP/2 fingerprint
// (see chromeTransport). The stdlib http.Client still owns the cookie jar,
// redirect-following and overall timeout; the transport only does one fingerprinted
// round trip per call. All connections are IPv4-only.
//
// This client is intended solely for authorized Facebook/Meta API calls.
func New(proxy string, timeout time.Duration) (*http.Client, error) {
	// Validate the proxy early so the worker can fall back to a direct
	// connection on a malformed proxy string (same behavior as before).
	if _, err := proxyFunc(proxy); err != nil {
		return nil, err
	}

	// A per-worker cookie jar. Never share a global jar between workers.
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &http.Client{
		Transport: newChromeTransport(proxy, timeout),
		Jar:       jar,
		Timeout:   timeout,
	}, nil
}
