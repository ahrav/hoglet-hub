// Package http provides HTTP server components for the hoglet-hub API.
package http

import (
	"context"
	"net/http"

	"github.com/trufflesecurity/hoglet-hub/api/v1/server"
	handler "github.com/trufflesecurity/hoglet-hub/internal/infra/adapters/http/handler"
)

// ServerAdapter implements the StrictServerInterface by delegating requests
// to the appropriate domain-specific handlers. It serves as an adapter between
// the generated API server and our business logic handlers.
type ServerAdapter struct {
	tenantHandler    *handler.TenantHandler
	operationHandler *handler.OperationHandler
}

// NewServerAdapter creates a new server adapter with the provided handlers.
// This constructor ensures all required handlers are properly initialized.
func NewServerAdapter(tenantHandler *handler.TenantHandler, operationHandler *handler.OperationHandler) *ServerAdapter {
	return &ServerAdapter{
		tenantHandler:    tenantHandler,
		operationHandler: operationHandler,
	}
}

// GetOperation delegates operation retrieval requests to the specialized operation handler.
// It implements part of the StrictServerInterface contract.
func (a *ServerAdapter) GetOperation(ctx context.Context, req server.GetOperationRequestObject) (server.GetOperationResponseObject, error) {
	return a.operationHandler.GetOperation(ctx, req)
}

// CreateTenant delegates tenant creation requests to the specialized tenant handler.
// It implements part of the StrictServerInterface contract.
func (a *ServerAdapter) CreateTenant(ctx context.Context, req server.CreateTenantRequestObject) (server.CreateTenantResponseObject, error) {
	return a.tenantHandler.CreateTenant(ctx, req)
}

// DeleteTenant delegates tenant deletion requests to the specialized tenant handler.
// It implements part of the StrictServerInterface contract.
func (a *ServerAdapter) DeleteTenant(ctx context.Context, req server.DeleteTenantRequestObject) (server.DeleteTenantResponseObject, error) {
	return a.tenantHandler.DeleteTenant(ctx, req)
}

// NewHTTPServer creates a configured HTTP server using the provided adapter.
// It wraps the server adapter with a strict handler to ensure request validation
// and proper error handling according to the API specification.
func NewHTTPServer(serverAdapter *ServerAdapter) http.Handler {
	strictHandler := server.NewStrictHandler(serverAdapter, nil)
	return server.Handler(strictHandler)
}
