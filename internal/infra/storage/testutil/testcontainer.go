package testutil

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// SetupTestContainer sets up a PostgreSQL container and runs migrations.
// It returns a connection pool and a cleanup function.
func SetupTestContainer(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
			return fmt.Sprintf("postgresql://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())
		}),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://test:test@localhost:%s/testdb?sslmode=disable", port.Port())

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	db := stdlib.OpenDBFromPool(pool)

	driver, err := pgx.WithInstance(db, &pgx.Config{})
	require.NoError(t, err)

	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..")

	migrationsPath := fmt.Sprintf("file://%s", filepath.Join(projectRoot, "db", "migrations"))

	migrations, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	require.NoError(t, err)

	// Apply all schema migrations.
	err = migrations.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = container.Terminate(ctx)
	}

	return pool, cleanup
}

// NoOpTracer returns a no-op tracer for testing
func NoOpTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
