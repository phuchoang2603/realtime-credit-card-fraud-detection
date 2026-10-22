package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/domain"
)

const tokenPrefix = "vt_"

var (
	ErrNotAllowListed = errors.New("card number is not an allow-listed test card")
	ErrTokenNotFound  = errors.New("token not found")
)

type StoredToken struct {
	MethodType        string
	Fingerprint       string
	FixtureCiphertext []byte
}

type NewToken struct {
	Digest            []byte
	IntegrationID     uuid.UUID
	MethodType        string
	Fingerprint       string
	FixtureCiphertext []byte
}

type TokenStore interface {
	InsertToken(ctx context.Context, token NewToken) error
	ActiveToken(ctx context.Context, digest []byte, integrationID uuid.UUID) (StoredToken, error)
}

type Sealer interface {
	Seal(plaintext, associatedData []byte) ([]byte, error)
	Open(ciphertext, associatedData []byte) ([]byte, error)
}

type Method struct {
	Type        string
	Fingerprint string
}

type Fixture struct {
	CardReference string
	Behavior      domain.Behavior
}

type fixturePayload struct {
	CardReference string          `json:"card_reference"`
	Behavior      domain.Behavior `json:"behavior"`
}

type Service struct {
	store          TokenStore
	sealer         Sealer
	fingerprintKey []byte
}

func NewService(store TokenStore, sealer Sealer, fingerprintKey []byte) *Service {
	return &Service{store: store, sealer: sealer, fingerprintKey: fingerprintKey}
}

func (s *Service) Tokenize(ctx context.Context, integrationID uuid.UUID, cardNumber string) (string, Method, error) {
	card, ok := domain.LookupCard(cardNumber)
	if !ok {
		return "", Method{}, ErrNotAllowListed
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", Method{}, fmt.Errorf("generate token: %w", err)
	}
	token := tokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	digest := tokenDigest(token)
	payload, err := json.Marshal(fixturePayload{CardReference: card.Ref, Behavior: card.Behavior})
	if err != nil {
		return "", Method{}, err
	}
	ciphertext, err := s.sealer.Seal(payload, digest)
	if err != nil {
		return "", Method{}, fmt.Errorf("seal fixture: %w", err)
	}
	method := Method{Type: domain.MethodTypeCard, Fingerprint: domain.Fingerprint(s.fingerprintKey, card.Ref)}
	err = s.store.InsertToken(ctx, NewToken{
		Digest:            digest,
		IntegrationID:     integrationID,
		MethodType:        method.Type,
		Fingerprint:       method.Fingerprint,
		FixtureCiphertext: ciphertext,
	})
	if err != nil {
		return "", Method{}, fmt.Errorf("store token: %w", err)
	}
	return token, method, nil
}

func (s *Service) Describe(ctx context.Context, integrationID uuid.UUID, token string) (Method, error) {
	stored, err := s.store.ActiveToken(ctx, tokenDigest(token), integrationID)
	if err != nil {
		return Method{}, err
	}
	return Method{Type: stored.MethodType, Fingerprint: stored.Fingerprint}, nil
}

func (s *Service) Resolve(ctx context.Context, integrationID uuid.UUID, token string) (Fixture, error) {
	digest := tokenDigest(token)
	stored, err := s.store.ActiveToken(ctx, digest, integrationID)
	if err != nil {
		return Fixture{}, err
	}
	plaintext, err := s.sealer.Open(stored.FixtureCiphertext, digest)
	if err != nil {
		return Fixture{}, fmt.Errorf("open fixture: %w", err)
	}
	var payload fixturePayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return Fixture{}, fmt.Errorf("decode fixture: %w", err)
	}
	return Fixture(payload), nil
}

func tokenDigest(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
