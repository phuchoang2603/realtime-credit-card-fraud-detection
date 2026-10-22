package domain

import (
	"math"
	"net/url"
	"regexp"
	"time"
	"unicode/utf8"
)

const (
	maxIdentifierLength = 128
	maxFieldLength      = 200
	maxReturnURLLength  = 2048
	maxItems            = 100
)

var (
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	countryPattern  = regexp.MustCompile(`^[A-Z]{2}$`)
)

type Money struct {
	MinorUnits int64
	Currency   string
}

type Item struct {
	ProductID string
	Category  string
	Condition string
	Quantity  int32
	UnitPrice int64
}

type Party struct {
	ExternalID       string
	AccountCreatedAt time.Time
}

type CheckoutInput struct {
	OrderReference string
	Seller         Party
	Buyer          Party
	Total          Money
	Items          []Item
	Shipping       Address
	ReturnURL      string
}

type ShippingLocation struct {
	Country            string
	Region             string
	PostalCode         string
	AddressFingerprint string
}

// Snapshot carries no recipient name, street lines, city or buyer contact data.
type Snapshot struct {
	OrderReference           string
	Total                    Money
	Items                    []Item
	MerchantAccountCreatedAt time.Time
	CustomerAccountCreatedAt time.Time
	Shipping                 ShippingLocation
}

func ValidCountry(code string) bool { return countryPattern.MatchString(code) }

func validText(value string, required bool, limit int) bool {
	if value == "" {
		return !required
	}
	return utf8.RuneCountInString(value) <= limit && utf8.ValidString(value)
}

func (c CheckoutInput) Validate() error {
	checks := []struct {
		ok     bool
		detail string
	}{
		{validText(c.OrderReference, true, maxIdentifierLength), "order_reference is required"},
		{validText(c.Seller.ExternalID, true, maxIdentifierLength), "seller.external_id is required"},
		{!c.Seller.AccountCreatedAt.IsZero(), "seller.account_created_at is required"},
		{validText(c.Buyer.ExternalID, true, maxIdentifierLength), "buyer.external_id is required"},
		{!c.Buyer.AccountCreatedAt.IsZero(), "buyer.account_created_at is required"},
		{c.Total.MinorUnits > 0, "amount.minor_units must be positive"},
		{currencyPattern.MatchString(c.Total.Currency), "amount.currency must be an ISO 4217 code"},
		{len(c.Items) > 0 && len(c.Items) <= maxItems, "items must hold between 1 and 100 entries"},
		{validText(c.Shipping.Name, true, maxFieldLength), "shipping_address.name is required"},
		{validText(c.Shipping.Line1, true, maxFieldLength), "shipping_address.line1 is required"},
		{validText(c.Shipping.Line2, false, maxFieldLength), "shipping_address.line2 is too long"},
		{validText(c.Shipping.City, true, maxFieldLength), "shipping_address.city is required"},
		{validText(c.Shipping.Region, false, maxFieldLength), "shipping_address.region is too long"},
		{validText(c.Shipping.PostalCode, false, maxFieldLength), "shipping_address.postal_code is too long"},
		{countryPattern.MatchString(c.Shipping.Country), "shipping_address.country must be an ISO 3166 alpha-2 code"},
	}
	for _, check := range checks {
		if !check.ok {
			return Invalid(check.detail)
		}
	}
	if err := c.validateItems(); err != nil {
		return err
	}
	return ValidateReturnURL(c.ReturnURL)
}

func (c CheckoutInput) validateItems() error {
	var sum int64
	for _, item := range c.Items {
		if !validText(item.ProductID, true, maxIdentifierLength) ||
			!validText(item.Category, true, maxIdentifierLength) ||
			!validText(item.Condition, true, maxIdentifierLength) {
			return Invalid("items require product_id, category and condition")
		}
		if item.Quantity <= 0 || item.UnitPrice <= 0 {
			return Invalid("items require a positive quantity and unit_price")
		}
		if item.UnitPrice > (math.MaxInt64-sum)/int64(item.Quantity) {
			return Invalid("items exceed the supported amount")
		}
		sum += int64(item.Quantity) * item.UnitPrice
	}
	if sum != c.Total.MinorUnits {
		return Invalid("items do not sum to amount")
	}
	return nil
}

func ValidateReturnURL(raw string) error {
	if raw == "" || len(raw) > maxReturnURLLength {
		return Invalid("return_url must be an absolute HTTPS URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return Invalid("return_url must be an absolute HTTPS URL")
	}
	return nil
}

func (c CheckoutInput) Snapshot(f Fingerprinter) Snapshot {
	return Snapshot{
		OrderReference:           c.OrderReference,
		Total:                    c.Total,
		Items:                    c.Items,
		MerchantAccountCreatedAt: c.Seller.AccountCreatedAt,
		CustomerAccountCreatedAt: c.Buyer.AccountCreatedAt,
		Shipping: ShippingLocation{
			Country:            c.Shipping.Country,
			Region:             c.Shipping.Region,
			PostalCode:         c.Shipping.PostalCode,
			AddressFingerprint: f.Address(c.Shipping),
		},
	}
}
