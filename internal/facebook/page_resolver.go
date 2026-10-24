package facebook

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// resolveBulkRoute calls POST /ajax/bulk-route-definitions/ to convert a page_id
// into the correct Messenger thread_key. This is the same API Facebook calls when
// the user clicks "Message" on a Page. Faster and more accurate than HTML scraping.
// Returns (threadKey, true) on API success (even if threadKey == rawID).
// Returns (rawID, false) on any network/parse error.
func resolveBulkRoute(ctx context.Context, client *http.Client, cookie string, tok messengerTokens, rawID string) (string, bool) {
	routeKey := "/messages/t/" + rawID
	form := url.Values{
		"route_urls[0]":     {routeKey},
		"routing_namespace": {"fb_comet"},
		"av":                {tok.uid},
		"__user":            {tok.uid},
		"__a":               {"1"},
		"__req":             {tok.req()},
		"fb_dtsg":           {tok.dtsg},
		"lsd":               {tok.lsd},
		"locale":            {tok.locale},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.facebook.com/ajax/bulk-route-definitions/",
		strings.NewReader(form.Encode()))
	if err != nil {
		log.Printf("messenger: bulkRoute: build request failed for %s: %v", rawID, err)
		return rawID, false
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
	req.Header.Set("Origin", "https://www.facebook.com")
	req.Header.Set("Referer", "https://www.facebook.com/")
	req.Header.Set("x-fb-lsd", tok.lsd)
	req.Header.Set("x-asbd-id", "359341")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("messenger: bulkRoute: request failed for %s: %v", rawID, err)
		return rawID, false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20)) // 4 MB — response includes JS resource refs
	log.Printf("messenger: bulkRoute HTTP %d for %s: %.300s", resp.StatusCode, rawID, raw)

	if resp.StatusCode != http.StatusOK {
		return rawID, false
	}

	// Strip for (;;); anti-hijacking prefix
	body := strings.TrimPrefix(string(raw), "for (;;);")
	var result struct {
		Payload struct {
			Payloads map[string]struct {
				Error  bool `json:"error"`
				Result *struct {
					Type    string `json:"type"`
					Exports *struct {
						RootView *struct {
							Props *struct {
								ThreadKey string `json:"thread_key"`
							} `json:"props"`
						} `json:"rootView"`
					} `json:"exports"`
					RedirectURL string `json:"redirect_url"`
				} `json:"result"`
			} `json:"payloads"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		log.Printf("messenger: bulkRoute: parse failed for %s: %v", rawID, err)
		return rawID, false
	}

	entry, ok := result.Payload.Payloads[routeKey]
	if !ok || entry.Error || entry.Result == nil {
		log.Printf("messenger: bulkRoute: no valid entry for %s", rawID)
		return rawID, false
	}

	// type=route_definition → thread_key directly
	if entry.Result.Exports != nil &&
		entry.Result.Exports.RootView != nil &&
		entry.Result.Exports.RootView.Props != nil &&
		entry.Result.Exports.RootView.Props.ThreadKey != "" {
		tk := entry.Result.Exports.RootView.Props.ThreadKey
		log.Printf("messenger: bulkRoute: %s -> %s", rawID, tk)
		return tk, true
	}

	log.Printf("messenger: bulkRoute: no thread_key in response for %s (type=%s)", rawID, entry.Result.Type)
	return rawID, false
}

// pageResolverPatterns converts raw page_id → actual Messenger message_box_id.
// Since Facebook merged Pages into Profiles, raw page_id ≠ message thread_id
// for many pages. Patterns tried in order, most specific first.
var pageResolverPatterns = []*regexp.Regexp{
	regexp.MustCompile(`"message_box_id"\s*:\s*"(\d+)"`),
	regexp.MustCompile(`"(?:profile_)?delegate_page_id"\s*:\s*"(\d+)"`),
	regexp.MustCompile(`"(?:profile_)?delegate_page"\s*:\s*\{[^}]*"id"\s*:\s*"(\d+)"`),
	regexp.MustCompile(`"pageMessengerInboxThreadKey"\s*:\s*"(\d+)"`),
	regexp.MustCompile(`"pageMessengerThreadKey"\s*:\s*"(\d+)"`),
	regexp.MustCompile(`"thread_key_str"\s*:\s*"(\d+)"`),
}

// reMessagesThreadID extracts the thread ID from a /messages/t/<id>/ URL.
var reMessagesThreadID = regexp.MustCompile(`/messages/t/(\d+)`)

// resolvePageID fetches facebook.com/profile.php?id={rawID} and extracts the
// actual Messenger thread ID. Returns rawID unchanged if resolution fails or
// no pattern matches. Errors are logged but not propagated (PageResolver
// best-effort: direct user/group threads may not need resolution at all).
//
// Mirrors Python PageResolver.resolve() with full debug logging.
func resolvePageID(ctx context.Context, client *http.Client, cookie string, tok messengerTokens, rawID string) string {
	page, _, err := fbGetWithFinalURL(ctx, client, cookie, "https://www.facebook.com/profile.php?id="+rawID, tok.profile, "same-origin")
	if err != nil {
		log.Printf("messenger: PageResolver: GET profile.php for %s failed: %v (using raw)", rawID, err)
		return rawID
	}
	log.Printf("messenger: PageResolver: raw_id=%s, HTML %d chars", rawID, len(page))

	// Mirrors Python: early exit if raw_id not in page at all — avoids
	// false-positive pattern matches on unrelated pages.
	if !strings.Contains(page, rawID) {
		log.Printf("messenger: PageResolver: raw_id=%s not found in page body, returning unchanged", rawID)
		return rawID
	}

	viewerID := firstMatch(reCUser, cookie)

	for _, re := range pageResolverPatterns {
		found := re.FindAllStringSubmatchIndex(page, -1)
		if len(found) == 0 {
			log.Printf("messenger: PageResolver: pattern %q — no match", re.String())
			continue
		}

		// Filter out viewer's own ID
		var candidates [][]int
		for _, loc := range found {
			id := page[loc[2]:loc[3]]
			if id != viewerID {
				candidates = append(candidates, loc)
			}
		}
		if len(candidates) == 0 {
			log.Printf("messenger: PageResolver: pattern %q — only viewer_id matches, skip", re.String())
			continue
		}

		first := candidates[0]
		resolvedID := page[first[2]:first[3]]
		matchOffset := first[0]

		// Find nearest occurrence of rawID in page (mirrors Python id_positions logic)
		nearestGap := -1
		searchPos := 0
		for {
			pos := strings.Index(page[searchPos:], rawID)
			if pos < 0 {
				break
			}
			absPos := searchPos + pos
			gap := matchOffset - absPos
			if gap < 0 {
				gap = -gap
			}
			if nearestGap < 0 || gap < nearestGap {
				nearestGap = gap
			}
			searchPos = absPos + len(rawID)
		}

		// Context: 80 chars before/after match
		ctxStart := matchOffset - 80
		if ctxStart < 0 {
			ctxStart = 0
		}
		ctxEnd := matchOffset + 80
		if ctxEnd > len(page) {
			ctxEnd = len(page)
		}

		log.Printf("messenger: PageResolver: pattern %q matched %d time(s), first=%s @ offset=%d (nearest gap to raw_id: %d) — ctx: %q",
			re.String(), len(found), resolvedID, matchOffset, nearestGap, page[ctxStart:ctxEnd])
		log.Printf("messenger: PageResolver: %s -> %s", rawID, resolvedID)
		return resolvedID
	}

	// Fallback: GET /messages/t/<rawID> — the redirect final URL or HTML
	// contains the actual Messenger thread_id for pages that don't embed it
	// in their profile HTML.
	log.Printf("messenger: PageResolver: %s — all HTML patterns failed, trying /messages/t/ fallback", rawID)
	messagesURL := "https://www.facebook.com/messages/t/" + rawID
	body, finalURL, err := fbGetWithFinalURL(ctx, client, cookie, messagesURL, tok.profile, "same-origin")
	if err != nil {
		log.Printf("messenger: PageResolver: /messages/t/ GET failed for %s: %v", rawID, err)
		log.Printf("messenger: PageResolver: %s unchanged", rawID)
		return rawID
	}
	log.Printf("messenger: PageResolver: /messages/t/ finalURL=%s", finalURL)

	// 1. Try final URL path: /messages/t/<thread_id>/
	if m := reMessagesThreadID.FindStringSubmatch(finalURL); m != nil && m[1] != rawID {
		log.Printf("messenger: PageResolver: %s -> %s (from redirect URL)", rawID, m[1])
		return m[1]
	}

	// 2. Try HTML patterns on the /messages/t/ response
	if !strings.Contains(body, rawID) {
		log.Printf("messenger: PageResolver: %s not in /messages/t/ body, unchanged", rawID)
		return rawID
	}
	for _, re := range pageResolverPatterns {
		found := re.FindAllStringSubmatchIndex(body, -1)
		if len(found) == 0 {
			continue
		}
		for _, loc := range found {
			id := body[loc[2]:loc[3]]
			if id != viewerID && id != rawID {
				log.Printf("messenger: PageResolver: %s -> %s (from /messages/t/ HTML, pattern %q)", rawID, id, re.String())
				return id
			}
		}
	}

	log.Printf("messenger: PageResolver: %s unchanged (no pattern matched in /messages/t/ HTML)", rawID)
	return rawID
}
