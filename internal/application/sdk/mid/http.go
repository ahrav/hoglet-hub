// Package mid provides app level middleware support.
package mid

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/ahrav/hoglet-hub/pkg/common/logger"
	"github.com/ahrav/hoglet-hub/pkg/web"
)

// HTTPMiddleware represents a standard Go HTTP middleware function. It wraps an HTTP
// handler and returns a new handler, allowing for pre and post-processing of requests.
type HTTPMiddleware func(http.Handler) http.Handler

// AsHTTP converts application-specific middleware (web.MidFunc) to standard Go HTTP
// middleware. This allows our custom middleware to be used with standard HTTP servers
// or third-party HTTP routers.
func AsHTTP(midware ...web.MidFunc) []HTTPMiddleware {
	var httpMw []HTTPMiddleware
	for _, mw := range midware {
		httpMw = append(httpMw, convertToHTTP(mw))
	}
	return httpMw
}

// convertToHTTP adapts a web.MidFunc to a standard HTTP middleware function.
func convertToHTTP(mw web.MidFunc) HTTPMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Store the response writer in context for potential middleware access.
			writerKey := contextKey("writer")
			ctx = context.WithValue(ctx, writerKey, w)

			sw := &statusWriter{ResponseWriter: w}

			// Bridge function connects our app middleware system with standard HTTP.
			bridge := func(ctx context.Context, r *http.Request) web.Encoder {
				next.ServeHTTP(sw, r.WithContext(ctx))
				return nil
			}

			handler := mw(bridge)
			_ = handler(ctx, r)
		})
	}
}

// contextKey is a type for keys stored in a context.
type contextKey string

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code and passes it to the wrapped ResponseWriter.
func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write captures a 200 status if WriteHeader hasn't been called yet.
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// LoggerHTTP provides a standard HTTP middleware for request logging. It logs the
// start and completion of HTTP requests along with important request metadata
// such as method, path, status code, and duration.
func LoggerHTTP(log *logger.Logger) HTTPMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()

			sw := &statusWriter{ResponseWriter: w}

			log.Info(ctx, "request started",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)

			next.ServeHTTP(sw, r)

			log.Info(ctx, "request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"status_code", sw.status,
				"took", time.Since(start).String(),
			)
		})
	}
}

// OtelHTTP provides a standard HTTP middleware for OpenTelemetry tracing. It creates
// a span for each request, propagates trace context from incoming headers, and records
// key request/response data as span attributes for observability.
func OtelHTTP(tracer trace.Tracer) HTTPMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			startTime := time.Now()

			// Extract trace context from request headers.
			ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))

			spanName := r.URL.Path
			attrs := []attribute.KeyValue{
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
			}

			ctx, span := tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
			defer span.End()

			sw := &statusWriter{ResponseWriter: w}

			next.ServeHTTP(sw, r.WithContext(ctx))

			span.SetAttributes(
				attribute.Int("http.status_code", sw.status),
				attribute.String("http.response_time", time.Since(startTime).String()),
			)
		})
	}
}

// GetMiddlewareChain returns a complete chain of standard HTTP middleware
// combining both direct HTTP middleware and converted application middleware.
// This provides a consistent middleware stack regardless of whether using the
// web.App framework or standard HTTP handlers.
func GetMiddlewareChain(log *logger.Logger, tracer trace.Tracer, metrics APIMetrics) []HTTPMiddleware {
	return []HTTPMiddleware{
		OtelHTTP(tracer),
		MetricsMiddleware(metrics),
		LoggerHTTP(log),
		convertToHTTP(Errors(log)),
		convertToHTTP(Panics()),
	}
}
