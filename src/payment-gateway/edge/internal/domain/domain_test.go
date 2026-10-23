package domain_test

import (
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/domain"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/testvectors"
)

func TestAddressFingerprintMatchesIdentityVector(t *testing.T) {
	vectors := testvectors.Load(t)
	vector := vectors.Fingerprint(t, "address")
	fingerprinter := domain.NewFingerprinter(vectors.SyntheticKey)

	got := fingerprinter.Address(domain.Address{Line1: vector.Input})
	if got != vector.Expected {
		t.Fatalf("address fingerprint = %s, want %s", got, vector.Expected)
	}
	if normalized := domain.NormalizeAddress(domain.Address{Line1: vector.Input}); normalized != vector.Normalized {
		t.Fatalf("normalized = %q, want %q", normalized, vector.Normalized)
	}
}

func TestNetworkFingerprintMatchesIdentityVector(t *testing.T) {
	vectors := testvectors.Load(t)
	vector := vectors.Fingerprint(t, "ip_network")
	got, ok := domain.NewFingerprinter(vectors.SyntheticKey).Network(netip.MustParseAddr(vector.Input))
	if !ok || got != vector.Expected {
		t.Fatalf("network fingerprint = %s (%v), want %s", got, ok, vector.Expected)
	}
	prefix, _ := domain.NetworkPrefix(netip.MustParseAddr(vector.Input))
	if prefix.String() != vector.Normalized {
		t.Fatalf("prefix = %s, want %s", prefix, vector.Normalized)
	}
}

func TestNetworkPrefixLengths(t *testing.T) {
	tests := map[string]string{
		"203.0.113.77":         "203.0.113.0/24",
		"::ffff:203.0.113.77":  "203.0.113.0/24",
		"2001:db8:1:2:3:4:5:6": "2001:db8:1::/48",
		"fe80::1%eth0":         "fe80::/48",
	}
	for input, want := range tests {
		prefix, ok := domain.NetworkPrefix(netip.MustParseAddr(input))
		if !ok || prefix.String() != want {
			t.Errorf("NetworkPrefix(%s) = %s (%v), want %s", input, prefix, ok, want)
		}
	}
	if _, ok := domain.NetworkPrefix(netip.Addr{}); ok {
		t.Error("zero address must not produce a network")
	}
}

func TestAddressFingerprintIgnoresFormattingAndRecipientName(t *testing.T) {
	fingerprinter := domain.NewFingerprinter("key")
	base := domain.Address{Name: "Ada Lovelace", Line1: "42 Main St", Line2: "Apt 5", City: "Springfield", Region: "IL", PostalCode: "62701", Country: "US"}
	reformatted := domain.Address{Name: "Someone Else", Line1: "  42  MAIN st ", Line2: "apt   5", City: " SPRINGFIELD", Region: "il", PostalCode: "62701 ", Country: "us"}
	if fingerprinter.Address(base) != fingerprinter.Address(reformatted) {
		t.Fatal("formatting or recipient name changed the address fingerprint")
	}
	moved := base
	moved.Line1 = "43 Main St"
	if fingerprinter.Address(base) == fingerprinter.Address(moved) {
		t.Fatal("different street lines produced equal fingerprints")
	}
}

func TestRealAndSyntheticKeysProduceDifferentFingerprints(t *testing.T) {
	address := domain.Address{Line1: "42 Main St", City: "Springfield", Country: "US"}
	real := domain.NewFingerprinter("a-secret-real-key").Address(address)
	synthetic := domain.NewFingerprinter("synthetic-fingerprint-v1").Address(address)
	if real == synthetic {
		t.Fatal("real and synthetic keys must not collide")
	}
}

func validInput() domain.CheckoutInput {
	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return domain.CheckoutInput{
		OrderReference: "order-1",
		Seller:         domain.Party{ExternalID: "seller-1", AccountCreatedAt: created},
		Buyer:          domain.Party{ExternalID: "buyer-1", AccountCreatedAt: created},
		Total:          domain.Money{MinorUnits: 3500, Currency: "USD"},
		Items: []domain.Item{
			{ProductID: "p1", Category: "phone", Condition: "good", Quantity: 2, UnitPrice: 1000},
			{ProductID: "p2", Category: "case", Condition: "new", Quantity: 1, UnitPrice: 1500},
		},
		Shipping:  domain.Address{Name: "Ada", Line1: "42 Main St", City: "Springfield", Region: "IL", PostalCode: "62701", Country: "US"},
		ReturnURL: "https://shop.example.com/return",
	}
}

func TestCheckoutInputValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.CheckoutInput)
		valid  bool
	}{
		{"valid", func(*domain.CheckoutInput) {}, true},
		{"items do not sum", func(c *domain.CheckoutInput) { c.Total.MinorUnits = 3499 }, false},
		{"http return url", func(c *domain.CheckoutInput) { c.ReturnURL = "http://shop.example.com/r" }, false},
		{"relative return url", func(c *domain.CheckoutInput) { c.ReturnURL = "/return" }, false},
		{"userinfo return url", func(c *domain.CheckoutInput) { c.ReturnURL = "https://user:pw@shop.example.com/r" }, false},
		{"bad currency", func(c *domain.CheckoutInput) { c.Total.Currency = "usd" }, false},
		{"no items", func(c *domain.CheckoutInput) { c.Items = nil }, false},
		{"zero quantity", func(c *domain.CheckoutInput) { c.Items[0].Quantity = 0 }, false},
		{"missing seller", func(c *domain.CheckoutInput) { c.Seller.ExternalID = "" }, false},
		{"missing buyer creation time", func(c *domain.CheckoutInput) { c.Buyer.AccountCreatedAt = time.Time{} }, false},
		{"bad country", func(c *domain.CheckoutInput) { c.Shipping.Country = "USA" }, false},
		{"overflow", func(c *domain.CheckoutInput) {
			c.Items = []domain.Item{{ProductID: "p", Category: "c", Condition: "n", Quantity: 1 << 30, UnitPrice: 1 << 40}}
		}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validInput()
			test.mutate(&input)
			err := input.Validate()
			if test.valid && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !test.valid && !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("error = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestSnapshotKeepsOnlyCoarseLocationAndFingerprint(t *testing.T) {
	input := validInput()
	snapshot := input.Snapshot(domain.NewFingerprinter("key"))
	if snapshot.Shipping.Country != "US" || snapshot.Shipping.Region != "IL" || snapshot.Shipping.PostalCode != "62701" {
		t.Fatalf("shipping = %+v", snapshot.Shipping)
	}
	if snapshot.Shipping.AddressFingerprint == "" || strings.Contains(snapshot.Shipping.AddressFingerprint, "Main") {
		t.Fatalf("address fingerprint = %q", snapshot.Shipping.AddressFingerprint)
	}
}

func TestCardHygiene(t *testing.T) {
	for _, field := range []string{"card_number", "NUMBER", "pan", "cvv", "cvc", "card_security_code", "card", "expiry"} {
		if !domain.IsCardField(field) {
			t.Errorf("%s must be a card field", field)
		}
	}
	if domain.IsCardField("payment_method_token") {
		t.Error("payment_method_token is not a card field")
	}
	if number, ok := domain.NormalizeCardNumber("4242 4242-4242 4242"); !ok || number != "4242424242424242" {
		t.Errorf("normalized = %q (%v)", number, ok)
	}
	for _, bad := range []string{"", "1234", "4242abcd42424242", strings.Repeat("1", 20)} {
		if _, ok := domain.NormalizeCardNumber(bad); ok {
			t.Errorf("%q must not normalize", bad)
		}
	}
}

func TestUserAgentFamily(t *testing.T) {
	tests := map[string]string{
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36":         "chrome",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36 Edg/126": "edge",
		"Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0":                "firefox",
		"Mozilla/5.0 (Macintosh) AppleWebKit/605.1.15 Version/17.5 Safari/605.1.15":             "safari",
		"curl/8.0": "other",
		"":         "other",
	}
	for agent, want := range tests {
		if got := domain.UserAgentFamily(agent); got != want {
			t.Errorf("UserAgentFamily(%q) = %s, want %s", agent, got, want)
		}
	}
}
