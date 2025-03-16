package testutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ahrav/hoglet-hub/internal/domain/operation"
)

// DefaultOperationTimeout is the default timeout for waiting for operations to complete.
const DefaultOperationTimeout = 10 * time.Second

// ErrOperationTimeout is returned when an operation doesn't complete in the expected time.
var ErrOperationTimeout = errors.New("operation timeout")

// WaitForOperationStatus polls for a specific operation status until it matches or times out.
func WaitForOperationStatus(
	ctx context.Context,
	t *testing.T,
	operationRepo operation.Repository,
	operationID int64,
	expectedStatus operation.Status,
	timeout time.Duration,
) (*operation.Operation, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use this ticker to help with debugging.
	// This gives us an idea of how long the operation is taking.
	// It also gives us insight into the status of the operation.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ErrOperationTimeout
		case <-ticker.C:
			op, err := operationRepo.FindByID(ctx, operationID)
			if err != nil {
				return nil, err
			}

			if op == nil {
				t.Logf("Operation %d not found", operationID)
				continue
			}

			t.Logf("Operation %d status: %s", operationID, op.Status)

			if op.Status == expectedStatus {
				return op, nil
			}

			// If the operation is in a terminal state but not the expected one, fail
			if op.Status == operation.StatusFailed || op.Status == operation.StatusCompleted {
				if op.Status != expectedStatus {
					t.Logf("Operation %d in terminal state %s, but expected %s",
						operationID, op.Status, expectedStatus)

					// Print additional debug information for failed operations
					if op.Status == operation.StatusFailed && op.ErrorMessage != nil {
						t.Logf("Error details for operation %d: %s", operationID, *op.ErrorMessage)
					}

					return op, nil
				}
			}
		}
	}
}
