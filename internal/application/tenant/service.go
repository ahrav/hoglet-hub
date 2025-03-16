package tenant

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ahrav/hoglet-hub/internal/application/workflow"
	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
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
	mu              sync.RWMutex
	activeWorkflows map[int64]workflow.Workflow

	logger *logger.Logger
	tracer trace.Tracer
}

// NewService creates a new tenant service with the required repositories.
// It initializes the workflow tracking map needed for asynchronous operations.
func NewService(
	tenantRepo tenant.Repository,
	operationRepo operation.Repository,
	logger *logger.Logger,
	tracer trace.Tracer,
) *Service {
	return &Service{
		tenantRepo:      tenantRepo,
		operationRepo:   operationRepo,
		activeWorkflows: make(map[int64]workflow.Workflow),
		logger:          logger.With("component", "tenant_service"),
		tracer:          tracer,
	}
}

// Create initiates tenant creation and returns tenant ID and operation information.
// It performs validation, creates necessary domain entities, and launches an async workflow.
func (s *Service) Create(ctx context.Context, params CreateParams) (*CreateResult, error) {
	name, region, tier, isolationGroupID := params.Name, params.Region, params.Tier, params.IsolationGroupID
	logger := logger.NewLoggerContext(s.logger.With(
		"operation_type", "create",
		"tenant_name", name,
		"region", region,
		"tier", tier,
		"isolation_group_id", isolationGroupID,
	))
	ctx, span := s.tracer.Start(ctx, "tenant.Create", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("region", string(region)),
		attribute.String("tier", string(tier)),
		attribute.Int64("isolation_group_id", *isolationGroupID),
	))
	defer span.End()

	existingTenant, err := s.tenantRepo.FindByName(ctx, name)
	if err != nil && !errors.Is(err, tenant.ErrTenantNotFound) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error checking existing tenant")
		return nil, fmt.Errorf("error checking existing tenant (%s): %w", name, err)
	}

	if existingTenant != nil {
		span.RecordError(tenant.ErrTenantAlreadyExists)
		span.SetStatus(codes.Error, "tenant already exists")
		return nil, tenant.ErrTenantAlreadyExists
	}

	newTenant, err := tenant.NewTenant(name, region, tier, isolationGroupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error creating tenant")
		return nil, err
	}
	span.AddEvent("tenant created")

	tenantID, err := s.tenantRepo.Create(ctx, newTenant)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error persisting tenant")
		return nil, fmt.Errorf("failed to persist tenant (%s): %w", name, err)
	}
	span.SetAttributes(attribute.Int64("tenant_id", tenantID))
	logger.Add("tenant_id", tenantID)
	span.AddEvent("tenant persisted")
	logger.Info(ctx, "tenant created")

	newOperation, err := operation.NewTenantCreateOperation(
		tenantID,
		name,
		string(region),
		string(tier),
		isolationGroupID,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error creating operation")
		return nil, fmt.Errorf("failed to create operation for tenant (%s): %w", name, err)
	}
	span.AddEvent("operation created")

	operationID, err := s.operationRepo.Create(ctx, newOperation)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error persisting operation")
		return nil, fmt.Errorf("failed to persist operation for tenant (%s): %w", name, err)
	}
	span.AddEvent("operation persisted")
	logger.Add("operation_id", operationID)
	span.SetAttributes(attribute.Int64("operation_id", operationID))
	logger.Info(ctx, "operation created")

	newOperation.ID = operationID
	creationWorkflow := workflow.NewTenantCreationWorkflow(
		newTenant,
		tenantID,
		newOperation,
		s.tenantRepo,
		s.operationRepo,
		s.logger,
		s.tracer,
	)
	span.AddEvent("create workflow created")

	s.mu.Lock()
	s.activeWorkflows[operationID] = creationWorkflow
	s.mu.Unlock()

	// Start workflow execution in background.
	creationWorkflow.Start(ctx)
	span.AddEvent(" async create workflow started")

	// Set up goroutine to handle workflow completion and cleanup.
	go s.handleWorkflowCompletion(ctx, operationID, creationWorkflow)

	logger.Info(ctx, "async create workflow started")
	span.AddEvent("async create workflow started")
	span.SetStatus(codes.Ok, "tenant creation process started")

	return &CreateResult{TenantID: tenantID, OperationID: operationID, Workflow: creationWorkflow}, nil
}

// Delete initiates tenant deletion and returns operation information.
// It verifies the tenant exists, creates a tracking operation, and launches an async workflow.
// TODO: Does this need to be async?
func (s *Service) Delete(ctx context.Context, tenantID int64) (*DeleteResult, error) {
	logger := logger.NewLoggerContext(s.logger.With("operation_type", "delete", "tenant_id", tenantID))
	ctx, span := s.tracer.Start(ctx, "tenant.Delete", trace.WithAttributes(
		attribute.Int64("tenant_id", tenantID),
	))
	defer span.End()

	t, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error finding tenant")
		return nil, fmt.Errorf("error finding tenant (%d): %w", tenantID, err)
	}

	if t == nil {
		span.RecordError(tenant.ErrTenantNotFound)
		span.SetStatus(codes.Error, "tenant not found")
		return nil, tenant.ErrTenantNotFound
	}

	newOperation, err := operation.NewTenantDeleteOperation(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error creating operation")
		return nil, fmt.Errorf("failed to create operation for tenant (%d): %w", tenantID, err)
	}
	span.AddEvent("operation created")

	operationID, err := s.operationRepo.Create(ctx, newOperation)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error persisting operation")
		return nil, fmt.Errorf("failed to persist operation for tenant (%d): %w", tenantID, err)
	}
	logger.Add("operation_id", operationID)
	span.SetAttributes(attribute.Int64("operation_id", operationID))
	span.AddEvent("operation persisted")
	logger.Info(ctx, "operation created")

	newOperation.ID = operationID
	deletionWorkflow := workflow.NewTenantDeletionWorkflow(
		t,
		tenantID,
		newOperation,
		s.tenantRepo,
		s.operationRepo,
		s.logger,
		s.tracer,
	)
	span.AddEvent("delete workflow created")

	s.mu.Lock()
	s.activeWorkflows[operationID] = deletionWorkflow
	s.mu.Unlock()

	// Start workflow execution in background.
	deletionWorkflow.Start(ctx)

	// Set up goroutine to handle workflow completion and cleanup.
	go s.handleWorkflowCompletion(ctx, operationID, deletionWorkflow)

	logger.Info(ctx, "async delete workflow started")
	span.AddEvent("async delete workflow started")
	span.SetStatus(codes.Ok, "tenant deletion process started")

	return &DeleteResult{OperationID: operationID, Workflow: deletionWorkflow}, nil
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
func (s *Service) handleWorkflowCompletion(ctx context.Context, operationID int64, workflow workflow.Workflow) {
	span := trace.SpanFromContext(ctx)
	defer span.End()

	span.AddEvent("waiting for workflow completion")
	s.logger.Debug(ctx, "waiting for workflow completion")
	// Wait for workflow to complete.
	// TODO: Handle the result we get back.
	<-workflow.ResultChan()
	span.AddEvent("workflow completed")

	s.mu.Lock()
	delete(s.activeWorkflows, operationID)
	s.mu.Unlock()

	s.logger.Info(ctx, "workflow cleanup complete")
	span.AddEvent("workflow cleanup complete")
}
