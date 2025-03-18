package metrics

import (
	"go.opentelemetry.io/otel/metric"

	"github.com/ahrav/hoglet-hub/internal/application/health"
	"github.com/ahrav/hoglet-hub/internal/application/sdk/mid"
	"github.com/ahrav/hoglet-hub/internal/application/security"
	"github.com/ahrav/hoglet-hub/internal/application/tenant"
)

const namespace = "hoglet_hub"

// Registry provides access to all metric implementations.
// It centralizes the creation and management of metrics instances.
type Registry struct {
	API      mid.APIMetrics
	Tenant   tenant.ProvisioningMetrics
	Security security.SecurityMetrics
	Health   health.HealthMetrics
}

// NewRegistry creates and initializes all metrics implementations.
// It uses a single meter provider to ensure consistent configuration.
func NewRegistry(mp metric.MeterProvider) (*Registry, error) {
	apiMetrics, err := newAPIMetrics(mp)
	if err != nil {
		return nil, err
	}

	tenantMetrics, err := newTenantMetrics(mp)
	if err != nil {
		return nil, err
	}

	securityMetrics, err := newSecurityMetrics(mp)
	if err != nil {
		return nil, err
	}

	healthMetrics, err := newHealthMetrics(mp)
	if err != nil {
		return nil, err
	}

	return &Registry{
		API:      apiMetrics,
		Tenant:   tenantMetrics,
		Security: securityMetrics,
		Health:   healthMetrics,
	}, nil
}
