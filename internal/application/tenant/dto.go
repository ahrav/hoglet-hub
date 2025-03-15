package tenant

import "github.com/ahrav/hoglet-hub/internal/domain/tenant"

// CreateTenantRequest represents input for tenant creation
type CreateTenantRequest struct {
	Name             string        `json:"name"`
	Region           tenant.Region `json:"region"`
	Tier             *tenant.Tier  `json:"tier,omitempty"`
	IsolationGroupID *int64        `json:"isolation_group_id,omitempty"`
}

// TenantCreatedResponse represents output from tenant creation
type TenantCreatedResponse struct {
	TenantID    int64             `json:"tenant_id"`
	Name        string            `json:"name"`
	OperationID int64             `json:"operation_id"`
	Status      string            `json:"status"`
	Links       map[string]string `json:"_links"`
}

// AsyncOperationResponse represents an async operation response
type AsyncOperationResponse struct {
	OperationID int64             `json:"operation_id"`
	Status      string            `json:"status"`
	TenantID    *int64            `json:"tenant_id,omitempty"`
	Links       map[string]string `json:"_links"`
}
