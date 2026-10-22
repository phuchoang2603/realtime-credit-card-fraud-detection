package testsupport

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/payments/gen/payments/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/domain"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/wire"
)

func Snapshot(orderReference string, minorUnits int64) *paymentsv1.CheckoutSnapshot {
	return &paymentsv1.CheckoutSnapshot{
		ExternalOrderId: orderReference,
		Total:           &paymentsv1.Money{MinorUnits: minorUnits, Currency: "USD"},
		Items: []*paymentsv1.LineItem{{
			ProductId: "sku-phone-001", Category: "phone", Condition: "good", Quantity: 1,
			UnitPrice: &paymentsv1.Money{MinorUnits: minorUnits, Currency: "USD"},
		}},
		MerchantAccountCreatedAt: timestamppb.New(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CustomerAccountCreatedAt: timestamppb.New(time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)),
		Shipping: &paymentsv1.ShippingLocation{
			Country: "US", Region: "CA", PostalCode: "94105",
			AddressFingerprint: "3e749dce5096ba78de0bd0faebbd401473ce860f9810c7f6571377c8ff8643a6",
		},
	}
}

func CreateRequest(integrationID uuid.UUID, key, orderReference string, minorUnits int64) *paymentsv1.CreatePaymentRequest {
	return &paymentsv1.CreatePaymentRequest{
		IntegrationId: integrationID.String(), IdempotencyKey: key, ExternalSellerId: "seller-01", ExternalBuyerId: "buyer-01",
		Snapshot: Snapshot(orderReference, minorUnits), ReturnUrl: "https://shop.example/return",
	}
}

func AttemptRequest(payment *paymentsv1.Payment, key, token string) *paymentsv1.StartAttemptRequest {
	return &paymentsv1.StartAttemptRequest{
		IntegrationId: payment.GetIntegrationId(), SessionId: payment.GetSessionId(), IdempotencyKey: key,
		PaymentMethodToken: token, CheckoutToken: payment.GetCheckoutToken(),
		Signals: &paymentsv1.AttemptSignals{PaymentMethod: &paymentsv1.PaymentMethod{Type: "card", Fingerprint: "fingerprint-1"}},
	}
}

func NewCreateCommand(integrationID uuid.UUID, orderReference string, now time.Time) domain.CreatePayment {
	id := integrationID.String()
	return domain.CreatePayment{
		Envelope:      domain.Envelope{EventID: uuid.NewString(), Now: now},
		PaymentID:     uuid.NewString(),
		IntegrationID: id,
		CorrelationID: uuid.NewString(),
		SessionID:     uuid.NewString(),
		ReturnURL:     "https://shop.example/return",
		ExpiresAt:     now.Add(15 * time.Minute).Truncate(time.Second),
		Merchant:      &domain.MerchantRef{MerchantID: domain.MerchantID(id, "seller-01").String(), ExternalSellerID: "seller-01"},
		Customer:      &domain.CustomerRef{CustomerID: domain.CustomerID(id, "buyer-01").String(), ExternalBuyerID: "buyer-01"},
		Snapshot:      wire.Snapshot(Snapshot(orderReference, 10000)),
	}
}
