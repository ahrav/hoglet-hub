package operation

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOpIsValid(t *testing.T) {
	tests := []struct {
		name     string
		opType   Op
		expected bool
	}{
		{"Valid - tenant create", OpTenantCreate, true},
		{"Valid - tenant delete", OpTenantDelete, true},
		{"Invalid - empty string", Op(""), false},
		{"Invalid - unsupported op", Op("unsupported.operation"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.opType.IsValid()
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestParseType_Success(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Op
	}{
		{"tenant create", "tenant.create", OpTenantCreate},
		{"tenant delete", "tenant.delete", OpTenantDelete},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseType(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestParseType_Error(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"unsupported operation", "unsupported.operation"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseType(tc.input)
			assert.Error(t, err)
			assert.Equal(t, Op(""), got)
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name           string
		field          string
		message        string
		expectedOutput string
	}{
		{
			name:           "Type validation error",
			field:          "type",
			message:        "must be a valid operation type",
			expectedOutput: "validation error on field 'type': must be a valid operation type",
		},
		{
			name:           "Required field error",
			field:          "tenant_id",
			message:        "cannot be empty",
			expectedOutput: "validation error on field 'tenant_id': cannot be empty",
		},
		{
			name:           "Format error",
			field:          "region",
			message:        "invalid format",
			expectedOutput: "validation error on field 'region': invalid format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := NewValidationError(tc.field, tc.message)
			assert.Equal(t, tc.expectedOutput, err.Error())
			assert.Equal(t, tc.field, err.Field)
			assert.Equal(t, tc.message, err.Message)
		})
	}
}

func TestNewOperation_Success(t *testing.T) {
	tenantID := int64(1234)
	params := map[string]any{
		"name":   "test-tenant",
		"region": "us-west",
		"tier":   "standard",
	}

	tests := []struct {
		name     string
		opType   Op
		tenantID *int64
		params   map[string]any
	}{
		{
			name:     "Valid operation - tenant create",
			opType:   OpTenantCreate,
			tenantID: &tenantID,
			params:   params,
		},
		{
			name:     "Valid operation - tenant delete",
			opType:   OpTenantDelete,
			tenantID: &tenantID,
			params:   map[string]any{"tenant_id": tenantID},
		},
		{
			name:     "Valid operation - nil tenant ID",
			opType:   OpTenantCreate,
			tenantID: nil,
			params:   params,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op, err := NewOperation(tc.opType, tc.tenantID, tc.params)
			assert.NoError(t, err)

			assert.Equal(t, tc.opType, op.Type)

			if tc.tenantID != nil {
				assert.NotNil(t, op.TenantID)
				assert.Equal(t, *tc.tenantID, *op.TenantID)
			} else {
				assert.Nil(t, op.TenantID)
			}

			assert.Equal(t, StatusPending, op.Status)
			assert.Equal(t, len(tc.params), len(op.Parameters))
		})
	}
}

func TestNewOperation_Error(t *testing.T) {
	tenantID := int64(1234)
	params := map[string]any{
		"name":   "test-tenant",
		"region": "us-west",
		"tier":   "standard",
	}

	tests := []struct {
		name       string
		opType     Op
		tenantID   *int64
		params     map[string]any
		errorField string
	}{
		{
			name:       "Invalid operation type",
			opType:     Op("invalid.type"),
			tenantID:   &tenantID,
			params:     params,
			errorField: "type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op, err := NewOperation(tc.opType, tc.tenantID, tc.params)
			assert.Error(t, err)
			assert.Nil(t, op)

			validationErr, ok := err.(ValidationError)
			assert.True(t, ok, "Expected ValidationError, got %T", err)
			assert.Equal(t, tc.errorField, validationErr.Field)
		})
	}
}

func TestNewTenantCreateOperation_Success(t *testing.T) {
	tenantID := int64(1234)
	isolationGroupID := int64(5678)

	tests := []struct {
		name             string
		tenantID         int64
		tenantName       string
		region           string
		tier             string
		isolationGroupID *int64
		hasIsolationKey  bool
	}{
		{
			name:             "Valid without isolation group",
			tenantID:         tenantID,
			tenantName:       "test-tenant",
			region:           "us-west",
			tier:             "standard",
			isolationGroupID: nil,
			hasIsolationKey:  false,
		},
		{
			name:             "Valid with isolation group",
			tenantID:         tenantID,
			tenantName:       "test-tenant",
			region:           "us-west",
			tier:             "standard",
			isolationGroupID: &isolationGroupID,
			hasIsolationKey:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op, err := NewTenantCreateOperation(tc.tenantID, tc.tenantName, tc.region, tc.tier, tc.isolationGroupID)
			assert.NoError(t, err)

			assert.Equal(t, OpTenantCreate, op.Type)
			assert.NotNil(t, op.TenantID)
			assert.Equal(t, tc.tenantID, *op.TenantID)
			assert.Equal(t, tc.tenantName, op.Parameters["name"])
			assert.Equal(t, tc.region, op.Parameters["region"])
			assert.Equal(t, tc.tier, op.Parameters["tier"])

			if tc.hasIsolationKey {
				isolationID, exists := op.Parameters["isolation_group_id"]
				assert.True(t, exists, "Operation.Parameters[\"isolation_group_id\"] should exist")
				assert.Equal(t, *tc.isolationGroupID, isolationID)
			} else {
				_, exists := op.Parameters["isolation_group_id"]
				assert.False(t, exists, "Operation.Parameters[\"isolation_group_id\"] should not exist")
			}
		})
	}
}

func TestNewTenantDeleteOperation_Success(t *testing.T) {
	tests := []struct {
		name     string
		tenantID int64
	}{
		{name: "Valid tenant delete operation", tenantID: 1234},
		{name: "Zero tenant ID", tenantID: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op, err := NewTenantDeleteOperation(tc.tenantID)

			assert.NoError(t, err)
			assert.Equal(t, OpTenantDelete, op.Type)
			assert.NotNil(t, op.TenantID)
			assert.Equal(t, tc.tenantID, *op.TenantID)

			paramTenantID, ok := op.Parameters["tenant_id"].(int64)
			assert.True(t, ok, "tenant_id parameter should be of type int64")
			assert.Equal(t, tc.tenantID, paramTenantID)
		})
	}
}

func TestOperationStateTransition_ToInProgress(t *testing.T) {
	op, _ := NewTenantCreateOperation(int64(1234), "test-tenant", "us-west", "standard", nil)

	beforeTransition := time.Now()
	op.Start()
	afterTransition := time.Now()

	assert.Equal(t, StatusInProgress, op.Status)
	assert.NotNil(t, op.StartedAt)
	assert.True(t, op.StartedAt.Equal(beforeTransition) || op.StartedAt.After(beforeTransition))
	assert.True(t, op.StartedAt.Equal(afterTransition) || op.StartedAt.Before(afterTransition))
	assert.NotNil(t, op.UpdatedAt)
}

func TestOperationStateTransition_ToCompleted(t *testing.T) {
	op, _ := NewTenantCreateOperation(int64(1234), "test-tenant", "us-west", "standard", nil)
	op.Start()

	expectedResults := map[string]any{"tenant_created": true}

	beforeTransition := time.Now()
	op.Complete(expectedResults)
	afterTransition := time.Now()

	assert.Equal(t, StatusCompleted, op.Status)
	assert.NotNil(t, op.CompletedAt)
	assert.True(t, op.CompletedAt.Equal(beforeTransition) || op.CompletedAt.After(beforeTransition))
	assert.True(t, op.CompletedAt.Equal(afterTransition) || op.CompletedAt.Before(afterTransition))
	assert.NotNil(t, op.UpdatedAt)
	assert.Equal(t, expectedResults, op.Result)
}

func TestOperationStateTransition_ToFailed(t *testing.T) {
	op, _ := NewTenantCreateOperation(int64(1234), "test-tenant", "us-west", "standard", nil)
	op.Start()

	expectedError := "database connection failed"

	beforeTransition := time.Now()
	op.Fail(expectedError)
	afterTransition := time.Now()

	assert.Equal(t, StatusFailed, op.Status)
	assert.NotNil(t, op.CompletedAt)
	assert.True(t, op.CompletedAt.Equal(beforeTransition) || op.CompletedAt.After(beforeTransition))
	assert.True(t, op.CompletedAt.Equal(afterTransition) || op.CompletedAt.Before(afterTransition))
	assert.NotNil(t, op.UpdatedAt)
	assert.NotNil(t, op.ErrorMessage)
	assert.Equal(t, expectedError, *op.ErrorMessage)
}

func TestOperationStateTransition_ToCancelled(t *testing.T) {
	// Test from InProgress to Cancelled.
	t.Run("from InProgress", func(t *testing.T) {
		op, _ := NewTenantCreateOperation(int64(1234), "test-tenant", "us-west", "standard", nil)
		op.Start()

		expectedError := "cancelled by user"

		beforeTransition := time.Now()
		op.Cancel(expectedError)
		afterTransition := time.Now()

		assert.Equal(t, StatusCancelled, op.Status)
		assert.NotNil(t, op.CompletedAt)
		assert.True(t, op.CompletedAt.Equal(beforeTransition) || op.CompletedAt.After(beforeTransition))
		assert.True(t, op.CompletedAt.Equal(afterTransition) || op.CompletedAt.Before(afterTransition))
		assert.NotNil(t, op.UpdatedAt)
		assert.NotNil(t, op.ErrorMessage)
		assert.Equal(t, expectedError, *op.ErrorMessage)
	})

	// Test from Pending to Cancelled.
	t.Run("from Pending", func(t *testing.T) {
		op, _ := NewTenantCreateOperation(int64(1234), "test-tenant", "us-west", "standard", nil)

		expectedError := "cancelled before start"

		beforeTransition := time.Now()
		op.Cancel(expectedError)
		afterTransition := time.Now()

		assert.Equal(t, StatusCancelled, op.Status)
		assert.NotNil(t, op.CompletedAt)
		assert.True(t, op.CompletedAt.Equal(beforeTransition) || op.CompletedAt.After(beforeTransition))
		assert.True(t, op.CompletedAt.Equal(afterTransition) || op.CompletedAt.Before(afterTransition))
		assert.NotNil(t, op.UpdatedAt)
		assert.NotNil(t, op.ErrorMessage)
		assert.Equal(t, expectedError, *op.ErrorMessage)
	})
}

func TestOperation_IsPending(t *testing.T) {
	// Pending operation should return true.
	t.Run("when pending", func(t *testing.T) {
		op, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		assert.True(t, op.IsPending())
	})

	// Non-pending operations should return false.
	t.Run("when not pending", func(t *testing.T) {
		// In progress.
		inProgressOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		inProgressOp.Start()
		assert.False(t, inProgressOp.IsPending())

		// Completed.
		completedOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		completedOp.Start()
		completedOp.Complete(map[string]any{})
		assert.False(t, completedOp.IsPending())

		// Failed.
		failedOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		failedOp.Start()
		failedOp.Fail("error")
		assert.False(t, failedOp.IsPending())

		// Cancelled.
		cancelledOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		cancelledOp.Start()
		cancelledOp.Cancel("cancelled")
		assert.False(t, cancelledOp.IsPending())
	})
}

func TestOperation_IsInProgress(t *testing.T) {
	// In progress operation should return true.
	t.Run("when in progress", func(t *testing.T) {
		op, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		op.Start()
		assert.True(t, op.IsInProgress())
	})

	// Non-in-progress operations should return false
	t.Run("when not in progress", func(t *testing.T) {
		// Pending.
		pendingOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		assert.False(t, pendingOp.IsInProgress())

		// Completed.
		completedOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		completedOp.Start()
		completedOp.Complete(map[string]any{})
		assert.False(t, completedOp.IsInProgress())

		// Failed.
		failedOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		failedOp.Start()
		failedOp.Fail("error")
		assert.False(t, failedOp.IsInProgress())

		// Cancelled.
		cancelledOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		cancelledOp.Start()
		cancelledOp.Cancel("cancelled")
		assert.False(t, cancelledOp.IsInProgress())
	})
}

func TestOperation_IsTerminal(t *testing.T) {
	// Terminal operations should return true.
	t.Run("when terminal", func(t *testing.T) {
		// Completed.
		completedOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		completedOp.Start()
		completedOp.Complete(map[string]any{})
		assert.True(t, completedOp.IsTerminal())

		// Failed.
		failedOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		failedOp.Start()
		failedOp.Fail("error")
		assert.True(t, failedOp.IsTerminal())

		// Cancelled.
		cancelledOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		cancelledOp.Start()
		cancelledOp.Cancel("cancelled")
		assert.True(t, cancelledOp.IsTerminal())
	})

	// Non-terminal operations should return false.
	t.Run("when not terminal", func(t *testing.T) {
		// Pending.
		pendingOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		assert.False(t, pendingOp.IsTerminal())

		// In progress.
		inProgressOp, _ := NewTenantCreateOperation(int64(1234), "test", "us-west", "standard", nil)
		inProgressOp.Start()
		assert.False(t, inProgressOp.IsTerminal())
	})
}

func TestOperationDuration_HasDuration(t *testing.T) {
	synctest.Run(func() {
		tenantID := int64(1234)

		tests := []struct {
			name  string
			setup func() *Operation
		}{
			{
				name: "Completed operation",
				setup: func() *Operation {
					op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
					op.Start()
					time.Sleep(100 * time.Millisecond) // Small delay for measurable duration
					op.Complete(map[string]any{})
					return op
				},
			},
			{
				name: "Failed operation",
				setup: func() *Operation {
					op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
					op.Start()
					time.Sleep(100 * time.Millisecond) // Small delay for measurable duration
					op.Fail("error")
					return op
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				op := tc.setup()
				duration := op.Duration()

				assert.NotNil(t, duration)
				assert.GreaterOrEqual(t, duration.Milliseconds(), int64(0))
			})
		}
	})
}

func TestOperationDuration_NoDuration(t *testing.T) {
	tenantID := int64(1234)

	tests := []struct {
		name  string
		setup func() *Operation
	}{
		{
			name: "Pending operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				return op
			},
		},
		{
			name: "InProgress operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				return op
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.setup()
			duration := op.Duration()
			assert.Nil(t, duration)
		})
	}
}

func TestOperationEstimateCompletionTime_HasEstimate(t *testing.T) {
	tenantID := int64(1234)

	tests := []struct {
		name      string
		setup     func() *Operation
		opType    Op
		reference string // whether to use "start" or "create" time
	}{
		{
			name: "Pending tenant create operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				return op
			},
			opType:    OpTenantCreate,
			reference: "create",
		},
		{
			name: "InProgress tenant delete operation",
			setup: func() *Operation {
				op, _ := NewTenantDeleteOperation(tenantID)
				op.Start()
				return op
			},
			opType:    OpTenantDelete,
			reference: "start",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.setup()
			estimate := op.EstimateCompletionTime()

			assert.NotNil(t, estimate)

			var expectedDuration time.Duration
			switch tc.opType {
			case OpTenantCreate:
				expectedDuration = 5 * time.Minute
			case OpTenantDelete:
				expectedDuration = 3 * time.Minute
			default:
				expectedDuration = 5 * time.Minute
			}

			// Determine reference time (creation or start time).
			var refTime time.Time
			if tc.reference == "start" {
				assert.NotNil(t, op.StartedAt)
				refTime = *op.StartedAt
			} else {
				refTime = op.CreatedAt
			}

			expectedTime := refTime.Add(expectedDuration)
			assert.True(t, estimate.Equal(expectedTime))
		})
	}
}

func TestOperationEstimateCompletionTime_NoEstimate(t *testing.T) {
	tenantID := int64(1234)

	tests := []struct {
		name  string
		setup func() *Operation
	}{
		{
			name: "Completed operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Complete(map[string]any{})
				return op
			},
		},
		{
			name: "Failed operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Fail("error")
				return op
			},
		},
		{
			name: "Cancelled operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Cancel("cancelled")
				return op
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.setup()
			estimate := op.EstimateCompletionTime()
			assert.Nil(t, estimate)
		})
	}
}

func TestOperationGetProgress_Success(t *testing.T) {
	tenantID := int64(1234)

	tests := []struct {
		name          string
		setup         func() *Operation
		expectedRange [2]int
	}{
		{
			name: "Pending operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				return op
			},
			expectedRange: [2]int{0, 0},
		},
		{
			name: "Just started operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				return op
			},
			expectedRange: [2]int{0, 20}, // Early in progress, but not 0
		},
		{
			name: "Completed operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Complete(map[string]any{})
				return op
			},
			expectedRange: [2]int{100, 100},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.setup()
			progress, err := op.GetProgress()

			assert.NoError(t, err)
			assert.GreaterOrEqual(t, progress, tc.expectedRange[0])
			assert.LessOrEqual(t, progress, tc.expectedRange[1])
		})
	}
}

func TestOperationGetProgress_Error(t *testing.T) {
	tenantID := int64(1234)

	tests := []struct {
		name  string
		setup func() *Operation
	}{
		{
			name: "Failed operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Fail("error")
				return op
			},
		},
		{
			name: "Cancelled operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Cancel("cancelled")
				return op
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.setup()
			progress, err := op.GetProgress()

			assert.Error(t, err)
			assert.Equal(t, 0, progress)
		})
	}
}

func TestOperationIsRetryable_Retryable(t *testing.T) {
	tenantID := int64(1234)

	tests := []struct {
		name  string
		setup func() *Operation
	}{
		{
			name: "Failed create operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Fail("database connection failed")
				return op
			},
		},
		{
			name: "Failed delete operation with no deletion",
			setup: func() *Operation {
				op, _ := NewTenantDeleteOperation(tenantID)
				op.Start()
				op.Fail("database connection failed")
				return op
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.setup()
			assert.True(t, op.IsRetryable())
		})
	}
}

func TestOperationIsRetryable_NotRetryable(t *testing.T) {
	tenantID := int64(1234)

	tests := []struct {
		name  string
		setup func() *Operation
	}{
		{
			name: "Pending operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				return op
			},
		},
		{
			name: "InProgress operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				return op
			},
		},
		{
			name: "Completed operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Complete(map[string]any{})
				return op
			},
		},
		{
			name: "Failed delete operation with deletion completed",
			setup: func() *Operation {
				op, _ := NewTenantDeleteOperation(tenantID)
				op.Start()
				op.Result = map[string]any{"status": "deleted"}
				op.Fail("network error after deletion")
				return op
			},
		},
		{
			name: "Cancelled operation",
			setup: func() *Operation {
				op, _ := NewTenantCreateOperation(tenantID, "test", "us-west", "standard", nil)
				op.Start()
				op.Cancel("cancelled by user")
				return op
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.setup()
			assert.False(t, op.IsRetryable())
		})
	}
}

func TestOperationString(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		opType   Op
		expected string
	}{
		{OpTenantCreate, "tenant.create"},
		{OpTenantDelete, "tenant.delete"},
		{Op("custom.type"), "custom.type"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(tc.expected, tc.opType.String())
		})
	}
}

func TestIsValidType(t *testing.T) {
	assert := assert.New(t)

	// Testing the unexported isValidType function indirectly through Op.IsValid.
	tests := []struct {
		opType   Op
		expected bool
	}{
		{OpTenantCreate, true},
		{OpTenantDelete, true},
		{Op(""), false},
		{Op("custom.type"), false},
	}

	for _, tc := range tests {
		t.Run(string(tc.opType), func(t *testing.T) {
			assert.Equal(tc.expected, tc.opType.IsValid())
		})
	}
}
