package application

import (
	"context"

	"github.com/google/uuid"
)

type correlationKey struct{}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey{}, id)
}

func CorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationKey{}).(string); ok && id != "" {
		return id
	}
	return uuid.NewString()
}
