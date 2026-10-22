package grpctransport_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"log/slog"
	"maps"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	vaultv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/vault/v1"
	healthcheck "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/runtime"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/shared/security/secretbox"
	testpostgres "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/testutil/postgres"
	testtls "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/testutil/tls"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/adapters/postgres"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/config"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/domain"
	grpctransport "github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/transport/grpc"
)

const (
	fingerprintKey = "synthetic-fingerprint-v1"
	edgeSecret     = "edge-secret"
	processorSec   = "processor-secret"
	paymentsSecret = "payments-secret"
)

var (
	integrationA = uuid.MustParse("01234567-89ab-4cde-8fab-0123456789ab")
	integrationB = uuid.MustParse("11111111-89ab-4cde-8fab-0123456789ab")
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type harness struct {
	databaseURL string
	certFile    string
	grpcAddr    string
	logs        *syncBuffer
	client      func(secret string) vaultv1.VaultServiceClient
	conn        func(secret string) *grpc.ClientConn
}

type bearer string

func (b bearer) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	if b == "" {
		return nil, nil
	}
	return map[string]string{"authorization": "Bearer " + string(b)}, nil
}

func (bearer) RequireTransportSecurity() bool { return true }

func credentialDigest(secret string) [32]byte {
	return sha256.Sum256([]byte(secret))
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return newHarnessWithMigrations(t, true)
}

func newHarnessWithMigrations(t *testing.T, migrate bool) *harness {
	t.Helper()
	databaseURL := testpostgres.SetupPostgres(t)
	ctx := context.Background()
	if migrate {
		if err := postgres.Migrate(ctx, databaseURL); err != nil {
			t.Fatal(err)
		}
	}
	store, err := postgres.NewStore(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)

	logs := &syncBuffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	sealer, err := secretbox.New([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	cert := testtls.NewSelfSignedCertificate(t, "vault-simulator-test")
	cfg := config.Config{
		ShutdownTimeout: 2 * time.Second,
		TLSCertFile:     cert.CertFile,
		TLSKeyFile:      cert.KeyFile,
		Callers: []config.Caller{
			{Name: "edge", CredentialSHA256: credentialDigest(edgeSecret), Permissions: map[string]bool{"tokenize": true, "describe": true}},
			{Name: "processor", CredentialSHA256: credentialDigest(processorSec), Permissions: map[string]bool{"resolve": true}},
			{Name: "payments", CredentialSHA256: credentialDigest(paymentsSecret), Permissions: map[string]bool{}},
		},
	}
	server, err := grpctransport.NewServer(cfg, application.NewService(store, sealer, []byte(fingerprintKey)), store)
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
	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(runCtx, grpcListener, metricsListener) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	pem, err := os.ReadFile(cert.CertFile)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pem)
	creds := credentials.NewTLS(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13})
	connect := func(secret string) *grpc.ClientConn {
		conn, err := grpc.NewClient(grpcListener.Addr().String(),
			grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(bearer(secret)))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		return conn
	}
	return &harness{
		databaseURL: databaseURL,
		certFile:    cert.CertFile,
		grpcAddr:    grpcListener.Addr().String(),
		logs:        logs,
		conn:        connect,
		client:      func(secret string) vaultv1.VaultServiceClient { return vaultv1.NewVaultServiceClient(connect(secret)) },
	}
}

func rpcContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func tokenize(t *testing.T, client vaultv1.VaultServiceClient, integration uuid.UUID, number string) (*vaultv1.TokenizeCardResponse, error) {
	t.Helper()
	ctx, cancel := rpcContext()
	defer cancel()
	return client.TokenizeCard(ctx, &vaultv1.TokenizeCardRequest{IntegrationId: integration.String(), CardNumber: number})
}

func expectedFingerprint(cardRef string) string {
	return domain.Fingerprint([]byte(fingerprintKey), cardRef)
}

func TestTokenizeIssuesOpaqueTokensWithStableFingerprint(t *testing.T) {
	h := newHarness(t)
	edge := h.client(edgeSecret)
	cards := []struct {
		number, ref string
		behavior    vaultv1.Behavior
	}{
		{"4242424242424242", "card-0001", vaultv1.Behavior_BEHAVIOR_APPROVE},
		{"4000000000009995", "card-0002", vaultv1.Behavior_BEHAVIOR_INSUFFICIENT_FUNDS},
		{"4000000000000002", "card-0003", vaultv1.Behavior_BEHAVIOR_ISSUER_DECLINE},
		{"4000000000000069", "card-0004", vaultv1.Behavior_BEHAVIOR_INVALID},
	}
	processor := h.client(processorSec)
	for _, card := range cards {
		first, err := tokenize(t, edge, integrationA, card.number)
		if err != nil {
			t.Fatal(err)
		}
		second, err := tokenize(t, edge, integrationA, card.number)
		if err != nil {
			t.Fatal(err)
		}
		if first.GetToken() == second.GetToken() {
			t.Fatalf("%s: reissued token is identical", card.ref)
		}
		want := expectedFingerprint(card.ref)
		for _, response := range []*vaultv1.TokenizeCardResponse{first, second} {
			token := response.GetToken()
			if !strings.HasPrefix(token, "vt_") || len(token) < 40 {
				t.Fatalf("unexpected token shape %q", token)
			}
			for _, leaked := range []string{card.ref, card.number, "ok", "nsf", "issuer", "invalid"} {
				if strings.Contains(strings.TrimPrefix(token, "vt_"), leaked) && len(leaked) > 3 {
					t.Fatalf("token %q embeds %q", token, leaked)
				}
			}
			if response.GetMetadata().GetType() != "card" || response.GetMetadata().GetFingerprint() != want {
				t.Fatalf("%s: metadata %v, want fingerprint %s", card.ref, response.GetMetadata(), want)
			}
			ctx, cancel := rpcContext()
			described, err := edge.DescribeMethod(ctx, &vaultv1.DescribeMethodRequest{IntegrationId: integrationA.String(), Token: token})
			cancel()
			if err != nil || described.GetFingerprint() != want || described.GetType() != "card" {
				t.Fatalf("describe: %v %v", described, err)
			}
			ctx, cancel = rpcContext()
			fixture, err := processor.ResolveForProcessing(ctx, &vaultv1.ResolveRequest{IntegrationId: integrationA.String(), Token: token})
			cancel()
			if err != nil || fixture.GetCardReference() != card.ref || fixture.GetBehavior() != card.behavior {
				t.Fatalf("resolve: %v %v", fixture, err)
			}
		}
	}
}

func TestTokenizeRejectsNumbersOutsideAllowList(t *testing.T) {
	h := newHarness(t)
	edge := h.client(edgeSecret)
	for _, number := range []string{"4111111111111111", "5555555555554444", "4242424242424243", "", "4242 4242 4242 4242", "4111000000000000000"} {
		_, err := tokenize(t, edge, integrationA, number)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("%q: got %v, want InvalidArgument", number, err)
		}
	}
	_, err := tokenize(t, edge, integrationA, "4242424242424242")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := rpcContext()
	defer cancel()
	_, err = edge.TokenizeCard(ctx, &vaultv1.TokenizeCardRequest{IntegrationId: "not-a-uuid", CardNumber: "4242424242424242"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad integration: %v", err)
	}

	pool, err := pgxpool.New(ctx, h.databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM tokens").Scan(&count); err != nil || count != 1 {
		t.Fatalf("rejected numbers stored rows: count=%d err=%v", count, err)
	}
}

func TestForeignIntegrationTokenIsNotFound(t *testing.T) {
	h := newHarness(t)
	edge := h.client(edgeSecret)
	processor := h.client(processorSec)
	issued, err := tokenize(t, edge, integrationA, "4242424242424242")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := rpcContext()
	defer cancel()
	_, err = edge.DescribeMethod(ctx, &vaultv1.DescribeMethodRequest{IntegrationId: integrationB.String(), Token: issued.GetToken()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("describe foreign token: %v", err)
	}
	_, err = processor.ResolveForProcessing(ctx, &vaultv1.ResolveRequest{IntegrationId: integrationB.String(), Token: issued.GetToken()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("resolve foreign token: %v", err)
	}
	_, err = edge.DescribeMethod(ctx, &vaultv1.DescribeMethodRequest{IntegrationId: integrationA.String(), Token: "vt_unknown"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("describe unknown token: %v", err)
	}
}

func TestInactiveTokenIsNotFound(t *testing.T) {
	h := newHarness(t)
	edge := h.client(edgeSecret)
	issued, err := tokenize(t, edge, integrationA, "4242424242424242")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := rpcContext()
	defer cancel()
	pool, err := pgxpool.New(ctx, h.databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "UPDATE tokens SET active = false"); err != nil {
		t.Fatal(err)
	}
	_, err = edge.DescribeMethod(ctx, &vaultv1.DescribeMethodRequest{IntegrationId: integrationA.String(), Token: issued.GetToken()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("inactive token: %v", err)
	}
}

func TestUnauthorizedCallsAreDenied(t *testing.T) {
	h := newHarness(t)
	issued, err := tokenize(t, h.client(edgeSecret), integrationA, "4242424242424242")
	if err != nil {
		t.Fatal(err)
	}
	token := issued.GetToken()
	calls := map[string]func(ctx context.Context, c vaultv1.VaultServiceClient) error{
		"tokenize": func(ctx context.Context, c vaultv1.VaultServiceClient) error {
			_, err := c.TokenizeCard(ctx, &vaultv1.TokenizeCardRequest{IntegrationId: integrationA.String(), CardNumber: "4242424242424242"})
			return err
		},
		"describe": func(ctx context.Context, c vaultv1.VaultServiceClient) error {
			_, err := c.DescribeMethod(ctx, &vaultv1.DescribeMethodRequest{IntegrationId: integrationA.String(), Token: token})
			return err
		},
		"resolve": func(ctx context.Context, c vaultv1.VaultServiceClient) error {
			_, err := c.ResolveForProcessing(ctx, &vaultv1.ResolveRequest{IntegrationId: integrationA.String(), Token: token})
			return err
		},
	}
	tests := []struct {
		caller  string
		secret  string
		allowed string
		other   codes.Code
	}{
		{"edge", edgeSecret, "tokenize,describe", codes.PermissionDenied},
		{"processor", processorSec, "resolve", codes.PermissionDenied},
		{"payments", paymentsSecret, "", codes.PermissionDenied},
		{"unknown credential", "wrong-secret", "", codes.Unauthenticated},
		{"no credential", "", "", codes.Unauthenticated},
	}
	for _, test := range tests {
		client := h.client(test.secret)
		for name, call := range calls {
			t.Run(test.caller+"/"+name, func(t *testing.T) {
				ctx, cancel := rpcContext()
				defer cancel()
				err := call(ctx, client)
				if strings.Contains(test.allowed, name) {
					if err != nil {
						t.Fatalf("expected success, got %v", err)
					}
					return
				}
				if status.Code(err) != test.other {
					t.Fatalf("got %v, want %v", err, test.other)
				}
				if strings.Contains(status.Convert(err).Message(), "card-") {
					t.Fatalf("denial leaks fixture details: %v", err)
				}
			})
		}
	}
}

func TestNoPlaintextFixtureOrCardNumberInStorageOrLogs(t *testing.T) {
	h := newHarness(t)
	edge := h.client(edgeSecret)
	processor := h.client(processorSec)
	number := "4000000000009995"
	issued, err := tokenize(t, edge, integrationA, number)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := rpcContext()
	defer cancel()
	if _, err := processor.ResolveForProcessing(ctx, &vaultv1.ResolveRequest{IntegrationId: integrationA.String(), Token: issued.GetToken()}); err != nil {
		t.Fatal(err)
	}
	_, _ = tokenize(t, edge, integrationA, "4111111111111111")
	_, _ = edge.DescribeMethod(ctx, &vaultv1.DescribeMethodRequest{IntegrationId: integrationB.String(), Token: issued.GetToken()})

	pool, err := pgxpool.New(ctx, h.databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	rows, err := pool.Query(ctx, "SELECT t::text, encode(t.fixture_ciphertext, 'escape'), encode(t.token_digest, 'hex') FROM tokens t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var stored strings.Builder
	rowCount := 0
	for rows.Next() {
		var rowText, ciphertext, digest string
		if err := rows.Scan(&rowText, &ciphertext, &digest); err != nil {
			t.Fatal(err)
		}
		stored.WriteString(rowText + ciphertext)
		rowCount++
		sum := sha256.Sum256([]byte(issued.GetToken()))
		if digest != hex.EncodeToString(sum[:]) {
			t.Fatal("stored digest is not sha256 of token")
		}
	}
	if rowCount != 1 {
		t.Fatalf("got %d rows", rowCount)
	}
	for _, forbidden := range []string{number, "card-0002", "card_reference", "behavior", "nsf", issued.GetToken(), strings.TrimPrefix(issued.GetToken(), "vt_"), "4111111111111111"} {
		if strings.Contains(stored.String(), forbidden) {
			t.Fatalf("storage contains %q", forbidden)
		}
	}
	for _, forbidden := range []string{number, "4111111111111111", issued.GetToken(), strings.TrimPrefix(issued.GetToken(), "vt_"), "card-0002", edgeSecret, processorSec} {
		if strings.Contains(h.logs.String(), forbidden) {
			t.Fatalf("logs contain %q", forbidden)
		}
	}
	if h.logs.String() == "" {
		t.Fatal("expected request logs to be captured")
	}
}

func TestHealthServicesAreServedWithoutCredentials(t *testing.T) {
	h := newHarness(t)
	health := healthpb.NewHealthClient(h.conn(""))
	for _, name := range []string{grpctransport.LivenessService, grpctransport.ReadinessService} {
		deadline := time.Now().Add(10 * time.Second)
		for {
			ctx, cancel := rpcContext()
			response, err := health.Check(ctx, &healthpb.HealthCheckRequest{Service: name})
			cancel()
			if err == nil && response.GetStatus() == healthpb.HealthCheckResponse_SERVING {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s never SERVING: %v %v", name, response, err)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestReadinessRequiresMigratedDatabase(t *testing.T) {
	databaseURL := testpostgres.SetupPostgres(t)
	ctx := context.Background()
	store, err := postgres.NewStore(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err == nil {
		t.Fatal("ready before migrations")
	}
	if err := postgres.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err != nil {
		t.Fatalf("not ready after migrations: %v", err)
	}
	store.Close()
	if err := store.Ready(ctx); err == nil {
		t.Fatal("ready with closed pool")
	}
}

func TestHealthcheckCommandExitSemantics(t *testing.T) {
	ready := newHarness(t)
	unready := newHarnessWithMigrations(t, false)

	lookupFor := func(h *harness, extra map[string]string) func(string) string {
		values := map[string]string{"VAULT_GRPC_ADDR": h.grpcAddr, "VAULT_TLS_CERT_FILE": h.certFile}
		maps.Copy(values, extra)
		return func(name string) string { return values[name] }
	}
	waitFor := func(h *harness, service string, want healthpb.HealthCheckResponse_ServingStatus) {
		t.Helper()
		health := healthpb.NewHealthClient(h.conn(""))
		deadline := time.Now().Add(10 * time.Second)
		for {
			ctx, cancel := rpcContext()
			response, err := health.Check(ctx, &healthpb.HealthCheckRequest{Service: service})
			cancel()
			if err == nil && response.GetStatus() == want {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s never %s", service, want)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	waitFor(ready, grpctransport.ReadinessService, healthpb.HealthCheckResponse_SERVING)
	waitFor(unready, grpctransport.LivenessService, healthpb.HealthCheckResponse_SERVING)
	waitFor(unready, grpctransport.ReadinessService, healthpb.HealthCheckResponse_NOT_SERVING)

	otherCert := testtls.NewSelfSignedCertificate(t, "vault-simulator-test")
	tests := []struct {
		name    string
		harness *harness
		service string
		extra   map[string]string
		wantErr bool
	}{
		{"liveness serving", ready, "liveness", nil, false},
		{"readiness serving", ready, "readiness", nil, false},
		{"explicit CA and server name", ready, "readiness", map[string]string{"VAULT_HEALTHCHECK_CA_FILE": ready.certFile, "VAULT_HEALTHCHECK_SERVER_NAME": "localhost"}, false},
		{"CA from VAULT_TLS_CA_FILE", ready, "readiness", map[string]string{"VAULT_TLS_CA_FILE": ready.certFile}, false},
		{"liveness serving while not ready", unready, "liveness", nil, false},
		{"readiness not serving", unready, "readiness", nil, true},
		{"unknown service", ready, "other", nil, true},
		{"untrusted CA", ready, "liveness", map[string]string{"VAULT_HEALTHCHECK_CA_FILE": otherCert.CertFile}, true},
		{"wrong server name", ready, "liveness", map[string]string{"VAULT_HEALTHCHECK_SERVER_NAME": "vault.invalid"}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := config.HealthcheckFromLookup(lookupFor(test.harness, test.extra))
			if err != nil {
				t.Fatal(err)
			}
			err = healthcheck.Check(context.Background(), cfg, test.service)
			if (err != nil) != test.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, test.wantErr)
			}
		})
	}

	t.Run("nothing listening", func(t *testing.T) {
		cfg, err := config.HealthcheckFromLookup(lookupFor(ready, map[string]string{"VAULT_GRPC_ADDR": "127.0.0.1:1", "VAULT_HEALTHCHECK_TIMEOUT": "500ms"}))
		if err != nil {
			t.Fatal(err)
		}
		if err := healthcheck.Check(context.Background(), cfg, "liveness"); err == nil {
			t.Fatal("expected failure")
		}
	})
}
