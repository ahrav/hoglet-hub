package operation

import "context"

// Repository defines the interface for operation data access operations.
// This interface abstracts the underlying storage mechanism to allow
// for different implementations (database, in-memory, etc.).
type Repository interface {
	// Create persists a new operation and returns its ID.
	// The operation is assigned a new unique identifier by the storage system.
	Create(ctx context.Context, op *Operation) (int64, error)

	// Update modifies an existing operation with the provided data.
	// The operation must already exist in the system or an error will be returned.
	Update(ctx context.Context, op *Operation) error

	// FindByID retrieves an operation by its unique identifier.
	// Returns nil if no operation is found with the given ID.
	FindByID(ctx context.Context, id int64) (*Operation, error)

	// FindByTenantID retrieves all operations associated with a specific tenant.
	// This enables multi-tenancy support by separating operations between different
	// organizational units.
	FindByTenantID(ctx context.Context, tenantID int64) ([]*Operation, error)

	// FindByStatus retrieves all operations with a specific status.
	// This allows filtering operations based on their current state in the workflow.
	FindByStatus(ctx context.Context, status Status) ([]*Operation, error)

	// FindIncomplete retrieves all operations that haven't reached a terminal status.
	// This is particularly useful for finding operations that might need attention
	// or are still in progress.
	FindIncomplete(ctx context.Context) ([]*Operation, error)
}
