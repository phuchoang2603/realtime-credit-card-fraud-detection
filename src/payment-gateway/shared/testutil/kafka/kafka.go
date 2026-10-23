package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	kafkamodule "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func SetupKafka(t testing.TB) []string {
	t.Helper()
	if testing.Short() {
		t.Skip("-short disables test containers")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := kafkamodule.Run(ctx, "confluentinc/confluent-local:8.3.2")
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("start kafka container: %v", err)
	}
	brokers, err := container.Brokers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return brokers
}
