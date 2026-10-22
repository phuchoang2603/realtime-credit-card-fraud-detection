package redis_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/redis"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/testsupport"
)

type counters struct{ hits, misses, errors atomic.Int64 }

func (c *counters) CacheHit()   { c.hits.Add(1) }
func (c *counters) CacheMiss()  { c.misses.Add(1) }
func (c *counters) CacheError() { c.errors.Add(1) }

func newCache(t *testing.T, addr string, ttl time.Duration) (*redis.Cache, *counters, application.CacheKey) {
	t.Helper()
	observed := &counters{}
	cache := redis.New(redis.Options{Addr: addr, KeyPrefix: "test:" + uuid.NewString(), TTL: ttl, OpTimeout: 50 * time.Millisecond, Observer: observed})
	t.Cleanup(func() { _ = cache.Close() })
	return cache, observed, application.CacheKey{IntegrationID: testsupport.IntegrationID, Scope: "attempt:s", Key: "key-1"}
}

func TestConcurrentClaimsOfOneKeyAreAtomic(t *testing.T) {
	cache, observed, key := newCache(t, testsupport.RedisAddr(t), time.Minute)
	var warm sync.WaitGroup
	for range 20 {
		warm.Add(1)
		go func() {
			defer warm.Done()
			_ = cache.Client().Ping(context.Background()).Err()
		}()
	}
	warm.Wait()
	var claims atomic.Int64
	var group sync.WaitGroup
	for range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			existing, claimed, err := cache.Claim(context.Background(), key, application.CacheRecord{Digest: "abc", State: application.CacheInFlight})
			if err != nil {
				t.Error(err)
				return
			}
			if claimed {
				claims.Add(1)
			} else if existing.State != application.CacheInFlight || existing.Digest != "abc" {
				t.Errorf("existing = %+v", existing)
			}
		}()
	}
	group.Wait()
	if claims.Load() != 1 {
		t.Fatalf("%d claims, want 1", claims.Load())
	}
	if observed.hits.Load() != 19 || observed.misses.Load() != 1 {
		t.Fatalf("hits = %d, misses = %d", observed.hits.Load(), observed.misses.Load())
	}
}

func TestCompletedRecordIsReturnedAndExpiresAfterTheConfiguredTTL(t *testing.T) {
	cache, _, key := newCache(t, testsupport.RedisAddr(t), 300*time.Millisecond)
	if _, claimed, err := cache.Claim(context.Background(), key, application.CacheRecord{Digest: "abc", State: application.CacheInFlight}); err != nil || !claimed {
		t.Fatalf("claim = %v, %v", claimed, err)
	}
	completed := application.CacheRecord{Digest: "abc", State: application.CacheCompleted, AttemptID: "a-1", AttemptStatus: "ATTEMPT_STATUS_SUCCEEDED"}
	if err := cache.Complete(context.Background(), key, completed); err != nil {
		t.Fatal(err)
	}
	existing, claimed, err := cache.Claim(context.Background(), key, application.CacheRecord{Digest: "abc", State: application.CacheInFlight})
	if err != nil || claimed || existing.State != application.CacheCompleted || existing.AttemptID != "a-1" {
		t.Fatalf("claim after completion = %+v, %v, %v", existing, claimed, err)
	}
	time.Sleep(450 * time.Millisecond)
	if _, claimed, err := cache.Claim(context.Background(), key, application.CacheRecord{Digest: "abc", State: application.CacheInFlight}); err != nil || !claimed {
		t.Fatalf("record outlived its TTL: claimed = %v, %v", claimed, err)
	}
}

func TestReleaseRemovesOnlyAnInFlightClaimOfTheSameDigest(t *testing.T) {
	cache, _, key := newCache(t, testsupport.RedisAddr(t), time.Minute)
	ctx := context.Background()
	if _, _, err := cache.Claim(ctx, key, application.CacheRecord{Digest: "abc", State: application.CacheInFlight}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Release(ctx, key, "other"); err != nil {
		t.Fatal(err)
	}
	if existing, claimed, _ := cache.Claim(ctx, key, application.CacheRecord{Digest: "abc"}); claimed || existing == nil {
		t.Fatal("release with another digest removed the claim")
	}
	if err := cache.Release(ctx, key, "abc"); err != nil {
		t.Fatal(err)
	}
	if _, claimed, _ := cache.Claim(ctx, key, application.CacheRecord{Digest: "abc"}); !claimed {
		t.Fatal("matching release did not remove the claim")
	}
	if err := cache.Complete(ctx, key, application.CacheRecord{Digest: "abc", State: application.CacheCompleted}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Release(ctx, key, "abc"); err != nil {
		t.Fatal(err)
	}
	if existing, claimed, _ := cache.Claim(ctx, key, application.CacheRecord{Digest: "abc"}); claimed || existing.State != application.CacheCompleted {
		t.Fatal("release removed a completed record")
	}
}

func TestUnreachableRedisFailsFastWithinTheOperationDeadline(t *testing.T) {
	cache, observed, key := newCache(t, "127.0.0.1:1", time.Minute)
	started := time.Now()
	_, _, err := cache.Claim(context.Background(), key, application.CacheRecord{Digest: "abc", State: application.CacheInFlight})
	if err == nil {
		t.Fatal("claim against an unreachable redis succeeded")
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("claim took %v", elapsed)
	}
	if observed.errors.Load() == 0 {
		t.Fatal("redis error was not counted")
	}
}
