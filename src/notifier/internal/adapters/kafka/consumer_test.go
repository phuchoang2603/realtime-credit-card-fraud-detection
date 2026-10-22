package kafka_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/adapters/kafka"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/adapters/postgres"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/domain"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/metrics"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/testutil"
)

func TestConsumerRecordsOnceAcrossRestartsAndGroups(t *testing.T) {
	brokersEnv := os.Getenv("TEST_KAFKA_BROKERS")
	if brokersEnv == "" {
		t.Skip("TEST_KAFKA_BROKERS not set")
	}
	brokers := strings.Split(brokersEnv, ",")
	pool, _ := testutil.NewPool(t)
	set, err := domain.ParseIntegrations([]byte(`[
	{"integration_id":"sub","webhook":{"url":"https://hooks.example.com/x","secret":"s","events":["attempt.declined","payment.succeeded"]}},
	{"integration_id":"bare"}]`))
	if err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().UnixNano()
	topic := fmt.Sprintf("notifier-test-%d", suffix)
	group := fmt.Sprintf("notifier-test-group-%d", suffix)

	producer, err := kgo.NewClient(kgo.SeedBrokers(brokers...), kgo.AllowAutoTopicCreation())
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	messages := []struct {
		key   string
		value []byte
	}{
		{"p1", testutil.Created(t, "k1", "p1", "sub", "order-1")},
		{"p1", testutil.Declined(t, "k2", "p1", "sub", "a1", "ISSUER_DECLINED")},
		{"p1", testutil.Succeeded(t, "k3", "p1", "sub", "a2")},
		{"p1", testutil.Expired(t, "k4", "p1", "sub")},
		{"p2", testutil.Created(t, "k5", "p2", "bare", "order-2")},
		{"p2", testutil.Succeeded(t, "k6", "p2", "bare", "a1")},
		{"p2", []byte{0xff, 0xff}},
	}
	for _, message := range messages {
		record := &kgo.Record{Topic: topic, Key: []byte(message.key), Value: message.value}
		if err := producer.ProduceSync(context.Background(), record).FirstErr(); err != nil {
			t.Fatalf("produce: %v", err)
		}
	}

	runConsumer := func(group string, wantEvents int) {
		t.Helper()
		ingestor := &application.Ingestor{Store: postgres.NewStore(pool), Integrations: set, Now: time.Now}
		m := metrics.New()
		consumer, err := kafka.NewConsumer(brokers, topic, group, ingestor, m)
		if err != nil {
			t.Fatal(err)
		}
		defer consumer.Close()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			consumer.Run(ctx)
			close(done)
		}()
		deadline := time.Now().Add(40 * time.Second)
		for time.Now().Before(deadline) {
			var refs int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM payment_refs").Scan(&refs); err != nil {
				t.Fatal(err)
			}
			if refs == wantEvents {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		time.Sleep(time.Second)
		cancel()
		<-done
	}

	runConsumer(group, 2)
	assertDeliveries := func() {
		t.Helper()
		rows, err := pool.Query(context.Background(), "SELECT event_id FROM webhook_deliveries ORDER BY event_id")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		if got := strings.Join(ids, ","); got != "k2,k3" {
			t.Fatalf("deliveries = %s, want k2,k3 (k4 unsubscribed, bare integration endpoint-less)", got)
		}
	}
	assertDeliveries()

	runConsumer(group, 2)
	assertDeliveries()

	runConsumer(group+"-replay", 2)
	assertDeliveries()
}
