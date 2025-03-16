//go:build integration

package tenant

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/ahrav/hoglet-hub/internal/application/tenant"
	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	tenantDomain "github.com/ahrav/hoglet-hub/internal/domain/tenant"
	operationRepo "github.com/ahrav/hoglet-hub/internal/infra/storage/operation/postgres"
	tenantRepo "github.com/ahrav/hoglet-hub/internal/infra/storage/tenant/postgres"
	"github.com/ahrav/hoglet-hub/internal/infra/storage/testutil"
	integrationTestUtil "github.com/ahrav/hoglet-hub/internal/test/integration/tesutil"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
)

// setupTenantService creates a test environment with database connection
// and tenant service for integration tests.
func setupTenantService(t *testing.T) (
	*tenant.Service,
	tenantDomain.Repository,
	operation.Repository,
	context.Context,
	func(),
) {
	t.Helper()

	pool, cleanup := testutil.SetupTestContainer(t)

	tenantRepo := setupTenantRepository(pool)
	operationRepo := setupOperationRepository(pool)

	log := logger.Noop()
	tracer := noop.NewTracerProvider().Tracer("test-integration")
	service := tenant.NewService(tenantRepo, operationRepo, log, tracer)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	return service, tenantRepo, operationRepo, ctx, cleanup
}

func setupTenantRepository(pool *pgxpool.Pool) tenantDomain.Repository {
	tracer := noop.NewTracerProvider().Tracer("test")
	return tenantRepo.NewTenantStore(pool, tracer)
}

func setupOperationRepository(pool *pgxpool.Pool) operation.Repository {
	tracer := noop.NewTracerProvider().Tracer("test")
	return operationRepo.NewOperationStore(pool, tracer)
}

// TestTenantCreateAndDeleteHappyPath tests the complete flow of creating and
// deleting a tenant successfully.
func TestTenantCreateAndDeleteHappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service, tenantRepo, operationRepo, ctx, cleanup := setupTenantService(t)
	defer cleanup()

	tenantName := fmt.Sprintf("test-tenant-%d", time.Now().UnixNano())
	createParams := tenant.CreateParams{
		Name:   tenantName,
		Region: tenantDomain.RegionEU1,
		Tier:   tenantDomain.TierFree,
	}

	createResult, err := service.Create(ctx, createParams)
	require.NoError(t, err, "Failed to create tenant")
	require.NotNil(t, createResult, "Create result should not be nil")

	tenantID := createResult.TenantID
	createOpID := createResult.OperationID

	t.Logf("Created tenant with ID: %d, operation ID: %d", tenantID, createOpID)

	// Wait for creation operation to complete.
	createOp, err := integrationTestUtil.WaitForOperationStatus(
		ctx,
		t,
		operationRepo,
		createOpID,
		operation.StatusCompleted,
		integrationTestUtil.DefaultOperationTimeout,
	)
	require.NoError(t, err, "Failed waiting for operation to complete")

	AssertOperationSuccess(t, createOp)
	_ = AssertTenantExists(
		t,
		ctx,
		tenantRepo,
		tenantID,
		createParams.Name,
		createParams.Region,
		createParams.Tier,
	)

	deleteResult, err := service.Delete(ctx, tenantID)
	require.NoError(t, err, "Failed to delete tenant")
	require.NotNil(t, deleteResult, "Delete result should not be nil")

	deleteOpID := deleteResult.OperationID
	t.Logf("Deleted tenant with ID: %d, operation ID: %d", tenantID, deleteOpID)

	// Wait for deletion operation to complete.
	deleteOp, err := integrationTestUtil.WaitForOperationStatus(
		ctx,
		t,
		operationRepo,
		deleteOpID,
		operation.StatusCompleted,
		integrationTestUtil.DefaultOperationTimeout,
	)
	require.NoError(t, err, "Failed waiting for delete operation to complete")

	if deleteOp.Status == operation.StatusFailed && deleteOp.ErrorMessage != nil {
		t.Logf("Delete operation failed with error: %s", *deleteOp.ErrorMessage)
	}

	AssertOperationSuccess(t, deleteOp)
	_, err = tenantRepo.FindByID(ctx, tenantID)
	assert.ErrorIs(t, err, tenantDomain.ErrTenantNotFound, "Tenant should not exist after deletion")
}

// TestTenantCreateDuplicateNameFails verifies that creating a tenant with an
// existing name fails with ErrTenantAlreadyExists.
func TestTenantCreateDuplicateNameFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service, _, operationRepo, ctx, cleanup := setupTenantService(t)
	defer cleanup()

	duplicateName := fmt.Sprintf("duplicate-test-tenant-%d", time.Now().UnixNano())
	firstParams := tenant.CreateParams{
		Name:   duplicateName,
		Region: tenantDomain.RegionEU1,
		Tier:   tenantDomain.TierFree,
	}

	firstResult, err := service.Create(ctx, firstParams)
	require.NoError(t, err, "Failed to create first tenant")

	// Wait for first tenant creation to complete.
	_, err = integrationTestUtil.WaitForOperationStatus(
		ctx,
		t,
		operationRepo,
		firstResult.OperationID,
		operation.StatusCompleted,
		integrationTestUtil.DefaultOperationTimeout,
	)
	require.NoError(t, err, "Failed waiting for first tenant creation")

	// Try to create second tenant with same name.
	duplicateParams := tenant.CreateParams{
		Name:   duplicateName,          // Same name
		Region: tenantDomain.RegionUS1, // Different region
		Tier:   tenantDomain.TierPro,   // Different tier
	}

	// Execute - attempt to create duplicate tenant.
	_, err = service.Create(ctx, duplicateParams)
	assert.ErrorIs(t, err, tenantDomain.ErrTenantAlreadyExists, "Expected tenant already exists error")

	deleteResult, err := service.Delete(ctx, firstResult.TenantID)
	require.NoError(t, err, "Failed to delete first tenant")

	deleteOp, err := integrationTestUtil.WaitForOperationStatus(
		ctx,
		t,
		operationRepo,
		deleteResult.OperationID,
		operation.StatusCompleted,
		integrationTestUtil.DefaultOperationTimeout,
	)
	require.NoError(t, err, "Failed waiting for tenant deletion")

	if deleteOp.Status == operation.StatusFailed && deleteOp.ErrorMessage != nil {
		t.Logf("Delete operation failed with error: %s", *deleteOp.ErrorMessage)
	}
}

// TestTenantDeleteNonExistentFails verifies that attempting to delete a
// non-existent tenant fails with ErrTenantNotFound.
func TestTenantDeleteNonExistentFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service, _, _, ctx, cleanup := setupTenantService(t)
	defer cleanup()

	_, err := service.Delete(ctx, 99999) // ID that doesn't exist
	assert.ErrorIs(t, err, tenantDomain.ErrTenantNotFound, "Expected tenant not found error")
}
