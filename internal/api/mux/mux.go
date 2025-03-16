package mux

import (
	"context"
	"embed"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/ahrav/hoglet-hub/internal/api/mid"
	operationApp "github.com/ahrav/hoglet-hub/internal/application/operation"
	tenantApp "github.com/ahrav/hoglet-hub/internal/application/tenant"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
	"github.com/ahrav/hoglet-hub/pkg/web"
)

// StaticSite represents a static site to run.
type StaticSite struct {
	react      bool
	static     embed.FS
	staticDir  string
	staticPath string
}

// Options represent optional parameters.
type Options struct {
	corsOrigin []string
	sites      []StaticSite
}

// WithCORS provides configuration options for CORS.
func WithCORS(origins []string) func(opts *Options) {
	return func(opts *Options) {
		opts.corsOrigin = origins
	}
}

// WithFileServer provides configuration options for file server.
func WithFileServer(react bool, static embed.FS, dir string, path string) func(opts *Options) {
	return func(opts *Options) {
		opts.sites = append(opts.sites, StaticSite{
			react:      react,
			static:     static,
			staticDir:  dir,
			staticPath: path,
		})
	}
}

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Build            string
	Log              *logger.Logger
	DB               *pgxpool.Pool
	Tracer           trace.Tracer
	TenantService    *tenantApp.Service
	OperationService *operationApp.Service
}

// RouteAdder defines behavior that sets the routes to bind for an instance
// of the service.
type RouteAdder interface {
	Add(app *web.App, cfg Config)
}

// HealthHandler provides health check endpoints for liveness and readiness probes.
type HealthHandler struct{ db *pgxpool.Pool }

// NewHealthHandler creates a new health handler with the provided database pool.
func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{db: db}
}

// Liveness returns a simple handler for liveness probe.
// The liveness probe is used to know when to restart a container.
func (h *HealthHandler) Liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"up"}`))
	}
}

// Readiness returns a handler for readiness probe.
// The readiness probe is used to know when a container is ready to start accepting traffic.
// It checks if the database connection is healthy.
func (h *HealthHandler) Readiness() http.HandlerFunc {
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

// WebAPI constructs a http.Handler with all application routes bound.
func WebAPI(cfg Config, routeAdder RouteAdder, options ...func(opts *Options)) http.Handler {
	logger := func(ctx context.Context, msg string, args ...any) {
		cfg.Log.Info(ctx, msg, args...)
	}

	app := web.NewApp(
		logger,
		cfg.Tracer,
		mid.Otel(cfg.Tracer),
		mid.Logger(cfg.Log),
		mid.Errors(cfg.Log),
		mid.Panics(),
	)

	var opts Options
	for _, option := range options {
		option(&opts)
	}

	if len(opts.corsOrigin) > 0 {
		app.EnableCORS(opts.corsOrigin)
	}

	routeAdder.Add(app, cfg)

	for _, site := range opts.sites {
		if site.react {
			app.FileServerReact(site.static, site.staticDir, site.staticPath)
		} else {
			app.FileServer(site.static, site.staticDir, site.staticPath)
		}
	}

	return app
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
	chain := mid.GetMiddlewareChain(cfg.Log, cfg.Tracer)

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

	// Create a new mux to attach health endpoints and then forward other requests
	// to the wrapped handler.
	mux := http.NewServeMux()

	// Register health check endpoints.
	healthHandler := NewHealthHandler(cfg.DB)
	mux.HandleFunc("/api/v1/health/liveness", healthHandler.Liveness())
	mux.HandleFunc("/api/v1/health/readiness", healthHandler.Readiness())

	// Forward all other requests to the original handler.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Skip health endpoints to prevent double handling.
		if r.URL.Path == "/api/v1/health/liveness" || r.URL.Path == "/api/v1/health/readiness" {
			return
		}
		handler.ServeHTTP(w, r)
	})

	// Apply the middleware chain to the mux with health endpoints.
	wrapped := http.Handler(mux)
	for _, middleware := range chain {
		wrapped = middleware(wrapped)
	}

	return wrapped
}
