package facebook

import "testing"

func TestJazoestMatchesSample(t *testing.T) {
	// The captured request used this fb_dtsg and jazoest=25623.
	const sampleDtsg = "NAfwPkHRQVJmRkXQfSOUZFTNEl2b9N3dfClSVvlytVCZRBiTXykyRrg:9:1776069563"
	if got := jazoest(sampleDtsg); got != "25623" {
		t.Fatalf("jazoest(sample) = %q, want 25623", got)
	}
}

func TestParseClickResponse(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name:    "success",
			raw:     `{"data":{"messenger_business_ads_sender":"119674836101452"},"extensions":{"is_final":true}}`,
			wantErr: false,
		},
		{name: "graphql errors", raw: `{"data":null,"errors":[{"message":"boom"}]}`, wantErr: true},
		{
			name:    "fb error envelope with prefix",
			raw:     `for (;;);{"error":1357053,"errorSummary":"Vui lòng xem lại tài khoản của bạn","errorDescription":"Tài khoản của bạn hiện bị hạn chế thực hiện hành động này."}`,
			wantErr: true,
		},
		{name: "empty object", raw: `{}`, wantErr: true},
		{name: "empty sender", raw: `{"data":{"messenger_business_ads_sender":""}}`, wantErr: true},
		{name: "garbage", raw: `not json`, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := parseClickResponse([]byte(c.raw))
			if (err != nil) != c.wantErr {
				t.Fatalf("parseClickResponse(%s) err=%v, wantErr=%v", c.raw, err, c.wantErr)
			}
		})
	}
}
