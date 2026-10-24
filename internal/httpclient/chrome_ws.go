package httpclient

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	utls "github.com/bogdanfinn/utls"
)

// NewChromeWSDialTLS returns a function suitable for gorilla/websocket's
// Dialer.NetDialTLSContext. It performs the TLS handshake with the SAME Chrome
// fingerprint as the HTTP path (bogdanfinn/utls, profile chosen from the account
// User-Agent); gorilla then runs the HTTP/1.1 Upgrade over the returned conn.
//
// ALPN is forced to http/1.1 only: gorilla/websocket speaks the HTTP/1.1 upgrade
// handshake, so if an edge negotiated h2 the Upgrade reader would parse an h2
// frame as a status line (the exact failure the old direct HTTP path hit). The
// ClientHello is otherwise the full Chrome spec (JA3 hashes extension *types*,
// and the ALPN extension is still present); only the advertised ALPN value is
// pinned to http/1.1 by this gorilla-based client.
//
// Connections are IPv4-only (project-wide tcp4 policy). When proxy != "" the
// handshake runs through an HTTP CONNECT tunnel to that proxy.
//
// Used solely for authorized Facebook/Meta WebSocket connections.
func NewChromeWSDialTLS(ua, proxy string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, _, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
		raw, err := dialTCP4(ctx, proxy, addr)
		if err != nil {
			return nil, err
		}
		spec, err := profileForMajor(chromeMajor(ua)).GetClientHelloSpec()
		if err != nil {
			raw.Close()
			return nil, err
		}
		forceHTTP1ALPN(&spec)
		uconn := utls.UClient(raw, &utls.Config{ServerName: host}, utls.HelloCustom, false, true, true)
		if err := uconn.ApplyPreset(&spec); err != nil {
			raw.Close()
			return nil, err
		}
		if err := uconn.HandshakeContext(ctx); err != nil {
			raw.Close()
			return nil, err
		}
		return uconn, nil
	}
}

// forceHTTP1ALPN rewrites the spec's ALPN extension to advertise only http/1.1.
func forceHTTP1ALPN(spec *utls.ClientHelloSpec) {
	for _, ext := range spec.Extensions {
		if a, ok := ext.(*utls.ALPNExtension); ok {
			a.AlpnProtocols = []string{"http/1.1"}
		}
	}
}

// dialTCP4 opens an IPv4 TCP connection to addr, tunneling through an HTTP
// CONNECT proxy when proxy != "". Mirrors the stdlib transport's CONNECT flow.
//
// Only http proxies are supported for WebSocket (the app uses http proxies
// exclusively; proxyFunc defaults a scheme-less proxy to http). A socks/https
// proxy is rejected with a clear error rather than mis-sent as an HTTP CONNECT.
func dialTCP4(ctx context.Context, proxy, addr string) (net.Conn, error) {
	d := &net.Dialer{}
	if proxy == "" {
		return d.DialContext(ctx, "tcp4", addr)
	}
	pu, err := url.Parse(normalizeProxyURL(proxy))
	if err != nil {
		return nil, err
	}
	if pu.Scheme != "http" {
		return nil, fmt.Errorf("websocket proxy scheme %q not supported (only http)", pu.Scheme)
	}
	conn, err := d.DialContext(ctx, "tcp4", pu.Host)
	if err != nil {
		return nil, err
	}
	// gorilla sets the connection deadline only AFTER this dial func returns, so
	// bound the CONNECT exchange here — otherwise a proxy that accepts then goes
	// silent hangs the worker and Stop can't free it. Cleared before returning.
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl)
	}
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}
	if pu.User != nil {
		user := pu.User.Username()
		pw, _ := pu.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(user + ":" + pw))
		req.Header.Set("Proxy-Authorization", "Basic "+auth)
	}
	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		conn.Close()
		return nil, err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		// note: Status only — never embed the proxy URL (carries credentials).
		return nil, fmt.Errorf("proxy CONNECT to %s failed: %s", addr, resp.Status)
	}
	if br.Buffered() > 0 {
		// The proxy sent bytes past the CONNECT response; those would be the
		// start of the TLS stream and we'd drop them reading from conn directly.
		conn.Close()
		return nil, fmt.Errorf("proxy CONNECT to %s returned %d unexpected buffered bytes", addr, br.Buffered())
	}
	// Hand a deadline-free conn back; utls HandshakeContext and gorilla's own
	// deadline wrapper manage timeouts from here.
	conn.SetDeadline(time.Time{})
	return conn, nil
}
