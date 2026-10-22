package workers_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/wire"

	"github.com/google/uuid"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/domain"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/testsupport"
	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/payments/v1"
)

func newStack(t *testing.T) *testsupport.Stack {
	t.Helper()
	return testsupport.NewStack(t, testsupport.StackOptions{Config: application.Config{
		PaymentExpiry: 15 * time.Minute, AttemptBudget: 2 * time.Second, FraudTimeout: 300 * time.Millisecond, ProcessorTimeout: 300 * time.Millisecond,
	}})
}

func create(t *testing.T, stack *testsupport.Stack, key string) *paymentsv1.Payment {
	t.Helper()
	payment, err := stack.Service.CreatePayment(context.Background(), testsupport.CreateRequest(testsupport.IntegrationID, key, "order-"+key, 12500))
	if err != nil {
		t.Fatal(err)
	}
	return payment
}

func status(t *testing.T, stack *testsupport.Stack, payment *paymentsv1.Payment) *paymentsv1.Payment {
	t.Helper()
	loaded, err := stack.Service.GetPayment(context.Background(), &paymentsv1.GetPaymentRequest{IntegrationId: payment.GetIntegrationId(), PaymentId: payment.GetPaymentId()})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func kinds(t *testing.T, stack *testsupport.Stack, payment *paymentsv1.Payment) []string {
	t.Helper()
	return testsupport.EventKinds(stack.DB.Events(t, payment.GetPaymentId()))
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func crashAfterIntent(t *testing.T, stack *testsupport.Stack, payment *paymentsv1.Payment, token string) (attemptID uuid.UUID, operationKey string) {
	t.Helper()
	ctx := context.Background()
	paymentID := uuid.MustParse(payment.GetPaymentId())
	attemptID = uuid.NewSHA1(paymentID, []byte("attempt:crash"))
	operationKey = "op-" + attemptID.String()
	envelope := func(version uint64) domain.Envelope {
		return domain.Envelope{EventID: uuid.NewString(), Now: stack.Clock.Now(), ExpectedVersion: version}
	}
	state, err := stack.Store.LoadPayment(ctx, paymentID)
	if err != nil {
		t.Fatal(err)
	}
	state, err = stack.Store.Transition(ctx, application.TransitionRequest{PaymentID: paymentID, Command: domain.StartAttempt{Envelope: envelope(state.Version), AttemptID: attemptID.String(), Signals: &domain.AttemptSignals{}}})
	if err != nil {
		t.Fatal(err)
	}
	state, err = stack.Store.Transition(ctx, application.TransitionRequest{PaymentID: paymentID, Command: domain.RecordRiskDecision{Envelope: envelope(state.Version), AttemptID: attemptID.String(), Assessment: wire.Assessment(testsupport.Approve())}})
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := stack.Sealer.Seal([]byte(token), attemptID[:])
	if err != nil {
		t.Fatal(err)
	}
	_, err = stack.Store.Transition(ctx, application.TransitionRequest{
		PaymentID: paymentID,
		Command:   domain.RequestProcessorOperation{Envelope: envelope(state.Version), AttemptID: attemptID.String(), OperationKey: operationKey},
		ProcessorRequest: &application.ProcessorRequestRecord{
			OperationKey: operationKey, PaymentID: paymentID, AttemptID: attemptID,
			IntegrationID: testsupport.IntegrationID, RequestHash: []byte("hash"), EncryptedToken: sealed,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return attemptID, operationKey
}

func TestRecoverySubmitsAnOperationRecordedBeforeACrashExactlyOnceWithTheSameKey(t *testing.T) {
	stack := newStack(t)
	payment := create(t, stack, "crash")
	attemptID, key := crashAfterIntent(t, stack, payment, "vault-token-crash")

	if recovered, err := stack.Workers.RecoverOnce(context.Background()); err != nil || recovered != 0 {
		t.Fatalf("recovery inside the grace period = %d, %v", recovered, err)
	}
	if stack.Processor.TotalSaleCalls() != 0 {
		t.Fatal("recovery acted before the grace period")
	}

	stack.Clock.Advance(11 * time.Second)
	for range 2 {
		if _, err := stack.Workers.RecoverOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if stack.Processor.SaleCalls(key) != 1 || stack.Processor.TokenFor(key) != "vault-token-crash" {
		t.Fatalf("sale calls = %d, token = %q; want one resubmission with the decrypted token", stack.Processor.SaleCalls(key), stack.Processor.TokenFor(key))
	}
	if got := kinds(t, stack, payment); !equal(got, []string{"created", "started", "risk", "requested", "succeeded"}) {
		t.Fatalf("events = %v", got)
	}
	if status(t, stack, payment).GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED {
		t.Fatal("payment is not succeeded")
	}
	record, err := stack.Store.ProcessorRequest(context.Background(), attemptID)
	if err != nil || len(record.EncryptedToken) != 0 {
		t.Fatalf("token not cleared after the terminal outcome: %+v, %v", record, err)
	}
}

func TestRecoveryResolvesAnUnknownOutcomeFromTheProcessorRecordWithoutResubmitting(t *testing.T) {
	for _, behavior := range []testsupport.SaleBehavior{testsupport.SaleTimeout, testsupport.SaleDecline} {
		stack := newStack(t)
		stack.Processor.SetBehavior(behavior)
		if behavior == testsupport.SaleDecline {
			stack.Processor.SetDelay(2 * time.Second)
		}
		payment := create(t, stack, "unknown")
		result, err := stack.Service.StartAttempt(context.Background(), testsupport.AttemptRequest(payment, "attempt-1", "vault-token"))
		if err != nil || result.GetStatus() != paymentsv1.AttemptStatus_ATTEMPT_STATUS_PENDING {
			t.Fatalf("result = %v, %v", result, err)
		}
		if got := kinds(t, stack, payment); !equal(got, []string{"created", "started", "risk", "requested", "unknown"}) {
			t.Fatalf("events = %v", got)
		}

		stack.Clock.Advance(11 * time.Second)
		if _, err := stack.Workers.RecoverOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		loaded := status(t, stack, payment)
		wantStatus, wantEvent := paymentsv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED, "succeeded"
		if behavior == testsupport.SaleDecline {
			wantStatus, wantEvent = paymentsv1.PaymentStatus_PAYMENT_STATUS_OPEN, "declined"
		}
		got := kinds(t, stack, payment)
		if loaded.GetStatus() != wantStatus || got[len(got)-1] != wantEvent {
			t.Fatalf("behavior %d: status = %v, events = %v", behavior, loaded.GetStatus(), got)
		}
		if stack.Processor.TotalSaleCalls() != 1 {
			t.Fatalf("recovery resubmitted a recorded operation: sale calls = %d", stack.Processor.TotalSaleCalls())
		}
		if _, err := stack.Workers.RecoverOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		if again := kinds(t, stack, payment); len(again) != len(got) {
			t.Fatalf("repeated recovery appended events: %v", again)
		}
	}
}

func TestExpiryWaitsForAPendingOutcomeAndALateSuccessWins(t *testing.T) {
	stack := newStack(t)
	stack.Processor.SetBehavior(testsupport.SaleTimeout)
	payment := create(t, stack, "late")
	if result, err := stack.Service.StartAttempt(context.Background(), testsupport.AttemptRequest(payment, "attempt-1", "vault-token")); err != nil || result.GetStatus() != paymentsv1.AttemptStatus_ATTEMPT_STATUS_PENDING {
		t.Fatalf("result = %v, %v", result, err)
	}

	stack.Clock.Advance(20 * time.Minute)
	if expired, err := stack.Workers.ExpireOnce(context.Background()); err != nil || expired != 0 {
		t.Fatalf("expiry pass = %d, %v; a pending outcome must block expiry", expired, err)
	}
	if status(t, stack, payment).GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_OPEN {
		t.Fatal("payment expired while its attempt outcome was unknown")
	}

	if _, err := stack.Workers.RecoverOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := kinds(t, stack, payment); got[len(got)-1] != "succeeded" || status(t, stack, payment).GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED {
		t.Fatalf("events = %v", got)
	}
	if _, err := stack.Workers.ExpireOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := kinds(t, stack, payment); got[len(got)-1] != "succeeded" {
		t.Fatalf("a succeeded payment expired: %v", got)
	}
}

func TestExpiryWorkerExpiresOpenPaymentsPastTheirExpiryExactlyOnce(t *testing.T) {
	stack := newStack(t)
	idle := create(t, stack, "idle")
	declined := create(t, stack, "declined")
	stack.Processor.SetBehavior(testsupport.SaleDecline)
	if _, err := stack.Service.StartAttempt(context.Background(), testsupport.AttemptRequest(declined, "attempt-1", "vault-token")); err != nil {
		t.Fatal(err)
	}

	if expired, err := stack.Workers.ExpireOnce(context.Background()); err != nil || expired != 0 {
		t.Fatalf("expired %d payments before their time: %v", expired, err)
	}
	stack.Clock.Advance(14*time.Minute + 30*time.Second)
	fresh := create(t, stack, "fresh")
	stack.Clock.Advance(time.Minute)

	var group sync.WaitGroup
	for range 3 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := stack.Workers.ExpireOnce(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()

	for _, payment := range []*paymentsv1.Payment{idle, declined} {
		if status(t, stack, payment).GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_EXPIRED {
			t.Errorf("payment %s was not expired", payment.GetPaymentId())
		}
		got := kinds(t, stack, payment)
		expiries := 0
		for _, kind := range got {
			if kind == "expired" {
				expiries++
			}
		}
		if expiries != 1 || got[len(got)-1] != "expired" {
			t.Errorf("events = %v, want exactly one trailing expiry", got)
		}
	}
	if status(t, stack, fresh).GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_OPEN {
		t.Error("a payment that is not yet due was expired")
	}
	if _, err := stack.Service.StartAttempt(context.Background(), testsupport.AttemptRequest(idle, "attempt-late", "vault-token")); err == nil {
		t.Error("attempt accepted on an expired payment")
	}
}

func TestConcurrentRecoveryRecordsTheOutcomeOnce(t *testing.T) {
	stack := newStack(t)
	payment := create(t, stack, "concurrent")
	_, key := crashAfterIntent(t, stack, payment, "vault-token")
	stack.Clock.Advance(time.Minute)

	var group sync.WaitGroup
	for range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, _ = stack.Workers.RecoverOnce(context.Background())
		}()
	}
	group.Wait()
	if got := kinds(t, stack, payment); !equal(got, []string{"created", "started", "risk", "requested", "succeeded"}) {
		t.Fatalf("events = %v", got)
	}
	if stack.Processor.Operation(key) == nil {
		t.Fatal("the processor never saw the operation")
	}
}

func TestRecoveryOfARequestedOperationWithAnUnreachableProcessorRecordsUnknown(t *testing.T) {
	stack := newStack(t)
	payment := create(t, stack, "outage")
	crashAfterIntent(t, stack, payment, "vault-token")
	stack.Processor.SetBehavior(testsupport.SaleUnavailable)
	stack.Clock.Advance(time.Minute)

	if recovered, err := stack.Workers.RecoverOnce(context.Background()); err != nil || recovered != 0 {
		t.Fatalf("recovery = %d, %v", recovered, err)
	}
	if got := kinds(t, stack, payment); got[len(got)-1] != "unknown" {
		t.Fatalf("events = %v, want the unresolved resubmission recorded as unknown", got)
	}
	stack.Processor.SetBehavior(testsupport.SaleApprove)
	stack.Clock.Advance(time.Minute)
	if _, err := stack.Workers.RecoverOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := kinds(t, stack, payment); got[len(got)-1] != "succeeded" {
		t.Fatalf("events = %v", got)
	}
}
