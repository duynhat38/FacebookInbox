package facebook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	messengerAppID      = "2220391788200892"
	messengerVersionID  = "28563113493286879"
	messengerTaskLabel  = "46"
	messengerSendText   = 1
	messengerSendAttach = 3
	messengerSendShare  = 6 // URL post share — plain "url" field, DGW (capture 2026-10-07)
)

// ── Token struct ──────────────────────────────────────────────────────────────

type messengerTokens struct {
	uid            string
	dtsg           string
	lsd            string
	clientRevision string
	hasteSession   string
	dyn            string
	csr            string
	deviceID       string
	profile        browserProfile // deterministic per uid — same account always same UA
	reqSeq         string         // __req form param; "" → use "1" as fallback
	locale         string         // account locale e.g. "en_US"; fallback "en_US"
}

// GraphQLHost is the Facebook hostname used for /api/graphql/ calls.
// Edit this to switch to a locale-specific domain, e.g. "en-gb.facebook.com".
// Default: "www.facebook.com"
var GraphQLHost = "en-gb.facebook.com"

// req returns the __req value for AJAX form params, defaulting to "1" when the
// session counter has not yet been set (e.g. during token extraction or direct calls).
func (t messengerTokens) req() string {
	if t.reqSeq == "" {
		return "1"
	}
	return t.reqSeq
}

// graphqlURL returns the full URL for Facebook's GraphQL endpoint using GraphQLHost.
func (t messengerTokens) graphqlURL() string {
	return "https://" + GraphQLHost + "/api/graphql/"
}

// ── Regexps ───────────────────────────────────────────────────────────────────

var (
	reLSD                  = regexp.MustCompile(`"LSD"[^}]*?"token":"([^"]+)"`)
	reDTSG                 = regexp.MustCompile(`"DTSGInitData"[^}]*?"token":"([^"]+)"`)
	reClientRevision       = regexp.MustCompile(`"client_revision"\s*:\s*(\d+)`)
	reHasteSession         = regexp.MustCompile(`"haste_session"\s*:\s*"([^"]+)"`)
	reDyn                  = regexp.MustCompile(`"__dyn"\s*:\s*"([^"]+)"`)
	reCSR                  = regexp.MustCompile(`"__csr"\s*:\s*"([^"]+)"`)
	reDeviceID             = regexp.MustCompile(`"(?:deviceId|clientID)"\s*:\s*"([^"]+)"`)
	reLocale               = regexp.MustCompile(`(?:\\"|")locale(?:\\"|")\s*:\s*(?:\\"|")([a-z]{2}_[A-Z]{2})`)
	reCUser                = regexp.MustCompile(`c_user=(\d+)`)
	reMarkFailed           = regexp.MustCompile(`"markOptimisticMessageFailed","[^"]*","([^"]+)"`)
	reTaskCompletionFailed = regexp.MustCompile(`mailboxTaskCompletionApiOnTaskCompletion.*?\],false`)
	// replaceOptimsiticMessage (FB typo: optimsitic not optimistic) returns: ["replaceOptimsiticMessage","<otid>","mid.$..."]
	reMessageIDReplace = regexp.MustCompile(`"replaceOptimsiticMessage","[^"]*","(mid\.[^"]+)"`)
	// lsInsertMessage returns: ["lsInsertMessage","mid.$...","<otid>",...]
	reMessageIDInsert = regexp.MustCompile(`"lsInsertMessage","(mid\.[^"]+)"`)
)

// firstMatch returns the first capturing group match or "".
func firstMatch(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1]
}

// ── extractTokens (mirrors Python TokenExtractor.extract) ────────────────────

// extractTokens GETs facebook.com/ and scrapes the JS tokens needed for
// upload and Lightspeed API calls.
func extractTokens(ctx context.Context, client *http.Client, cookie string, profileIdx int) (messengerTokens, error) {
	uid := firstMatch(reCUser, cookie)
	profile := profileForIndex(profileIdx)
	log.Printf("messenger: extractTokens: fetching facebook.com for c_user=%s ua=%.50s", uid, profile.userAgent)

	// Pass "" so the http.Client's jar (seeded in messengerSession.init) sends
	// cookies automatically and captures any Set-Cookie updates from this response.
	page, _, err := fbGetWithFinalURL(ctx, client, "", "https://www.facebook.com/", profile, "none")
	if err != nil {
		return messengerTokens{}, fmt.Errorf("messenger: fetch facebook.com: %w", err)
	}
	log.Printf("messenger: extractTokens: page loaded %d chars", len(page))

	// Detect checkpoint before extracting any tokens
	if strings.Contains(page, "/checkpoint/") && strings.Contains(page, "ServerRedirect") {
		return messengerTokens{}, fmt.Errorf("messenger: tài khoản %s bị checkpoint — đăng nhập lại và xác minh để lấy cookie mới", uid)
	}

	lsd := firstMatch(reLSD, page)
	if lsd == "" {
		preview := page
		if len(preview) > 300 {
			preview = preview[:300]
		}
		log.Printf("messenger: LSD not found. Page preview: %s", preview)
		return messengerTokens{}, errors.New("messenger: LSD token not found — cookie expired or blocked")
	}
	log.Printf("messenger: LSD: %s", lsd)

	dtsg, err := extractDTSG(ctx, client, cookie, page, lsd, profile)
	if err != nil {
		return messengerTokens{}, err
	}

	clientRevision := firstMatch(reClientRevision, page)
	if clientRevision == "" {
		return messengerTokens{}, errors.New("messenger: client_revision not found — cookie may be expired")
	}

	hasteSession := firstMatch(reHasteSession, page)
	if hasteSession == "" {
		log.Printf("messenger: haste_session not found, upload may fail")
	}

	deviceID := firstMatch(reDeviceID, page)
	if deviceID == "" {
		deviceID = strconv.Itoa(msgrRandInt(100_000_000, 999_999_999))
		log.Printf("messenger: deviceId not found in page, generated: %s", deviceID)
	}

	locale := firstMatch(reLocale, page)
	if locale == "" {
		locale = "en_US"
		log.Printf("messenger: locale not found in page, defaulting to en_US")
	} else {
		log.Printf("messenger: locale: %s", locale)
	}

	return messengerTokens{
		uid:            uid,
		dtsg:           dtsg,
		lsd:            lsd,
		clientRevision: clientRevision,
		hasteSession:   hasteSession,
		dyn:            firstMatch(reDyn, page),
		csr:            firstMatch(reCSR, page),
		deviceID:       deviceID,
		profile:        profile,
		locale:         locale,
	}, nil
}

// extractDTSG mirrors Python TokenExtractor._extract_dtsg: tries DTSGInitData
// in page first, falls back to async /ajax/dtsg/ if missing or equals LSD.
func extractDTSG(ctx context.Context, client *http.Client, cookie, page, lsd string, p browserProfile) (string, error) {
	dtsg := firstMatch(reDTSG, page)
	if dtsg == "" || dtsg == lsd {
		log.Printf("messenger: DTSGInitData empty or bled through to LSD, trying async /ajax/dtsg/")
		return fetchDTSGAsync(ctx, client, cookie, lsd, p)
	}
	log.Printf("messenger: DTSG: %.20s...", dtsg)
	return dtsg, nil
}

// fetchDTSGAsync POSTs /ajax/dtsg/ to get the DTSG token asynchronously.
// Mirrors Python TokenExtractor._fetch_dtsg_async().
func fetchDTSGAsync(ctx context.Context, client *http.Client, cookie, lsd string, p browserProfile) (string, error) {
	form := url.Values{"lsd": {lsd}, "__a": {"1"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.facebook.com/ajax/dtsg/",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	p.setHeaders(req)
	p.setDeviceHints(req)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("x-fb-lsd", lsd)
	req.Header.Set("Referer", "https://www.facebook.com/")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Origin", "https://www.facebook.com")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("messenger: dtsg fetch: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))

	data, err := parseFBJSON(string(raw))
	if err != nil {
		return "", errors.New("messenger: /ajax/dtsg/ did not return JSON")
	}
	payload, _ := data["payload"].(map[string]interface{})
	token, _ := payload["token"].(string)
	if token == "" {
		log.Printf("messenger: async DTSG returned no token. Response: %.300s", raw)
		return "", errors.New("messenger: /ajax/dtsg/ returned no token")
	}
	log.Printf("messenger: DTSG from async: %.20s...", token)
	return token, nil
}
