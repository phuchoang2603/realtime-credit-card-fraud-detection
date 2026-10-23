package application

import (
	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/domain"
	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/payments/v1"
)

func outcomeOf(event *paymentsv1.PaymentEvent) (domain.Outcome, bool) {
	switch payload := event.GetPayload().(type) {
	case *paymentsv1.PaymentEvent_AttemptDeclined:
		return domain.Outcome{
			Type:      domain.EventAttemptDeclined,
			Status:    domain.StatusDeclined,
			AttemptID: payload.AttemptDeclined.GetAttemptId(),
			Reason:    payload.AttemptDeclined.GetReason(),
		}, true
	case *paymentsv1.PaymentEvent_PaymentSucceeded:
		return domain.Outcome{
			Type:      domain.EventPaymentSucceeded,
			Status:    domain.StatusSucceeded,
			AttemptID: payload.PaymentSucceeded.GetAttemptId(),
		}, true
	case *paymentsv1.PaymentEvent_PaymentExpired:
		return domain.Outcome{Type: domain.EventPaymentExpired, Status: domain.StatusExpired}, true
	default:
		return domain.Outcome{}, false
	}
}
