package operation

import (
	"context"
	"fmt"
	"time"

	"github.com/ahrav/hoglet-hub/internal/domain/operation"
)

// Service provides operation-related application services.
// It coordinates operation state transitions and manages lifecycle events,
// abstracting the underlying data persistence.
type Service struct {
	repo operation.Repository
}

// NewService creates a new operation service with the provided repository.
// The repository is used for persisting and retrieving operation data.
func NewService(repo operation.Repository) *Service {
	return &Service{repo: repo}
}

// GetByID retrieves an operation by its ID.
// Returns the operation or an appropriate error if not found or if retrieval fails.
func (s *Service) GetByID(ctx context.Context, operationID int64) (*operation.Operation, error) {
	op, err := s.repo.FindByID(ctx, operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve operation: %w", err)
	}

	if op == nil {
		return nil, operation.ErrOperationNotFound
	}

	return op, nil
}

// StartOperation transitions an operation from pending to in-progress state.
// This is typically called when execution of the operation begins.
func (s *Service) StartOperation(ctx context.Context, operationID int64) error {
	op, err := s.repo.FindByID(ctx, operationID)
	if err != nil {
		return fmt.Errorf("failed to retrieve operation: %w", err)
	}

	if op == nil {
		return operation.ErrOperationNotFound
	}

	op.Start()

	if err := s.repo.Update(ctx, op); err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}

	return nil
}

// CompleteOperation marks an operation as successfully completed with result data.
// The result map contains operation-specific outputs needed by consumers.
func (s *Service) CompleteOperation(ctx context.Context, operationID int64, result map[string]interface{}) error {
	op, err := s.repo.FindByID(ctx, operationID)
	if err != nil {
		return fmt.Errorf("failed to retrieve operation: %w", err)
	}

	if op == nil {
		return operation.ErrOperationNotFound
	}

	op.Complete(result)

	if err := s.repo.Update(ctx, op); err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}

	return nil
}

// FailOperation marks an operation as failed with a specific error message.
// Used when an operation encounters an unrecoverable error condition.
func (s *Service) FailOperation(ctx context.Context, operationID int64, errorMsg string) error {
	op, err := s.repo.FindByID(ctx, operationID)
	if err != nil {
		return fmt.Errorf("failed to retrieve operation: %w", err)
	}

	if op == nil {
		return operation.ErrOperationNotFound
	}

	op.Fail(errorMsg)

	if err := s.repo.Update(ctx, op); err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}

	return nil
}

// CancelOperation marks an operation as cancelled with a reason.
// Used when an operation is deliberately halted before completion.
func (s *Service) CancelOperation(ctx context.Context, operationID int64, reason string) error {
	op, err := s.repo.FindByID(ctx, operationID)
	if err != nil {
		return fmt.Errorf("failed to retrieve operation: %w", err)
	}

	if op == nil {
		return operation.ErrOperationNotFound
	}

	op.Cancel(reason)

	if err := s.repo.Update(ctx, op); err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}

	return nil
}

// ListIncompleteOperations returns all operations that haven't reached a terminal state.
// Useful for finding operations that may need attention or monitoring.
func (s *Service) ListIncompleteOperations(ctx context.Context) ([]*operation.Operation, error) {
	ops, err := s.repo.FindIncomplete(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve incomplete operations: %w", err)
	}

	return ops, nil
}

// ListStalledOperations returns operations that have been running longer than the specified threshold.
// Helps identify operations that may be stuck or need intervention.
func (s *Service) ListStalledOperations(ctx context.Context, threshold time.Duration) ([]*operation.Operation, error) {
	inProgressOps, err := s.repo.FindByStatus(ctx, operation.StatusInProgress)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve in-progress operations: %w", err)
	}

	var stalledOps []*operation.Operation
	now := time.Now()

	for _, op := range inProgressOps {
		if op.StartedAt != nil {
			duration := now.Sub(*op.StartedAt)
			if duration > threshold {
				stalledOps = append(stalledOps, op)
			}
		}
	}

	return stalledOps, nil
}

// GetOperationsByTenant returns all operations associated with a specific tenant.
// Enables tenant-specific operation monitoring and management.
func (s *Service) GetOperationsByTenant(ctx context.Context, tenantID int64) ([]*operation.Operation, error) {
	ops, err := s.repo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve operations for tenant %d: %w", tenantID, err)
	}

	return ops, nil
}

// GetOperationProgress returns the completion progress of an operation as a percentage (0-100).
// Provides insight into how far along an operation is toward completion.
func (s *Service) GetOperationProgress(ctx context.Context, operationID int64) (int, error) {
	op, err := s.repo.FindByID(ctx, operationID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve operation: %w", err)
	}

	if op == nil {
		return 0, operation.ErrOperationNotFound
	}

	progress, err := op.GetProgress()
	if err != nil {
		return 0, fmt.Errorf("failed to calculate operation progress: %w", err)
	}

	return progress, nil
}

// GetOperationEstimatedCompletion returns the estimated time when an operation will complete.
// Returns nil if the operation doesn't have enough information to make an estimate.
func (s *Service) GetOperationEstimatedCompletion(ctx context.Context, operationID int64) (*time.Time, error) {
	op, err := s.repo.FindByID(ctx, operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve operation: %w", err)
	}

	if op == nil {
		return nil, operation.ErrOperationNotFound
	}

	estimatedTime := op.EstimateCompletionTime()

	return estimatedTime, nil
}

// RetryOperation attempts to restart a failed operation from its beginning.
// Only operations in certain states (like failed) can be retried.
func (s *Service) RetryOperation(ctx context.Context, operationID int64) error {
	op, err := s.repo.FindByID(ctx, operationID)
	if err != nil {
		return fmt.Errorf("failed to retrieve operation: %w", err)
	}

	if op == nil {
		return operation.ErrOperationNotFound
	}

	if !op.IsRetryable() {
		return fmt.Errorf("operation cannot be retried")
	}

	// Reset operation to pending state
	op.Status = operation.StatusPending
	op.ErrorMessage = nil
	now := time.Now()
	op.UpdatedAt = &now

	if err := s.repo.Update(ctx, op); err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}

	return nil
}
