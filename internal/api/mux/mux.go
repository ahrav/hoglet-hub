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

	// Apply the middleware chain to the original handler.
	wrapped := handler
	for _, middleware := range chain {
		wrapped = middleware(wrapped)
	}

	return wrapped
}
