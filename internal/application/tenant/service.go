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
// It provides the tenant ID and operation ID which can be used to track
// the asynchronous provisioning progress through the operations API.
//
// Following the Go idiom "accept interfaces, return concrete types", we expose
// only the operation identifiers rather than returning workflow interfaces.
// This makes the API more stable and decouples consumers from implementation details.
type CreateResult struct {
	TenantID    int64
	OperationID int64
}

// DeleteResult contains the output of a tenant deletion operation.
// It provides the operation ID which can be used to track
// the asynchronous deletion progress through the operations API.
//
// Following the Go idiom "accept interfaces, return concrete types", we expose
// only the operation identifier rather than returning workflow interfaces.
// This makes the API more stable and decouples consumers from implementation details.
type DeleteResult struct{ OperationID int64 }

// WorkflowFactory creates workflows for tenant operations.
//
// This factory pattern provides several important architectural benefits:
//  1. Separation of concerns: It decouples workflow creation logic from the tenant service,
//     following the Single Responsibility Principle.
//  2. Extensibility: New workflow types or provisioning strategies can be added without
//     modifying the core tenant service code.
//
// IMPLEMENTATION NOTE: In an ideal Go design following "accept interfaces, return concrete types",
// this interface would return concrete workflow types. However, since we need to maintain workflow
// implementation details in their own package and support multiple workflow implementations,
// we use a workflow.Workflow interface as a return type. This is a deliberate trade-off between
// strict adherence to Go idioms and maintaining a clean architecture with proper separation of concerns.
//
// The workflow.Workflow interface is considered an internal implementation detail
// of the tenant service, and is never exposed directly to external API clients.
type WorkflowFactory interface {
	NewTenantCreationWorkflow(
		t *tenant.Tenant,
		tenantID int64,
		op *operation.Operation,
	) workflow.Workflow
	NewTenantDeletionWorkflow(
		t *tenant.Tenant,
		tenantID int64,
		op *operation.Operation,
	) workflow.Workflow
}

// DefaultWorkflowFactory is the default implementation of WorkflowFactory.
//
// This implementation creates standard workflows for tenant operations in production
// environments. It encapsulates the details of workflow creation including
// dependency management and initialization logic. By isolating these concerns,
// the tenant service remains focused on orchestration rather than implementation
// details of each workflow.
//
// The factory pattern also allows us to extend the system with specialized factories
// for different scenarios such as different cloud providers, regional requirements,
// or tenant tiers with different resource allocations.
type DefaultWorkflowFactory struct {
	tenantRepo    tenant.Repository
	operationRepo operation.Repository
	logger        *logger.Logger
	tracer        trace.Tracer
}

// NewDefaultWorkflowFactory creates a new default workflow factory
func NewDefaultWorkflowFactory(
	tenantRepo tenant.Repository,
	operationRepo operation.Repository,
	logger *logger.Logger,
	tracer trace.Tracer,
) *DefaultWorkflowFactory {
	return &DefaultWorkflowFactory{tenantRepo, operationRepo, logger, tracer}
}

// NewTenantCreationWorkflow creates a new tenant creation workflow
func (f *DefaultWorkflowFactory) NewTenantCreationWorkflow(
	t *tenant.Tenant,
	tenantID int64,
	op *operation.Operation,
) workflow.Workflow {
	return workflow.NewTenantCreationWorkflow(
		t,
		tenantID,
		op,
		f.tenantRepo,
		f.operationRepo,
		f.logger,
		f.tracer,
	)
}

// NewTenantDeletionWorkflow creates a new tenant deletion workflow
func (f *DefaultWorkflowFactory) NewTenantDeletionWorkflow(
	t *tenant.Tenant,
	tenantID int64,
	op *operation.Operation,
) workflow.Workflow {
	return workflow.NewTenantDeletionWorkflow(
		t,
		tenantID,
		op,
		f.tenantRepo,
		f.operationRepo,
		f.logger,
		f.tracer,
	)
}

// Service provides tenant-related application services.
// It orchestrates tenant lifecycle operations and manages the associated workflows.
//
// This service follows the "accept interfaces, return concrete types" principle by:
// 1. Accepting interfaces (repositories, factories) for flexibility and testability
// 2. Returning concrete result types (CreateResult, DeleteResult) to clients
// 3. Never exposing internal workflow interfaces to external consumers
//
// The service maintains internal references to workflows via the workflow.Workflow
// interface, but these are an implementation detail hidden from API clients.
type Service struct {
	tenantRepo    tenant.Repository
	operationRepo operation.Repository
	// Track active workflows for monitoring and management.
	mu              sync.RWMutex
	activeWorkflows map[int64]workflow.Workflow
	workflowFactory WorkflowFactory

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
	factory := NewDefaultWorkflowFactory(tenantRepo, operationRepo, logger, tracer)
	return &Service{
		tenantRepo:      tenantRepo,
		operationRepo:   operationRepo,
		activeWorkflows: make(map[int64]workflow.Workflow),
		workflowFactory: factory,
		logger:          logger.With("component", "tenant_service"),
		tracer:          tracer,
	}
}

// NewServiceWithWorkflowFactory creates a new tenant service with a custom workflow factory.
//
// This constructor supports the Strategy pattern by allowing different workflow
// creation strategies to be injected. This is valuable in several scenarios:
//
// 1. Different cloud providers might require different provisioning workflows
// 2. Development/staging environments may use simplified workflows
// 3. Different regions might have specialized compliance or infrastructure requirements
// 4. Future tenant tiers might need differentiated provisioning strategies
//
// This flexibility makes the system more adaptable to changing business requirements
// without requiring modifications to the core tenant service.
func NewServiceWithWorkflowFactory(
	tenantRepo tenant.Repository,
	operationRepo operation.Repository,
	workflowFactory WorkflowFactory,
	logger *logger.Logger,
	tracer trace.Tracer,
) *Service {
	return &Service{
		tenantRepo:      tenantRepo,
		operationRepo:   operationRepo,
		activeWorkflows: make(map[int64]workflow.Workflow),
		workflowFactory: workflowFactory,
		logger:          logger.With("component", "tenant_service"),
		tracer:          tracer,
	}
}

// Create initiates tenant creation and returns tenant ID and operation information.
// It performs validation, creates necessary domain entities, and launches an async workflow.
// TODO: Come back and deal with isolation group ID.
func (s *Service) Create(ctx context.Context, params CreateParams) (*CreateResult, error) {
	name, region, tier, isolationGroupID := params.Name, params.Region, params.Tier, params.IsolationGroupID
	logger := logger.NewLoggerContext(s.logger.With(
		"operation_type", "create",
		"tenant_name", name,
		"region", region,
		"tier", tier,
	))
	ctx, span := s.tracer.Start(ctx, "tenant.Create", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("region", string(region)),
		attribute.String("tier", string(tier)),
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

	creationWorkflow := s.workflowFactory.NewTenantCreationWorkflow(
		newTenant,
		tenantID,
		newOperation,
	)
	span.AddEvent("create workflow created")

	s.mu.Lock()
	s.activeWorkflows[operationID] = creationWorkflow
	s.mu.Unlock()

	// Start workflow execution in background.
	// Using an asynchronous design pattern ensures the API remains responsive
	// while long-running provisioning operations execute. This separation between
	// request handling and resource provisioning is crucial for scalability
	// and enables progress tracking through the operations API.
	creationWorkflow.Start(ctx)
	span.AddEvent("async create workflow started")

	// Set up goroutine to handle workflow completion and cleanup.
	go s.handleWorkflowCompletion(ctx, operationID, creationWorkflow)

	logger.Info(ctx, "async create workflow started")
	span.AddEvent("async create workflow started")
	span.SetStatus(codes.Ok, "tenant creation process started")

	return &CreateResult{TenantID: tenantID, OperationID: operationID}, nil
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

	// Create workflow using the factory
	// The factory pattern encapsulates the complexity of workflow creation and allows
	// for different provisioning strategies based on tenant characteristics, infrastructure
	// requirements, and regional constraints. This improves maintainability and makes
	// the system more adaptable to changing business requirements.
	deletionWorkflow := s.workflowFactory.NewTenantDeletionWorkflow(
		t,
		tenantID,
		newOperation,
	)
	span.AddEvent("delete workflow created")

	s.mu.Lock()
	s.activeWorkflows[operationID] = deletionWorkflow
	s.mu.Unlock()

	// Start workflow execution in background.
	// This asynchronous execution allows the deletion process to proceed independently
	// of the API request, ensuring good user experience while potentially lengthy
	// resource cleanup operations occur. The operation can be monitored through
	// the operations API.
	deletionWorkflow.Start(ctx)

	// Set up goroutine to handle workflow completion and cleanup.
	go s.handleWorkflowCompletion(ctx, operationID, deletionWorkflow)

	logger.Info(ctx, "async delete workflow started")
	span.AddEvent("async delete workflow started")
	span.SetStatus(codes.Ok, "tenant deletion process started")

	return &DeleteResult{OperationID: operationID}, nil
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
//
// The method is deliberately designed to run in its own goroutine to avoid blocking
// the service API. This is important since workflows can take a long time to complete.
//
// Design considerations:
// 1. Asynchronous cleanup: Runs in a separate goroutine to avoid blocking API responses
// 2. Resource management: Prevents memory leaks by removing completed workflows
// 3. Wait mechanism: Blocks on the workflow's result channel to ensure proper sequencing
// 4. Thread safety: Uses mutex to safely manage the shared activeWorkflows map
// 5. Observability: Adds tracing spans and logging for monitoring workflow completion
//
// This design pattern complements the factory pattern by handling the lifecycle
// of workflow objects created by the factory, ensuring proper resource cleanup regardless
// of which specific workflow implementation was created.
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
