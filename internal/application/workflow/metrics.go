package workflow

import (
	"context"
	"time"
)

// ProvisioningMetrics defines metrics for tenant provisioning operations.
type ProvisioningMetrics interface {
	// IncProvisioningSuccess increments the count of successful tenant provisioning operations.
	IncProvisioningSuccess(ctx context.Context, tenantTier string, region string)

	// IncProvisioningFailure increments the count of failed tenant provisioning operations.
	IncProvisioningFailure(ctx context.Context, tenantTier string, region string, reason string)

	// ObserveProvisioningDuration records how long it took to provision a tenant.
	ObserveProvisioningDuration(ctx context.Context, tenantTier string, region string, duration time.Duration)

	// ObserveProvisioningStageDuration records how long a specific provisioning stage took.
	ObserveProvisioningStageDuration(ctx context.Context, stage string, tenantTier string, region string, duration time.Duration)

	// SetConcurrentProvisioningOps sets the number of concurrent provisioning operations.
	// SetConcurrentProvisioningOps(ctx context.Context, count int)

	// IncTenantDeletionSuccess increments the count of successful tenant deletions.
	IncTenantDeletionSuccess(ctx context.Context, tenantTier string, region string)

	// IncTenantDeletionFailure increments the count of failed tenant deletions.
	IncTenantDeletionFailure(ctx context.Context, tenantTier string, region string, reason string)

	// ObserveTenantDeletionDuration records how long it took to delete a tenant.
	ObserveTenantDeletionDuration(ctx context.Context, tenantTier string, region string, duration time.Duration)

	// TODO: Add domain metrics.
}
