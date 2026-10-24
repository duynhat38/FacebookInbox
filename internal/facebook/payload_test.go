package facebook

import (
	"sort"
	"strings"
	"testing"
)

func keysOf(m map[string]interface{}) string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return strings.Join(ks, ",")
}

// TestBuildSharePayloadMatchesCapture locks the send_type=6 URL-share payload to
// exactly the fields of the captured working browser request (Reqable record 6002,
// 2026-10-07): a plain "url" field, no XMA/url_attachment. thread_id must be an
// int64 (not string) to match the capture.
func TestBuildSharePayloadMatchesCapture(t *testing.T) {
	p := buildSharePayload("1285261987995134", "ahihi", "https://www.facebook.com/share/p/1RoHerxpsu/")

	const wantTop = "initiating_source,mark_thread_read,otid,send_type,source,sync_group,text,thread_id,url"
	if got := keysOf(p); got != sortedCSV(wantTop) {
		t.Fatalf("top-level keys = %q, want %q", got, sortedCSV(wantTop))
	}
	if p["send_type"] != messengerSendShare {
		t.Fatalf("send_type = %v, want %d", p["send_type"], messengerSendShare)
	}
	if p["thread_id"] != int64(1285261987995134) {
		t.Fatalf("thread_id = %v (%T), want int64 1285261987995134", p["thread_id"], p["thread_id"])
	}
	if p["url"] != "https://www.facebook.com/share/p/1RoHerxpsu/" {
		t.Fatalf("url = %v", p["url"])
	}
}

func sortedCSV(csv string) string {
	ks := strings.Split(csv, ",")
	sort.Strings(ks)
	return strings.Join(ks, ",")
}
