package httpclient

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const testUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// newTestClient builds the real chromeTransport but injects a tls-client that
// trusts the httptest self-signed cert, so the adapter's stdlib<->fhttp
// conversion (redirects, jar, body, decompression) is exercised end-to-end
// without a network dependency. Everything else is production code.
func newTestClient(t *testing.T) *http.Client {
	t.Helper()
	tr := newChromeTransport("", 10*time.Second)
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithClientProfile(profiles.Chrome_131),
		tls_client.WithNotFollowRedirects(), // stdlib http.Client owns redirects
		tls_client.WithInsecureSkipVerify(), // httptest cert is self-signed
	)
	if err != nil {
		t.Fatalf("build tls-client: %v", err)
	}
	tr.clients[chromeMajor(testUA)] = c
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return &http.Client{Transport: tr, Jar: jar, Timeout: 10 * time.Second}
}

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	// hop 1: set a cookie, then 302 to /final
	mux.HandleFunc("/redir", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "abc123", Path: "/"})
		http.Redirect(w, r, "/final", http.StatusFound)
	})
	// hop 2: only succeeds if the jar carried the cookie from hop 1
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		if ck, err := r.Cookie("sid"); err != nil || ck.Value != "abc123" {
			http.Error(w, "missing cookie", http.StatusBadRequest)
			return
		}
		io.WriteString(w, "FINAL-OK")
	})
	// echoes body length + declared Content-Length
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		io.WriteString(w, "cl="+strconv.FormatInt(r.ContentLength, 10)+" body="+string(body))
	})
	// always gzip-encodes, regardless of Accept-Encoding
	mux.HandleFunc("/gz", func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write([]byte("hello-gzip"))
		zw.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(buf.Bytes())
	})
	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	return srv
}

func do(t *testing.T, client *http.Client, method, url string, body io.Reader) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("User-Agent", testUA)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, string(b)
}

func TestRoundTripRedirectAndJar(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	client := newTestClient(t)

	resp, body := do(t, client, http.MethodGet, srv.URL+"/redir", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (redirect not followed or cookie lost)", resp.StatusCode)
	}
	if body != "FINAL-OK" {
		t.Fatalf("body = %q, want FINAL-OK", body)
	}
	if !strings.HasSuffix(resp.Request.URL.Path, "/final") {
		t.Fatalf("final URL = %q, want .../final (final-URL tracking broken)", resp.Request.URL.String())
	}
}

func TestRoundTripPostContentLength(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	client := newTestClient(t)

	const payload = "fb_dtsg=redacted&lsd=redacted&message=hi"
	resp, body := do(t, client, http.MethodPost, srv.URL+"/echo", strings.NewReader(payload))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	want := "cl=" + strconv.Itoa(len(payload)) + " body=" + payload
	if body != want {
		t.Fatalf("echo = %q, want %q (POST body or Content-Length wrong)", body, want)
	}
}

func TestRoundTripGzipDecoded(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	client := newTestClient(t)

	resp, body := do(t, client, http.MethodGet, srv.URL+"/gz", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if body != "hello-gzip" {
		t.Fatalf("body = %q, want hello-gzip (gzip not decoded)", body)
	}
}

func TestRoundTripDoesNotMutateCallerHeaders(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	client := newTestClient(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/final", nil)
	req.Header.Set("User-Agent", testUA)
	req.Header.Set("X-Custom", "v")
	// pre-seed the cookie the /final handler requires, via the jar is simpler,
	// but here we only care about header integrity, so hit /gz instead.
	req.URL.Path = "/gz"
	before := len(req.Header)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	resp.Body.Close()
	if len(req.Header) != before {
		t.Fatalf("caller header count changed %d -> %d", before, len(req.Header))
	}
	if _, bad := req.Header["Header-Order:"]; bad {
		t.Fatal("fhttp HeaderOrderKey leaked into caller's header map")
	}
}
