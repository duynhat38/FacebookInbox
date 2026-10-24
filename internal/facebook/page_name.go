package facebook

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// pageNamePlaceholder is replaced in message/share text with the target page's
// display name. Users type it in content or the share caption.
const pageNamePlaceholder = "{namepage}"

// rePageName extracts the page display name from the profile_header_renderer
// block Facebook embeds in the page HTML (facebook.com/<page_id>):
//
//	"path":["user","profile_header_renderer","user"],"data":{"name":"<name>"
//
// The capture allows backslash escapes (\", \uXXXX) so a name is matched whole;
// it is JSON-decoded afterwards. note: ceiling — a name containing an un-escaped
// '"' (never valid JSON) would truncate; page names in practice don't.
var rePageName = regexp.MustCompile(
	`"path":\["user","profile_header_renderer","user"\],"data":\{"name":"((?:[^"\\]|\\.)*)"`)

// fetchPageName GETs facebook.com/<pageID> and returns the page's display name,
// or "" if the page can't be loaded or the name isn't found. pageID is the raw
// target page id (not the resolved Messenger thread id). Best-effort: all errors
// are logged, never propagated — a missing name just leaves the placeholder.
func fetchPageName(ctx context.Context, client *http.Client, cookie string, tok messengerTokens, pageID string) string {
	page, _, err := fbGetWithFinalURL(ctx, client, cookie, "https://www.facebook.com/"+pageID, tok.profile, "same-origin")
	if err != nil {
		log.Printf("messenger: fetchPageName: GET for %s failed: %v", pageID, err)
		return ""
	}
	m := rePageName.FindStringSubmatch(page)
	if m == nil {
		log.Printf("messenger: fetchPageName: name not found in page %s (%d chars)", pageID, len(page))
		return ""
	}
	// JSON-decode the captured value so \uXXXX / \" / \/ escapes become real chars.
	var name string
	if err := json.Unmarshal([]byte(`"`+m[1]+`"`), &name); err != nil {
		name = m[1] // fall back to the raw capture if decoding fails
	}
	name = strings.TrimSpace(name)
	log.Printf("messenger: fetchPageName: %s -> %q", pageID, name)
	return name
}
