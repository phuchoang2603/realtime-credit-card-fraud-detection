package grpcclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/requestid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const requestIDMetadata = "x-request-id"

// Interceptor bounds every outbound call by timeout and propagates the request ID.
func Interceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if id := requestid.From(ctx); id != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, requestIDMetadata, id)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

type bearer struct{ token string }

func (b bearer) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + b.token}, nil
}

func (bearer) RequireTransportSecurity() bool { return true }

func BearerCredentials(token string) credentials.PerRPCCredentials { return bearer{token: token} }

func TLSFromCAFile(path string) (credentials.TransportCredentials, error) {
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("CA file %s holds no certificates", path)
	}
	return credentials.NewTLS(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}), nil
}

func Dial(target string, timeout time.Duration, transport credentials.TransportCredentials, extra ...grpc.DialOption) (*grpc.ClientConn, error) {
	if transport == nil {
		transport = insecure.NewCredentials()
	}
	options := append([]grpc.DialOption{
		grpc.WithTransportCredentials(transport),
		grpc.WithChainUnaryInterceptor(Interceptor(timeout)),
	}, extra...)
	return grpc.NewClient(target, options...)
}
