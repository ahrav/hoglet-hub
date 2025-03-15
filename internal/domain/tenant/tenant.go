package tenant

import (
	"errors"
	"regexp"
	"time"
)

// Common errors
var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantAlreadyExists = errors.New("tenant already exists")
	ErrInvalidName         = errors.New("invalid tenant name")
	ErrInvalidRegion       = errors.New("invalid region")
	ErrInvalidTier         = errors.New("invalid tier")
)

// Region represents a deployment region
type Region string

// Predefined deployment regions
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

// Tier represents a tenant's subscription tier
type Tier string

// Predefined tier levels
const (
	TierEnterprise Tier = "enterprise"
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
)

// Status represents the tenant's current status
type Status string

// Predefined tenant statuses
const (
	StatusProvisioning Status = "provisioning"
	StatusActive       Status = "active"
	StatusSuspended    Status = "suspended"
	StatusDeleting     Status = "deleting"
	StatusDeleted      Status = "deleted"
)

// Tenant represents a customer tenant in the system
type Tenant struct {
	ID               int64
	Name             string
	Region           Region
	Tier             Tier
	Status           Status
	IsolationGroupID *int64
	CreatedAt        time.Time
	UpdatedAt        *time.Time
	DeletedAt        *time.Time
}

// NewTenant creates a new tenant with validation
func NewTenant(name string, region Region, tier Tier, isolationGroupID *int64) (*Tenant, error) {
	// Validate name (lowercase letters, numbers, hyphens)
	if !isValidName(name) {
		return nil, ErrInvalidName
	}

	// Validate region
	if !isValidRegion(region) {
		return nil, ErrInvalidRegion
	}

	// Validate tier
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

// isValidName validates the tenant name format
func isValidName(name string) bool {
	pattern := regexp.MustCompile(`^[a-z0-9-]+$`)
	return pattern.MatchString(name)
}

// isValidRegion checks if the region is valid
func isValidRegion(region Region) bool {
	switch region {
	case RegionEU1, RegionEU2, RegionEU3, RegionEU4,
		RegionUS1, RegionUS2, RegionUS3, RegionUS4:
		return true
	default:
		return false
	}
}

// isValidTier checks if the tier is valid
func isValidTier(tier Tier) bool {
	switch tier {
	case TierEnterprise, TierFree, TierPro:
		return true
	default:
		return false
	}
}

// Activate marks the tenant as active
func (t *Tenant) Activate() {
	t.Status = StatusActive
	now := time.Now()
	t.UpdatedAt = &now
}

// Suspend marks the tenant as suspended
func (t *Tenant) Suspend() {
	t.Status = StatusSuspended
	now := time.Now()
	t.UpdatedAt = &now
}

// MarkForDeletion changes the tenant status to deleting
func (t *Tenant) MarkForDeletion() {
	t.Status = StatusDeleting
	now := time.Now()
	t.UpdatedAt = &now
}

// Delete marks the tenant as deleted
func (t *Tenant) Delete() {
	t.Status = StatusDeleted
	now := time.Now()
	t.UpdatedAt = &now
	t.DeletedAt = &now
}

// UpgradeTier upgrades the tenant to a new tier
func (t *Tenant) UpgradeTier(newTier Tier) error {
	if !isValidTier(newTier) {
		return ErrInvalidTier
	}

	t.Tier = newTier
	now := time.Now()
	t.UpdatedAt = &now
	return nil
}

// ChangeRegion moves the tenant to a different region
func (t *Tenant) ChangeRegion(newRegion Region) error {
	if !isValidRegion(newRegion) {
		return ErrInvalidRegion
	}

	t.Region = newRegion
	now := time.Now()
	t.UpdatedAt = &now
	return nil
}

// IsActive checks if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == StatusActive
}

// IsDeleted checks if the tenant is deleted
func (t *Tenant) IsDeleted() bool {
	return t.Status == StatusDeleted
}
