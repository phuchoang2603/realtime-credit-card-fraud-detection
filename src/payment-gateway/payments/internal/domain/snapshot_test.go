package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestSnapshotPartitionsAndBoundaries(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*CheckoutSnapshot)
		valid  bool
	}{
		{"valid", func(*CheckoutSnapshot) {}, true},
		{"zero amount", func(s *CheckoutSnapshot) { s.Total.MinorUnits = 0; s.Items[0].UnitPrice.MinorUnits = 0 }, true},
		{"negative total", func(s *CheckoutSnapshot) { s.Total.MinorUnits = -1 }, false},
		{"maximum amount", func(s *CheckoutSnapshot) {
			s.Total.MinorUnits = math.MaxInt64
			s.Items[0].UnitPrice.MinorUnits = math.MaxInt64
		}, true},
		{"multiplication overflow", func(s *CheckoutSnapshot) { s.Items[0].Quantity = 2; s.Items[0].UnitPrice.MinorUnits = math.MaxInt64 }, false},
		{"addition overflow", func(s *CheckoutSnapshot) {
			s.Total.MinorUnits = math.MaxInt64
			s.Items = append(s.Items, &LineItem{ProductID: "extra", Category: "goods", Condition: "new", Quantity: 1, UnitPrice: &Money{MinorUnits: math.MaxInt64, Currency: "USD"}})
		}, false},
		{"zero quantity", func(s *CheckoutSnapshot) { s.Items[0].Quantity = 0 }, false},
		{"negative quantity", func(s *CheckoutSnapshot) { s.Items[0].Quantity = -1 }, false},
		{"missing total", func(s *CheckoutSnapshot) { s.Total = nil }, false},
		{"no items", func(s *CheckoutSnapshot) { s.Items = nil }, false},
		{"nil item", func(s *CheckoutSnapshot) { s.Items[0] = nil }, false},
		{"missing price", func(s *CheckoutSnapshot) { s.Items[0].UnitPrice = nil }, false},
		{"negative price", func(s *CheckoutSnapshot) { s.Items[0].UnitPrice.MinorUnits = -1 }, false},
		{"currency mismatch", func(s *CheckoutSnapshot) { s.Items[0].UnitPrice.Currency = "EUR" }, false},
		{"malformed currency", func(s *CheckoutSnapshot) { s.Total.Currency = "usd" }, false},
		{"missing order", func(s *CheckoutSnapshot) { s.ExternalOrderID = "" }, false},
		{"missing item identity", func(s *CheckoutSnapshot) { s.Items[0].ProductID = "" }, false},
		{"sum mismatch", func(s *CheckoutSnapshot) { s.Total.MinorUnits++ }, false},
		{"missing account time", func(s *CheckoutSnapshot) { s.CustomerAccountCreatedAt = time.Time{} }, false},
		{"missing shipping", func(s *CheckoutSnapshot) { s.Shipping = nil }, false},
		{"missing fingerprint", func(s *CheckoutSnapshot) { s.Shipping.AddressFingerprint = "" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := &CheckoutSnapshot{
				ExternalOrderID: "order", Total: &Money{MinorUnits: 1, Currency: "USD"},
				Items:                    []*LineItem{{ProductID: "product", Category: "goods", Condition: "new", Quantity: 1, UnitPrice: &Money{MinorUnits: 1, Currency: "USD"}}},
				MerchantAccountCreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				CustomerAccountCreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				Shipping:                 &ShippingLocation{Country: "US", AddressFingerprint: "fingerprint"},
			}
			test.mutate(snapshot)
			err := ValidateSnapshot(snapshot)
			if (err == nil) != test.valid || (err != nil && !errors.Is(err, ErrInvalidSnapshot)) {
				t.Fatalf("snapshot validity = %t, want %t: %v", err == nil, test.valid, err)
			}
		})
	}
	if err := ValidateSnapshot(nil); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("nil snapshot: %v", err)
	}
}
