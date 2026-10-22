package application

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/domain"
)

var (
	ErrInvalidRequest   = errors.New("invalid sale request")
	ErrKeyConflict      = errors.New("operation key was used with a different request")
	ErrUnknownToken     = errors.New("payment method token is unknown or inactive")
	ErrVaultUnavailable = errors.New("vault is unavailable")
	ErrNotFound         = errors.New("operation not found")
	ErrInjectedTimeout  = errors.New("injected timeout")
)

const (
	tokenPrefix       = "vt_"
	maxOperationKey   = 200
	minTokenLength    = 20
	maxTokenLength    = 128
	referencePrefix   = "proc_"
	referenceByteSize = 12
)

var (
	operationKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)
	tokenBodyPattern    = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	currencyPattern     = regexp.MustCompile(`^[A-Z]{3}$`)
)

type Record struct {
	Operation   domain.Operation
	RequestHash []byte
}

type NewRecord struct {
	Sale               domain.Sale
	RequestHash        []byte
	Outcome            domain.Outcome
	DeclineReason      string
	ProcessorReference string
}

type Store interface {
	Find(ctx context.Context, operationKey string) (Record, error)
	Insert(ctx context.Context, record NewRecord) (Record, bool, error)
}

type Vault interface {
	Resolve(ctx context.Context, integrationID, token string) (domain.Behavior, error)
}

type Service struct {
	store       Store
	vault       Vault
	faults      domain.Faults
	timeoutHold time.Duration
}

func NewService(store Store, vault Vault, faults domain.Faults, timeoutHold time.Duration) *Service {
	return &Service{store: store, vault: vault, faults: faults, timeoutHold: timeoutHold}
}

func (s *Service) Sale(ctx context.Context, sale domain.Sale) (domain.Operation, error) {
	if err := validate(sale); err != nil {
		return domain.Operation{}, err
	}
	hash := sale.Hash()
	existing, err := s.store.Find(ctx, sale.OperationKey)
	switch {
	case err == nil:
		return replay(existing, hash)
	case !errors.Is(err, ErrNotFound):
		return domain.Operation{}, err
	}

	behavior, err := s.vault.Resolve(ctx, sale.IntegrationID, sale.Token)
	if err != nil {
		return domain.Operation{}, err
	}
	outcome, reason, ok := domain.Decide(behavior)
	if !ok {
		return domain.Operation{}, fmt.Errorf("vault returned unsupported behavior")
	}
	reference, err := newReference()
	if err != nil {
		return domain.Operation{}, err
	}
	record, inserted, err := s.store.Insert(ctx, NewRecord{
		Sale:               sale,
		RequestHash:        hash,
		Outcome:            outcome,
		DeclineReason:      reason,
		ProcessorReference: reference,
	})
	if err != nil {
		return domain.Operation{}, err
	}
	if !inserted {
		return replay(record, hash)
	}

	plan := s.faults.Plan(sale.OperationKey)
	if plan.Timeout {
		if err := sleep(ctx, s.timeoutHold); err != nil {
			return domain.Operation{}, err
		}
		return domain.Operation{}, ErrInjectedTimeout
	}
	if err := sleep(ctx, plan.Latency); err != nil {
		return domain.Operation{}, err
	}
	return record.Operation, nil
}

func (s *Service) GetOperation(ctx context.Context, operationKey string) (domain.Operation, error) {
	if operationKey == "" || len(operationKey) > maxOperationKey {
		return domain.Operation{}, fmt.Errorf("%w: operation_key", ErrInvalidRequest)
	}
	record, err := s.store.Find(ctx, operationKey)
	if err != nil {
		return domain.Operation{}, err
	}
	return record.Operation, nil
}

func replay(record Record, hash []byte) (domain.Operation, error) {
	if subtle.ConstantTimeCompare(record.RequestHash, hash) != 1 {
		return domain.Operation{}, ErrKeyConflict
	}
	return record.Operation, nil
}

func validate(sale domain.Sale) error {
	if sale.OperationKey == "" || len(sale.OperationKey) > maxOperationKey || !operationKeyPattern.MatchString(sale.OperationKey) {
		return fmt.Errorf("%w: operation_key", ErrInvalidRequest)
	}
	if _, err := uuid.Parse(sale.IntegrationID); err != nil {
		return fmt.Errorf("%w: integration_id must be a UUID", ErrInvalidRequest)
	}
	if !isSyntheticToken(sale.Token) {
		return fmt.Errorf("%w: payment_method_token must be a vault token", ErrInvalidRequest)
	}
	if sale.AmountMinorUnit <= 0 {
		return fmt.Errorf("%w: amount_minor_units must be positive", ErrInvalidRequest)
	}
	if !currencyPattern.MatchString(sale.Currency) {
		return fmt.Errorf("%w: currency must be an ISO 4217 code", ErrInvalidRequest)
	}
	return nil
}

func isSyntheticToken(token string) bool {
	if len(token) < minTokenLength || len(token) > maxTokenLength || token[:min(len(token), len(tokenPrefix))] != tokenPrefix {
		return false
	}
	return tokenBodyPattern.MatchString(token[len(tokenPrefix):])
}

func newReference() (string, error) {
	raw := make([]byte, referenceByteSize)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate processor reference: %w", err)
	}
	return referencePrefix + hex.EncodeToString(raw), nil
}

func sleep(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
