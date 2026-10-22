package domain

import (
	"errors"
	"fmt"
	"math"
	"regexp"
)

var (
	ErrInvalidSnapshot = errors.New("invalid checkout snapshot")
	currencyPattern    = regexp.MustCompile(`^[A-Z]{3}$`)
)

func ValidateSnapshot(snapshot *CheckoutSnapshot) error {
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidSnapshot, fmt.Sprintf(format, args...))
	}
	if snapshot == nil {
		return invalid("snapshot is required")
	}
	if snapshot.ExternalOrderID == "" {
		return invalid("external_order_id is required")
	}
	total := snapshot.Total
	if total == nil || total.MinorUnits < 0 || !currencyPattern.MatchString(total.Currency) {
		return invalid("total must be a nonnegative amount with an ISO 4217 currency")
	}
	if len(snapshot.Items) == 0 {
		return invalid("at least one item is required")
	}
	var sum int64
	for _, item := range snapshot.Items {
		if item == nil {
			return invalid("item is required")
		}
		price := item.UnitPrice
		switch {
		case item.ProductID == "" || item.Category == "" || item.Condition == "":
			return invalid("item product_id, category and condition are required")
		case item.Quantity <= 0:
			return invalid("item quantity must be positive")
		case price == nil || price.MinorUnits < 0 || price.Currency != total.Currency:
			return invalid("item unit_price must be nonnegative in the total currency")
		case price.MinorUnits > math.MaxInt64/int64(item.Quantity):
			return invalid("item amount overflows")
		}
		line := price.MinorUnits * int64(item.Quantity)
		if sum > math.MaxInt64-line {
			return invalid("item amounts overflow")
		}
		sum += line
	}
	if sum != total.MinorUnits {
		return invalid("items sum to %d but total is %d", sum, total.MinorUnits)
	}
	if snapshot.MerchantAccountCreatedAt.IsZero() || snapshot.CustomerAccountCreatedAt.IsZero() {
		return invalid("account creation times are required")
	}
	shipping := snapshot.Shipping
	if shipping == nil || shipping.Country == "" || shipping.AddressFingerprint == "" {
		return invalid("shipping country and address_fingerprint are required")
	}
	return nil
}
