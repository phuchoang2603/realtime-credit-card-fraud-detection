package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const LeaderLockKey int64 = 0x6e6f74696679

type Leader struct {
	pool      *pgxpool.Pool
	key       int64
	retry     time.Duration
	heartbeat time.Duration
}

func NewLeader(pool *pgxpool.Pool, key int64) *Leader {
	return &Leader{pool: pool, key: key, retry: 5 * time.Second, heartbeat: 3 * time.Second}
}

// Run executes fn only while holding the session-level advisory lock; fn's context is cancelled if the lock connection is lost.
func (l *Leader) Run(ctx context.Context, fn func(context.Context)) {
	for ctx.Err() == nil {
		if l.runOnce(ctx, fn) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(l.retry):
		}
	}
}

func (l *Leader) runOnce(ctx context.Context, fn func(context.Context)) bool {
	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return false
	}
	defer conn.Release()
	var locked bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", l.key).Scan(&locked); err != nil || !locked {
		return false
	}
	slog.Info("delivery leadership acquired")

	leaderCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(leaderCtx)
	}()

	ticker := time.NewTicker(l.heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			cancel()
			<-done
			_, _ = conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", l.key)
			return true
		case <-done:
			cancel()
			_, _ = conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", l.key)
			return true
		case <-ticker.C:
			var one int
			if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
				slog.Warn("delivery leadership lost", "error", err)
				cancel()
				<-done
				_ = conn.Conn().Close(context.WithoutCancel(ctx))
				return true
			}
		}
	}
}
