package postgres_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/adapters/postgres"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/testutil"
)

func TestReadinessRequiresLatestMigration(t *testing.T) {
	pool, dbURL := testutil.NewPool(t)
	readiness, err := postgres.NewReadiness(pool)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = readiness.Close() }()
	ctx := context.Background()
	if err := readiness.Check(ctx); err != nil {
		t.Fatalf("migrated database must be ready: %v", err)
	}
	if err := postgres.Migrate(ctx, dbURL); err != nil {
		t.Fatalf("second migrate must be a no-op: %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM goose_db_version WHERE version_id = (SELECT max(version_id) FROM goose_db_version)"); err != nil {
		t.Fatal(err)
	}
	if err := readiness.Check(ctx); err == nil {
		t.Fatal("database without the latest migration must not be ready")
	}
}

func TestReadinessFailsWhenDatabaseUnreachable(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://nobody:nobody@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	readiness, err := postgres.NewReadiness(pool)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := readiness.Check(ctx); err == nil {
		t.Fatal("expected error")
	}
}

func TestOnlyOneLeaderRunsAtATime(t *testing.T) {
	pool, _ := testutil.NewPool(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var active, maxActive atomic.Int32
	run := func(ctx context.Context) {
		now := active.Add(1)
		for {
			if current := maxActive.Load(); now <= current || maxActive.CompareAndSwap(current, now) {
				break
			}
		}
		<-ctx.Done()
		active.Add(-1)
	}
	first := make(chan struct{})
	go func() {
		postgres.NewLeader(pool, postgres.LeaderLockKey).Run(ctx, func(ctx context.Context) {
			close(first)
			run(ctx)
		})
	}()
	<-first
	secondCtx, secondCancel := context.WithCancel(ctx)
	secondDone := make(chan struct{})
	go func() {
		postgres.NewLeader(pool, postgres.LeaderLockKey).Run(secondCtx, run)
		close(secondDone)
	}()
	time.Sleep(1500 * time.Millisecond)
	if got := active.Load(); got != 1 {
		t.Fatalf("active leaders = %d, want 1", got)
	}
	secondCancel()
	<-secondDone
	if maxActive.Load() != 1 {
		t.Fatalf("max concurrent leaders = %d", maxActive.Load())
	}
}
