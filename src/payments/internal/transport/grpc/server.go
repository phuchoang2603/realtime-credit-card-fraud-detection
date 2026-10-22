package grpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/payments/gen/payments/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/domain"
)

const (
	RequestIDHeader    = "x-request-id"
	maxRequestIDLength = 128
)

type Server struct {
	paymentsv1.UnimplementedPaymentServiceServer
	service *application.Service
}

func NewServer(service *application.Service) *Server {
	return &Server{service: service}
}

func (s *Server) CreatePayment(ctx context.Context, request *paymentsv1.CreatePaymentRequest) (*paymentsv1.Payment, error) {
	payment, err := s.service.CreatePayment(ctx, request)
	return payment, toStatus(err)
}

func (s *Server) StartAttempt(ctx context.Context, request *paymentsv1.StartAttemptRequest) (*paymentsv1.AttemptResult, error) {
	result, err := s.service.StartAttempt(ctx, request)
	return result, toStatus(err)
}

func (s *Server) GetPayment(ctx context.Context, request *paymentsv1.GetPaymentRequest) (*paymentsv1.Payment, error) {
	payment, err := s.service.GetPayment(ctx, request)
	return payment, toStatus(err)
}

func (s *Server) GetCheckoutSession(ctx context.Context, request *paymentsv1.GetCheckoutSessionRequest) (*paymentsv1.Payment, error) {
	payment, err := s.service.GetCheckoutSession(ctx, request)
	return payment, toStatus(err)
}

func toStatus(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok && status.Code(err) != codes.Unknown {
		return err
	}
	var netErr net.Error
	switch {
	case errors.Is(err, application.ErrInvalidRequest), errors.Is(err, domain.ErrInvalidSnapshot):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, application.ErrIdempotencyConflict), errors.Is(err, application.ErrOrderConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, application.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, application.ErrPaymentNotOpen), errors.Is(err, application.ErrAttemptActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, application.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.As(err, &netErr):
		return status.Error(codes.Unavailable, "dependency unavailable")
	}
	return status.Error(codes.Internal, "internal error")
}

func CorrelationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requestID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(RequestIDHeader); len(values) > 0 {
				requestID = sanitize(values[0])
			}
		}
		if requestID == "" {
			requestID = uuid.NewString()
		}
		ctx = application.WithCorrelationID(ctx, requestID)
		started := time.Now()
		response, err := handler(ctx, request)
		slog.Info("payments rpc", "service", "payments", "method", info.FullMethod, "code", status.Code(err).String(),
			"duration_ms", time.Since(started).Milliseconds(), "request_id", requestID)
		return response, err
	}
}

func sanitize(value string) string {
	value = strings.TrimSpace(strings.Map(func(character rune) rune {
		if character >= 32 && character < 127 {
			return character
		}
		return -1
	}, value))
	if runes := []rune(value); len(runes) > maxRequestIDLength {
		value = string(runes[:maxRequestIDLength])
	}
	return value
}
