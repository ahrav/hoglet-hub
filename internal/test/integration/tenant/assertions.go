package tenant

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
)

// AssertTenantExists verifies that a tenant with the given ID exists and matches expected values
func AssertTenantExists(
	t *testing.T,
	ctx context.Context,
	repo tenant.Repository,
	tenantID int64,
	expectedName string,
	expectedRegion tenant.Region,
	expectedTier tenant.Tier,
) *tenant.Tenant {
	t.Helper()

	ten, err := repo.FindByID(ctx, tenantID)
	require.NoError(t, err, "Failed to find tenant by ID")
	require.NotNil(t, ten, "Tenant should exist")

	assert.Equal(t, expectedName, ten.Name, "Tenant name should match")
	assert.Equal(t, expectedRegion, ten.Region, "Tenant region should match")
	assert.Equal(t, expectedTier, ten.Tier, "Tenant tier should match")

	return ten
}

// AssertTenantDoesNotExist verifies that a tenant with the given ID doesn't exist.
func AssertTenantDoesNotExist(
	t *testing.T,
	ctx context.Context,
	repo tenant.Repository,
	tenantID int64,
) {
	t.Helper()

	ten, err := repo.FindByID(ctx, tenantID)
	assert.ErrorIs(t, err, tenant.ErrTenantNotFound, "Should get tenant not found error")
	assert.Nil(t, ten, "Tenant should not exist")
}

// AssertOperationSuccess verifies that an operation completed successfully.
func AssertOperationSuccess(
	t *testing.T,
	op *operation.Operation,
) {
	t.Helper()

	assert.Equal(t, operation.StatusCompleted, op.Status, "Operation should be completed")
	assert.Nil(t, op.ErrorMessage, "Operation should not have error message")
	assert.NotNil(t, op.CompletedAt, "Operation should have completion timestamp")
}

// AssertOperationFailed verifies that an operation failed with expected error.
func AssertOperationFailed(
	t *testing.T,
	op *operation.Operation,
	expectedErrorSubstring string,
) {
	t.Helper()

	assert.Equal(t, operation.StatusFailed, op.Status, "Operation should be failed")
	assert.NotNil(t, op.ErrorMessage, "Operation should have error message")
	if expectedErrorSubstring != "" && op.ErrorMessage != nil {
		assert.Contains(t, *op.ErrorMessage, expectedErrorSubstring, "Error message should contain expected substring")
	}
	assert.NotNil(t, op.CompletedAt, "Operation should have completion timestamp")
}
