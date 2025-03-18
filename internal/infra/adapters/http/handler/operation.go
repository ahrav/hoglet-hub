package httphandler

import (
	"context"
	"errors"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/ahrav/hoglet-hub/api/v1/server"
	appOperation "github.com/ahrav/hoglet-hub/internal/application/operation"
	"github.com/ahrav/hoglet-hub/internal/domain/operation"
)

// OperationHandler implements the operation-related API endpoints.
// It serves as the HTTP interface layer for operation management functionalities,
// translating between HTTP requests/responses and application service calls.
type OperationHandler struct{ operationService *appOperation.Service }

// NewOperationHandler creates a new operation handler with the given operation service.
// The operation service is required to handle business logic operations.
func NewOperationHandler(operationService *appOperation.Service) *OperationHandler {
	return &OperationHandler{operationService: operationService}
}

// GetOperation handles HTTP requests for retrieving operation details by ID.
// It maps domain entities to API response objects and handles error cases
// with appropriate HTTP status codes and error messages.
func (h *OperationHandler) GetOperation(ctx context.Context, req server.GetOperationRequestObject) (server.GetOperationResponseObject, error) {
	// Fetch operation details from application service.
	op, err := h.operationService.GetByID(ctx, req.OperationId)
	if err != nil {
		switch {
		case errors.Is(err, operation.ErrOperationNotFound):
			return server.GetOperation404JSONResponse{
				Error:   "operation_not_found",
				Message: "The specified operation does not exist",
			}, nil
		default:
			return server.GetOperation500JSONResponse{
				Error:   "internal_error",
				Message: "An internal error occurred",
				Details: &map[string]any{
					"error": err.Error(),
				},
			}, nil
		}
	}

	var status server.OperationStatus
	switch op.Status {
	case operation.StatusPending:
		status = server.Pending
	case operation.StatusInProgress:
		status = server.InProgress
	case operation.StatusCompleted:
		status = server.Completed
	case operation.StatusFailed:
		status = server.Failed
	case operation.StatusCancelled:
		status = server.Cancelled
	}

	// Construct HATEOAS links for API discoverability.
	links := server.Links{
		"self": fmt.Sprintf("/operations/%d", op.ID),
	}
	if op.TenantID != nil {
		links["tenant"] = fmt.Sprintf("/tenants/%d", *op.TenantID)
	}

	var createdBy *openapi_types.Email
	if op.CreatedBy != nil {
		email := openapi_types.Email(*op.CreatedBy)
		createdBy = &email
	}

	return server.GetOperation200JSONResponse{
		Links:         links,
		Id:            op.ID,
		OperationType: op.Type.String(),
		Status:        status,
		TenantId:      op.TenantID,
		CreatedAt:     op.CreatedAt,
		StartedAt:     op.StartedAt,
		CompletedAt:   op.CompletedAt,
		UpdatedAt:     op.UpdatedAt,
		ErrorMessage:  op.ErrorMessage,
		Parameters:    &op.Parameters,
		Result:        &op.Result,
		CreatedBy:     createdBy,
	}, nil
}
