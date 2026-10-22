package grpctransport

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	processorv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/gen/processor/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/domain"
)

const (
	LivenessService  = "liveness"
	ReadinessService = "readiness"
)

type Processor interface {
	Sale(ctx context.Context, sale domain.Sale) (domain.Operation, error)
	GetOperation(ctx context.Context, operationKey string) (domain.Operation, error)
}

type Readiness interface {
	Ready(ctx context.Context) error
}

type Options struct {
	ShutdownTimeout   time.Duration
	ReadinessInterval time.Duration
}

type Server struct {
	processorv1.UnimplementedProcessorServiceServer
	processor Processor
	readiness Readiness
	options   Options
	health    *health.Server
	registry  *prometheus.Registry
	requests  *prometheus.CounterVec
	outcomes  *prometheus.CounterVec
}

func NewServer(options Options, processor Processor, readiness Readiness) *Server {
	server := &Server{
		processor: processor,
		readiness: readiness,
		options:   options,
		health:    health.NewServer(),
		registry:  prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "processor_requests_total", Help: "Processor gRPC requests by method and status code.",
		}, []string{"method", "code"}),
		outcomes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "processor_sale_outcomes_total", Help: "Sale outcomes returned to callers.",
		}, []string{"outcome"}),
	}
	server.registry.MustRegister(server.requests, server.outcomes)
	server.health.SetServingStatus(LivenessService, healthpb.HealthCheckResponse_SERVING)
	server.health.SetServingStatus(ReadinessService, healthpb.HealthCheckResponse_NOT_SERVING)
	return server
}

func (s *Server) Run(ctx context.Context, grpcListener, metricsListener net.Listener) error {
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(s.observe),
	)
	processorv1.RegisterProcessorServiceServer(grpcServer, s)
	healthpb.RegisterHealthServer(grpcServer, s.health)

	metrics := http.NewServeMux()
	metrics.Handle("/metrics", promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{}))
	metricsServer := &http.Server{Handler: metrics, ReadHeaderTimeout: 5 * time.Second}

	errs := make(chan error, 2)
	go func() { errs <- grpcServer.Serve(grpcListener) }()
	go func() {
		if err := metricsServer.Serve(metricsListener); !errors.Is(err, http.ErrServerClosed) {
			errs <- err
			return
		}
		errs <- nil
	}()
	healthCtx, stopHealth := context.WithCancel(ctx)
	defer stopHealth()
	go s.pollReadiness(healthCtx)

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-errs:
	}
	s.health.Shutdown()
	stopHealth()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.options.ShutdownTimeout)
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		_ = metricsServer.Close()
	}
	return runErr
}

func (s *Server) pollReadiness(ctx context.Context) {
	check := func() {
		checkCtx, cancel := context.WithTimeout(ctx, s.options.ReadinessInterval)
		defer cancel()
		state := healthpb.HealthCheckResponse_SERVING
		if err := s.readiness.Ready(checkCtx); err != nil {
			state = healthpb.HealthCheckResponse_NOT_SERVING
		}
		if ctx.Err() == nil {
			s.health.SetServingStatus(ReadinessService, state)
		}
	}
	check()
	ticker := time.NewTicker(s.options.ReadinessInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

func (s *Server) observe(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	code := status.Code(err)
	s.requests.WithLabelValues(info.FullMethod, code.String()).Inc()
	slog.InfoContext(ctx, "processor request",
		"method", info.FullMethod,
		"code", code.String(),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return resp, err
}

func (s *Server) Sale(ctx context.Context, req *processorv1.SaleRequest) (*processorv1.Operation, error) {
	operation, err := s.processor.Sale(ctx, domain.Sale{
		OperationKey:    req.GetOperationKey(),
		IntegrationID:   req.GetIntegrationId(),
		Token:           req.GetPaymentMethodToken(),
		AmountMinorUnit: req.GetAmountMinorUnits(),
		Currency:        req.GetCurrency(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	s.outcomes.WithLabelValues(string(operation.Outcome)).Inc()
	return toProto(operation), nil
}

func (s *Server) GetOperation(ctx context.Context, req *processorv1.GetOperationRequest) (*processorv1.Operation, error) {
	operation, err := s.processor.GetOperation(ctx, req.GetOperationKey())
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(operation), nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, application.ErrInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, application.ErrUnknownToken):
		return status.Error(codes.InvalidArgument, application.ErrUnknownToken.Error())
	case errors.Is(err, application.ErrKeyConflict):
		return status.Error(codes.AlreadyExists, application.ErrKeyConflict.Error())
	case errors.Is(err, application.ErrNotFound):
		return status.Error(codes.NotFound, "operation not found")
	case errors.Is(err, application.ErrVaultUnavailable):
		return status.Error(codes.Unavailable, "vault is unavailable")
	case errors.Is(err, application.ErrInjectedTimeout), errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "canceled")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func toProto(operation domain.Operation) *processorv1.Operation {
	outcome := processorv1.OperationOutcome_OPERATION_OUTCOME_APPROVED
	if operation.Outcome == domain.OutcomeDeclined {
		outcome = processorv1.OperationOutcome_OPERATION_OUTCOME_DECLINED
	}
	return &processorv1.Operation{
		OperationKey:       operation.Key,
		Outcome:            outcome,
		DeclineReason:      operation.DeclineReason,
		ProcessorReference: operation.ProcessorReference,
		RecordedAt:         timestamppb.New(operation.RecordedAt),
	}
}
