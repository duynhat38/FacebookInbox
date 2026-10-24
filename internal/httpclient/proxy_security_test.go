package httpclient

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestProxyCredNotLeakedInError(t *testing.T) {
	const pw = "SUPERSECRETPW"
	c, err := New("http://user:"+pw+"@127.0.0.1:1/", 3*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, "https://www.facebook.com/", nil)
	req.Header.Set("User-Agent", testUA)
	_, err = c.Do(req)
	if err == nil {
		t.Fatal("expected error from dead proxy")
	}
	t.Logf("error text: %v", err)
	if strings.Contains(err.Error(), pw) {
		t.Fatalf("PROXY PASSWORD LEAKED in error: %v", err)
	}
}
