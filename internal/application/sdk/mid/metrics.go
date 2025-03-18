// Package mid provides app level middleware support.
package mid

import (
	"context"
	"net/http"
	"time"
)

// MetricsResponseWriter wraps an http.ResponseWriter to capture the status code
type MetricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before delegating to the wrapped ResponseWriter
func (w *MetricsResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// StatusCode returns the HTTP status code of the response
func (w *MetricsResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK // Default to 200 if WriteHeader was never called
	}
	return w.statusCode
}

// APIMetrics defines metrics for API operations.
type APIMetrics interface {
	// ObserveRequestLatency records the latency of API requests.
	ObserveRequestLatency(ctx context.Context, endpoint string, method string, statusCode int, duration time.Duration)

	// IncRequestCount increments the count of requests by endpoint and status.
	IncRequestCount(ctx context.Context, endpoint string, method string, statusCode int)

	// TrackConcurrentRequests tracks the number of concurrent requests.
	TrackConcurrentRequests(ctx context.Context, endpoint string, f func() error) error
}

// MetricsMiddleware creates middleware that records API metrics.
func MetricsMiddleware(metrics APIMetrics) HTTPMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			endpoint := r.URL.Path
			method := r.Method

			metricsWriter := &MetricsResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default to 200
			}

			metrics.IncRequestCount(r.Context(), endpoint, method, http.StatusOK) // Increment before knowing status
			err := metrics.TrackConcurrentRequests(r.Context(), endpoint, func() error {
				next.ServeHTTP(metricsWriter, r)
				return nil
			})
			duration := time.Since(start)
			statusCode := metricsWriter.StatusCode()

			metrics.ObserveRequestLatency(r.Context(), endpoint, method, statusCode, duration)

			// If there was an error in the middleware itself (not from the handler).
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		})
	}
}
