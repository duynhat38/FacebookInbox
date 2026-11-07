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

	// ── Chrome 137-153 × Windows ──────────────────────────────────────────────
	{ // Chrome 137 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="137", "Chromium";v="137", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="137.0.7151.55", "Chromium";v="137.0.7151.55", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 138 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="138", "Not(A:Brand";v="8", "Google Chrome";v="138"`,
		secChUAFullVersion: `"Chromium";v="138.0.7204.100", "Not(A:Brand";v="8.0.0.0", "Google Chrome";v="138.0.7204.100"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 139 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="139", "Chromium";v="139", "Not/A_Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="139.0.7258.65", "Chromium";v="139.0.7258.65", "Not/A_Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 140 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="140", "Google Chrome";v="140", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="140.0.7321.84", "Google Chrome";v="140.0.7321.84", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 141 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="141", "Chromium";v="141", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="141.0.7395.73", "Chromium";v="141.0.7395.73", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 142 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="142", "Not_A Brand";v="24", "Google Chrome";v="142"`,
		secChUAFullVersion: `"Chromium";v="142.0.7471.61", "Not_A Brand";v="24.0.0.0", "Google Chrome";v="142.0.7471.61"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 143 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="143", "Chromium";v="143", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="143.0.7549.52", "Chromium";v="143.0.7549.52", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 144 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="144", "Not;A=Brand";v="24", "Google Chrome";v="144"`,
		secChUAFullVersion: `"Chromium";v="144.0.7637.45", "Not;A=Brand";v="24.0.0.0", "Google Chrome";v="144.0.7637.45"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 145 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="145", "Not=A?Brand";v="8", "Chromium";v="145"`,
		secChUAFullVersion: `"Google Chrome";v="145.0.7724.58", "Not=A?Brand";v="8.0.0.0", "Chromium";v="145.0.7724.58"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 146 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="146", "Google Chrome";v="146", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="146.0.7821.79", "Google Chrome";v="146.0.7821.79", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 147 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="147", "Chromium";v="147", "Not(A:Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="147.0.7931.55", "Chromium";v="147.0.7931.55", "Not(A:Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 148 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="148", "Not/A_Brand";v="24", "Google Chrome";v="148"`,
		secChUAFullVersion: `"Chromium";v="148.0.8035.88", "Not/A_Brand";v="24.0.0.0", "Google Chrome";v="148.0.8035.88"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 149 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="149", "Chromium";v="149", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="149.0.8127.64", "Chromium";v="149.0.8127.64", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 150 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="150", "Google Chrome";v="150", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Chromium";v="150.0.8230.73", "Google Chrome";v="150.0.8230.73", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 151 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="151", "Chromium";v="151", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="151.0.8342.49", "Chromium";v="151.0.8342.49", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 152 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="152", "Not?A_Brand";v="99", "Google Chrome";v="152"`,
		secChUAFullVersion: `"Chromium";v="152.0.8455.68", "Not?A_Brand";v="99.0.0.0", "Google Chrome";v="152.0.8455.68"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Chrome 153 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/153.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="153", "Chromium";v="153", "Not;A=Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="153.0.8570.52", "Chromium";v="153.0.8570.52", "Not;A=Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},

	// ── Chrome 137-153 × macOS ────────────────────────────────────────────────
	{ // Chrome 137 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="137", "Chromium";v="137", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="137.0.7151.55", "Chromium";v="137.0.7151.55", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 138 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="138", "Not(A:Brand";v="8", "Google Chrome";v="138"`,
		secChUAFullVersion: `"Chromium";v="138.0.7204.100", "Not(A:Brand";v="8.0.0.0", "Google Chrome";v="138.0.7204.100"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 139 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="139", "Chromium";v="139", "Not/A_Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="139.0.7258.65", "Chromium";v="139.0.7258.65", "Not/A_Brand";v="24.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 140 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="140", "Google Chrome";v="140", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="140.0.7321.84", "Google Chrome";v="140.0.7321.84", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 141 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="141", "Chromium";v="141", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="141.0.7395.73", "Chromium";v="141.0.7395.73", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 142 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="142", "Not_A Brand";v="24", "Google Chrome";v="142"`,
		secChUAFullVersion: `"Chromium";v="142.0.7471.61", "Not_A Brand";v="24.0.0.0", "Google Chrome";v="142.0.7471.61"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 143 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="143", "Chromium";v="143", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="143.0.7549.52", "Chromium";v="143.0.7549.52", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 144 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="144", "Not;A=Brand";v="24", "Google Chrome";v="144"`,
		secChUAFullVersion: `"Chromium";v="144.0.7637.45", "Not;A=Brand";v="24.0.0.0", "Google Chrome";v="144.0.7637.45"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 145 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="145", "Not=A?Brand";v="8", "Chromium";v="145"`,
		secChUAFullVersion: `"Google Chrome";v="145.0.7724.58", "Not=A?Brand";v="8.0.0.0", "Chromium";v="145.0.7724.58"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 146 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="146", "Google Chrome";v="146", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="146.0.7821.79", "Google Chrome";v="146.0.7821.79", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 147 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="147", "Chromium";v="147", "Not(A:Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="147.0.7931.55", "Chromium";v="147.0.7931.55", "Not(A:Brand";v="8.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 148 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="148", "Not/A_Brand";v="24", "Google Chrome";v="148"`,
		secChUAFullVersion: `"Chromium";v="148.0.8035.88", "Not/A_Brand";v="24.0.0.0", "Google Chrome";v="148.0.8035.88"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 149 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="149", "Chromium";v="149", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="149.0.8127.64", "Chromium";v="149.0.8127.64", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 150 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="150", "Google Chrome";v="150", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Chromium";v="150.0.8230.73", "Google Chrome";v="150.0.8230.73", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 151 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="151", "Chromium";v="151", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="151.0.8342.49", "Chromium";v="151.0.8342.49", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.1.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 152 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="152", "Not?A_Brand";v="99", "Google Chrome";v="152"`,
		secChUAFullVersion: `"Chromium";v="152.0.8455.68", "Not?A_Brand";v="99.0.0.0", "Google Chrome";v="152.0.8455.68"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.1.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Chrome 153 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/153.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="153", "Chromium";v="153", "Not;A=Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="153.0.8570.52", "Chromium";v="153.0.8570.52", "Not;A=Brand";v="24.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"15.1.0"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},

	// ── Edge 134-148 × Windows ────────────────────────────────────────────────
	{ // Edge 134 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 Edg/134.0.0.0",
		secChUA:            `"Microsoft Edge";v="134", "Chromium";v="134", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Microsoft Edge";v="134.0.3124.72", "Chromium";v="134.0.6998.165", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 135 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0",
		secChUA:            `"Microsoft Edge";v="135", "Chromium";v="135", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Microsoft Edge";v="135.0.3179.98", "Chromium";v="135.0.7049.114", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 136 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36 Edg/136.0.0.0",
		secChUA:            `"Microsoft Edge";v="136", "Chromium";v="136", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="136.0.3240.92", "Chromium";v="136.0.7103.92", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 137 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36 Edg/137.0.0.0",
		secChUA:            `"Microsoft Edge";v="137", "Chromium";v="137", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Microsoft Edge";v="137.0.3296.55", "Chromium";v="137.0.7151.55", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 138 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36 Edg/138.0.0.0",
		secChUA:            `"Microsoft Edge";v="138", "Chromium";v="138", "Not;A=Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="138.0.3351.100", "Chromium";v="138.0.7204.100", "Not;A=Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 139 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0",
		secChUA:            `"Microsoft Edge";v="139", "Chromium";v="139", "Not=A?Brand";v="8"`,
		secChUAFullVersion: `"Microsoft Edge";v="139.0.3405.65", "Chromium";v="139.0.7258.65", "Not=A?Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 140 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36 Edg/140.0.0.0",
		secChUA:            `"Microsoft Edge";v="140", "Chromium";v="140", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Microsoft Edge";v="140.0.3467.84", "Chromium";v="140.0.7321.84", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 141 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0",
		secChUA:            `"Microsoft Edge";v="141", "Chromium";v="141", "Not(A:Brand";v="8"`,
		secChUAFullVersion: `"Microsoft Edge";v="141.0.3540.73", "Chromium";v="141.0.7395.73", "Not(A:Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 142 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0",
		secChUA:            `"Microsoft Edge";v="142", "Chromium";v="142", "Not/A_Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="142.0.3616.61", "Chromium";v="142.0.7471.61", "Not/A_Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 143 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 Edg/143.0.0.0",
		secChUA:            `"Microsoft Edge";v="143", "Chromium";v="143", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Microsoft Edge";v="143.0.3693.52", "Chromium";v="143.0.7549.52", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 144 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 Edg/144.0.0.0",
		secChUA:            `"Microsoft Edge";v="144", "Chromium";v="144", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Microsoft Edge";v="144.0.3780.45", "Chromium";v="144.0.7637.45", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 145 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36 Edg/145.0.0.0",
		secChUA:            `"Microsoft Edge";v="145", "Chromium";v="145", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="145.0.3866.58", "Chromium";v="145.0.7724.58", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 146 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0",
		secChUA:            `"Microsoft Edge";v="146", "Chromium";v="146", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Microsoft Edge";v="146.0.3965.79", "Chromium";v="146.0.7821.79", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 147 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36 Edg/147.0.0.0",
		secChUA:            `"Microsoft Edge";v="147", "Chromium";v="147", "Not;A=Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="147.0.4077.55", "Chromium";v="147.0.7931.55", "Not;A=Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},
	{ // Edge 148 · Windows
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36 Edg/148.0.0.0",
		secChUA:            `"Microsoft Edge";v="148", "Chromium";v="148", "Not=A?Brand";v="8"`,
		secChUAFullVersion: `"Microsoft Edge";v="148.0.4183.88", "Chromium";v="148.0.8035.88", "Not=A?Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
	},

	// ── Chrome 128-136 + 154 × Windows · viewport 1920 ────────────────────────
	{ // Chrome 154 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="154", "Google Chrome";v="154", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="154.0.8037.98", "Google Chrome";v="154.0.8037.98", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 136 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="136", "Not(A:Brand";v="8", "Google Chrome";v="136"`,
		secChUAFullVersion: `"Chromium";v="136.0.7103.92", "Not(A:Brand";v="8.0.0.0", "Google Chrome";v="136.0.7103.92"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 135 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="135", "Chromium";v="135", "Not/A_Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="135.0.7049.114", "Chromium";v="135.0.7049.114", "Not/A_Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 134 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="134", "Google Chrome";v="134", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="134.0.6998.165", "Google Chrome";v="134.0.6998.165", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 133 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="133", "Chromium";v="133", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="133.0.6943.141", "Chromium";v="133.0.6943.141", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 132 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="132", "Not(A:Brand";v="24", "Google Chrome";v="132"`,
		secChUAFullVersion: `"Chromium";v="132.0.6834.159", "Not(A:Brand";v="24.0.0.0", "Google Chrome";v="132.0.6834.159"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 131 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="131.0.6778.205", "Chromium";v="131.0.6778.205", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 130 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="130.0.6723.191", "Google Chrome";v="130.0.6723.191", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 129 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="129", "Not=A?Brand";v="8", "Chromium";v="129"`,
		secChUAFullVersion: `"Google Chrome";v="129.0.6668.100", "Not=A?Brand";v="8.0.0.0", "Chromium";v="129.0.6668.100"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},
	{ // Chrome 128 · Windows · 1920px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="128", "Not;A=Brand";v="24", "Google Chrome";v="128"`,
		secChUAFullVersion: `"Chromium";v="128.0.6613.120", "Not;A=Brand";v="24.0.0.0", "Google Chrome";v="128.0.6613.120"`,
		platform:           `"Windows"`,
		platformVersion:    `"15.0.0"`,
		dpr:                "1",
		viewportWidth:      "1920",
	},

	// ── Chrome 137-149 × Windows · viewport 1366 ─────────────────────────────
	{ // Chrome 137 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="137", "Chromium";v="137", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="137.0.7151.55", "Chromium";v="137.0.7151.55", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 138 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="138", "Not(A:Brand";v="8", "Google Chrome";v="138"`,
		secChUAFullVersion: `"Chromium";v="138.0.7204.100", "Not(A:Brand";v="8.0.0.0", "Google Chrome";v="138.0.7204.100"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 139 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="139", "Chromium";v="139", "Not/A_Brand";v="24"`,
		secChUAFullVersion: `"Google Chrome";v="139.0.7258.65", "Chromium";v="139.0.7258.65", "Not/A_Brand";v="24.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 140 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="140", "Google Chrome";v="140", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="140.0.7321.84", "Google Chrome";v="140.0.7321.84", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 141 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="141", "Chromium";v="141", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="141.0.7395.73", "Chromium";v="141.0.7395.73", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 142 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="142", "Not_A Brand";v="24", "Google Chrome";v="142"`,
		secChUAFullVersion: `"Chromium";v="142.0.7471.61", "Not_A Brand";v="24.0.0.0", "Google Chrome";v="142.0.7471.61"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 143 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="143", "Chromium";v="143", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="143.0.7549.52", "Chromium";v="143.0.7549.52", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 144 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="144", "Not;A=Brand";v="24", "Google Chrome";v="144"`,
		secChUAFullVersion: `"Chromium";v="144.0.7637.45", "Not;A=Brand";v="24.0.0.0", "Google Chrome";v="144.0.7637.45"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 145 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="145", "Not=A?Brand";v="8", "Chromium";v="145"`,
		secChUAFullVersion: `"Google Chrome";v="145.0.7724.58", "Not=A?Brand";v="8.0.0.0", "Chromium";v="145.0.7724.58"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 146 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="146", "Google Chrome";v="146", "Not A(Brand";v="99"`,
		secChUAFullVersion: `"Chromium";v="146.0.7821.79", "Google Chrome";v="146.0.7821.79", "Not A(Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 147 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="147", "Chromium";v="147", "Not(A:Brand";v="8"`,
		secChUAFullVersion: `"Google Chrome";v="147.0.7931.55", "Chromium";v="147.0.7931.55", "Not(A:Brand";v="8.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 148 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36",
		secChUA:            `"Chromium";v="148", "Not/A_Brand";v="24", "Google Chrome";v="148"`,
		secChUAFullVersion: `"Chromium";v="148.0.8035.88", "Not/A_Brand";v="24.0.0.0", "Google Chrome";v="148.0.8035.88"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},
	{ // Chrome 149 · Windows · 1366px
		userAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
		secChUA:            `"Google Chrome";v="149", "Chromium";v="149", "Not:A-Brand";v="99"`,
		secChUAFullVersion: `"Google Chrome";v="149.0.8127.64", "Chromium";v="149.0.8127.64", "Not:A-Brand";v="99.0.0.0"`,
		platform:           `"Windows"`,
		platformVersion:    `"19.0.0"`,
		dpr:                "1",
		viewportWidth:      "1366",
	},

	// ── Edge 130-133 × macOS ──────────────────────────────────────────────────
	{ // Edge 133 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 Edg/133.0.0.0",
		secChUA:            `"Microsoft Edge";v="133", "Chromium";v="133", "Not.A%Brand";v="8"`,
		secChUAFullVersion: `"Microsoft Edge";v="133.0.3065.92", "Chromium";v="133.0.6943.141", "Not.A%Brand";v="8.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Edge 132 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0",
		secChUA:            `"Microsoft Edge";v="132", "Chromium";v="132", "Not(A:Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="132.0.2957.140", "Chromium";v="132.0.6834.159", "Not(A:Brand";v="24.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Edge 131 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
		secChUA:            `"Microsoft Edge";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		secChUAFullVersion: `"Microsoft Edge";v="131.0.2903.86", "Chromium";v="131.0.6778.205", "Not_A Brand";v="24.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
	{ // Edge 130 · macOS
		userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
		secChUA:            `"Microsoft Edge";v="130", "Chromium";v="130", "Not?A_Brand";v="99"`,
		secChUAFullVersion: `"Microsoft Edge";v="130.0.2849.80", "Chromium";v="130.0.6723.191", "Not?A_Brand";v="99.0.0.0"`,
		platform:           `"macOS"`,
		platformVersion:    `"14.7.1"`,
		dpr:                "2",
		viewportWidth:      "1440",
	},
}

// profileForIndex picks a browser profile by worker index so each of the first
// 100 concurrent accounts gets a distinct fingerprint. Falls back to modulo when
// there are more workers than profiles.
func profileForIndex(idx int) browserProfile {
	if idx < 0 {
		idx = 0
	}
	return browserProfiles[idx%len(browserProfiles)]
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
