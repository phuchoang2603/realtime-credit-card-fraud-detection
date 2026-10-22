package domain

import (
	"errors"
	"fmt"
	"math"
	"regexp"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/payments/gen/payments/v1"
)

var (
	ErrInvalidSnapshot = errors.New("invalid checkout snapshot")
	currencyPattern    = regexp.MustCompile(`^[A-Z]{3}$`)
)

func ValidateSnapshot(snapshot *paymentsv1.CheckoutSnapshot) error {
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidSnapshot, fmt.Sprintf(format, args...))
	}
	if snapshot == nil {
		return invalid("snapshot is required")
	}
	if snapshot.GetExternalOrderId() == "" {
		return invalid("external_order_id is required")
	}
	total := snapshot.GetTotal()
	if total == nil || total.GetMinorUnits() < 0 || !currencyPattern.MatchString(total.GetCurrency()) {
		return invalid("total must be a nonnegative amount with an ISO 4217 currency")
	}
	if len(snapshot.GetItems()) == 0 {
		return invalid("at least one item is required")
	}
	var sum int64
	for _, item := range snapshot.GetItems() {
		price := item.GetUnitPrice()
		switch {
		case item.GetProductId() == "" || item.GetCategory() == "" || item.GetCondition() == "":
			return invalid("item product_id, category and condition are required")
		case item.GetQuantity() <= 0:
			return invalid("item quantity must be positive")
		case price == nil || price.GetMinorUnits() < 0 || price.GetCurrency() != total.GetCurrency():
			return invalid("item unit_price must be nonnegative in the total currency")
		case price.GetMinorUnits() > math.MaxInt64/int64(item.GetQuantity()):
			return invalid("item amount overflows")
		}
		line := price.GetMinorUnits() * int64(item.GetQuantity())
		if sum > math.MaxInt64-line {
			return invalid("item amounts overflow")
		}
		sum += line
	}
	if sum != total.GetMinorUnits() {
		return invalid("items sum to %d but total is %d", sum, total.GetMinorUnits())
	}
	if snapshot.GetMerchantAccountCreatedAt() == nil || snapshot.GetCustomerAccountCreatedAt() == nil {
		return invalid("account creation times are required")
	}
	shipping := snapshot.GetShipping()
	if shipping == nil || shipping.GetCountry() == "" || shipping.GetAddressFingerprint() == "" {
		return invalid("shipping country and address_fingerprint are required")
	}
	return nil
}
