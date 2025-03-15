package postgres

import (
	"context"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trufflesecurity/hoglet-hub/internal/db"
	"github.com/trufflesecurity/hoglet-hub/internal/domain/tenant"
	"github.com/trufflesecurity/hoglet-hub/internal/infra/storage/testutil"
)

func setupTenantTest(t *testing.T) (context.Context, *tenantStore, func()) {
	t.Helper()

	pool, cleanup := testutil.SetupTestContainer(t)
	store := &tenantStore{
		q:      db.New(pool),
		pool:   pool,
		tracer: testutil.NoOpTracer(),
	}
	ctx := context.Background()

	return ctx, store, cleanup
}

func TestTenantStore_Create(t *testing.T) {
	t.Parallel()

	ctx, store, cleanup := setupTenantTest(t)
	defer cleanup()

	newTenant, err := tenant.NewTenant("test-tenant", tenant.RegionUS1, tenant.TierFree, nil)
	require.NoError(t, err)

	id, err := store.Create(ctx, newTenant)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	savedTenant, err := store.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, newTenant.Name, savedTenant.Name)
	assert.Equal(t, newTenant.Region, savedTenant.Region)
	assert.Equal(t, newTenant.Tier, savedTenant.Tier)
	assert.Equal(t, tenant.StatusProvisioning, savedTenant.Status)
}

func TestTenantStore_FindByName(t *testing.T) {
	t.Parallel()

	ctx, store, cleanup := setupTenantTest(t)
	defer cleanup()

	newTenant, err := tenant.NewTenant("find-by-name-test", tenant.RegionUS1, tenant.TierFree, nil)
	require.NoError(t, err)

	id, err := store.Create(ctx, newTenant)
	require.NoError(t, err)

	found, err := store.FindByName(ctx, newTenant.Name)
	require.NoError(t, err)
	assert.Equal(t, id, found.ID)
	assert.Equal(t, newTenant.Name, found.Name)
}

func TestTenantStore_FindByName_NotFound(t *testing.T) {
	t.Parallel()

	ctx, store, cleanup := setupTenantTest(t)
	defer cleanup()

	_, err := store.FindByName(ctx, "non-existent-tenant")
	assert.ErrorIs(t, err, tenant.ErrTenantNotFound)
}

func TestTenantStore_Update(t *testing.T) {
	t.Parallel()

	ctx, store, cleanup := setupTenantTest(t)
	defer cleanup()

	newTenant, err := tenant.NewTenant("update-test", tenant.RegionUS1, tenant.TierFree, nil)
	require.NoError(t, err)

	id, err := store.Create(ctx, newTenant)
	require.NoError(t, err)

	found, err := store.FindByID(ctx, id)
	require.NoError(t, err)

	found.Activate()
	err = store.Update(ctx, found)
	require.NoError(t, err)

	updated, err := store.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, tenant.StatusActive, updated.Status)
}

func TestTenantStore_Delete(t *testing.T) {
	t.Parallel()

	ctx, store, cleanup := setupTenantTest(t)
	defer cleanup()

	newTenant, err := tenant.NewTenant("delete-test", tenant.RegionUS1, tenant.TierFree, nil)
	require.NoError(t, err)

	id, err := store.Create(ctx, newTenant)
	require.NoError(t, err)

	err = store.Delete(ctx, id)
	require.NoError(t, err)

	// Verify it's marked for deletion (but not actually deleted)
	// In our schema, we're setting status to 'deleting' rather than physically deleting.
	found, err := store.FindByID(ctx, id)
	require.Error(t, err)
	assert.ErrorIs(t, err, tenant.ErrTenantNotFound)
	assert.Nil(t, found)
}

func TestTenantStore_FindByID_NotFound(t *testing.T) {
	t.Parallel()

	ctx, store, cleanup := setupTenantTest(t)
	defer cleanup()

	_, err := store.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, tenant.ErrTenantNotFound)
}

func TestTenantStore_Create_SameName(t *testing.T) {
	t.Parallel()

	ctx, store, cleanup := setupTenantTest(t)
	defer cleanup()

	tenantName := "duplicate-tenant"
	newTenant, err := tenant.NewTenant(tenantName, tenant.RegionUS1, tenant.TierFree, nil)
	require.NoError(t, err)

	_, err = store.Create(ctx, newTenant)
	require.NoError(t, err)

	duplicateTenant, err := tenant.NewTenant(tenantName, tenant.RegionUS2, tenant.TierPro, nil)
	require.NoError(t, err)

	_, err = store.Create(ctx, duplicateTenant)
	assert.Error(t, err)
}

func TestTenantStore_UpgradeTier(t *testing.T) {
	t.Parallel()

	ctx, store, cleanup := setupTenantTest(t)
	defer cleanup()

	newTenant, err := tenant.NewTenant("upgrade-tenant", tenant.RegionUS1, tenant.TierFree, nil)
	require.NoError(t, err)

	id, err := store.Create(ctx, newTenant)
	require.NoError(t, err)

	found, err := store.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, tenant.TierFree, found.Tier)

	err = found.UpgradeTier(tenant.TierPro)
	require.NoError(t, err)

	err = store.Update(ctx, found)
	require.NoError(t, err)

	updatedTenant, err := store.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, tenant.TierPro, updatedTenant.Tier)
}
