package testsupport

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	testredis "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/testutil/redis"

	"github.com/google/uuid"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/fraud"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/postgres"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/processor"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/redis"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application/workers"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/domain"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/metrics"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/shared/security/secretbox"
)

type Clock struct {
	mu  sync.Mutex
	now time.Time
}

func NewClock() *Clock {
	return &Clock{now: time.Now().UTC().Truncate(time.Microsecond)}
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type localElector struct{}

func (localElector) Run(ctx context.Context, work func(context.Context)) { work(ctx) }

type StackOptions struct {
	Config        application.Config
	WithRedis     bool
	RedisAddr     string
	Database      *Database
	RedisObserver redis.Observer
	RedisPrefix   string
	Clock         *Clock
}

type Stack struct {
	Redis     *testredis.Redis
	DB        *Database
	Store     *postgres.Store
	Service   *application.Service
	Workers   *workers.Workers
	Fraud     *FakeFraud
	Processor *FakeProcessor
	Cache     *redis.Cache
	Clock     *Clock
	Metrics   *metrics.Metrics
	Tokens    domain.CheckoutTokens
	Sealer    *secretbox.Box
	EncKey    []byte
	Prefix    string
}

func DefaultConfig() application.Config {
	return application.Config{
		PaymentExpiry: 15 * time.Minute, AttemptBudget: 5 * time.Second,
		FraudTimeout: time.Second, ProcessorTimeout: 3 * time.Second,
	}
}

func NewStack(t testing.TB, options StackOptions) *Stack {
	t.Helper()
	database := options.Database
	if database == nil {
		database = NewDatabase(t)
	}
	if options.Config.AttemptBudget == 0 {
		options.Config = DefaultConfig()
	}
	fraudFake, fraudConn := NewFakeFraud(t)
	processorFake, processorConn := NewFakeProcessor(t)
	key := make([]byte, secretbox.KeySize)
	_, _ = rand.Read(key)
	sealer, err := secretbox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	stack := &Stack{
		DB: database, Store: postgres.NewStore(database.App), Fraud: fraudFake, Processor: processorFake,
		Clock: options.Clock, Prefix: options.RedisPrefix, Metrics: metrics.New(), Tokens: domain.NewCheckoutTokens([]byte("checkout-token-key-for-tests-0123456789")),
		Sealer: sealer, EncKey: key,
	}
	if stack.Clock == nil {
		stack.Clock = NewClock()
	}
	if stack.Prefix == "" {
		stack.Prefix = "test:" + uuid.NewString()
	}
	var cache application.IdempotencyCache
	if options.WithRedis {
		addr := options.RedisAddr
		if addr == "" {
			stack.Redis = testredis.SetupRedis(t)
			addr = stack.Redis.Address
		}
		stack.Cache = redis.New(redis.Options{Addr: addr, KeyPrefix: stack.Prefix, TTL: time.Hour, OpTimeout: 50 * time.Millisecond, Observer: stack.Metrics})
		t.Cleanup(func() { _ = stack.Cache.Close() })
		cache = stack.Cache
	}
	stack.Service = application.NewService(options.Config, application.Dependencies{
		Store: stack.Store, Cache: cache, Fraud: fraud.NewClient(fraudConn), Processor: processor.NewClient(processorConn),
		Sealer: sealer, Tokens: stack.Tokens, Metrics: stack.Metrics, Clock: stack.Clock.Now,
	})
	stack.Workers = workers.New(workers.Config{RecoveryGrace: 10 * time.Second, Interval: time.Hour}, stack.Service, localElector{}, stack.Metrics)
	return stack
}

func (s *Stack) Restart(t testing.TB, options StackOptions) *Stack {
	t.Helper()
	options.Database = s.DB
	options.Clock = s.Clock
	options.RedisPrefix = s.Prefix
	if options.WithRedis && options.RedisAddr == "" && s.Cache != nil {
		options.RedisAddr = s.Cache.Client().Options().Addr
	}
	return NewStack(t, options)
}

func (s *Stack) FlushRedis(t testing.TB) {
	t.Helper()
	ctx := context.Background()
	client := s.Cache.Client()
	iterator := client.Scan(ctx, 0, s.Prefix+":*", 1000).Iterator()
	for iterator.Next(ctx) {
		client.Del(ctx, iterator.Val())
	}
	if err := iterator.Err(); err != nil {
		t.Fatal(err)
	}
}

func (s *Stack) RedisEntries(t testing.TB) map[string]string {
	t.Helper()
	ctx := context.Background()
	client := s.Cache.Client()
	entries := map[string]string{}
	iterator := client.Scan(ctx, 0, s.Prefix+":*", 1000).Iterator()
	for iterator.Next(ctx) {
		entries[iterator.Val()] = client.Get(ctx, iterator.Val()).Val()
	}
	if err := iterator.Err(); err != nil {
		t.Fatal(err)
	}
	return entries
}

var IntegrationID = uuid.MustParse("01234567-89ab-4cde-8fab-0123456789ab")
