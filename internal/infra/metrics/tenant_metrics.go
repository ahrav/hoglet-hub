package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/ahrav/hoglet-hub/internal/application/workflow"
)

var _ workflow.ProvisioningMetrics = (*tenantMetrics)(nil)

type tenantMetrics struct {
	// Provisioning metrics.
	provisioningSuccess       metric.Int64Counter
	provisioningFailure       metric.Int64Counter
	provisioningDuration      metric.Float64Histogram
	provisioningStageDuration metric.Float64Histogram
	concurrentProvisioningOps metric.Int64UpDownCounter
	tenantDeletionSuccess     metric.Int64Counter
	tenantDeletionFailure     metric.Int64Counter
	tenantDeletionDuration    metric.Float64Histogram

	// TODO: Add domain metrics.
}

// newTenantMetrics creates a new TenantMetrics instance.
func newTenantMetrics(mp metric.MeterProvider) (*tenantMetrics, error) {
	meter := mp.Meter(namespace, metric.WithInstrumentationVersion("v0.1.0"))

	m := new(tenantMetrics)
	var err error

	// Initialize provisioning metrics
	if m.provisioningSuccess, err = meter.Int64Counter(
		"tenant_provisioning_success_total",
		metric.WithDescription("Total number of successful tenant provisioning operations"),
	); err != nil {
		return nil, err
	}

	if m.provisioningFailure, err = meter.Int64Counter(
		"tenant_provisioning_failure_total",
		metric.WithDescription("Total number of failed tenant provisioning operations"),
	); err != nil {
		return nil, err
	}

	if m.provisioningDuration, err = meter.Float64Histogram(
		"tenant_provisioning_duration_seconds",
		metric.WithDescription("Duration of tenant provisioning operations in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if m.provisioningStageDuration, err = meter.Float64Histogram(
		"tenant_provisioning_stage_duration_seconds",
		metric.WithDescription("Duration of tenant provisioning stages in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if m.concurrentProvisioningOps, err = meter.Int64UpDownCounter(
		"tenant_concurrent_provisioning_operations",
		metric.WithDescription("Number of concurrent tenant provisioning operations"),
	); err != nil {
		return nil, err
	}

	if m.tenantDeletionSuccess, err = meter.Int64Counter(
		"tenant_deletion_success_total",
		metric.WithDescription("Total number of successful tenant deletion operations"),
	); err != nil {
		return nil, err
	}

	if m.tenantDeletionFailure, err = meter.Int64Counter(
		"tenant_deletion_failure_total",
		metric.WithDescription("Total number of failed tenant deletion operations"),
	); err != nil {
		return nil, err
	}

	if m.tenantDeletionDuration, err = meter.Float64Histogram(
		"tenant_deletion_duration_seconds",
		metric.WithDescription("Duration of tenant deletion operations in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *tenantMetrics) IncProvisioningSuccess(ctx context.Context, tenantTier string, region string) {
	m.provisioningSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("tier", tenantTier),
		attribute.String("region", region),
	))
}

func (m *tenantMetrics) IncProvisioningFailure(ctx context.Context, tenantTier string, region string, reason string) {
	m.provisioningFailure.Add(ctx, 1, metric.WithAttributes(
		attribute.String("tier", tenantTier),
		attribute.String("region", region),
		attribute.String("reason", reason),
	))
}

func (m *tenantMetrics) ObserveProvisioningDuration(ctx context.Context, tenantTier string, region string, duration time.Duration) {
	m.provisioningDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("tier", tenantTier),
		attribute.String("region", region),
	))
}

func (m *tenantMetrics) ObserveProvisioningStageDuration(ctx context.Context, stage string, tenantTier string, region string, duration time.Duration) {
	m.provisioningStageDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("stage", stage),
		attribute.String("tier", tenantTier),
		attribute.String("region", region),
	))
}

// func (m *tenantMetrics) SetConcurrentProvisioningOps(ctx context.Context, count int) {
// 	// First reset to 0, then set to the new count to handle cases where count has decreased
// 	m.concurrentProvisioningOps.Add(ctx, int64(count)-getCurrentCount(m.concurrentProvisioningOps, ctx))
// }

func (m *tenantMetrics) IncTenantDeletionSuccess(ctx context.Context, tenantTier string, region string) {
	m.tenantDeletionSuccess.Add(ctx, 1, metric.WithAttributes(
		attribute.String("tier", tenantTier),
		attribute.String("region", region),
	))
}

func (m *tenantMetrics) IncTenantDeletionFailure(ctx context.Context, tenantTier string, region string, reason string) {
	m.tenantDeletionFailure.Add(ctx, 1, metric.WithAttributes(
		attribute.String("tier", tenantTier),
		attribute.String("region", region),
		attribute.String("reason", reason),
	))
}

func (m *tenantMetrics) ObserveTenantDeletionDuration(ctx context.Context, tenantTier string, region string, duration time.Duration) {
	m.tenantDeletionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("tier", tenantTier),
		attribute.String("region", region),
	))
}
