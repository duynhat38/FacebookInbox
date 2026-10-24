package facebook

import (
	"hash/fnv"
	"net/http"
	"strings"
)

// browserProfile represents a real browser fingerprint (UA + client hints).
// All profiles are desktop-only (?0 for mobile) — matches Messenger web app usage.
type browserProfile struct {
	userAgent            string
	secChUA              string // Sec-Ch-Ua
	secChUAFullVersion   string // Sec-Ch-Ua-Full-Version-List
	platform             string // Sec-Ch-Ua-Platform  (e.g. `"Windows"`)
	platformVersion      string // Sec-Ch-Ua-Platform-Version (e.g. `"19.0.0"`)
	dpr                  string // device pixel ratio (e.g. "1", "2")
	viewportWidth        string // viewport width in CSS px (e.g. "1440")
	prefersColorScheme   string // "dark" | "light"; empty = derived from UA hash (~50/50)
}

// setHeaders applies UA + client hint headers to req.
// Call this on every HTTP request that impersonates a browser.
// Does NOT include Dpr/Viewport-Width — call setDeviceHints for HTTP requests
// where Facebook has signalled it wants those hints (all API and nav requests).
// Not called for WebSocket upgrades which don't carry these device metrics.
func (p browserProfile) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Sec-Ch-Ua", p.secChUA)
	req.Header.Set("Sec-Ch-Ua-Full-Version-List", p.secChUAFullVersion)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Model", `""`)
	req.Header.Set("Sec-Ch-Ua-Platform", p.platform)
	req.Header.Set("Sec-Ch-Ua-Platform-Version", p.platformVersion)
	req.Header.Set("Sec-Ch-Prefers-Color-Scheme", p.colorScheme())
}

// colorScheme returns "dark" or "light" for this profile.
// When prefersColorScheme is explicitly set, returns it directly.
// Otherwise derives from the UA string via FNV-32a hash — deterministic per
// profile, distributes ~50/50 across the pool so not all accounts look identical.
func (p browserProfile) colorScheme() string {
	if p.prefersColorScheme != "" {
		return p.prefersColorScheme
	}
	h := fnv.New32a()
	h.Write([]byte(p.userAgent))
	if h.Sum32()%2 == 0 {
		return "dark"
	}
	return "light"
}

// setDeviceHints adds Dpr and Viewport-Width — the low-entropy device metrics
// Facebook requests via Accept-CH. Call on every HTTP request (navigation and
// fetch/XHR) but NOT on WebSocket upgrades.
// Falls back to sensible defaults when the profile fields are empty:
//   - macOS → DPR 2 (Retina), Windows/other → DPR 1
//   - viewport 1440 CSS px (most common 1440×900 / 1440×1024 desktop size)
func (p browserProfile) setDeviceHints(req *http.Request) {
	dpr := p.dpr
	if dpr == "" {
		if strings.Contains(p.platform, "macOS") {
			dpr = "2" // Retina default
		} else {
			dpr = "1" // standard Windows DPI
		}
	}
	vw := p.viewportWidth
	if vw == "" {
		vw = "1440"
	}
	req.Header.Set("Dpr", dpr)
	req.Header.Set("Viewport-Width", vw)
}

// browserProfiles is the pool of Chrome/Edge versions on Windows and macOS.
// Pool size = 24 → with FNV hash, different UIDs spread evenly across profiles.
// Chrome 128-154 covers ~2 years of stable releases.
// Edge 130-133 adds a distinct browser brand for further diversity.
// note: update this list periodically as newer Chrome stable versions ship.
var browserProfiles = []browserProfile{
	// ── Chrome × Windows ──────────────────────────────────────────────────────
	{ // Chrome 154 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="154", "Google Chrome";v="154", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="154.0.8037.98", "Google Chrome";v="154.0.8037.98", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 136 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="136", "Not(A:Brand";v="8", "Google Chrome";v="136"`,
		secChUAFullVersion: `"Chromium";v="136.0.7103.92", "Not(A:Brand";v="8.0.0.0", "Google Chrome";v="136.0.7103.92"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 135 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="135", "Chromium";v="135", "Not/A_Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="135.0.7049.114", "Chromium";v="135.0.7049.114", "Not/A_Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 134 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="134", "Google Chrome";v="134", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="134.0.6998.165", "Google Chrome";v="134.0.6998.165", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 133 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="133", "Chromium";v="133", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="133.0.6943.141", "Chromium";v="133.0.6943.141", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 132 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="132", "Not(A:Brand";v="24", "Google Chrome";v="132"`,
		secChUAFullVersion: `"Chromium";v="132.0.6834.159", "Not(A:Brand";v="24.0.0.0", "Google Chrome";v="132.0.6834.159"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
	},
	{ // Chrome 131 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="131.0.6778.205", "Chromium";v="131.0.6778.205", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
	},
	{ // Chrome 130 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="130.0.6723.191", "Google Chrome";v="130.0.6723.191", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
	},
	{ // Chrome 129 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="129", "Not=A?Brand";v="8", "Chromium";v="129"`,
		secChUAFullVersion: `"Google Chrome";v="129.0.6668.100", "Not=A?Brand";v="8.0.0.0", "Chromium";v="129.0.6668.100"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
	},
	{ // Chrome 128 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="128", "Not;A=Brand";v="24", "Google Chrome";v="128"`,
		secChUAFullVersion: `"Chromium";v="128.0.6613.120", "Not;A=Brand";v="24.0.0.0", "Google Chrome";v="128.0.6613.120"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
	},

	// ── Chrome × macOS ────────────────────────────────────────────────────────
	{ // Chrome 154 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="154", "Google Chrome";v="154", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="154.0.8037.98", "Google Chrome";v="154.0.8037.98", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
	},
	{ // Chrome 136 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="136", "Not(A:Brand";v="8", "Google Chrome";v="136"`,
		secChUAFullVersion: `"Chromium";v="136.0.7103.92", "Not(A:Brand";v="8.0.0.0", "Google Chrome";v="136.0.7103.92"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
	},
	{ // Chrome 135 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="135", "Chromium";v="135", "Not/A_Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="135.0.7049.114", "Chromium";v="135.0.7049.114", "Not/A_Brand";v="24.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.5.0"`,
	},
	{ // Chrome 134 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="134", "Google Chrome";v="134", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="134.0.6998.165", "Google Chrome";v="134.0.6998.165", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.3.0"`,
	},
	{ // Chrome 133 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="133", "Chromium";v="133", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="133.0.6943.141", "Chromium";v="133.0.6943.141", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.2.0"`,
	},
	{ // Chrome 132 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="132", "Not(A:Brand";v="24", "Google Chrome";v="132"`,
		secChUAFullVersion: `"Chromium";v="132.0.6834.159", "Not(A:Brand";v="24.0.0.0", "Google Chrome";v="132.0.6834.159"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.1.0"`,
	},
	{ // Chrome 131 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="131.0.6778.205", "Chromium";v="131.0.6778.205", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"13.7.0"`,
	},
	{ // Chrome 130 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="130.0.6723.191", "Google Chrome";v="130.0.6723.191", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"13.6.0"`,
	},
	{ // Chrome 129 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="129", "Not=A?Brand";v="8", "Chromium";v="129"`,
		secChUAFullVersion: `"Google Chrome";v="129.0.6668.100", "Not=A?Brand";v="8.0.0.0", "Chromium";v="129.0.6668.100"`,
		platform:           `"macOS"`,
		platformVersion:    `"13.5.0"`,
	},
	{ // Chrome 128 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="128", "Not;A=Brand";v="24", "Google Chrome";v="128"`,
		secChUAFullVersion: `"Chromium";v="128.0.6613.120", "Not;A=Brand";v="24.0.0.0", "Google Chrome";v="128.0.6613.120"`,
		platform:           `"macOS"`,
		platformVersion:    `"13.4.0"`,
	},

	// ── Microsoft Edge × Windows (Chromium-based, distinct brand) ────────────
	{ // Edge 133 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 Edg/133.0.0.0",
		secChUA:            `"Microsoft Edge";v="133", "Chromium";v="133", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Microsoft Edge";v="133.0.3065.92", "Chromium";v="133.0.6943.141", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 132 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0",
		secChUA:            `"Microsoft Edge";v="132", "Chromium";v="132", "Not(A:Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="132.0.2957.140", "Chromium";v="132.0.6834.159", "Not(A:Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
	},
	{ // Edge 131 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
		secChUA:            `"Microsoft Edge";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="131.0.2903.86", "Chromium";v="131.0.6778.205", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
	},
	{ // Edge 130 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
		secChUA:            `"Microsoft Edge";v="130", "Chromium";v="130", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Microsoft Edge";v="130.0.2849.80", "Chromium";v="130.0.6723.191", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
	},
}

// profileForUID picks a browser profile deterministically from the account UID.
// Same UID → same profile (stable per account); different UIDs → different profiles.
// FNV-1a 64-bit: cheap and well-distributed for short numeric strings.
func profileForUID(uid string) browserProfile {
	h := fnv.New64a()
	h.Write([]byte(uid))
	return browserProfiles[h.Sum64()%uint64(len(browserProfiles))]
}

// profileFromCookie extracts c_user from cookie and returns the matching profile.
// Used by code paths that don't yet have messengerTokens (e.g. live check).
func profileFromCookie(cookie string) browserProfile {
	uid := firstMatch(reCUser, cookie)
	return profileForUID(uid)
}
