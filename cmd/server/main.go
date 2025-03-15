package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	operationApp "github.com/trufflesecurity/hoglet-hub/internal/application/operation"
	tenantApp "github.com/trufflesecurity/hoglet-hub/internal/application/tenant"
	"github.com/trufflesecurity/hoglet-hub/internal/domain/operation"
	"github.com/trufflesecurity/hoglet-hub/internal/domain/tenant"
	httpServer "github.com/trufflesecurity/hoglet-hub/internal/infra/adapters/http"
	handler "github.com/trufflesecurity/hoglet-hub/internal/infra/adapters/http/handler"
)

func main() {
	log.Println("Starting tenant management service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		host := os.Getenv("POSTGRES_HOST")
		dbname := os.Getenv("POSTGRES_DB")

		if user == "" {
			user = "postgres"
		}
		if password == "" {
			password = "postgres"
		}
		if host == "" {
			host = "postgres"
		}
		if dbname == "" {
			dbname = "hoglet-hub"
		}

		dsn = fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
			user, password, host, dbname)
	}

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("failed to parse db config: %v", err)
		os.Exit(1)
	}
	poolCfg.MinConns = 5
	poolCfg.MaxConns = 20
	poolCfg.ConnConfig.Tracer = otelpgx.NewTracer()
	// TODO: Collect metrics for the pool and expose them via prometheus.

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
		os.Exit(1)
	}

	// Initialize repositories
	// tenantRepo := memory.NewTenantRepository()
	// operationRepo := memory.NewOperationRepository()

	var tenantRepo tenant.Repository
	var operationRepo operation.Repository

	// Initialize application services
	operationService := operationApp.NewService(operationRepo)
	tenantService := tenantApp.NewService(tenantRepo, operationRepo)

	// Initialize HTTP handlers
	tenantHandler := handler.NewTenantHandler(tenantService)
	operationHandler := handler.NewOperationHandler(operationService)

	// Initialize server adapter
	serverAdapter := httpServer.NewServerAdapter(tenantHandler, operationHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      httpServer.NewHTTPServer(serverAdapter),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("HTTP server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// TODO: consider moving this to an init container.
// runMigrations uses golang-migrate to apply all up migrations from "db/migrations".
// runMigrations acquires a single pgx connection from the pool, runs migrations,
// and then releases the connection back to the pool.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Acquire a connection from the pool
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("could not acquire connection: %w", err)
	}
	defer conn.Release() // Ensure the connection is released

	db := stdlib.OpenDBFromPool(pool)
	if err != nil {
		return fmt.Errorf("could not open db from pool: %w", err)
	}

	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("could not create pgx driver: %w", err)
	}

	const migrationsPath = "file:///app/db/migrations"
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	// Run the migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}

	return nil
}
