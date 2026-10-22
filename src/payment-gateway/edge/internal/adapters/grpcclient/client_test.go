package grpcclient_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/adapters/grpcclient"
	vaultadapter "github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/adapters/vault"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/domain"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/requestid"
	vaultv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/vault/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type recordingVault struct {
	vaultv1.UnimplementedVaultServiceServer
	authorization chan string
	requestID     chan string
	block         bool
}

func (v recordingVault) DescribeMethod(ctx context.Context, _ *vaultv1.DescribeMethodRequest) (*vaultv1.MethodMetadata, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	v.authorization <- first(md.Get("authorization"))
	v.requestID <- first(md.Get("x-request-id"))
	if v.block {
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	return &vaultv1.MethodMetadata{Type: "card", Fingerprint: "fp"}, nil
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func writeCA(t *testing.T) (certPath string, serverCert tls.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "vault.test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPath = filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func startVault(t *testing.T, serverCert tls.Certificate, vault recordingVault) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{serverCert}, MinVersion: tls.VersionTLS12})))
	vaultv1.RegisterVaultServiceServer(server, vault)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return listener.Addr().String()
}

func TestVaultCallsUseVerifiedTLSBearerCredentialAndRequestID(t *testing.T) {
	caPath, cert := writeCA(t)
	vault := recordingVault{authorization: make(chan string, 4), requestID: make(chan string, 4)}
	address := startVault(t, cert, vault)

	transport, err := grpcclient.TLSFromCAFile(caPath)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := grpcclient.Dial(address, 3*time.Second, transport, grpc.WithPerRPCCredentials(grpcclient.BearerCredentials("edge-secret")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	client := vaultadapter.New(conn)
	method, err := client.Describe(requestid.With(context.Background(), "req-9"), "int", "tok")
	if err != nil || method.Fingerprint != "fp" {
		t.Fatalf("describe = %+v, %v", method, err)
	}
	if got := <-vault.authorization; got != "Bearer edge-secret" {
		t.Fatalf("authorization = %q", got)
	}
	if got := <-vault.requestID; got != "req-9" {
		t.Fatalf("request id = %q", got)
	}
}

func TestVaultCertificateIsVerifiedAgainstTheConfiguredCA(t *testing.T) {
	_, cert := writeCA(t)
	otherCA, _ := writeCA(t)
	vault := recordingVault{authorization: make(chan string, 4), requestID: make(chan string, 4)}
	address := startVault(t, cert, vault)

	transport, err := grpcclient.TLSFromCAFile(otherCA)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := grpcclient.Dial(address, 2*time.Second, transport, grpc.WithPerRPCCredentials(grpcclient.BearerCredentials("edge-secret")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := vaultadapter.New(conn).Describe(context.Background(), "int", "tok"); err == nil {
		t.Fatal("an untrusted certificate must fail")
	}
	select {
	case <-vault.authorization:
		t.Fatal("the credential was sent over an unverified connection")
	default:
	}
}

func TestBearerCredentialRefusesPlaintextTransport(t *testing.T) {
	if !grpcclient.BearerCredentials("x").RequireTransportSecurity() {
		t.Fatal("the bearer credential must require transport security")
	}
	if _, err := grpcclient.TLSFromCAFile(filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Fatal("a missing CA file must fail")
	}
	empty := filepath.Join(t.TempDir(), "empty.pem")
	if err := os.WriteFile(empty, []byte("nothing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := grpcclient.TLSFromCAFile(empty); err == nil {
		t.Fatal("a CA file without certificates must fail")
	}
}

func TestEveryOutboundCallHasAGatewaySideDeadline(t *testing.T) {
	caPath, cert := writeCA(t)
	vault := recordingVault{authorization: make(chan string, 4), requestID: make(chan string, 4), block: true}
	address := startVault(t, cert, vault)
	transport, _ := grpcclient.TLSFromCAFile(caPath)
	conn, err := grpcclient.Dial(address, 150*time.Millisecond, transport, grpc.WithPerRPCCredentials(grpcclient.BearerCredentials("edge-secret")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	start := time.Now()
	_, err = vaultadapter.New(conn).Describe(context.Background(), "int", "tok")
	if err == nil || time.Since(start) > 3*time.Second {
		t.Fatalf("err = %v after %s", err, time.Since(start))
	}
	if !isTimeout(err) {
		t.Fatalf("err = %v, want a timeout", err)
	}
}

func isTimeout(err error) bool {
	return status.Code(err) == codes.DeadlineExceeded || errorsIs(err, domain.ErrTimeout)
}

func errorsIs(err, target error) bool { return errors.Is(err, target) }
