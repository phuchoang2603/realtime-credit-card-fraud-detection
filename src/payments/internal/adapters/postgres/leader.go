package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Leader struct {
	pool     *pgxpool.Pool
	lockID   int64
	interval time.Duration
}

func NewLeader(pool *pgxpool.Pool, lockID int64, interval time.Duration) *Leader {
	return &Leader{pool: pool, lockID: lockID, interval: interval}
}

func (l *Leader) Run(ctx context.Context, work func(ctx context.Context)) {
	for ctx.Err() == nil {
		l.lead(ctx, work)
		select {
		case <-ctx.Done():
			return
		case <-time.After(l.interval):
		}
	}
}

func (l *Leader) lead(ctx context.Context, work func(ctx context.Context)) {
	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		slog.Warn("leader election: acquire connection", "error", err)
		return
	}
	var held bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, l.lockID).Scan(&held); err != nil || !held {
		conn.Release()
		return
	}
	leaderCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		ticker := time.NewTicker(l.interval)
		defer ticker.Stop()
		for {
			select {
			case <-leaderCtx.Done():
				return
			case <-ticker.C:
				probe, probeCancel := context.WithTimeout(leaderCtx, max(l.interval, time.Second))
				err := conn.Conn().Ping(probe)
				probeCancel()
				if err != nil && leaderCtx.Err() == nil {
					cancel()
					return
				}
			}
		}
	}()
	work(leaderCtx)
	cancel()
	<-monitorDone
	unlockCtx, unlockCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer unlockCancel()
	if _, err := conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, l.lockID); err != nil {
		_ = conn.Conn().Close(unlockCtx)
	}
	conn.Release()
}
