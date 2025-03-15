package workflow

import (
	"context"
	"time"

	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
)

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
}

// NewTenantCreationWorkflow creates a new workflow for tenant provisioning.
// It sets up the necessary steps and manages the operation for tracking the provisioning process.
func NewTenantCreationWorkflow(
	t *tenant.Tenant,
	tenantID int64,
	op *operation.Operation,
	tenantRepo tenant.Repository,
	operationRepo operation.Repository,
) *TenantCreationWorkflow {
	workflow := &TenantCreationWorkflow{
		tenant:        t,
		tenantID:      tenantID,
		operation:     op,
		tenantRepo:    tenantRepo,
		operationRepo: operationRepo,
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

	return workflow
}

// Start begins the workflow execution in a goroutine and manages the operation state.
// The workflow result is delivered through the result channel provided by the BaseWorkflow.
func (w *TenantCreationWorkflow) Start(ctx context.Context) {
	go func() {
		// Update operation status to in progress
		w.operation.Start()
		if err := w.operationRepo.Update(ctx, w.operation); err != nil {
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

		// Execute all steps
		result := w.ExecuteSteps(ctx)

		// Add tenant ID to result for downstream consumers
		result.Result["tenant_id"] = w.tenantID

		// Update operation based on workflow result
		if result.Success {
			w.operation.Complete(result.Result)
		} else {
			w.operation.Fail(result.Error.Error())
		}

		// Update operation in repository
		if err := w.operationRepo.Update(ctx, w.operation); err != nil {
			// Log error but don't fail the workflow
		}

		w.resultChan <- result
		close(w.resultChan)
	}()
}

// Step implementation methods

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
}

// NewTenantDeletionWorkflow creates a new workflow for tenant removal.
// It sets up the necessary steps and manages the operation for tracking the deletion process.
func NewTenantDeletionWorkflow(
	t *tenant.Tenant,
	tenantID int64,
	op *operation.Operation,
	tenantRepo tenant.Repository,
	operationRepo operation.Repository,
) *TenantDeletionWorkflow {
	workflow := &TenantDeletionWorkflow{
		tenant:        t,
		tenantID:      tenantID,
		operation:     op,
		tenantRepo:    tenantRepo,
		operationRepo: operationRepo,
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

	return workflow
}

// Start begins the workflow execution in a goroutine and manages the operation state.
// The workflow result is delivered through the result channel provided by the BaseWorkflow.
func (w *TenantDeletionWorkflow) Start(ctx context.Context) {
	go func() {
		// Update operation status to in progress
		w.operation.Start()
		if err := w.operationRepo.Update(ctx, w.operation); err != nil {
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

		// Execute all steps
		result := w.ExecuteSteps(ctx)

		// Add tenant ID to result for downstream consumers
		result.Result["tenant_id"] = w.tenantID

		// Update operation based on workflow result
		if result.Success {
			w.operation.Complete(result.Result)
		} else {
			w.operation.Fail(result.Error.Error())
		}

		// Update operation in repository
		if err := w.operationRepo.Update(ctx, w.operation); err != nil {
			// Log error but don't fail the workflow
		}

		w.resultChan <- result
		close(w.resultChan)
	}()
}

// Step implementation methods

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
