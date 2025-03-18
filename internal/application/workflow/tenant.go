package workflow

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
)

// OperationType defines the type of tenant operation to be performed.
// This type enables polymorphic behavior of workflows through a discriminated union pattern.
type OperationType string

const (
	// OperationTypeCreate represents tenant creation operations.
	// This operation handles the provisioning of new tenant resources.
	OperationTypeCreate OperationType = "create"

	// OperationTypeDelete represents tenant deletion operations.
	// This operation handles the decommissioning of tenant resources.
	OperationTypeDelete OperationType = "delete"

	// TODO: Keep going...
	// Additional operation types like Update, Upgrade, Migrate can be added here
	// without modifying existing workflow implementations.
)

// TenantOperationWorkflow is a generic workflow that orchestrates tenant lifecycle operations.
// It applies the Strategy pattern by adapting its behavior based on the operation type.
type TenantOperationWorkflow struct {
	*BaseWorkflow
	operationType OperationType

	tenant        *tenant.Tenant
	tenantID      int64
	operation     *operation.Operation
	tenantRepo    tenant.Repository
	operationRepo operation.Repository

	logger  *logger.Logger
	tracer  trace.Tracer
	metrics ProvisioningMetrics
}

// NewTenantOperationWorkflow creates a new workflow for tenant operations (create/delete).
// This factory function dynamically constructs the appropriate workflow based on the operation type.
func NewTenantOperationWorkflow(
	opType OperationType,
	t *tenant.Tenant,
	tenantID int64,
	op *operation.Operation,
	tenantRepo tenant.Repository,
	operationRepo operation.Repository,
	logger *logger.Logger,
	tracer trace.Tracer,
	metrics ProvisioningMetrics,
) (*TenantOperationWorkflow, error) {
	workflow := &TenantOperationWorkflow{
		operationType: opType,
		tenant:        t,
		tenantID:      tenantID,
		operation:     op,
		tenantRepo:    tenantRepo,
		operationRepo: operationRepo,
		tracer:        tracer,
		metrics:       metrics,
	}

	// Define steps based on operation type.
	var steps []Step
	var componentName string // Used for tracing and logging.

	switch opType {
	case OperationTypeCreate:
		componentName = "tenant_creation_workflow"
		steps = []Step{
			{
				Name:        "initialize",
				Description: "Initialize tenant resources",
				Execute:     workflow.initializeTenant,
			},
			{
				Name:        "provision-database",
				Description: "Provision tenant database schema",
				Execute:     workflow.provisionDatabase,
			},
			{
				Name:        "setup-secrets",
				Description: "Set up tenant secrets",
				Execute:     workflow.setupSecrets,
			},
			{
				Name:        "deploy-resources",
				Description: "Deploy tenant resources",
				Execute:     workflow.deployResources,
			},
			{
				Name:        "finalize",
				Description: "Finalize tenant creation",
				Execute:     workflow.finalizeTenant,
			},
		}
	case OperationTypeDelete:
		componentName = "tenant_deletion_workflow"
		steps = []Step{
			{
				Name:        "deactivate",
				Description: "Deactivate tenant",
				Execute:     workflow.deactivateTenant,
			},
			{
				Name:        "remove-resources",
				Description: "Remove tenant resources",
				Execute:     workflow.removeResources,
			},
			{
				Name:        "cleanup-secrets",
				Description: "Clean up tenant secrets",
				Execute:     workflow.cleanupSecrets,
			},
			{
				Name:        "remove-database",
				Description: "Remove tenant database schema",
				Execute:     workflow.removeDatabase,
			},
			{
				Name:        "finalize",
				Description: "Finalize tenant deletion",
				Execute:     workflow.finalizeDeletion,
			},
		}
	default:
		return nil, fmt.Errorf("HOW! invalid operation type: %s", opType)
	}

	workflow.BaseWorkflow = NewBaseWorkflow(steps)
	workflow.logger = logger.With(
		"component", componentName,
		"tenant_id", tenantID,
		"operation_id", op.ID,
		"tenant_name", t.Name,
		"region", t.Region,
		"number_of_steps", len(steps),
	)

	return workflow, nil
}

// Start begins the workflow execution in a goroutine and manages the operation state.
// The workflow result is delivered through the result channel provided by the BaseWorkflow.
//
// Implementation details:
//  1. Asynchronous Execution: Uses goroutines to allow the caller to continue without waiting
//     for potentially long-running operations to complete
//  2. Background Context: Creates a detached context to prevent cancellation when the original
//     request completes
//  3. Consistent Operation State Management: Updates operation status at defined points
//     throughout execution
func (w *TenantOperationWorkflow) Start(ctx context.Context) {
	go func() {
		logger := logger.NewLoggerContext(w.logger.With(
			"operation_type", "start",
			"tenant_id", w.tenantID,
			"operation_id", w.operation.ID,
			"tenant_name", w.tenant.Name,
			"region", string(w.tenant.Region),
		))
		ctx, span := w.tracer.Start(ctx, "TenantOperationWorkflow.Start", trace.WithAttributes(
			attribute.String("operation_type", string(w.operationType)),
			attribute.Int64("tenant_id", w.tenantID),
			attribute.Int64("operation_id", w.operation.ID),
			attribute.String("tenant_name", w.tenant.Name),
			attribute.String("region", string(w.tenant.Region)),
		))
		defer span.End()

		w.operation.Start()
		span.AddEvent("operation started")
		if err := w.operationRepo.Update(ctx, w.operation); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "error updating operation")
			logger.Error(ctx, "error updating operation", "error", err)
			// If operation update fails, report failure but don't block workflow.
			result := WorkflowResult{
				Success: false,
				Error:   err,
				Result:  map[string]any{"tenant_id": w.tenantID},
			}
			w.resultChan <- result
			close(w.resultChan)
			return
		}
		span.AddEvent("operation updated")
		logger.Info(ctx, "operation updated")

		result := w.ExecuteSteps(ctx)
		span.AddEvent("workflow completed")
		logger.Info(ctx, "workflow completed")

		result.Result["tenant_id"] = w.tenantID

		// Update operation based on workflow result.
		if result.Success {
			span.AddEvent("operation completed")
			logger.Info(ctx, "operation completed")
			w.operation.Complete(result.Result)
		} else {
			span.AddEvent("operation failed")
			logger.Error(ctx, "operation failed", "error", result.Error)
			w.operation.Fail(result.Error.Error())
		}

		span.AddEvent("persisting operation")
		logger.Info(ctx, "persisting operation")
		if err := w.operationRepo.Update(ctx, w.operation); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "error persisting updated operation")
			logger.Error(ctx, "error persisting updated operation", "error", err)
		}
		logger.Info(ctx, "operation persisted")
		span.AddEvent("operation persisted")

		w.resultChan <- result
		close(w.resultChan)

		logger.Info(ctx, "workflow result delivered")
		span.AddEvent("workflow result delivered")
		span.SetStatus(codes.Ok, "workflow completed")
	}()
}

// TODO: All this stuff...

// Step implementation methods for creating tenants
func (w *TenantOperationWorkflow) initializeTenant(ctx context.Context) error {
	// This would include generating namespaces, IDs, etc.
	time.Sleep(100 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantOperationWorkflow) provisionDatabase(ctx context.Context) error {
	// This would create the tenant schema in the database
	time.Sleep(500 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantOperationWorkflow) setupSecrets(ctx context.Context) error {
	// This would set up secrets in the secret manager
	time.Sleep(300 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantOperationWorkflow) deployResources(ctx context.Context) error {
	// This would deploy Kubernetes resources
	time.Sleep(1 * time.Second) // Simulate work
	return nil
}

func (w *TenantOperationWorkflow) finalizeTenant(ctx context.Context) error {
	// Update tenant status to active
	w.tenant.Activate()

	// Update tenant in repository
	return w.tenantRepo.Update(ctx, w.tenant)
}

// Step implementation methods for deleting tenants
func (w *TenantOperationWorkflow) deactivateTenant(ctx context.Context) error {
	// Mark tenant for deletion
	w.tenant.MarkForDeletion()

	// Update tenant in repository
	return w.tenantRepo.Update(ctx, w.tenant)
}

func (w *TenantOperationWorkflow) removeResources(ctx context.Context) error {
	// This would remove Kubernetes resources
	time.Sleep(1 * time.Second) // Simulate work
	return nil
}

func (w *TenantOperationWorkflow) cleanupSecrets(ctx context.Context) error {
	// This would clean up secrets from the secret manager
	time.Sleep(300 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantOperationWorkflow) removeDatabase(ctx context.Context) error {
	// This would remove the tenant schema from the database
	time.Sleep(500 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantOperationWorkflow) finalizeDeletion(ctx context.Context) error {
	// Mark tenant as deleted
	w.tenant.Delete()

	// Update tenant in repository
	return w.tenantRepo.Update(ctx, w.tenant)
}
