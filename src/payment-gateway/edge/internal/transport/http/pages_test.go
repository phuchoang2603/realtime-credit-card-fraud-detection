package httptransport

import "testing"

func TestMerchantOrigin(t *testing.T) {
	cases := map[string]string{
		"https://shop.example.com/return":        "https://shop.example.com",
		"https://shop.example.com:8443/checkout": "https://shop.example.com:8443",
		"http://shop.example.com/return":         "",
		"https:///return":                        "",
		"https://bad host/return":                "",
		"":                                       "",
	}
	for raw, want := range cases {
		if got := merchantOrigin(raw); got != want {
			t.Fatalf("merchantOrigin(%q) = %q, want %q", raw, got, want)
		}
	}
}
