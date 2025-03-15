package tenant

import (
	"context"
	"fmt"

	"github.com/ahrav/hoglet-hub/internal/application/workflow"
	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
)

// CreateParams contains parameters for creating a new tenant.
// These parameters define the tenant's characteristics and resource allocation.
type CreateParams struct {
	Name             string
	Region           tenant.Region
	Tier             tenant.Tier
	IsolationGroupID *int64
}

// CreateResult contains the output of a tenant creation operation.
// It provides access to both synchronous results (IDs) and the asynchronous workflow.
type CreateResult struct {
	TenantID    int64
	OperationID int64
	Workflow    workflow.Workflow
}

// DeleteResult contains the output of a tenant deletion operation.
// It provides access to both the operation tracking ID and the asynchronous workflow.
type DeleteResult struct {
	OperationID int64
	Workflow    workflow.Workflow
}

// Service provides tenant-related application services.
// It orchestrates tenant lifecycle operations and manages the associated workflows.
type Service struct {
	tenantRepo    tenant.Repository
	operationRepo operation.Repository
	// Track active workflows for monitoring and management.
	activeWorkflows map[int64]workflow.Workflow
}

// NewService creates a new tenant service with the required repositories.
// It initializes the workflow tracking map needed for asynchronous operations.
func NewService(tenantRepo tenant.Repository, operationRepo operation.Repository) *Service {
	return &Service{
		tenantRepo:      tenantRepo,
		operationRepo:   operationRepo,
		activeWorkflows: make(map[int64]workflow.Workflow),
	}
}

// Create initiates tenant creation and returns tenant ID and operation information.
// It performs validation, creates necessary domain entities, and launches an async workflow.
func (s *Service) Create(ctx context.Context, params CreateParams) (*CreateResult, error) {
	existingTenant, err := s.tenantRepo.FindByName(ctx, params.Name)
	if err != nil {
		return nil, fmt.Errorf("error checking existing tenant: %w", err)
	}

	if existingTenant != nil {
		return nil, tenant.ErrTenantAlreadyExists
	}

	newTenant, err := tenant.NewTenant(
		params.Name,
		params.Region,
		params.Tier,
		params.IsolationGroupID,
	)

	if err != nil {
		return nil, err
	}

	tenantID, err := s.tenantRepo.Create(ctx, newTenant)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	newOperation, err := operation.NewTenantCreateOperation(
		tenantID,
		params.Name,
		string(params.Region),
		string(params.Tier),
		params.IsolationGroupID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create operation: %w", err)
	}

	operationID, err := s.operationRepo.Create(ctx, newOperation)
	if err != nil {
		return nil, fmt.Errorf("failed to persist operation: %w", err)
	}

	newOperation.ID = operationID
	creationWorkflow := workflow.NewTenantCreationWorkflow(
		newTenant,
		tenantID,
		newOperation,
		s.tenantRepo,
		s.operationRepo,
	)

	s.activeWorkflows[operationID] = creationWorkflow

	// Start workflow execution in background.
	creationWorkflow.Start(ctx)

	// Set up goroutine to handle workflow completion and cleanup.
	go s.handleWorkflowCompletion(operationID, creationWorkflow.ResultChan())

	return &CreateResult{
		TenantID:    tenantID,
		OperationID: operationID,
		Workflow:    creationWorkflow,
	}, nil
}

// Delete initiates tenant deletion and returns operation information.
// It verifies the tenant exists, creates a tracking operation, and launches an async workflow.
func (s *Service) Delete(ctx context.Context, tenantID int64) (*DeleteResult, error) {
	t, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error finding tenant: %w", err)
	}

	if t == nil {
		return nil, tenant.ErrTenantNotFound
	}

	newOperation, err := operation.NewTenantDeleteOperation(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create operation: %w", err)
	}

	operationID, err := s.operationRepo.Create(ctx, newOperation)
	if err != nil {
		return nil, fmt.Errorf("failed to persist operation: %w", err)
	}

	newOperation.ID = operationID
	deletionWorkflow := workflow.NewTenantDeletionWorkflow(
		t,
		tenantID,
		newOperation,
		s.tenantRepo,
		s.operationRepo,
	)

	s.activeWorkflows[operationID] = deletionWorkflow

	// Start workflow execution in background.
	deletionWorkflow.Start(ctx)

	// Set up goroutine to handle workflow completion and cleanup.
	go s.handleWorkflowCompletion(operationID, deletionWorkflow.ResultChan())

	return &DeleteResult{
		OperationID: operationID,
		Workflow:    deletionWorkflow,
	}, nil
}

// GetOperationStatus retrieves the current status of an operation.
// This provides visibility into the progress of asynchronous tenant operations.
func (s *Service) GetOperationStatus(ctx context.Context, operationID int64) (*operation.Operation, error) {
	op, err := s.operationRepo.FindByID(ctx, operationID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving operation: %w", err)
	}

	if op == nil {
		return nil, operation.ErrOperationNotFound
	}

	return op, nil
}

// handleWorkflowCompletion cleans up workflow resources after completion.
// This prevents memory leaks by removing references to completed workflows.
func (s *Service) handleWorkflowCompletion(operationID int64, resultChan <-chan workflow.WorkflowResult) {
	// Wait for workflow to complete.
	<-resultChan
	delete(s.activeWorkflows, operationID)
}
