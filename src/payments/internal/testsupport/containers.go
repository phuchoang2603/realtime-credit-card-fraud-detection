package testsupport

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresImage = "postgres:17.6-alpine"
	valkeyImage   = "valkey/valkey:8.1.10"
	startTimeout  = 2 * time.Minute
)

type service struct {
	once  sync.Once
	value string
	err   error
}

var (
	postgresService service
	valkeyService   service
	valkeyContainer testcontainers.Container
)

func resolve(t testing.TB, svc *service, start func(context.Context) (string, error)) string {
	t.Helper()
	if testing.Short() {
		t.Skip("-short disables test containers")
	}
	svc.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
		defer cancel()
		svc.value, svc.err = start(ctx)
	})
	if svc.err != nil {
		t.Fatalf("start test container (is Docker available?): %v", svc.err)
	}
	return svc.value
}

func postgresURL(t testing.TB) string {
	t.Helper()
	return resolve(t, &postgresService, func(ctx context.Context) (string, error) {
		container, err := pgmodule.Run(ctx, postgresImage,
			pgmodule.WithDatabase("postgres"),
			pgmodule.WithUsername("postgres"),
			pgmodule.WithPassword("postgres"),
			testcontainers.WithCmd("postgres", "-c", "fsync=off", "-c", "wal_level=logical"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(startTimeout),
			),
		)
		if err != nil {
			return "", err
		}
		return container.ConnectionString(ctx, "sslmode=disable")
	})
}

func RedisAddr(t testing.TB) string {
	t.Helper()
	return resolve(t, &valkeyService, func(ctx context.Context) (string, error) {
		container, err := testcontainers.Run(ctx, valkeyImage,
			testcontainers.WithExposedPorts("6379/tcp"),
			testcontainers.WithCmd("valkey-server", "--save", "", "--appendonly", "no"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("Ready to accept connections"),
				wait.ForListeningPort("6379/tcp"),
			),
		)
		if err != nil {
			return "", err
		}
		valkeyContainer = container
		return container.PortEndpoint(ctx, "6379/tcp", "")
	})
}

func RestartRedis(t testing.TB) string {
	t.Helper()
	RedisAddr(t)
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()
	if err := valkeyContainer.Stop(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := valkeyContainer.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := wait.ForListeningPort("6379/tcp").WaitUntilReady(ctx, valkeyContainer); err != nil {
		t.Fatal(err)
	}
	address, err := valkeyContainer.PortEndpoint(ctx, "6379/tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	valkeyService.value = address
	return address
}
