package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/ahrav/hoglet-hub/internal/application/health"
)

var _ health.HealthMetrics = (*healthMetrics)(nil)

type healthMetrics struct {
	systemHealth metric.Int64UpDownCounter
}

func newHealthMetrics(mp metric.MeterProvider) (*healthMetrics, error) {
	meter := mp.Meter(namespace, metric.WithInstrumentationVersion("v0.1.0"))

	m := new(healthMetrics)
	var err error

	if m.systemHealth, err = meter.Int64UpDownCounter(
		"system_health",
		metric.WithDescription("System health status"),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *healthMetrics) SetSystemHealth(ctx context.Context, status bool) {
	if status {
		m.systemHealth.Add(ctx, 1, metric.WithAttributes(
			attribute.Bool("status", status),
		))
	} else {
		m.systemHealth.Add(ctx, -1, metric.WithAttributes(
			attribute.Bool("status", status),
		))
	}
}
