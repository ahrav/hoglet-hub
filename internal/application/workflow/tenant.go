package workflow

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
)

// TODO: Consider if there is a way refactor the specific workflows into a more generic
// workflow that can be used for multiple tenant operations. (ex: Create, Delete, Update, etc.)
// We can probably have multiple constructors that can be used to create the specific workflows.
// This would allow us to avoid duplicating code and make the workflows more maintainable.

// TenantCreationWorkflow orchestrates the multi-step process of tenant provisioning.
// It handles all aspects from initialization through resource deployment and tracks
// the operation state for observability.
type TenantCreationWorkflow struct {
	*BaseWorkflow
	tenant        *tenant.Tenant
	tenantID      int64
	operation     *operation.Operation
	tenantRepo    tenant.Repository
	operationRepo operation.Repository

	logger *logger.Logger
	tracer trace.Tracer
}

// NewTenantCreationWorkflow creates a new workflow for tenant provisioning.
// It sets up the necessary steps and manages the operation for tracking the provisioning process.
func NewTenantCreationWorkflow(
	t *tenant.Tenant,
	tenantID int64,
	op *operation.Operation,
	tenantRepo tenant.Repository,
	operationRepo operation.Repository,
	logger *logger.Logger,
	tracer trace.Tracer,
) *TenantCreationWorkflow {
	workflow := &TenantCreationWorkflow{
		tenant:        t,
		tenantID:      tenantID,
		operation:     op,
		tenantRepo:    tenantRepo,
		operationRepo: operationRepo,
		tracer:        tracer,
	}

	// Define the workflow steps.
	steps := []Step{
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
	workflow.BaseWorkflow = NewBaseWorkflow(steps)
	workflow.logger = logger.With(
		"component", "tenant_creation_workflow",
		"tenant_id", tenantID,
		"operation_id", op.ID,
		"tenant_name", t.Name,
		"region", t.Region,
		"number_of_steps", len(steps),
	)

	return workflow
}

// Start begins the workflow execution in a goroutine and manages the operation state.
// The workflow result is delivered through the result channel provided by the BaseWorkflow.
func (w *TenantCreationWorkflow) Start(ctx context.Context) {
	go func() {
		logger := logger.NewLoggerContext(w.logger.With(
			"operation_type", "start",
			"tenant_id", w.tenantID,
			"operation_id", w.operation.ID,
			"tenant_name", w.tenant.Name,
			"region", string(w.tenant.Region),
		))
		ctx, span := w.tracer.Start(ctx, "TenantCreationWorkflow.Start", trace.WithAttributes(
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

// Step implementation methods.

func (w *TenantCreationWorkflow) initializeTenant(ctx context.Context) error {
	// This would include generating namespaces, IDs, etc.
	time.Sleep(100 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantCreationWorkflow) provisionDatabase(ctx context.Context) error {
	// This would create the tenant schema in the database
	time.Sleep(500 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantCreationWorkflow) setupSecrets(ctx context.Context) error {
	// This would set up secrets in the secret manager
	time.Sleep(300 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantCreationWorkflow) deployResources(ctx context.Context) error {
	// This would deploy Kubernetes resources
	time.Sleep(1 * time.Second) // Simulate work
	return nil
}

func (w *TenantCreationWorkflow) finalizeTenant(ctx context.Context) error {
	// Update tenant status to active
	w.tenant.Activate()

	// Update tenant in repository
	return w.tenantRepo.Update(ctx, w.tenant)
}

// TenantDeletionWorkflow orchestrates the multi-step process of tenant removal.
// It ensures that all tenant resources are properly decommissioned and tracks
// the operation state for observability.
type TenantDeletionWorkflow struct {
	*BaseWorkflow
	tenant        *tenant.Tenant
	tenantID      int64
	operation     *operation.Operation
	tenantRepo    tenant.Repository
	operationRepo operation.Repository

	logger *logger.Logger
	tracer trace.Tracer
}

// NewTenantDeletionWorkflow creates a new workflow for tenant removal.
// It sets up the necessary steps and manages the operation for tracking the deletion process.
func NewTenantDeletionWorkflow(
	t *tenant.Tenant,
	tenantID int64,
	op *operation.Operation,
	tenantRepo tenant.Repository,
	operationRepo operation.Repository,
	logger *logger.Logger,
	tracer trace.Tracer,
) *TenantDeletionWorkflow {
	workflow := &TenantDeletionWorkflow{
		tenant:        t,
		tenantID:      tenantID,
		operation:     op,
		tenantRepo:    tenantRepo,
		operationRepo: operationRepo,
		tracer:        tracer,
	}

	// Define the workflow steps
	steps := []Step{
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

	workflow.BaseWorkflow = NewBaseWorkflow(steps)
	workflow.logger = logger.With(
		"component", "tenant_deletion_workflow",
		"tenant_id", tenantID,
		"operation_id", op.ID,
		"tenant_name", t.Name,
		"region", t.Region,
		"number_of_steps", len(steps),
	)

	return workflow
}

// Start begins the workflow execution in a goroutine and manages the operation state.
// The workflow result is delivered through the result channel provided by the BaseWorkflow.
func (w *TenantDeletionWorkflow) Start(ctx context.Context) {
	go func() {
		logger := logger.NewLoggerContext(w.logger.With(
			"operation_type", "start",
			"tenant_id", w.tenantID,
			"operation_id", w.operation.ID,
			"tenant_name", w.tenant.Name,
			"region", string(w.tenant.Region),
		))
		ctx, span := w.tracer.Start(ctx, "TenantDeletionWorkflow.Start", trace.WithAttributes(
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
			// If operation update fails, report failure but don't block workflow
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

		// Update operation based on workflow result
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

// Step implementation methods.

func (w *TenantDeletionWorkflow) deactivateTenant(ctx context.Context) error {
	// Mark tenant for deletion
	w.tenant.MarkForDeletion()

	// Update tenant in repository
	return w.tenantRepo.Update(ctx, w.tenant)
}

func (w *TenantDeletionWorkflow) removeResources(ctx context.Context) error {
	// This would remove Kubernetes resources
	time.Sleep(1 * time.Second) // Simulate work
	return nil
}

func (w *TenantDeletionWorkflow) cleanupSecrets(ctx context.Context) error {
	// This would clean up secrets from the secret manager
	time.Sleep(300 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantDeletionWorkflow) removeDatabase(ctx context.Context) error {
	// This would remove the tenant schema from the database
	time.Sleep(500 * time.Millisecond) // Simulate work
	return nil
}

func (w *TenantDeletionWorkflow) finalizeDeletion(ctx context.Context) error {
	// Mark tenant as deleted
	w.tenant.Delete()

	// Update tenant in repository
	return w.tenantRepo.Update(ctx, w.tenant)
}
