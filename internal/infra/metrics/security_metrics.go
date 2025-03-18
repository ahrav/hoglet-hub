package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/ahrav/hoglet-hub/internal/application/security"
)

var _ security.SecurityMetrics = (*securityMetrics)(nil)

type securityMetrics struct {
	actorActivity metric.Int64Counter
	ipAnomalies   metric.Int64Counter
}

// newSecurityMetrics creates a new SecurityMetrics instance.
func newSecurityMetrics(mp metric.MeterProvider) (*securityMetrics, error) {
	meter := mp.Meter(namespace, metric.WithInstrumentationVersion("v0.1.0"))

	m := new(securityMetrics)
	var err error

	// Initialize security metrics
	if m.actorActivity, err = meter.Int64Counter(
		"security_actor_activity_total",
		metric.WithDescription("Total activities by actors"),
	); err != nil {
		return nil, err
	}

	if m.ipAnomalies, err = meter.Int64Counter(
		"security_ip_anomalies_total",
		metric.WithDescription("Total IP anomalies detected"),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *securityMetrics) RecordActorActivity(ctx context.Context, actorID string, actorType string, activity string) {
	m.actorActivity.Add(ctx, 1, metric.WithAttributes(
		attribute.String("actor_id", actorID),
		attribute.String("actor_type", actorType),
		attribute.String("activity", activity),
	))
}

func (m *securityMetrics) RecordIPAnomaly(ctx context.Context, ip string, anomalyType string, severity int) {
	m.ipAnomalies.Add(ctx, 1, metric.WithAttributes(
		attribute.String("ip", ip),
		attribute.String("anomaly_type", anomalyType),
		attribute.Int("severity", severity),
	))
}
