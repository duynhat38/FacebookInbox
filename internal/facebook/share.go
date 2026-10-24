package facebook

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const docIDCreateShareURL = "30568280579452205" // useLinkSharingCreateWrappedUrlMutation

// createWrappedURL calls useLinkSharingCreateWrappedUrlMutation to convert an
// original Facebook post URL into a short /share/p/... wrapped URL.
// Per capture 2026-10-06: Facebook always creates this before attaching to message.
// Returns wrapped_url on success, originalURL unchanged on any error.
func createWrappedURL(ctx context.Context, client *http.Client, cookie string, tok messengerTokens, originalURL string) string {
	variables, _ := json.Marshal(map[string]interface{}{
		"input": map[string]string{
			"actor_id":             tok.uid,
			"client_mutation_id":   "1",
			"original_content_url": originalURL,
			"product_type":         "FEED",
		},
	})
	form := url.Values{
		"av":                       {tok.uid},
		"__user":                   {tok.uid},
		"__req":                    {tok.req()},
		"fb_api_caller_class":      {"RelayModern"},
		"fb_api_req_friendly_name": {"useLinkSharingCreateWrappedUrlMutation"},
		"fb_dtsg":                  {tok.dtsg},
		"lsd":                      {tok.lsd},
		"variables":                {string(variables)},
		"doc_id":                   {docIDCreateShareURL},
		"server_timestamps":        {"true"},
		"locale":                   {tok.locale},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		tok.graphqlURL(),
		strings.NewReader(form.Encode()))
	if err != nil {
		log.Printf("messenger: createWrappedURL: build failed: %v", err)
		return originalURL
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	tok.profile.setHeaders(req)
	tok.profile.setDeviceHints(req)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("x-fb-friendly-name", "useLinkSharingCreateWrappedUrlMutation")
	req.Header.Set("x-fb-lsd", tok.lsd)
	req.Header.Set("x-asbd-id", "359341")
	req.Header.Set("Origin", "https://www.facebook.com")
	req.Header.Set("Referer", "https://www.facebook.com/")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("messenger: createWrappedURL: request failed: %v", err)
		return originalURL
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	log.Printf("messenger: createWrappedURL HTTP %d: %.1500s", resp.StatusCode, raw)

	if resp.StatusCode != http.StatusOK {
		return originalURL
	}

	// Strip the for(;;); anti-hijacking prefix Facebook prepends to AJAX JSON responses.
	body := strings.TrimPrefix(string(raw), "for (;;);")

	var result struct {
		Data struct {
			CreateShareURLWrapper struct {
				ShareURLWrapper struct {
					WrappedURL string `json:"wrapped_url"`
				} `json:"share_url_wrapper"`
			} `json:"xfb_create_share_url_wrapper"`
		} `json:"data"`
	}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&result); err != nil {
		log.Printf("messenger: createWrappedURL: parse failed: %v", err)
		return originalURL
	}
	wrapped := result.Data.CreateShareURLWrapper.ShareURLWrapper.WrappedURL
	if wrapped == "" {
		log.Printf("messenger: createWrappedURL: no wrapped_url in response")
		return originalURL
	}
	log.Printf("messenger: createWrappedURL: %s -> %s", originalURL, wrapped)
	return wrapped
}
