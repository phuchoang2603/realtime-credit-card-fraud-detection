package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/db"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/adapters/postgres/sqlcgen"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/domain"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	latest  int64
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	latest, err := db.LatestVersion()
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool, queries: sqlcgen.New(pool), latest: latest}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ready(ctx context.Context) error {
	var version int64
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&version)
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if version != s.latest {
		return fmt.Errorf("migration version %d, want %d", version, s.latest)
	}
	return nil
}

func (s *Store) Find(ctx context.Context, operationKey string) (application.Record, error) {
	row, err := s.queries.GetOperation(ctx, operationKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.Record{}, application.ErrNotFound
	}
	if err != nil {
		return application.Record{}, err
	}
	return toRecord(row.OperationKey, row.RequestHash, row.Outcome, row.DeclineReason, row.ProcessorReference, row.RecordedAt), nil
}

func (s *Store) Insert(ctx context.Context, record application.NewRecord) (application.Record, bool, error) {
	integrationID, err := uuid.Parse(record.Sale.IntegrationID)
	if err != nil {
		return application.Record{}, false, err
	}
	row, err := s.queries.InsertOperation(ctx, sqlcgen.InsertOperationParams{
		OperationKey:       record.Sale.OperationKey,
		RequestHash:        record.RequestHash,
		IntegrationID:      integrationID,
		AmountMinorUnits:   record.Sale.AmountMinorUnit,
		Currency:           record.Sale.Currency,
		Outcome:            string(record.Outcome),
		DeclineReason:      pgtype.Text{String: record.DeclineReason, Valid: record.DeclineReason != ""},
		ProcessorReference: record.ProcessorReference,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		existing, findErr := s.Find(ctx, record.Sale.OperationKey)
		return existing, false, findErr
	}
	if err != nil {
		return application.Record{}, false, err
	}
	return toRecord(row.OperationKey, row.RequestHash, row.Outcome, row.DeclineReason, row.ProcessorReference, row.RecordedAt), true, nil
}

func toRecord(key string, hash []byte, outcome string, reason pgtype.Text, reference string, recordedAt time.Time) application.Record {
	return application.Record{
		RequestHash: hash,
		Operation: domain.Operation{
			Key:                key,
			Outcome:            domain.Outcome(outcome),
			DeclineReason:      reason.String,
			ProcessorReference: reference,
			RecordedAt:         recordedAt,
		},
	}
}

func Migrate(ctx context.Context, databaseURL string) error {
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	conn := stdlib.OpenDB(*config)
	defer closeQuietly(conn)
	provider, err := goose.NewProvider(goose.DialectPostgres, conn, db.Migrations())
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func closeQuietly(conn *sql.DB) {
	_ = conn.Close()
}
