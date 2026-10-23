package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"time"

	"github.com/google/uuid"
)

var identityNamespace = uuid.MustParse("71088b85-53a3-5ed5-bf27-8ea372bf796d")

func MerchantID(integrationID, externalSellerID string) uuid.UUID {
	return uuid.NewSHA1(identityNamespace, []byte(integrationID+":merchant:"+externalSellerID))
}

func CustomerID(integrationID, externalBuyerID string) uuid.UUID {
	return uuid.NewSHA1(identityNamespace, []byte(integrationID+":customer:"+externalBuyerID))
}

var ErrInvalidCheckoutToken = errors.New("invalid checkout token")

const checkoutTokenLength = 16 + 8 + sha256.Size

type CheckoutTokens struct {
	key []byte
}

func NewCheckoutTokens(key []byte) CheckoutTokens {
	return CheckoutTokens{key: key}
}

func (t CheckoutTokens) Issue(sessionID uuid.UUID, expiresAt time.Time) string {
	raw := make([]byte, 0, checkoutTokenLength)
	raw = append(raw, sessionID[:]...)
	raw = binary.BigEndian.AppendUint64(raw, uint64(expiresAt.Unix()))
	raw = append(raw, t.sign(raw)...)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func (t CheckoutTokens) Verify(token string) (uuid.UUID, time.Time, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != checkoutTokenLength {
		return uuid.Nil, time.Time{}, ErrInvalidCheckoutToken
	}
	body, signature := raw[:24], raw[24:]
	if !hmac.Equal(signature, t.sign(body)) {
		return uuid.Nil, time.Time{}, ErrInvalidCheckoutToken
	}
	sessionID, err := uuid.FromBytes(body[:16])
	if err != nil {
		return uuid.Nil, time.Time{}, ErrInvalidCheckoutToken
	}
	return sessionID, time.Unix(int64(binary.BigEndian.Uint64(body[16:])), 0).UTC(), nil
}

func (t CheckoutTokens) sign(body []byte) []byte {
	mac := hmac.New(sha256.New, t.key)
	mac.Write(body)
	return mac.Sum(nil)
}
