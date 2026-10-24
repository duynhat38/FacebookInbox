package httpclient

import "testing"

func TestChromeMajor(t *testing.T) {
	cases := map[string]int{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36":                    154,
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0":      131, // Edge carries a Chrome token
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36":              128,
		"":          0,
		"Firefox/1": 0,
	}
	for ua, want := range cases {
		if got := chromeMajor(ua); got != want {
			t.Errorf("chromeMajor(%.40q) = %d, want %d", ua, got, want)
		}
	}
}

func TestProfileForMajor(t *testing.T) {
	// Every major the pool's UAs (Chrome 128–154 + Edge 130–133) and the unknown
	// fallback can resolve to must yield a real tls-client profile whose
	// ClientHello spec builds — guards against referencing a non-existent or
	// zero-value profile (e.g. the tls-client library dropping a version).
	for _, major := range []int{0, 100, 124, 128, 130, 131, 133, 136, 144, 146, 150, 152, 154} {
		p := profileForMajor(major)
		if p.GetClientHelloStr() == "" {
			t.Errorf("profileForMajor(%d) returned a profile with empty ClientHello id", major)
		}
		if _, err := p.GetClientHelloSpec(); err != nil {
			t.Errorf("profileForMajor(%d) ClientHello spec failed to build: %v", major, err)
		}
		if len(p.GetSettings()) == 0 {
			t.Errorf("profileForMajor(%d) has no HTTP/2 settings (not a real profile)", major)
		}
	}
}
