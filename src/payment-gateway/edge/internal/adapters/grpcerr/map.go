package grpcerr

import (
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Only the status code survives: server messages may carry caller data and never reach logs or responses.
func Map(err error, operation string) error {
	if err == nil {
		return nil
	}
	code := status.Code(err)
	detail := operation + ": " + code.String()
	switch code {
	case codes.InvalidArgument:
		return domain.New(domain.ErrInvalid, detail)
	case codes.Unauthenticated:
		return domain.New(domain.ErrUnauthorized, detail)
	case codes.NotFound, codes.PermissionDenied:
		return domain.New(domain.ErrNotFound, detail)
	case codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition:
		return domain.New(domain.ErrConflict, detail)
	case codes.DeadlineExceeded:
		return domain.New(domain.ErrTimeout, detail)
	case codes.Unavailable, codes.ResourceExhausted, codes.Canceled:
		return domain.New(domain.ErrUnavailable, detail)
	default:
		return domain.New(domain.ErrInternal, detail)
	}
}
