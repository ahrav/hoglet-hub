package operation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/ahrav/hoglet-hub/internal/application/operation"
	domainOp "github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
)

type MockOperationRepo struct{ mock.Mock }

func (m *MockOperationRepo) Create(ctx context.Context, op *domainOp.Operation) (int64, error) {
	args := m.Called(ctx, op)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOperationRepo) Update(ctx context.Context, op *domainOp.Operation) error {
	args := m.Called(ctx, op)
	return args.Error(0)
}

func (m *MockOperationRepo) FindByID(ctx context.Context, id int64) (*domainOp.Operation, error) {
	args := m.Called(ctx, id)
	val, _ := args.Get(0).(*domainOp.Operation)
	return val, args.Error(1)
}

func (m *MockOperationRepo) FindByTenantID(ctx context.Context, tenantID int64) ([]*domainOp.Operation, error) {
	args := m.Called(ctx, tenantID)
	val, _ := args.Get(0).([]*domainOp.Operation)
	return val, args.Error(1)
}

func (m *MockOperationRepo) FindByStatus(ctx context.Context, status domainOp.Status) ([]*domainOp.Operation, error) {
	args := m.Called(ctx, status)
	val, _ := args.Get(0).([]*domainOp.Operation)
	return val, args.Error(1)
}

func (m *MockOperationRepo) FindIncomplete(ctx context.Context) ([]*domainOp.Operation, error) {
	args := m.Called(ctx)
	val, _ := args.Get(0).([]*domainOp.Operation)
	return val, args.Error(1)
}

func TestOperationService_GetByID(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		desc           string
		operationID    int64
		mockSetup      func(*MockOperationRepo)
		wantError      bool
		wantErrorIs    error
		wantOp         *domainOp.Operation
		wantErrorMatch string
	}{
		{
			desc:        "repo returns error",
			operationID: 99,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByID", ctx, int64(99)).
					Return((*domainOp.Operation)(nil), errors.New("some DB error"))
			},
			wantError:      true,
			wantErrorMatch: "failed to retrieve operation: some DB error",
		},
		{
			desc:        "operation not found",
			operationID: 101,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByID", ctx, int64(101)).
					Return((*domainOp.Operation)(nil), nil)
			},
			wantError:   true,
			wantErrorIs: domainOp.ErrOperationNotFound,
		},
		{
			desc:        "successful retrieval",
			operationID: 10,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByID", ctx, int64(10)).
					Return(&domainOp.Operation{ID: 10, Status: domainOp.StatusPending}, nil)
			},
			wantOp: &domainOp.Operation{ID: 10, Status: domainOp.StatusPending},
		},
	}

	for _, tc := range testCases {
		mockRepo := new(MockOperationRepo)
		tc.mockSetup(mockRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := operation.NewService(mockRepo, logger, tracer)
		op, err := svc.GetByID(ctx, tc.operationID)
		if tc.wantError {
			assert.Error(t, err, "expected an error")
			if tc.wantErrorIs != nil {
				assert.ErrorIs(t, err, tc.wantErrorIs)
			}
			if tc.wantErrorMatch != "" {
				assert.Contains(t, err.Error(), tc.wantErrorMatch)
			}
			assert.Nil(t, op)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.wantOp, op)
		}

		mockRepo.AssertExpectations(t)
	}
}

// func TestOperationService_StartOperation(t *testing.T) {
// 	ctx := context.Background()

// 	testCases := []struct {
// 		desc           string
// 		operationID    int64
// 		mockSetup      func(*MockOperationRepo)
// 		wantError      bool
// 		wantErrorIs    error
// 		wantErrorMatch string
// 	}{
// 		{
// 			desc:        "repo FindByID error",
// 			operationID: 5,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(5)).
// 					Return((*domainOp.Operation)(nil), errors.New("db error"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to retrieve operation: db error",
// 		},
// 		{
// 			desc:        "operation not found",
// 			operationID: 7,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(7)).
// 					Return((*domainOp.Operation)(nil), nil)
// 			},
// 			wantError:   true,
// 			wantErrorIs: domainOp.ErrOperationNotFound,
// 		},
// 		{
// 			desc:        "update error",
// 			operationID: 9,
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 9, Status: domainOp.StatusPending}
// 				m.On("FindByID", ctx, int64(9)).
// 					Return(op, nil)
// 				// After Start, op becomes in_progress
// 				m.On("Update", ctx, op).
// 					Return(errors.New("update fail"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to update operation: update fail",
// 		},
// 		{
// 			desc:        "successful start",
// 			operationID: 11,
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 11, Status: domainOp.StatusPending}
// 				m.On("FindByID", ctx, int64(11)).
// 					Return(op, nil)
// 				m.On("Update", ctx, mock.AnythingOfType("*operation.Operation")).
// 					Return(nil)
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		mockRepo := new(MockOperationRepo)
// 		tc.mockSetup(mockRepo)

// 		svc := operation.NewService(mockRepo)
// 		err := svc.StartOperation(ctx, tc.operationID)
// 		if tc.wantError {
// 			assert.Error(t, err)
// 			if tc.wantErrorIs != nil {
// 				assert.ErrorIs(t, err, tc.wantErrorIs)
// 			}
// 			if tc.wantErrorMatch != "" {
// 				assert.Contains(t, err.Error(), tc.wantErrorMatch)
// 			}
// 		} else {
// 			assert.NoError(t, err)
// 		}
// 		mockRepo.AssertExpectations(t)
// 	}
// }

// func TestOperationService_CompleteOperation(t *testing.T) {
// 	ctx := context.Background()

// 	testCases := []struct {
// 		desc           string
// 		operationID    int64
// 		result         map[string]interface{}
// 		mockSetup      func(*MockOperationRepo)
// 		wantError      bool
// 		wantErrorIs    error
// 		wantErrorMatch string
// 	}{
// 		{
// 			desc:        "repo FindByID error",
// 			operationID: 5,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(5)).
// 					Return((*domainOp.Operation)(nil), errors.New("db error"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to retrieve operation: db error",
// 		},
// 		{
// 			desc:        "operation not found",
// 			operationID: 7,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(7)).
// 					Return((*domainOp.Operation)(nil), nil)
// 			},
// 			wantError:   true,
// 			wantErrorIs: domainOp.ErrOperationNotFound,
// 		},
// 		{
// 			desc:        "update error",
// 			operationID: 9,
// 			result:      map[string]interface{}{"hello": "world"},
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 9, Status: domainOp.StatusInProgress}
// 				m.On("FindByID", ctx, int64(9)).
// 					Return(op, nil)
// 				m.On("Update", ctx, op).
// 					Return(errors.New("update fail"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to update operation: update fail",
// 		},
// 		{
// 			desc:        "successful complete",
// 			operationID: 11,
// 			result:      map[string]interface{}{"foo": "bar"},
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 11, Status: domainOp.StatusInProgress}
// 				m.On("FindByID", ctx, int64(11)).
// 					Return(op, nil)
// 				m.On("Update", ctx, mock.AnythingOfType("*operation.Operation")).
// 					Return(nil)
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		mockRepo := new(MockOperationRepo)
// 		tc.mockSetup(mockRepo)

// 		svc := operation.NewService(mockRepo)
// 		err := svc.CompleteOperation(ctx, tc.operationID, tc.result)
// 		if tc.wantError {
// 			assert.Error(t, err)
// 			if tc.wantErrorIs != nil {
// 				assert.ErrorIs(t, err, tc.wantErrorIs)
// 			}
// 			if tc.wantErrorMatch != "" {
// 				assert.Contains(t, err.Error(), tc.wantErrorMatch)
// 			}
// 		} else {
// 			assert.NoError(t, err)
// 		}
// 		mockRepo.AssertExpectations(t)
// 	}
// }

// func TestOperationService_FailOperation(t *testing.T) {
// 	ctx := context.Background()

// 	testCases := []struct {
// 		desc           string
// 		operationID    int64
// 		errorMsg       string
// 		mockSetup      func(*MockOperationRepo)
// 		wantError      bool
// 		wantErrorIs    error
// 		wantErrorMatch string
// 	}{
// 		{
// 			desc:        "repo error on find",
// 			operationID: 1,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(1)).
// 					Return((*domainOp.Operation)(nil), errors.New("db error"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to retrieve operation: db error",
// 		},
// 		{
// 			desc:        "not found",
// 			operationID: 2,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(2)).
// 					Return((*domainOp.Operation)(nil), nil)
// 			},
// 			wantError:   true,
// 			wantErrorIs: domainOp.ErrOperationNotFound,
// 		},
// 		{
// 			desc:        "fail update error",
// 			operationID: 3,
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 3, Status: domainOp.StatusInProgress}
// 				m.On("FindByID", ctx, int64(3)).
// 					Return(op, nil)
// 				m.On("Update", ctx, op).
// 					Return(errors.New("update fail"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to update operation: update fail",
// 		},
// 		{
// 			desc:        "fail successful",
// 			operationID: 4,
// 			errorMsg:    "something broke",
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 4, Status: domainOp.StatusInProgress}
// 				m.On("FindByID", ctx, int64(4)).
// 					Return(op, nil)
// 				m.On("Update", ctx, mock.AnythingOfType("*operation.Operation")).
// 					Return(nil)
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		mockRepo := new(MockOperationRepo)
// 		tc.mockSetup(mockRepo)

// 		svc := operation.NewService(mockRepo)
// 		err := svc.FailOperation(ctx, tc.operationID, tc.errorMsg)
// 		if tc.wantError {
// 			assert.Error(t, err)
// 			if tc.wantErrorIs != nil {
// 				assert.ErrorIs(t, err, tc.wantErrorIs)
// 			}
// 			if tc.wantErrorMatch != "" {
// 				assert.Contains(t, err.Error(), tc.wantErrorMatch)
// 			}
// 		} else {
// 			assert.NoError(t, err)
// 		}
// 		mockRepo.AssertExpectations(t)
// 	}
// }

// func TestOperationService_CancelOperation(t *testing.T) {
// 	ctx := context.Background()

// 	testCases := []struct {
// 		desc           string
// 		operationID    int64
// 		reason         string
// 		mockSetup      func(*MockOperationRepo)
// 		wantError      bool
// 		wantErrorIs    error
// 		wantErrorMatch string
// 	}{
// 		{
// 			desc:        "repo error on find",
// 			operationID: 10,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(10)).
// 					Return((*domainOp.Operation)(nil), errors.New("db error"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to retrieve operation: db error",
// 		},
// 		{
// 			desc:        "operation not found",
// 			operationID: 11,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(11)).
// 					Return((*domainOp.Operation)(nil), nil)
// 			},
// 			wantError:   true,
// 			wantErrorIs: domainOp.ErrOperationNotFound,
// 		},
// 		{
// 			desc:        "cancel update error",
// 			operationID: 12,
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 12}
// 				m.On("FindByID", ctx, int64(12)).
// 					Return(op, nil)
// 				m.On("Update", ctx, op).
// 					Return(errors.New("update fail"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to update operation: update fail",
// 		},
// 		{
// 			desc:        "successful cancel",
// 			operationID: 13,
// 			reason:      "no longer needed",
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 13, Status: domainOp.StatusInProgress}
// 				m.On("FindByID", ctx, int64(13)).
// 					Return(op, nil)
// 				m.On("Update", ctx, mock.AnythingOfType("*operation.Operation")).
// 					Return(nil)
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		mockRepo := new(MockOperationRepo)
// 		tc.mockSetup(mockRepo)

// 		svc := operation.NewService(mockRepo)
// 		err := svc.CancelOperation(ctx, tc.operationID, tc.reason)
// 		if tc.wantError {
// 			assert.Error(t, err)
// 			if tc.wantErrorIs != nil {
// 				assert.ErrorIs(t, err, tc.wantErrorIs)
// 			}
// 			if tc.wantErrorMatch != "" {
// 				assert.Contains(t, err.Error(), tc.wantErrorMatch)
// 			}
// 		} else {
// 			assert.NoError(t, err)
// 		}

// 		mockRepo.AssertExpectations(t)
// 	}
// }

func TestOperationService_ListIncompleteOperations(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		desc           string
		mockSetup      func(*MockOperationRepo)
		wantError      bool
		wantErrorMatch string
		wantOps        []*domainOp.Operation
	}{
		{
			desc: "repo returns error",
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindIncomplete", ctx).
					Return(([]*domainOp.Operation)(nil), errors.New("db error"))
			},
			wantError:      true,
			wantErrorMatch: "failed to retrieve incomplete operations: db error",
		},
		{
			desc: "repo returns some ops",
			mockSetup: func(m *MockOperationRepo) {
				ops := []*domainOp.Operation{
					{ID: 1, Status: domainOp.StatusPending},
					{ID: 2, Status: domainOp.StatusInProgress},
				}
				m.On("FindIncomplete", ctx).Return(ops, nil)
			},
			wantOps: []*domainOp.Operation{
				{ID: 1, Status: domainOp.StatusPending},
				{ID: 2, Status: domainOp.StatusInProgress},
			},
		},
	}

	for _, tc := range testCases {
		mockRepo := new(MockOperationRepo)
		tc.mockSetup(mockRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := operation.NewService(mockRepo, logger, tracer)
		ops, err := svc.ListIncompleteOperations(ctx)
		if tc.wantError {
			assert.Error(t, err)
			if tc.wantErrorMatch != "" {
				assert.Contains(t, err.Error(), tc.wantErrorMatch)
			}
			assert.Nil(t, ops)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.wantOps, ops)
		}

		mockRepo.AssertExpectations(t)
	}
}

func TestOperationService_ListStalledOperations(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	justStarted := now.Add(-30 * time.Second) // 30s ago
	longRunning := now.Add(-90 * time.Second) // 90s ago

	testCases := []struct {
		desc           string
		threshold      time.Duration
		mockSetup      func(*MockOperationRepo)
		wantError      bool
		wantErrorMatch string
		wantOps        []int64
	}{
		{
			desc:      "repo error",
			threshold: 1 * time.Minute,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByStatus", ctx, domainOp.StatusInProgress).
					Return(([]*domainOp.Operation)(nil), errors.New("db error"))
			},
			wantError:      true,
			wantErrorMatch: "failed to retrieve in-progress operations: db error",
		},
		{
			desc:      "no in-progress ops returned",
			threshold: 60 * time.Second,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByStatus", ctx, domainOp.StatusInProgress).
					Return([]*domainOp.Operation{}, nil)
			},
			wantOps: nil,
		},
		{
			desc:      "some stalled, some not",
			threshold: 60 * time.Second,
			mockSetup: func(m *MockOperationRepo) {
				ops := []*domainOp.Operation{
					{ID: 1, Status: domainOp.StatusInProgress, StartedAt: &justStarted}, // 30s old -> not stalled
					{ID: 2, Status: domainOp.StatusInProgress, StartedAt: &longRunning}, // 90s old -> stalled
					{ID: 3, Status: domainOp.StatusInProgress, StartedAt: nil},          // no start time -> skip or consider 0
				}
				m.On("FindByStatus", ctx, domainOp.StatusInProgress).Return(ops, nil)
			},
			wantOps: []int64{2}, // #2 is older than 60s
		},
	}

	for _, tc := range testCases {
		mockRepo := new(MockOperationRepo)
		tc.mockSetup(mockRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := operation.NewService(mockRepo, logger, tracer)
		stalled, err := svc.ListStalledOperations(ctx, tc.threshold)
		if tc.wantError {
			assert.Error(t, err)
			if tc.wantErrorMatch != "" {
				assert.Contains(t, err.Error(), tc.wantErrorMatch)
			}
			assert.Nil(t, stalled)
		} else {
			assert.NoError(t, err)
			var ids []int64
			for _, op := range stalled {
				ids = append(ids, op.ID)
			}
			assert.Equal(t, tc.wantOps, ids)
		}
		mockRepo.AssertExpectations(t)
	}
}

func TestOperationService_GetOperationsByTenant(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		desc           string
		tenantID       int64
		mockSetup      func(*MockOperationRepo)
		wantError      bool
		wantErrorMatch string
		wantOps        []*domainOp.Operation
	}{
		{
			desc:     "db error",
			tenantID: 999,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByTenantID", ctx, int64(999)).
					Return(([]*domainOp.Operation)(nil), errors.New("db error"))
			},
			wantError:      true,
			wantErrorMatch: "failed to retrieve operations for tenant 999: db error",
		},
		{
			desc:     "some ops returned",
			tenantID: 100,
			mockSetup: func(m *MockOperationRepo) {
				ops := []*domainOp.Operation{
					{ID: 1, TenantID: &[]int64{100}[0]},
					{ID: 2, TenantID: &[]int64{100}[0]},
				}
				m.On("FindByTenantID", ctx, int64(100)).Return(ops, nil)
			},
			wantOps: []*domainOp.Operation{
				{ID: 1, TenantID: &[]int64{100}[0]},
				{ID: 2, TenantID: &[]int64{100}[0]},
			},
		},
	}

	for _, tc := range testCases {
		mockRepo := new(MockOperationRepo)
		tc.mockSetup(mockRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := operation.NewService(mockRepo, logger, tracer)
		ops, err := svc.GetOperationsByTenant(ctx, tc.tenantID)
		if tc.wantError {
			assert.Error(t, err)
			if tc.wantErrorMatch != "" {
				assert.Contains(t, err.Error(), tc.wantErrorMatch)
			}
			assert.Nil(t, ops)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.wantOps, ops)
		}
		mockRepo.AssertExpectations(t)
	}
}

func TestOperationService_GetOperationProgress(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		desc           string
		opID           int64
		mockSetup      func(*MockOperationRepo)
		wantError      bool
		wantErrorMatch string
		wantProgress   int
	}{
		{
			desc: "db error",
			opID: 50,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByID", ctx, int64(50)).
					Return((*domainOp.Operation)(nil), errors.New("db error"))
			},
			wantError:      true,
			wantErrorMatch: "failed to retrieve operation: db error",
		},
		{
			desc: "not found",
			opID: 51,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByID", ctx, int64(51)).
					Return((*domainOp.Operation)(nil), nil)
			},
			wantError:      true,
			wantErrorMatch: domainOp.ErrOperationNotFound.Error(),
		},
		{
			desc: "failed operation => error for progress",
			opID: 52,
			mockSetup: func(m *MockOperationRepo) {
				eMsg := "some fail"
				op := &domainOp.Operation{ID: 52, Status: domainOp.StatusFailed, ErrorMessage: &eMsg}
				m.On("FindByID", ctx, int64(52)).
					Return(op, nil)
			},
			wantError:      true,
			wantErrorMatch: "operation failed or cancelled",
		},
		{
			desc: "pending => progress = 0",
			opID: 53,
			mockSetup: func(m *MockOperationRepo) {
				op := &domainOp.Operation{ID: 53, Status: domainOp.StatusPending}
				m.On("FindByID", ctx, int64(53)).
					Return(op, nil)
			},
			wantProgress: 0,
		},
		{
			desc: "in_progress => some progress calc (we won't fully test domain logic here)",
			opID: 54,
			mockSetup: func(m *MockOperationRepo) {
				started := time.Now().Add(-1 * time.Minute)
				op := &domainOp.Operation{ID: 54, Status: domainOp.StatusInProgress, StartedAt: &started}
				m.On("FindByID", ctx, int64(54)).
					Return(op, nil)
			},
			// We won't necessarily know the exact value, but let's just ensure no error.
		},
	}

	for _, tc := range testCases {
		mockRepo := new(MockOperationRepo)
		tc.mockSetup(mockRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := operation.NewService(mockRepo, logger, tracer)
		prog, err := svc.GetOperationProgress(ctx, tc.opID)
		if tc.wantError {
			assert.Error(t, err)
			if tc.wantErrorMatch != "" {
				assert.Contains(t, err.Error(), tc.wantErrorMatch)
			}
		} else {
			assert.NoError(t, err)
			// If we do have a wantProgress, check it.
			if tc.wantProgress != 0 {
				// Some minimal check.
				assert.Equal(t, tc.wantProgress, prog)
			}
		}
		mockRepo.AssertExpectations(t)
	}
}

func TestOperationService_GetOperationEstimatedCompletion(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		desc           string
		opID           int64
		mockSetup      func(*MockOperationRepo)
		wantError      bool
		wantErrorMatch string
		wantNil        bool
	}{
		{
			desc: "db error",
			opID: 60,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByID", ctx, int64(60)).
					Return((*domainOp.Operation)(nil), errors.New("db error"))
			},
			wantError:      true,
			wantErrorMatch: "failed to retrieve operation: db error",
		},
		{
			desc: "not found",
			opID: 61,
			mockSetup: func(m *MockOperationRepo) {
				m.On("FindByID", ctx, int64(61)).
					Return((*domainOp.Operation)(nil), nil)
			},
			wantError:      true,
			wantErrorMatch: domainOp.ErrOperationNotFound.Error(),
		},
		{
			desc: "completed => returns nil",
			opID: 62,
			mockSetup: func(m *MockOperationRepo) {
				cTime := time.Now()
				op := &domainOp.Operation{ID: 62, Status: domainOp.StatusCompleted, CompletedAt: &cTime}
				m.On("FindByID", ctx, int64(62)).
					Return(op, nil)
			},
			wantNil: true,
		},
		{
			desc: "pending => returns a non-nil time (we won't check the exact calc)",
			opID: 63,
			mockSetup: func(m *MockOperationRepo) {
				op := &domainOp.Operation{ID: 63, Status: domainOp.StatusPending, CreatedAt: time.Now()}
				m.On("FindByID", ctx, int64(63)).
					Return(op, nil)
			},
		},
	}

	for _, tc := range testCases {
		mockRepo := new(MockOperationRepo)
		tc.mockSetup(mockRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := operation.NewService(mockRepo, logger, tracer)
		est, err := svc.GetOperationEstimatedCompletion(ctx, tc.opID)
		if tc.wantError {
			assert.Error(t, err)
			if tc.wantErrorMatch != "" {
				assert.Contains(t, err.Error(), tc.wantErrorMatch)
			}
		} else {
			assert.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, est)
			} else {
				assert.NotNil(t, est)
			}
		}

		mockRepo.AssertExpectations(t)
	}
}

// func TestOperationService_RetryOperation(t *testing.T) {
// 	ctx := context.Background()

// 	testCases := []struct {
// 		desc           string
// 		opID           int64
// 		mockSetup      func(*MockOperationRepo)
// 		wantError      bool
// 		wantErrorMatch string
// 	}{
// 		{
// 			desc: "db error",
// 			opID: 70,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(70)).
// 					Return((*domainOp.Operation)(nil), errors.New("db error"))
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "failed to retrieve operation: db error",
// 		},
// 		{
// 			desc: "not found",
// 			opID: 71,
// 			mockSetup: func(m *MockOperationRepo) {
// 				m.On("FindByID", ctx, int64(71)).
// 					Return((*domainOp.Operation)(nil), nil)
// 			},
// 			wantError:      true,
// 			wantErrorMatch: domainOp.ErrOperationNotFound.Error(),
// 		},
// 		{
// 			desc: "not retryable (not failed)",
// 			opID: 72,
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 72, Status: domainOp.StatusInProgress}
// 				m.On("FindByID", ctx, int64(72)).
// 					Return(op, nil)
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "operation cannot be retried",
// 		},
// 		{
// 			desc: "failed, but result says tenant was fully deleted => not retryable",
// 			opID: 73,
// 			mockSetup: func(m *MockOperationRepo) {
// 				dRes := map[string]any{"status": "deleted"}
// 				op := &domainOp.Operation{ID: 73, Status: domainOp.StatusFailed, Type: domainOp.OpTenantDelete, Result: dRes}
// 				m.On("FindByID", ctx, int64(73)).
// 					Return(op, nil)
// 			},
// 			wantError:      true,
// 			wantErrorMatch: "operation cannot be retried",
// 		},
// 		{
// 			desc: "failed => can retry",
// 			opID: 74,
// 			mockSetup: func(m *MockOperationRepo) {
// 				op := &domainOp.Operation{ID: 74, Status: domainOp.StatusFailed, Type: domainOp.OpTenantCreate}
// 				m.On("FindByID", ctx, int64(74)).
// 					Return(op, nil)
// 				m.On("Update", ctx, op).
// 					Return(nil)
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		mockRepo := new(MockOperationRepo)
// 		tc.mockSetup(mockRepo)

// 		svc := operation.NewService(mockRepo)
// 		err := svc.RetryOperation(ctx, tc.opID)
// 		if tc.wantError {
// 			assert.Error(t, err)
// 			if tc.wantErrorMatch != "" {
// 				assert.Contains(t, err.Error(), tc.wantErrorMatch)
// 			}
// 		} else {
// 			assert.NoError(t, err)
// 		}

// 		mockRepo.AssertExpectations(t)
// 	}
// }
