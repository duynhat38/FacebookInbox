package httpclient

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	utls "github.com/bogdanfinn/utls"
)

const wsTestUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

func alpnOf(spec *utls.ClientHelloSpec) []string {
	for _, e := range spec.Extensions {
		if a, ok := e.(*utls.ALPNExtension); ok {
			return a.AlpnProtocols
		}
	}
	return nil
}

// forceHTTP1ALPN must not mutate state shared with the HTTP path: a second spec
// from the same profile must still advertise h2. Guards against a future
// tls-client/utls change that returns cached extension pointers from ToSpec.
func TestForceHTTP1ALPNDoesNotLeak(t *testing.T) {
	p := profileForMajor(131)
	s1, err := p.GetClientHelloSpec()
	if err != nil {
		t.Fatalf("spec 1: %v", err)
	}
	forceHTTP1ALPN(&s1)
	if got := alpnOf(&s1); !slices.Equal(got, []string{"http/1.1"}) {
		t.Fatalf("after force, ALPN = %v, want [http/1.1]", got)
	}
	s2, err := p.GetClientHelloSpec()
	if err != nil {
		t.Fatalf("spec 2: %v", err)
	}
	if got := alpnOf(&s2); !slices.Contains(got, "h2") {
		t.Fatalf("second spec ALPN = %v, lost h2 — mutation leaked across specs", got)
	}
}

func TestDialTCP4RejectsNonHTTPProxy(t *testing.T) {
	_, err := dialTCP4(context.Background(), "socks5://127.0.0.1:1080", "edge-chat.facebook.com:443")
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("err = %v, want 'not supported'", err)
	}
}

// rawListener accepts TCP4 conns and runs handler per conn; closed via cleanup.
func rawListener(t *testing.T, handler func(net.Conn)) string {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handler(c)
		}
	}()
	return ln.Addr().String()
}

// A CONNECT proxy that always rejects with 407 — error must not leak the password.
func TestDialTCP4ProxyAuthErrorNoLeak(t *testing.T) {
	addr := rawListener(t, func(c net.Conn) {
		defer c.Close()
		http.ReadRequest(bufio.NewReader(c))
		io.WriteString(c, "HTTP/1.1 407 Proxy Authentication Required\r\n\r\n")
	})
	const pw = "SUPERSECRETPW"
	_, err := dialTCP4(context.Background(), "http://user:"+pw+"@"+addr, "edge-chat.facebook.com:443")
	if err == nil {
		t.Fatal("expected CONNECT failure")
	}
	t.Logf("error: %v", err)
	if strings.Contains(err.Error(), pw) {
		t.Fatalf("PROXY PASSWORD LEAKED: %v", err)
	}
	if !strings.Contains(err.Error(), "407") {
		t.Fatalf("error should mention the 407 status: %v", err)
	}
}

// The direct (no-proxy) path: a target that completes TCP but never speaks TLS
// must fail at the ctx deadline, not hang — exercises the utls HandshakeContext.
func TestWSDialDeadlineOnSilentTarget(t *testing.T) {
	addr := rawListener(t, func(c net.Conn) {
		defer c.Close()
		io.Copy(io.Discard, c) // accept TCP, never complete the TLS handshake
	})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := NewChromeWSDialTLS(wsTestUA, "")(ctx, "tcp", addr)
	if err == nil {
		t.Fatal("expected a handshake deadline error")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("handshake hung %v past the 300ms deadline", d)
	}
}

// A proxy that accepts then never replies must not hang past the ctx deadline.
func TestDialTCP4DeadlineOnSilentProxy(t *testing.T) {
	addr := rawListener(t, func(c net.Conn) {
		defer c.Close()
		io.Copy(io.Discard, c) // read forever, never respond
	})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := dialTCP4(ctx, "http://"+addr, "edge-chat.facebook.com:443")
	if err == nil {
		t.Fatal("expected a deadline error from the silent proxy")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("dial hung %v past the 300ms deadline", d)
	}
}
