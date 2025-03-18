package mux

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	operationApp "github.com/ahrav/hoglet-hub/internal/application/operation"
	"github.com/ahrav/hoglet-hub/internal/application/sdk/mid"
	tenantApp "github.com/ahrav/hoglet-hub/internal/application/tenant"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
)

// Options represent optional parameters.
type Options struct {
	corsOrigin []string
}

// WithCORS provides configuration options for CORS.
func WithCORS(origins []string) func(opts *Options) {
	return func(opts *Options) {
		opts.corsOrigin = origins
	}
}

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Build            string
	Log              *logger.Logger
	DB               *pgxpool.Pool
	Tracer           trace.Tracer
	APIMetrics       mid.APIMetrics
	TenantService    *tenantApp.Service
	OperationService *operationApp.Service
}

// healthHandler provides health check endpoints for liveness and readiness probes.
type healthHandler struct{ db *pgxpool.Pool }

// newHealthHandler creates a new health handler with the provided database pool.
func newHealthHandler(db *pgxpool.Pool) *healthHandler { return &healthHandler{db: db} }

// Liveness returns a simple handler for liveness probe.
// The liveness probe is used to know when to restart a container.
func (h *healthHandler) Liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"up"}`))
	}
}

// Readiness returns a handler for readiness probe.
// The readiness probe is used to know when a container is ready to start accepting traffic.
// It checks if the database connection is healthy.
func (h *healthHandler) Readiness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := h.db.Ping(ctx)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"down","reason":"database unavailable"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"up"}`))
	}
}

// WrapWithMiddleware applies the standard middleware stack to an existing HTTP handler.
// This allows existing HTTP servers (like OpenAPI-generated ones) to benefit from
// the middleware infrastructure without changing their routing logic.
func WrapWithMiddleware(cfg Config, handler http.Handler, options ...func(opts *Options)) http.Handler {
	var opts Options
	for _, option := range options {
		option(&opts)
	}

	// Create a middleware chain using our mid package.
	chain := mid.GetMiddlewareChain(cfg.Log, cfg.Tracer, cfg.APIMetrics)

	if len(opts.corsOrigin) > 0 {
		chain = append(chain, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodOptions {
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					w.Header().Set("Access-Control-Max-Age", "86400")

					origin := r.Header.Get("Origin")
					for _, host := range opts.corsOrigin {
						if host == "*" || host == origin {
							w.Header().Set("Access-Control-Allow-Origin", origin)
							break
						}
					}
					w.WriteHeader(http.StatusOK)
					return
				}

				// Handle regular requests.
				origin := r.Header.Get("Origin")
				for _, host := range opts.corsOrigin {
					if host == "*" || host == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}

				next.ServeHTTP(w, r)
			})
		})
	}

	// Apply the middleware chain to the original handler.
	wrappedHandler := handler
	for _, middleware := range chain {
		wrappedHandler = middleware(wrappedHandler)
	}

	// Create a mux to integrate health endpoints with the middleware-wrapped handler.
	finalMux := http.NewServeMux()

	// Register health check endpoints directly on the mux WITHOUT middleware.
	healthHandler := newHealthHandler(cfg.DB)
	finalMux.HandleFunc("/api/v1/health/liveness", healthHandler.Liveness())
	finalMux.HandleFunc("/api/v1/health/readiness", healthHandler.Readiness())

	// Register the middleware-wrapped handler for all other paths.
	finalMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Skip health endpoints to prevent double handling
		if r.URL.Path == "/api/v1/health/liveness" || r.URL.Path == "/api/v1/health/readiness" {
			return
		}
		wrappedHandler.ServeHTTP(w, r)
	})

	return finalMux
}
