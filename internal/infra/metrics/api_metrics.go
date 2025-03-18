package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/ahrav/hoglet-hub/internal/application/sdk/mid"
)

var _ mid.APIMetrics = (*apiMetrics)(nil)

// apiMetrics implements TenantMetrics.
type apiMetrics struct {
	requestLatency     metric.Float64Histogram
	requestCount       metric.Int64Counter
	concurrentRequests metric.Int64UpDownCounter
}

// newAPIMetrics creates a new APIStats instance.
func newAPIMetrics(mp metric.MeterProvider) (*apiMetrics, error) {
	meter := mp.Meter(namespace, metric.WithInstrumentationVersion("v0.1.0"))

	m := new(apiMetrics)
	var err error

	// Initialize API metrics
	if m.requestLatency, err = meter.Float64Histogram(
		"api_request_latency_seconds",
		metric.WithDescription("Latency of API requests in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if m.requestCount, err = meter.Int64Counter(
		"api_request_total",
		metric.WithDescription("Total number of API requests"),
	); err != nil {
		return nil, err
	}

	if m.concurrentRequests, err = meter.Int64UpDownCounter(
		"api_concurrent_requests",
		metric.WithDescription("Number of concurrent API requests"),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *apiMetrics) ObserveRequestLatency(ctx context.Context, endpoint string, method string, statusCode int, duration time.Duration) {
	m.requestLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("method", method),
		attribute.Int("status_code", statusCode),
	))
}

// IncRequestCount increments the count of requests by endpoint and status.
func (m *apiMetrics) IncRequestCount(ctx context.Context, endpoint string, method string, statusCode int) {
	m.requestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("method", method),
		attribute.Int("status_code", statusCode),
	))
}

// TrackConcurrentRequests tracks the number of concurrent requests.
func (m *apiMetrics) TrackConcurrentRequests(ctx context.Context, endpoint string, f func() error) error {
	m.concurrentRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
	))
	defer m.concurrentRequests.Add(ctx, -1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
	))

	return f()
}
