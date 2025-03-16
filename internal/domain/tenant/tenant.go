package tenant

import (
	"errors"
	"regexp"
	"time"
)

// Common errors that can be returned by tenant functions.
var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantAlreadyExists = errors.New("tenant already exists")
	ErrInvalidName         = errors.New("invalid tenant name")
	ErrInvalidRegion       = errors.New("invalid region")
	ErrInvalidTier         = errors.New("invalid tier")
)

// Region represents a deployment region for tenant resources.
type Region string

// Predefined deployment regions available for tenant provisioning.
const (
	RegionEU1 Region = "eu1"
	RegionEU2 Region = "eu2"
	RegionEU3 Region = "eu3"
	RegionEU4 Region = "eu4"
	RegionUS1 Region = "us1"
	RegionUS2 Region = "us2"
	RegionUS3 Region = "us3"
	RegionUS4 Region = "us4"
)

// Tier represents a tenant's subscription level which determines
// available features and resource limits.
type Tier string

// Predefined subscription tiers with different feature sets and pricing.
const (
	TierEnterprise Tier = "enterprise"
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
)

// Status represents the tenant's current lifecycle state.
type Status string

// Predefined tenant lifecycle states.
const (
	StatusProvisioning Status = "provisioning" // Initial state during setup
	StatusActive       Status = "active"       // Normal operating state
	StatusSuspended    Status = "suspended"    // Temporarily disabled
	StatusDeleting     Status = "deleting"     // Being removed from the system
	StatusDeleted      Status = "deleted"      // Logically deleted
)

// Tenant represents a customer tenant in the system with all its configuration
// and state information.
type Tenant struct {
	ID               int64      // Unique identifier
	Name             string     // Unique tenant name (used in URLs)
	Region           Region     // Deployment region
	Tier             Tier       // Subscription tier
	Status           Status     // Current lifecycle state
	IsolationGroupID *int64     // Optional group for resource isolation
	CreatedAt        time.Time  // Creation timestamp
	UpdatedAt        *time.Time // Last update timestamp
	DeletedAt        *time.Time // Deletion timestamp (if deleted)
}

// NewTenant creates a new tenant with validation of all fields.
// Returns a pointer to the new tenant or an error if validation fails.
func NewTenant(name string, region Region, tier Tier, isolationGroupID *int64) (*Tenant, error) {
	// Validate name (lowercase letters, numbers, hyphens)
	if !isValidName(name) {
		return nil, ErrInvalidName
	}

	if !isValidRegion(region) {
		return nil, ErrInvalidRegion
	}

	if !isValidTier(tier) {
		return nil, ErrInvalidTier
	}

	now := time.Now()
	return &Tenant{
		Name:             name,
		Region:           region,
		Tier:             tier,
		Status:           StatusProvisioning,
		IsolationGroupID: isolationGroupID,
		CreatedAt:        now,
	}, nil
}

var validNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// isValidName validates the tenant name format.
// Names must contain only lowercase letters, numbers, and hyphens.
func isValidName(name string) bool {
	return validNamePattern.MatchString(name)
}

// isValidRegion checks if the region is one of the predefined valid regions.
func isValidRegion(region Region) bool {
	switch region {
	case RegionEU1, RegionEU2, RegionEU3, RegionEU4,
		RegionUS1, RegionUS2, RegionUS3, RegionUS4:
		return true
	default:
		return false
	}
}

// isValidTier checks if the tier is one of the predefined valid tiers.
func isValidTier(tier Tier) bool {
	switch tier {
	case TierEnterprise, TierFree, TierPro:
		return true
	default:
		return false
	}
}

// Activate marks the tenant as active, indicating it's ready for use.
// Updates the tenant's status and timestamp.
func (t *Tenant) Activate() {
	t.Status = StatusActive
	now := time.Now()
	t.UpdatedAt = &now
}

// Suspend marks the tenant as suspended, temporarily disabling access.
// This is typically used for billing issues or policy violations.
func (t *Tenant) Suspend() {
	t.Status = StatusSuspended
	now := time.Now()
	t.UpdatedAt = &now
}

// MarkForDeletion changes the tenant status to deleting, indicating
// that deletion is in progress but not yet complete.
func (t *Tenant) MarkForDeletion() {
	t.Status = StatusDeleting
	now := time.Now()
	t.UpdatedAt = &now
}

// Delete marks the tenant as logically deleted in the system.
// This sets both UpdatedAt and DeletedAt timestamps.
func (t *Tenant) Delete() {
	t.Status = StatusDeleted
	now := time.Now()
	t.UpdatedAt = &now
	t.DeletedAt = &now
}

// UpgradeTier changes the tenant to a new subscription tier after validation.
// Returns an error if the new tier is invalid.
func (t *Tenant) UpgradeTier(newTier Tier) error {
	if !isValidTier(newTier) {
		return ErrInvalidTier
	}

	t.Tier = newTier
	now := time.Now()
	t.UpdatedAt = &now
	return nil
}

// ChangeRegion moves the tenant to a different deployment region after validation.
// Returns an error if the new region is invalid.
func (t *Tenant) ChangeRegion(newRegion Region) error {
	if !isValidRegion(newRegion) {
		return ErrInvalidRegion
	}

	t.Region = newRegion
	now := time.Now()
	t.UpdatedAt = &now
	return nil
}

// IsActive checks if the tenant is in the active state and available for use.
func (t *Tenant) IsActive() bool {
	return t.Status == StatusActive
}

// IsDeleted checks if the tenant has been logically deleted.
func (t *Tenant) IsDeleted() bool {
	return t.Status == StatusDeleted
}
