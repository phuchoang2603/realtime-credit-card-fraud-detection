package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/fraud"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/postgres"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/processor"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/redis"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application/workers"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/config"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/domain"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/metrics"
	transport "github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/transport/grpc"
	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/payments/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/shared/security/secretbox"
)

const (
	readinessInterval = 2 * time.Second
	leaderLockID      = 0x7061796d656e7473
)

type App struct {
	config    config.Config
	pool      *pgxpool.Pool
	metrics   *metrics.Metrics
	cache     *redis.Cache
	conns     []*grpc.ClientConn
	service   *application.Service
	workers   *workers.Workers
	tlsConfig *tls.Config
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	app := &App{config: cfg, pool: pool, metrics: metrics.New()}
	if cfg.TLSCertFile != "" {
		certificate, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("load gRPC server certificate: %w", err)
		}
		app.tlsConfig = &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS13}
	}
	app.metrics.Registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	sealer, err := secretbox.New(cfg.EncryptionKey)
	if err != nil {
		app.Close()
		return nil, err
	}
	var cache application.IdempotencyCache
	if cfg.RedisAddr != "" {
		app.cache = redis.New(redis.Options{
			Addr: cfg.RedisAddr, Password: cfg.RedisPassword, TTL: cfg.IdempotencyTTL, OpTimeout: cfg.RedisTimeout, Observer: app.metrics,
		})
		cache = app.cache
	}
	fraudConn, err := dial(cfg.FraudAddr)
	if err != nil {
		app.Close()
		return nil, fmt.Errorf("create fraud client: %w", err)
	}
	app.conns = append(app.conns, fraudConn)
	processorConn, err := dial(cfg.ProcessorAddr)
	if err != nil {
		app.Close()
		return nil, fmt.Errorf("create processor client: %w", err)
	}
	app.conns = append(app.conns, processorConn)

	app.service = application.NewService(application.Config{
		PaymentExpiry: cfg.PaymentExpiry, AttemptBudget: cfg.AttemptBudget,
		FraudTimeout: cfg.FraudTimeout, ProcessorTimeout: cfg.ProcessorTimeout,
	}, application.Dependencies{
		Store: postgres.NewStore(pool), Cache: cache, Fraud: fraud.NewClient(fraudConn), Processor: processor.NewClient(processorConn),
		Sealer: sealer, Tokens: domain.NewCheckoutTokens(cfg.CheckoutTokenKey), Metrics: app.metrics,
	})
	app.workers = workers.New(workers.Config{RecoveryGrace: cfg.RecoveryGrace, Interval: cfg.WorkerInterval}, app.service,
		postgres.NewLeader(pool, leaderLockID, cfg.WorkerInterval), app.metrics)
	return app, nil
}

func dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
}

func (a *App) Service() *application.Service {
	return a.service
}

func (a *App) Close() {
	for _, conn := range a.conns {
		_ = conn.Close()
	}
	if a.cache != nil {
		_ = a.cache.Close()
	}
	a.pool.Close()
}

func (a *App) Serve(ctx context.Context, grpcListener, metricsListener net.Listener) error {
	defer a.Close()
	healthServer := health.NewServer()
	healthServer.SetServingStatus(transport.LivenessService, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(transport.ReadinessService, healthpb.HealthCheckResponse_NOT_SERVING)

	serverOptions := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(transport.CorrelationInterceptor()),
	}
	if a.tlsConfig != nil {
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(a.tlsConfig)))
	}
	grpcServer := grpc.NewServer(serverOptions...)
	paymentsv1.RegisterPaymentServiceServer(grpcServer, transport.NewServer(a.service))
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	metricsServer := &http.Server{
		Handler:           metricsMux(a.metrics.Registry),
		ReadHeaderTimeout: 5 * time.Second,
	}

	background, stopBackground := context.WithCancel(context.WithoutCancel(ctx))
	var group sync.WaitGroup
	run := func(work func()) {
		group.Add(1)
		go func() {
			defer group.Done()
			work()
		}()
	}
	run(func() {
		transport.WatchReadiness(background, healthServer, readinessInterval, func(ctx context.Context) error {
			return postgres.Ready(ctx, a.pool)
		})
	})
	run(func() { a.workers.Run(background) })

	errCh := make(chan error, 2)
	go func() { errCh <- grpcServer.Serve(grpcListener) }()
	go func() {
		if err := metricsServer.Serve(metricsListener); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	var serveErr error
	select {
	case serveErr = <-errCh:
	case <-ctx.Done():
	}

	healthServer.Shutdown()
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
	defer cancel()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		slog.Warn("graceful stop timed out; forcing stop")
		grpcServer.Stop()
		<-stopped
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		_ = metricsServer.Close()
	}
	stopBackground()
	group.Wait()
	return serveErr
}

func metricsMux(registry *prometheus.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return mux
}
