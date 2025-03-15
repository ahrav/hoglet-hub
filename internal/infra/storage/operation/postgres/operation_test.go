package postgres

import (
	"context"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/ahrav/hoglet-hub/internal/db"
	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
	tenantRepo "github.com/ahrav/hoglet-hub/internal/infra/storage/tenant/postgres"
	"github.com/ahrav/hoglet-hub/internal/infra/storage/testutil"
)

func setupOperationTest(t *testing.T) (context.Context, *operationStore, tenant.Repository, func()) {
	t.Helper()

	pool, cleanup := testutil.SetupTestContainer(t)
	tracer := noop.NewTracerProvider().Tracer("test")
	opStore := &operationStore{
		q:      db.New(pool),
		pool:   pool,
		tracer: tracer,
	}
	tenantStore := tenantRepo.NewTenantStore(pool, tracer)
	ctx := context.Background()

	return ctx, opStore, tenantStore, cleanup
}

func createTestTenant(t *testing.T, ctx context.Context, store tenant.Repository) int64 {
	t.Helper()

	newTenant, err := tenant.NewTenant("test-tenant", tenant.RegionUS1, tenant.TierFree, nil)
	require.NoError(t, err)

	id, err := store.Create(ctx, newTenant)
	require.NoError(t, err)
	return id
}

func TestOperationStore_Create(t *testing.T) {
	t.Parallel()

	ctx, opStore, tenantStore, cleanup := setupOperationTest(t)
	defer cleanup()

	tenantID := createTestTenant(t, ctx, tenantStore)

	op, err := operation.NewTenantCreateOperation(tenantID, "test-tenant", "us1", "free", nil)
	require.NoError(t, err)

	id, err := opStore.Create(ctx, op)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	savedOp, err := opStore.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, operation.OpTenantCreate, savedOp.Type)
	assert.Equal(t, operation.StatusPending, savedOp.Status)
	assert.Equal(t, &tenantID, savedOp.TenantID)
}

func TestOperationStore_Update(t *testing.T) {
	t.Parallel()

	ctx, opStore, tenantStore, cleanup := setupOperationTest(t)
	defer cleanup()

	tenantID := createTestTenant(t, ctx, tenantStore)

	op, err := operation.NewTenantCreateOperation(tenantID, "test-tenant", "us1", "free", nil)
	require.NoError(t, err)

	id, err := opStore.Create(ctx, op)
	require.NoError(t, err)

	savedOp, err := opStore.FindByID(ctx, id)
	require.NoError(t, err)

	// Update the operation.
	savedOp.Start()
	err = opStore.Update(ctx, savedOp)
	require.NoError(t, err)

	updatedOp, err := opStore.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, operation.StatusInProgress, updatedOp.Status)
	assert.NotNil(t, updatedOp.StartedAt)
}

func TestOperationStore_FindByTenantID(t *testing.T) {
	t.Parallel()

	ctx, opStore, tenantStore, cleanup := setupOperationTest(t)
	defer cleanup()

	tenantID := createTestTenant(t, ctx, tenantStore)

	op1, err := operation.NewTenantCreateOperation(tenantID, "test-tenant", "us1", "free", nil)
	require.NoError(t, err)

	id1, err := opStore.Create(ctx, op1)
	require.NoError(t, err)

	op2, err := operation.NewTenantDeleteOperation(tenantID)
	require.NoError(t, err)

	id2, err := opStore.Create(ctx, op2)
	require.NoError(t, err)

	ops, err := opStore.FindByTenantID(ctx, tenantID)
	require.NoError(t, err)
	assert.Len(t, ops, 2)

	// The most recent operation should be first (tenantDelete).
	assert.Equal(t, id2, ops[0].ID)
	assert.Equal(t, operation.OpTenantDelete, ops[0].Type)

	assert.Equal(t, id1, ops[1].ID)
	assert.Equal(t, operation.OpTenantCreate, ops[1].Type)
}

func TestOperationStore_FindByStatus(t *testing.T) {
	t.Parallel()

	ctx, opStore, tenantStore, cleanup := setupOperationTest(t)
	defer cleanup()

	tenantID := createTestTenant(t, ctx, tenantStore)

	op, err := operation.NewTenantCreateOperation(tenantID, "test-tenant", "us1", "free", nil)
	require.NoError(t, err)

	id, err := opStore.Create(ctx, op)
	require.NoError(t, err)

	pendingOps, err := opStore.FindByStatus(ctx, operation.StatusPending)
	require.NoError(t, err)
	assert.Greater(t, len(pendingOps), 0)
	foundPending := false
	for _, op := range pendingOps {
		if op.ID == id {
			foundPending = true
			break
		}
	}
	assert.True(t, foundPending, "Should find the pending operation")

	// Update operation to completed
	savedOp, err := opStore.FindByID(ctx, id)
	require.NoError(t, err)

	savedOp.Complete(map[string]any{"result": "success"})
	err = opStore.Update(ctx, savedOp)
	require.NoError(t, err)

	// Find completed operations
	completedOps, err := opStore.FindByStatus(ctx, operation.StatusCompleted)
	require.NoError(t, err)
	assert.Greater(t, len(completedOps), 0)
	foundCompleted := false
	for _, op := range completedOps {
		if op.ID == id {
			foundCompleted = true
			break
		}
	}
	assert.True(t, foundCompleted, "Should find the completed operation")
}

func TestOperationStore_FindIncomplete(t *testing.T) {
	t.Parallel()

	ctx, opStore, tenantStore, cleanup := setupOperationTest(t)
	defer cleanup()

	// Create a tenant
	tenantID := createTestTenant(t, ctx, tenantStore)

	// Create operations in different states
	pendingOp, err := operation.NewTenantCreateOperation(tenantID, "test-tenant", "us1", "free", nil)
	require.NoError(t, err)

	pendingID, err := opStore.Create(ctx, pendingOp)
	require.NoError(t, err)

	inProgressOp, err := operation.NewTenantDeleteOperation(tenantID)
	require.NoError(t, err)
	inProgressOp.Start()

	inProgressID, err := opStore.Create(ctx, inProgressOp)
	require.NoError(t, err)

	completedOp, err := operation.NewTenantCreateOperation(tenantID, "completed-tenant", "us1", "free", nil)
	require.NoError(t, err)
	completedOp.Complete(map[string]any{"result": "success"})

	completedID, err := opStore.Create(ctx, completedOp)
	require.NoError(t, err)

	// Find incomplete operations
	incompleteOps, err := opStore.FindIncomplete(ctx)
	require.NoError(t, err)

	foundPending := false
	foundInProgress := false
	foundCompleted := false

	for _, op := range incompleteOps {
		if op.ID == pendingID {
			foundPending = true
		} else if op.ID == inProgressID {
			foundInProgress = true
		} else if op.ID == completedID {
			foundCompleted = true
		}
	}

	assert.True(t, foundPending, "Should find the pending operation")
	assert.True(t, foundInProgress, "Should find the in-progress operation")
	assert.False(t, foundCompleted, "Should not find the completed operation")
}

func TestOperationStore_FindByID_NotFound(t *testing.T) {
	t.Parallel()

	ctx, opStore, _, cleanup := setupOperationTest(t)
	defer cleanup()

	_, err := opStore.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, operation.ErrOperationNotFound)
}

func TestOperationStore_CompleteOperation(t *testing.T) {
	t.Parallel()

	ctx, opStore, tenantStore, cleanup := setupOperationTest(t)
	defer cleanup()

	tenantID := createTestTenant(t, ctx, tenantStore)

	op, err := operation.NewTenantCreateOperation(tenantID, "test-tenant", "us1", "free", nil)
	require.NoError(t, err)

	id, err := opStore.Create(ctx, op)
	require.NoError(t, err)

	savedOp, err := opStore.FindByID(ctx, id)
	require.NoError(t, err)

	savedOp.Start()
	err = opStore.Update(ctx, savedOp)
	require.NoError(t, err)

	result := map[string]any{
		"tenant_id": tenantID,
		"success":   true,
		"resources": []string{"db", "namespace", "services"},
	}

	savedOp.Complete(result)
	err = opStore.Update(ctx, savedOp)
	require.NoError(t, err)

	completedOp, err := opStore.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, operation.StatusCompleted, completedOp.Status)
	assert.NotNil(t, completedOp.CompletedAt)
	assert.Equal(t, true, completedOp.Result["success"])
	assert.Equal(t, float64(tenantID), completedOp.Result["tenant_id"])
	assert.Len(t, completedOp.Result["resources"].([]any), 3)
}

func TestOperationStore_FailOperation(t *testing.T) {
	t.Parallel()

	ctx, opStore, tenantStore, cleanup := setupOperationTest(t)
	defer cleanup()

	tenantID := createTestTenant(t, ctx, tenantStore)

	op, err := operation.NewTenantCreateOperation(tenantID, "test-tenant", "us1", "free", nil)
	require.NoError(t, err)

	id, err := opStore.Create(ctx, op)
	require.NoError(t, err)

	savedOp, err := opStore.FindByID(ctx, id)
	require.NoError(t, err)

	savedOp.Start()
	err = opStore.Update(ctx, savedOp)
	require.NoError(t, err)

	errorMsg := "Failed to provision database: timeout exceeded"
	savedOp.Fail(errorMsg)
	err = opStore.Update(ctx, savedOp)
	require.NoError(t, err)

	failedOp, err := opStore.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, operation.StatusFailed, failedOp.Status)
	assert.NotNil(t, failedOp.CompletedAt)
	assert.Equal(t, errorMsg, *failedOp.ErrorMessage)
}
