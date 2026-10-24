package httpclient

import (
	"fmt"
	"net/http"
	"net/url"
)

// proxyFunc returns an http.Transport proxy function for the given proxy URL.
// An empty proxy string means "no proxy" (direct connection).
func proxyFunc(proxy string) (func(*http.Request) (*url.URL, error), error) {
	if proxy == "" {
		return nil, nil
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return nil, fmt.Errorf("parse proxy %q: %w", proxy, err)
	}
	if u.Scheme == "" {
		// Default to http when the scheme was omitted (e.g. "127.0.0.1:8001").
		u, err = url.Parse("http://" + proxy)
		if err != nil {
			return nil, fmt.Errorf("parse proxy %q: %w", proxy, err)
		}
	}
	return http.ProxyURL(u), nil
}
