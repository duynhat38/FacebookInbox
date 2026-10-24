package facebook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// docIDClickMessenger is the persisted GraphQL query that registers the account
// as a messenger_business_ads_sender for a page — the request Facebook fires when
// the user clicks "Message" on a page's ad entry point. doc_id values rotate when
// Facebook redeploys, so it is kept as a single named constant to update in one place.
const docIDClickMessenger = "6799089560137001"

// jazoest derives the jazoest form param from fb_dtsg the same way Facebook's JS
// does: the literal "2" followed by the sum of the UTF-16 code units of the token.
// For the captured dtsg this yields "25623", matching the sample request.
func jazoest(dtsg string) string {
	sum := 0
	for _, r := range dtsg {
		sum += int(r)
	}
	return "2" + strconv.Itoa(sum)
}

// clickMessenger registers the account (actor_id = uid) as the sender for pageID
// via the messenger ads-click GraphQL mutation. It must run before the first
// message so Facebook accepts the thread as an ad-initiated conversation.
// pageID is the raw target page id (not the resolved Messenger thread_id).
// Returns nil only when the response carries messenger_business_ads_sender.
func clickMessenger(ctx context.Context, client *http.Client, cookie string, tok messengerTokens, pageID string) error {
	variables, _ := json.Marshal(map[string]interface{}{
		"input": map[string]interface{}{
			"page_id":            pageID,
			"post_id":            nil,
			"actor_id":           tok.uid,
			"client_mutation_id": "1",
		},
	})
	spin := strconv.FormatInt(time.Now().Unix(), 10)
	form := url.Values{
		"av":                       {tok.uid},
		"__user":                   {tok.uid},
		"__a":                      {"1"},
		"__req":                    {tok.req()},
		"dpr":                      {"1"},
		"__ccg":                    {"EXCELLENT"},
		"__comet_req":              {"1"},
		"__rev":                    {tok.clientRevision},
		"__spin_r":                 {tok.clientRevision},
		"__spin_b":                 {"trunk"},
		"__spin_t":                 {spin},
		"__hs":                     {tok.hasteSession},
		"__dyn":                    {tok.dyn},
		"__csr":                    {tok.csr},
		"fb_dtsg":                  {tok.dtsg},
		"jazoest":                  {jazoest(tok.dtsg)},
		"lsd":                      {tok.lsd},
		"fb_api_caller_class":      {"RelayModern"},
		"fb_api_req_friendly_name": {"MWChatBusinessCTAAdsSenderMutation"},
		"variables":                {string(variables)},
		"doc_id":                   {docIDClickMessenger},
		"locale":                   {tok.locale},
		"server_timestamps":        {"true"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		tok.graphqlURL(),
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("messenger: clickMessenger: build request: %w", err)
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
	req.Header.Set("x-fb-lsd", tok.lsd)
	req.Header.Set("x-fb-friendly-name", "MWChatBusinessCTAAdsSenderMutation")
	req.Header.Set("x-asbd-id", "359341")
	req.Header.Set("Origin", "https://www.facebook.com")
	req.Header.Set("Referer", "https://www.facebook.com/")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("messenger: clickMessenger: request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	log.Printf("messenger: clickMessenger HTTP %d page=%s: %.1000s", resp.StatusCode, pageID, raw)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("messenger: clickMessenger: HTTP %d", resp.StatusCode)
	}
	return parseClickResponse(raw)
}

// parseClickResponse returns nil only if the GraphQL response contains a
// non-empty data.messenger_business_ads_sender. It strips the for(;;); anti-JSON
// prefix and decodes the first JSON object (GraphQL may stream concatenated
// chunks, which json.Unmarshal rejects). On a Facebook error envelope
// ({"error":...,"errorDescription":...}) it surfaces the human-readable reason
// (e.g. account action-restricted). The error never contains credentials.
func parseClickResponse(raw []byte) error {
	body := strings.TrimPrefix(string(raw), "for (;;);")
	var result struct {
		Data struct {
			Sender string `json:"messenger_business_ads_sender"`
		} `json:"data"`
		ErrorSummary     string `json:"errorSummary"`
		ErrorDescription string `json:"errorDescription"`
	}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&result); err != nil {
		return fmt.Errorf("messenger: clickMessenger: parse response: %w (%s)", err, snippet(body))
	}
	if result.Data.Sender != "" {
		return nil
	}
	// Facebook rejection: report its own message so the user sees why (restricted
	// account, etc.) instead of a raw dump.
	if msg := result.ErrorDescription; msg != "" {
		if result.ErrorSummary != "" {
			msg = result.ErrorSummary + ": " + msg
		}
		return errors.New("messenger: clickMessenger: " + msg)
	}
	if result.ErrorSummary != "" {
		return errors.New("messenger: clickMessenger: " + result.ErrorSummary)
	}
	return errors.New("messenger: clickMessenger: no messenger_business_ads_sender in response (" + snippet(body) + ")")
}

func snippet(s string) string {
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
