package testutil

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	postgresOnce sync.Once
	postgresURL  string
	postgresErr  error

	kafkaOnce    sync.Once
	kafkaBrokers []string
	kafkaErr     error
)

func PostgresURL(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("short mode: skipping tests that need PostgreSQL")
	}
	postgresOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		container, err := pgmodule.Run(ctx, "postgres:17.6-alpine",
			pgmodule.WithDatabase("postgres"),
			pgmodule.WithUsername("postgres"),
			pgmodule.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(2*time.Minute),
			),
		)
		if err != nil {
			postgresErr = err
			return
		}
		postgresURL, postgresErr = container.ConnectionString(ctx, "sslmode=disable")
	})
	if postgresErr != nil {
		t.Fatalf("start postgres container (is Docker available?): %v", postgresErr)
	}
	return postgresURL
}

func KafkaBrokers(t *testing.T) []string {
	t.Helper()
	if testing.Short() {
		t.Skip("short mode: skipping tests that need Kafka")
	}
	kafkaOnce.Do(func() {
		kafkaBrokers, kafkaErr = startKafka()
	})
	if kafkaErr != nil {
		t.Fatalf("start kafka container (is Docker available?): %v", kafkaErr)
	}
	return kafkaBrokers
}

const kafkaStarterScript = "/tmp/start-kafka.sh"

// The advertised listener needs the host-mapped port, which exists only after start, so the entrypoint waits for a script generated post-start.
// The trailing marker keeps the entrypoint from running a half-extracted script.
func startKafka() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var address string
	_, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "apache/kafka:4.3.1",
			ExposedPorts: []string{"9092/tcp"},
			Entrypoint:   []string{"sh"},
			Cmd:          []string{"-c", "until grep -qs '^# complete$' " + kafkaStarterScript + "; do sleep 0.1; done; exec sh " + kafkaStarterScript},
			Env: map[string]string{
				"KAFKA_NODE_ID":                                  "1",
				"KAFKA_PROCESS_ROLES":                            "broker,controller",
				"KAFKA_LISTENERS":                                "PLAINTEXT://:9092,CONTROLLER://:9093",
				"KAFKA_CONTROLLER_LISTENER_NAMES":                "CONTROLLER",
				"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":           "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT",
				"KAFKA_CONTROLLER_QUORUM_VOTERS":                 "1@localhost:9093",
				"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":         "1",
				"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": "1",
				"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":            "1",
				"KAFKA_AUTO_CREATE_TOPICS_ENABLE":                "true",
			},
			LifecycleHooks: []testcontainers.ContainerLifecycleHooks{{
				PostStarts: []testcontainers.ContainerHook{
					func(ctx context.Context, c testcontainers.Container) error {
						host, err := c.Host(ctx)
						if err != nil {
							return err
						}
						port, err := c.MappedPort(ctx, "9092/tcp")
						if err != nil {
							return err
						}
						address = net.JoinHostPort(host, strconv.Itoa(int(port.Num())))
						script := fmt.Sprintf("export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://%s\nexec /etc/kafka/docker/run\n# complete\n", address)
						if err := c.CopyToContainer(ctx, []byte(script), kafkaStarterScript, 0o755); err != nil {
							return err
						}
						return wait.ForLog("Kafka Server started").WithStartupTimeout(2*time.Minute).WaitUntilReady(ctx, c)
					},
				},
			}},
		},
	})
	if err != nil {
		return nil, err
	}
	return []string{address}, nil
}
