package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/ahrav/hoglet-hub/internal/api/debug"
	"github.com/ahrav/hoglet-hub/internal/api/mux"
	operationApp "github.com/ahrav/hoglet-hub/internal/application/operation"
	tenantApp "github.com/ahrav/hoglet-hub/internal/application/tenant"
	httpServer "github.com/ahrav/hoglet-hub/internal/infra/adapters/http"
	handler "github.com/ahrav/hoglet-hub/internal/infra/adapters/http/handler"
	operationRepo "github.com/ahrav/hoglet-hub/internal/infra/storage/operation/postgres"
	tenantRepo "github.com/ahrav/hoglet-hub/internal/infra/storage/tenant/postgres"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
	"github.com/ahrav/hoglet-hub/pkg/common/otel"
)

var build = "develop"

const (
	serviceType = "tenant-management"
)

func main() {
	// Set the correct number of threads for the service
	_, _ = maxprocs.Set()

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	// Set up a context for the application
	ctx := context.Background()

	// Initialize a simple logger with service name and hostname
	svcName := fmt.Sprintf("TENANT-MGMT-%s", hostname)

	// Create metadata for the logger
	metadata := map[string]string{
		"service":  svcName,
		"hostname": hostname,
		"pod":      os.Getenv("POD_NAME"),
		"app":      serviceType,
	}

	// Initialize events for error logging
	logEvents := logger.Events{
		Error: func(ctx context.Context, r logger.Record) {
			errorAttrs := map[string]any{
				"error_message": r.Message,
				"error_time":    r.Time.UTC().Format(time.RFC3339),
				"trace_id":      otel.GetTraceID(ctx),
			}

			// Add any error-specific attributes
			for k, v := range r.Attributes {
				errorAttrs[k] = v
			}

			errorAttrsJSON, err := json.Marshal(errorAttrs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to marshal error attributes: %v\n", err)
				return
			}

			// Output the error event with valid JSON details
			fmt.Fprintf(os.Stderr, "Error event: %s, details: %s\n",
				r.Message, errorAttrsJSON)
		},
	}

	// Define a trace ID function that will be used by the logger
	traceIDFn := func(ctx context.Context) string {
		return otel.GetTraceID(ctx)
	}

	// Create the structured logger with all configurations
	log := logger.NewWithMetadata(os.Stdout, logger.LevelDebug, svcName, traceIDFn, logEvents, metadata)

	// Run the application with the configured logger
	if err := run(ctx, log, hostname); err != nil {
		log.Error(ctx, "startup error", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger, hostname string) error {
	// -------------------------------------------------------------------------
	// GOMAXPROCS
	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// -------------------------------------------------------------------------
	// Configuration
	cfg := struct {
		Web struct {
			ReadTimeout        time.Duration
			WriteTimeout       time.Duration
			IdleTimeout        time.Duration
			ShutdownTimeout    time.Duration
			APIHost            string
			APIPort            string
			DebugHost          string
			CORSAllowedOrigins []string
		}
		Tempo struct {
			Host        string
			ServiceName string
			Probability float64
		}
	}{}

	// Set default configuration values
	cfg.Web.ReadTimeout = 5 * time.Second
	cfg.Web.WriteTimeout = 10 * time.Second
	cfg.Web.IdleTimeout = 120 * time.Second
	cfg.Web.ShutdownTimeout = 20 * time.Second
	cfg.Web.APIHost = "0.0.0.0"
	cfg.Web.APIPort = "8080"
	cfg.Web.DebugHost = "0.0.0.0:8090"         // Always include port for debug host
	cfg.Web.CORSAllowedOrigins = []string{"*"} // All origins by default

	// Default Tempo configuration
	cfg.Tempo.Host = "tempo:4317"
	cfg.Tempo.ServiceName = serviceType
	cfg.Tempo.Probability = 0.05

	// Override with environment variables if provided
	if port := os.Getenv("API_PORT"); port != "" {
		cfg.Web.APIPort = port
	}
	if host := os.Getenv("API_HOST"); host != "" {
		cfg.Web.APIHost = host
	}
	if debugHost := os.Getenv("DEBUG_HOST"); debugHost != "" {
		if !strings.Contains(debugHost, ":") {
			debugHost = debugHost + ":8090"
		}
		cfg.Web.DebugHost = debugHost
	}
	if tempoHost := os.Getenv("TEMPO_HOST"); tempoHost != "" {
		cfg.Tempo.Host = tempoHost
	}
	if tempoServiceName := os.Getenv("TEMPO_SERVICE_NAME"); tempoServiceName != "" {
		cfg.Tempo.ServiceName = tempoServiceName
	}
	if tempoProbStr := os.Getenv("TEMPO_SAMPLING_PROBABILITY"); tempoProbStr != "" {
		if prob, err := strconv.ParseFloat(tempoProbStr, 64); err == nil {
			cfg.Tempo.Probability = prob
		}
	}

	// -------------------------------------------------------------------------
	// Start Tracing Support
	log.Info(ctx, "startup", "status", "initializing tracing support")

	// Configure and initialize OpenTelemetry
	traceProvider, teardown, err := otel.InitTelemetry(log, otel.Config{
		ServiceName:      cfg.Tempo.ServiceName,
		ExporterEndpoint: cfg.Tempo.Host,
		ExcludedRoutes: map[string]struct{}{
			"/api/v1/health/readiness": {},
			"/api/v1/health/liveness":  {},
			"/debug/pprof/":            {},
			"/debug/vars":              {},
			"/healthz":                 {},
		},
		Probability: cfg.Tempo.Probability,
		ResourceAttributes: map[string]string{
			"library.language": "go",
			"k8s.pod.name":     os.Getenv("POD_NAME"),
			"k8s.namespace":    os.Getenv("POD_NAMESPACE"),
			"k8s.container.id": hostname,
		},
		InsecureExporter: true, // TODO: Configure TLS for production
	})
	if err != nil {
		log.Error(ctx, "failed to initialize telemetry", "error", err)
		os.Exit(1)
	}
	defer teardown(ctx)

	// Get the tracer from the provider
	tracer := traceProvider.Tracer(cfg.Tempo.ServiceName)

	// -------------------------------------------------------------------------
	// Database Configuration
	log.Info(ctx, "startup", "status", "initializing database")

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
		return fmt.Errorf("parsing db config: %w", err)
	}
	poolCfg.MinConns = 5
	poolCfg.MaxConns = 20
	poolCfg.ConnConfig.Tracer = otelpgx.NewTracer()
	// TODO: Collect metrics for the pool and expose them via prometheus.

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("creating db pool: %w", err)
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// -------------------------------------------------------------------------
	// Start Debug Service
	go func() {
		log.Info(ctx, "startup", "status", "debug router started", "host", cfg.Web.DebugHost)
		if err := http.ListenAndServe(cfg.Web.DebugHost, debug.Mux()); err != nil {
			log.Error(ctx, "shutdown", "status", "debug router closed", "host", cfg.Web.DebugHost, "msg", err)
		}
	}()

	// -------------------------------------------------------------------------
	// Initialize repositories, services and handlers.
	log.Info(ctx, "startup", "status", "initializing repositories and services")

	// Initialize repositories using the tracer.
	tenantRepository := tenantRepo.NewTenantStore(pool, tracer)
	operationRepository := operationRepo.NewOperationStore(pool, tracer)

	// Initialize application services.
	operationService := operationApp.NewService(operationRepository)
	tenantService := tenantApp.NewService(tenantRepository, operationRepository, log, tracer)

	// Initialize HTTP handlers.
	tenantHandler := handler.NewTenantHandler(tenantService)
	operationHandler := handler.NewOperationHandler(operationService)

	// Initialize server adapter.
	serverAdapter := httpServer.NewServerAdapter(tenantHandler, operationHandler)

	// -------------------------------------------------------------------------
	// Start API Service.
	log.Info(ctx, "startup", "status", "initializing API support")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Create the base OpenAPI server
	openAPIHandler := httpServer.NewHTTPServer(serverAdapter)

	// Initialize centralized mux configuration with all dependencies.
	webCfg := mux.Config{
		Build:            build,
		Log:              log,
		DB:               pool,
		Tracer:           tracer,
		TenantService:    tenantService,
		OperationService: operationService,
	}

	// Wrap the OpenAPI server with our middleware infrastructure.
	// This provides consistent logging, error handling, and tracing.
	webAPI := mux.WrapWithMiddleware(
		webCfg,
		openAPIHandler,
		mux.WithCORS(cfg.Web.CORSAllowedOrigins),
	)

	// Configure and start the API server.
	apiAddr := fmt.Sprintf("%s:%s", cfg.Web.APIHost, cfg.Web.APIPort)
	api := http.Server{
		Addr:         apiAddr,
		Handler:      webAPI,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info(ctx, "startup", "status", "api router started", "host", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// -------------------------------------------------------------------------
	// Shutdown

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := api.Shutdown(ctx); err != nil {
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
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
