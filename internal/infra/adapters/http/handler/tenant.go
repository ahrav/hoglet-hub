package httphandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/ahrav/hoglet-hub/api/v1/server"
	appTenant "github.com/ahrav/hoglet-hub/internal/application/tenant"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
)

// TenantHandler implements the tenant-related API endpoints by translating
// HTTP requests to application service calls and mapping responses back to HTTP.
type TenantHandler struct{ tenantService *appTenant.Service }

// NewTenantHandler creates a new tenant handler with the provided tenant service.
// The tenant service is used to execute the business logic for tenant operations.
func NewTenantHandler(tenantService *appTenant.Service) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

// CreateTenant handles tenant creation requests by validating input,
// transforming API models to domain models, and delegating to the tenant service.
// It returns appropriate HTTP responses based on the operation result.
func (h *TenantHandler) CreateTenant(ctx context.Context, req server.CreateTenantRequestObject) (server.CreateTenantResponseObject, error) {
	if req.Body == nil {
		return server.CreateTenant400JSONResponse{
			Error:   "invalid_request",
			Message: "Missing request body",
		}, nil
	}

	// Map API tier enum to domain tier.
	var tier tenant.Tier
	if req.Body.Tier != nil {
		switch *req.Body.Tier {
		case server.TenantCreateTierEnterprise:
			tier = tenant.TierEnterprise
		case server.TenantCreateTierPro:
			tier = tenant.TierPro
		case server.TenantCreateTierFree:
			tier = tenant.TierFree
		default:
			tier = tenant.TierFree
		}
	} else {
		tier = tenant.TierFree
	}

	// Map API region enum to domain region.
	var region tenant.Region
	switch req.Body.Region {
	case server.Eu1:
		region = tenant.RegionEU1
	case server.Eu2:
		region = tenant.RegionEU2
	case server.Eu3:
		region = tenant.RegionEU3
	case server.Eu4:
		region = tenant.RegionEU4
	case server.Us1:
		region = tenant.RegionUS1
	case server.Us2:
		region = tenant.RegionUS2
	case server.Us3:
		region = tenant.RegionUS3
	case server.Us4:
		region = tenant.RegionUS4
	}

	params := appTenant.CreateParams{
		Name:             req.Body.Name,
		Region:           region,
		Tier:             tier,
		IsolationGroupID: req.Body.IsolationGroupId,
	}

	// Delegate to application service and handle domain-specific errors.
	result, err := h.tenantService.Create(ctx, params)
	if err != nil {
		switch {
		case errors.Is(err, tenant.ErrTenantAlreadyExists):
			return server.CreateTenant409JSONResponse{
				Error:   "tenant_already_exists",
				Message: "A tenant with this name already exists",
			}, nil
		case errors.Is(err, tenant.ErrInvalidName):
			return server.CreateTenant400JSONResponse{
				Error:   "invalid_tenant_name",
				Message: "Tenant name must contain only lowercase letters, numbers, and hyphens",
			}, nil
		case errors.Is(err, tenant.ErrInvalidRegion):
			return server.CreateTenant400JSONResponse{
				Error:   "invalid_region",
				Message: "Invalid region specified",
			}, nil
		case errors.Is(err, tenant.ErrInvalidTier):
			return server.CreateTenant400JSONResponse{
				Error:   "invalid_tier",
				Message: "Invalid tier specified",
			}, nil
		default:
			return server.CreateTenant500JSONResponse{
				Error:   "internal_error",
				Message: "An internal error occurred",
				Details: &map[string]any{
					"error": err.Error(),
				},
			}, nil
		}
	}

	return server.CreateTenant202JSONResponse{
		Links: server.Links{
			"self":      fmt.Sprintf("/tenants/%d", result.TenantID),
			"operation": fmt.Sprintf("/operations/%d", result.OperationID),
		},
		Name:        req.Body.Name,
		OperationId: result.OperationID,
		Status:      server.Pending,
		TenantId:    result.TenantID,
	}, nil
}

// DeleteTenant handles tenant deletion requests by delegating to the tenant service
// and mapping the result to appropriate HTTP responses. It initiates an asynchronous
// deletion operation and returns information about the operation.
func (h *TenantHandler) DeleteTenant(
	ctx context.Context,
	req server.DeleteTenantRequestObject,
) (server.DeleteTenantResponseObject, error) {
	result, err := h.tenantService.Delete(ctx, req.TenantId)
	if err != nil {
		switch {
		case errors.Is(err, tenant.ErrTenantNotFound):
			return server.DeleteTenant404JSONResponse{
				Error:   "tenant_not_found",
				Message: "The specified tenant does not exist",
			}, nil
		default:
			return server.DeleteTenant500JSONResponse{
				Error:   "internal_error",
				Message: "An internal error occurred",
				Details: &map[string]any{
					"error": err.Error(),
				},
			}, nil
		}
	}

	tenantID := req.TenantId
	return server.DeleteTenant202JSONResponse{
		Links: server.Links{
			"self":   fmt.Sprintf("/operations/%d", result.OperationID),
			"tenant": fmt.Sprintf("/tenants/%d", tenantID),
		},
		OperationId: result.OperationID,
		Status:      server.Pending,
		TenantId:    &tenantID,
	}, nil
}
