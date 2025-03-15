package tenant

import "context"

// Repository defines the interface for tenant data access operations.
// This interface abstracts the underlying storage mechanism to allow
// for different implementations (database, in-memory, etc.).
type Repository interface {
	// Create persists a new tenant to the storage system and returns the assigned ID.
	// If a tenant with the same name already exists, an error should be returned.
	Create(ctx context.Context, tenant *Tenant) (int64, error)

	// Update modifies an existing tenant's properties.
	// The tenant is identified by its ID field; all other fields will be updated
	// with the provided values.
	Update(ctx context.Context, tenant *Tenant) error

	// FindByName retrieves a tenant by its unique name.
	// Returns nil and an error if the tenant cannot be found.
	FindByName(ctx context.Context, name string) (*Tenant, error)

	// FindByID retrieves a tenant by its unique identifier.
	// Returns nil and an error if the tenant cannot be found.
	FindByID(ctx context.Context, id int64) (*Tenant, error)

	// Delete permanently removes a tenant from the storage system.
	// This operation cannot be undone, so callers should implement
	// any necessary validation or confirmation before invoking.
	Delete(ctx context.Context, id int64) error
}
