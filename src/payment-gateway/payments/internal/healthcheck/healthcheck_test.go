package healthcheck_test

import (
	"context"
	"net"
	"testing"
	"time"

	testtls "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/testutil/tls"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/app"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/config"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/testsupport"
	healthcheck "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/runtime"
)

func serve(t *testing.T, cfg config.Config) string {
	t.Helper()
	application, err := app.New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	grpcListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	metricsListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = application.Serve(ctx, grpcListener, metricsListener)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return grpcListener.Addr().String()
}

func TestHealthcheckOverTLSReportsLivenessAndReadiness(t *testing.T) {
	db := testsupport.NewDatabase(t)
	cert := testtls.NewSelfSignedCertificate(t, "payments.payment-gateway.svc")
	certFile, keyFile := cert.CertFile, cert.KeyFile
	addr := serve(t, config.Config{
		GRPCAddr: "127.0.0.1:0", MetricsAddr: "127.0.0.1:0", ShutdownTimeout: time.Second, DatabaseURL: db.AppURL,
		FraudAddr: "127.0.0.1:1", ProcessorAddr: "127.0.0.1:1", TLSCertFile: certFile, TLSKeyFile: keyFile,
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"), CheckoutTokenKey: []byte("checkout-token-key-for-tests-0123456789"),
		IdempotencyTTL: time.Hour, PaymentExpiry: time.Minute, AttemptBudget: time.Second, FraudTimeout: time.Second,
		ProcessorTimeout: time.Second, RecoveryGrace: time.Hour, WorkerInterval: time.Hour, RedisTimeout: 50 * time.Millisecond,
	})
	_, port, _ := net.SplitHostPort(addr)

	lookup := func(values map[string]string) func(string) string {
		return func(name string) string { return values[name] }
	}
	cfg, err := config.HealthcheckFromLookup(lookup(map[string]string{"PAYMENTS_GRPC_ADDR": ":" + port, "PAYMENTS_TLS_CERT_FILE": certFile}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "payments.payment-gateway.svc" || cfg.CAFile != certFile || cfg.Address != "127.0.0.1:"+port {
		t.Fatalf("derived healthcheck config = %+v", cfg)
	}

	testsupport.WaitFor(t, 5*time.Second, func() bool { return healthcheck.Check(context.Background(), cfg, "readiness") == nil })
	if err := healthcheck.Check(context.Background(), cfg, "liveness"); err != nil {
		t.Fatalf("liveness: %v", err)
	}

	wrongName := cfg
	wrongName.ServerName = "other.example"
	if err := healthcheck.Check(context.Background(), wrongName, "liveness"); err == nil {
		t.Fatal("healthcheck accepted a certificate for another server name")
	}
	otherCert := testtls.NewSelfSignedCertificate(t, "payments.payment-gateway.svc")
	untrusted := cfg
	untrusted.CAFile = otherCert.CertFile
	if err := healthcheck.Check(context.Background(), untrusted, "liveness"); err == nil {
		t.Fatal("healthcheck accepted a server whose certificate the CA does not vouch for")
	}
	plaintext := cfg
	plaintext.CAFile = ""
	if err := healthcheck.Check(context.Background(), plaintext, "liveness"); err == nil {
		t.Fatal("plaintext healthcheck succeeded against a TLS listener")
	}
}

func TestHealthcheckConfigDefaultsToPlaintextWithoutTLS(t *testing.T) {
	cfg, err := config.HealthcheckFromLookup(func(string) string { return "" })
	if err != nil || cfg.CAFile != "" || cfg.Address != "127.0.0.1:8000" {
		t.Fatalf("config = %+v, %v", cfg, err)
	}
}
