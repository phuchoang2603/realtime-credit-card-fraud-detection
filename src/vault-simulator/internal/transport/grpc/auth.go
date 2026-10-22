package grpctransport

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	vaultv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/vault/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/config"
)

const healthMethodPrefix = "/grpc.health.v1.Health/"

var methodPermissions = map[string]string{
	vaultv1.VaultService_TokenizeCard_FullMethodName:         config.PermissionTokenize,
	vaultv1.VaultService_DescribeMethod_FullMethodName:       config.PermissionDescribe,
	vaultv1.VaultService_ResolveForProcessing_FullMethodName: config.PermissionResolve,
}

type callerKey struct{}

type Authenticator struct {
	callers []config.Caller
}

func NewAuthenticator(callers []config.Caller) *Authenticator {
	return &Authenticator{callers: callers}
}

func CallerName(ctx context.Context) string {
	name, _ := ctx.Value(callerKey{}).(string)
	return name
}

func (a *Authenticator) authenticate(ctx context.Context) (config.Caller, bool) {
	values := metadata.ValueFromIncomingContext(ctx, "authorization")
	if len(values) != 1 {
		return config.Caller{}, false
	}
	credential, found := strings.CutPrefix(values[0], "Bearer ")
	if !found || credential == "" {
		return config.Caller{}, false
	}
	digest := sha256.Sum256([]byte(credential))
	var matched config.Caller
	ok := false
	for _, caller := range a.callers {
		if subtle.ConstantTimeCompare(digest[:], caller.CredentialSHA256[:]) == 1 {
			matched = caller
			ok = true
		}
	}
	return matched, ok
}

func (a *Authenticator) Unary(onDenied func(method, reason string)) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, healthMethodPrefix) {
			return handler(ctx, req)
		}
		required, known := methodPermissions[info.FullMethod]
		if !known {
			return nil, status.Error(codes.Unimplemented, "unknown method")
		}
		caller, ok := a.authenticate(ctx)
		if !ok {
			onDenied(info.FullMethod, "unauthenticated")
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		if !caller.Permissions[required] {
			onDenied(info.FullMethod, "permission_denied")
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		}
		return handler(context.WithValue(ctx, callerKey{}, caller.Name), req)
	}
}
