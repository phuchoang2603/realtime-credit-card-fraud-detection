package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application"
)

type Observer interface {
	CacheHit()
	CacheMiss()
	CacheError()
}

type Options struct {
	Addr      string
	KeyPrefix string
	Password  string
	TTL       time.Duration
	OpTimeout time.Duration
	Observer  Observer
}

type Cache struct {
	client    *goredis.Client
	prefix    string
	ttl       time.Duration
	opTimeout time.Duration
	observer  Observer
}

var _ application.IdempotencyCache = (*Cache)(nil)

func New(options Options) *Cache {
	client := goredis.NewClient(&goredis.Options{
		Addr:            options.Addr,
		Password:        options.Password,
		DialTimeout:     options.OpTimeout,
		ReadTimeout:     options.OpTimeout,
		WriteTimeout:    options.OpTimeout,
		MaxRetries:      -1,
		PoolSize:        32,
		DisableIdentity: true,
	})
	prefix := options.KeyPrefix
	if prefix == "" {
		prefix = "payments:idem"
	}
	return &Cache{client: client, prefix: prefix, ttl: options.TTL, opTimeout: options.OpTimeout, observer: options.Observer}
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Client() *goredis.Client {
	return c.client
}

func (c *Cache) redisKey(key application.CacheKey) string {
	return fmt.Sprintf("%s:%s:%s:%s", c.prefix, key.IntegrationID, key.Scope, key.Key)
}

func (c *Cache) Claim(ctx context.Context, key application.CacheKey, record application.CacheRecord) (*application.CacheRecord, bool, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, false, fmt.Errorf("encode idempotency record: %w", err)
	}
	setCtx, cancelSet := context.WithTimeout(ctx, c.opTimeout)
	claimed, err := c.client.SetNX(setCtx, c.redisKey(key), encoded, c.ttl).Result()
	cancelSet()
	if err != nil {
		c.observer.CacheError()
		return nil, false, fmt.Errorf("claim idempotency key: %w", err)
	}
	if claimed {
		c.observer.CacheMiss()
		return nil, true, nil
	}
	getCtx, cancelGet := context.WithTimeout(ctx, c.opTimeout)
	defer cancelGet()
	raw, err := c.client.Get(getCtx, c.redisKey(key)).Bytes()
	if errors.Is(err, goredis.Nil) {
		c.observer.CacheMiss()
		return nil, false, errors.New("idempotency record vanished after claim attempt")
	}
	if err != nil {
		c.observer.CacheError()
		return nil, false, fmt.Errorf("read idempotency record: %w", err)
	}
	existing := &application.CacheRecord{}
	if err := json.Unmarshal(raw, existing); err != nil {
		c.observer.CacheError()
		return nil, false, fmt.Errorf("decode idempotency record: %w", err)
	}
	c.observer.CacheHit()
	return existing, false, nil
}

func (c *Cache) Complete(ctx context.Context, key application.CacheKey, record application.CacheRecord) error {
	ctx, cancel := context.WithTimeout(ctx, c.opTimeout)
	defer cancel()
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode idempotency record: %w", err)
	}
	if err := c.client.Set(ctx, c.redisKey(key), encoded, c.ttl).Err(); err != nil {
		c.observer.CacheError()
		return fmt.Errorf("complete idempotency record: %w", err)
	}
	return nil
}

var releaseScript = goredis.NewScript(`
local current = redis.call('GET', KEYS[1])
if not current then return 0 end
local record = cjson.decode(current)
if record['state'] == 'in_flight' and record['digest'] == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0`)

func (c *Cache) Release(ctx context.Context, key application.CacheKey, digest string) error {
	ctx, cancel := context.WithTimeout(ctx, c.opTimeout)
	defer cancel()
	if err := releaseScript.Run(ctx, c.client, []string{c.redisKey(key)}, digest).Err(); err != nil && !errors.Is(err, goredis.Nil) {
		c.observer.CacheError()
		return fmt.Errorf("release idempotency record: %w", err)
	}
	return nil
}
