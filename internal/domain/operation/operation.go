package operation

import (
	"errors"
	"fmt"
	"time"
)

// Common errors that can be returned by operation functions.
var (
	ErrOperationNotFound  = errors.New("operation not found")
	ErrOperationFailed    = errors.New("operation failed")
	ErrOperationCancelled = errors.New("operation cancelled")
	ErrOperationCompleted = errors.New("operation completed")
)

// Op represents the operation type in the system.
// This type is used to categorize different operations that can be performed.
type Op string

// Predefined operation types supported by the system.
const (
	OpTenantCreate Op = "tenant.create"
	OpTenantDelete Op = "tenant.delete"
	// OpTenantUpdate  Op = "tenant.update"
	// OpTenantUpgrade Op = "tenant.upgrade"
	// OpTenantMigrate Op = "tenant.migrate"
)

// IsValid checks if the operation type is valid by comparing it against
// the predefined set of supported operations.
func (t Op) IsValid() bool {
	switch t {
	case OpTenantCreate, OpTenantDelete:
		return true
	default:
		return false
	}
}

// String returns the string representation of the operation type.
func (t Op) String() string {
	return string(t)
}

// ParseType converts a string to an operation type with validation.
// Returns an error if the string does not represent a valid operation type.
func ParseType(s string) (Op, error) {
	t := Op(s)
	if !t.IsValid() {
		return "", fmt.Errorf("invalid operation type: %s", s)
	}
	return t, nil
}

// ValidationError represents a domain validation error.
// It provides context about which field failed validation and why.
type ValidationError struct {
	Field   string
	Message string
}

// Error returns the error message for a ValidationError,
// implementing the error interface.
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError with the given field and message.
func NewValidationError(field, message string) ValidationError {
	return ValidationError{
		Field:   field,
		Message: message,
	}
}

// Status represents the current state of an operation.
type Status string

// Predefined operation statuses that represent the lifecycle of an operation.
const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// Operation represents an asynchronous operation in the system.
// It tracks the state, timing, and results of operations such as
// tenant creation and deletion.
type Operation struct {
	ID           int64
	Type         Op
	Status       Status
	TenantID     *int64
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
	UpdatedAt    *time.Time
	CreatedBy    *string
	ErrorMessage *string
	Parameters   map[string]any
	Result       map[string]any
}

// NewTenantCreateOperation creates a new tenant creation operation.
// This is a convenience function that sets up the appropriate parameters
// for creating a tenant.
func NewTenantCreateOperation(tenantID int64, name string, region, tier string, isolationGroupID *int64) (*Operation, error) {
	params := map[string]interface{}{
		"name":   name,
		"region": region,
		"tier":   tier,
	}

	if isolationGroupID != nil {
		params["isolation_group_id"] = *isolationGroupID
	}

	return NewOperation(OpTenantCreate, &tenantID, params)
}

// NewTenantDeleteOperation creates a new tenant deletion operation.
// It sets up the necessary parameters for deleting a tenant.
func NewTenantDeleteOperation(tenantID int64) (*Operation, error) {
	params := map[string]interface{}{
		"tenant_id": tenantID,
	}

	return NewOperation(OpTenantDelete, &tenantID, params)
}

// NewOperation creates a new operation with the given type, tenant ID, and parameters.
// It initializes the operation in the pending state with the current timestamp.
func NewOperation(opType Op, tenantID *int64, params map[string]any) (*Operation, error) {
	if !opType.IsValid() {
		return nil, NewValidationError("type", "invalid operation type")
	}

	now := time.Now()
	return &Operation{
		Type:       opType,
		TenantID:   tenantID,
		Status:     StatusPending,
		Parameters: params,
		CreatedAt:  now,
	}, nil
}

// isValidType checks if the operation type is valid.
// This is an internal helper function.
func isValidType(opType Op) bool {
	return opType.IsValid()
}

// Start marks the operation as in progress and sets the start time.
// This should be called when the system begins executing the operation.
func (o *Operation) Start() {
	o.Status = StatusInProgress
	now := time.Now()
	o.StartedAt = &now
	o.UpdatedAt = &now
}

// Complete marks the operation as completed and stores the result.
// This should be called when the operation has successfully finished.
func (o *Operation) Complete(result map[string]any) {
	o.Status = StatusCompleted
	o.Result = result
	now := time.Now()
	o.CompletedAt = &now
	o.UpdatedAt = &now
}

// Fail marks the operation as failed with the provided error message.
// This should be called when the operation encounters an error.
func (o *Operation) Fail(errMsg string) {
	o.Status = StatusFailed
	o.ErrorMessage = &errMsg
	now := time.Now()
	o.CompletedAt = &now
	o.UpdatedAt = &now
}

// Cancel marks the operation as cancelled with the provided reason.
// This should be called when the operation is manually or automatically cancelled.
func (o *Operation) Cancel(reason string) {
	o.Status = StatusCancelled
	o.ErrorMessage = &reason
	now := time.Now()
	o.CompletedAt = &now
	o.UpdatedAt = &now
}

// IsTerminal checks if the operation is in a terminal state (completed, failed, or cancelled).
// Terminal operations cannot transition to other states.
func (o *Operation) IsTerminal() bool {
	return o.Status == StatusCompleted || o.Status == StatusFailed || o.Status == StatusCancelled
}

// IsInProgress checks if the operation is currently being executed.
func (o *Operation) IsInProgress() bool {
	return o.Status == StatusInProgress
}

// IsPending checks if the operation is waiting to be started.
func (o *Operation) IsPending() bool {
	return o.Status == StatusPending
}

// Duration returns the duration of the operation if it has both started and completed.
// Returns nil if the operation hasn't started or completed.
func (o *Operation) Duration() *time.Duration {
	if o.StartedAt == nil || o.CompletedAt == nil {
		return nil
	}

	duration := o.CompletedAt.Sub(*o.StartedAt)
	return &duration
}

// EstimateCompletionTime predicts when the operation will complete based on its type.
// Returns nil for operations that are already in a terminal state.
func (o *Operation) EstimateCompletionTime() *time.Time {
	if o.Status != StatusPending && o.Status != StatusInProgress {
		return nil
	}

	var durationEstimate time.Duration

	// Different operation types have different expected durations
	switch o.Type {
	case OpTenantCreate:
		durationEstimate = 5 * time.Minute
	case OpTenantDelete:
		durationEstimate = 3 * time.Minute
	default:
		durationEstimate = 5 * time.Minute
	}

	var startTime time.Time
	if o.StartedAt != nil {
		startTime = *o.StartedAt
	} else {
		startTime = o.CreatedAt
	}

	estimatedCompletion := startTime.Add(durationEstimate)
	return &estimatedCompletion
}

// GetProgress returns an estimated progress percentage (0-100) for the operation.
// The percentage is based on the elapsed time compared to the estimated total duration.
// Returns 100 for completed operations and an error for failed or cancelled operations.
func (o *Operation) GetProgress() (int, error) {
	if o.IsTerminal() {
		if o.Status == StatusCompleted {
			return 100, nil
		}
		return 0, errors.New("operation failed or cancelled")
	}

	if o.Status == StatusPending {
		return 0, nil
	}

	if o.StartedAt == nil {
		return 5, nil
	}

	estimatedCompletionTime := o.EstimateCompletionTime()
	if estimatedCompletionTime == nil {
		return 50, nil // Default to 50% if we can't estimate
	}

	now := time.Now()
	totalDuration := estimatedCompletionTime.Sub(*o.StartedAt)
	elapsedDuration := now.Sub(*o.StartedAt)

	// Calculate progress as a percentage
	if totalDuration <= 0 {
		return 99, nil
	}

	progress := int((float64(elapsedDuration) / float64(totalDuration)) * 100)

	// Clamp progress between 0 and 99 (100 is only for completed operations)
	if progress < 0 {
		progress = 0
	}
	if progress > 99 {
		progress = 99
	}

	return progress, nil
}

// IsRetryable checks if a failed operation can be retried.
// Some operations may not be retryable if they've already had partial effects.
func (o *Operation) IsRetryable() bool {
	if o.Status != StatusFailed {
		return false
	}

	// Tenant deletion is not retryable if the tenant was actually deleted
	if o.Type == OpTenantDelete && o.Result != nil {
		if status, ok := o.Result["status"]; ok && status == "deleted" {
			return false
		}
	}

	return true
}
