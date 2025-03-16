package tenant_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/ahrav/hoglet-hub/internal/application/tenant"
	"github.com/ahrav/hoglet-hub/internal/application/workflow"
	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	tenantDomain "github.com/ahrav/hoglet-hub/internal/domain/tenant"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
)

// MockTenantRepo is a testify mock for tenant.Repository.
type MockTenantRepo struct{ mock.Mock }

func (m *MockTenantRepo) Create(ctx context.Context, t *tenantDomain.Tenant) (int64, error) {
	args := m.Called(ctx, t)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTenantRepo) Update(ctx context.Context, t *tenantDomain.Tenant) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTenantRepo) FindByName(ctx context.Context, name string) (*tenantDomain.Tenant, error) {
	args := m.Called(ctx, name)
	tenant, _ := args.Get(0).(*tenantDomain.Tenant)
	return tenant, args.Error(1)
}

func (m *MockTenantRepo) FindByID(ctx context.Context, id int64) (*tenantDomain.Tenant, error) {
	args := m.Called(ctx, id)
	tenant, _ := args.Get(0).(*tenantDomain.Tenant)
	return tenant, args.Error(1)
}

func (m *MockTenantRepo) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockOperationRepo is a testify mock for operation.Repository.
type MockOperationRepo struct{ mock.Mock }

func (m *MockOperationRepo) Create(ctx context.Context, op *operation.Operation) (int64, error) {
	args := m.Called(ctx, op)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOperationRepo) Update(ctx context.Context, op *operation.Operation) error {
	args := m.Called(ctx, op)
	return args.Error(0)
}

func (m *MockOperationRepo) FindByID(ctx context.Context, id int64) (*operation.Operation, error) {
	args := m.Called(ctx, id)
	op, _ := args.Get(0).(*operation.Operation)
	return op, args.Error(1)
}

func (m *MockOperationRepo) FindByTenantID(ctx context.Context, tenantID int64) ([]*operation.Operation, error) {
	args := m.Called(ctx, tenantID)
	ops, _ := args.Get(0).([]*operation.Operation)
	return ops, args.Error(1)
}

func (m *MockOperationRepo) FindByStatus(ctx context.Context, status operation.Status) ([]*operation.Operation, error) {
	args := m.Called(ctx, status)
	ops, _ := args.Get(0).([]*operation.Operation)
	return ops, args.Error(1)
}

func (m *MockOperationRepo) FindIncomplete(ctx context.Context) ([]*operation.Operation, error) {
	args := m.Called(ctx)
	ops, _ := args.Get(0).([]*operation.Operation)
	return ops, args.Error(1)
}

// MockWorkflow is a testify mock for workflow.Workflow.
type MockWorkflow struct {
	mock.Mock
	resultChan chan workflow.WorkflowResult
}

// Ensure MockWorkflow satisfies workflow.Workflow
var _ workflow.Workflow = (*MockWorkflow)(nil)

func NewMockWorkflow() *MockWorkflow {
	return &MockWorkflow{resultChan: make(chan workflow.WorkflowResult, 1)}
}

func (m *MockWorkflow) Start(ctx context.Context) {
	m.Called(ctx)
	// TODO: Use synctest.
	// In a real test, we can optionally simulate asynchronous completion
	// For example, we can push a result right away to the channel to test completion logic:
	// go func() {
	//     time.Sleep(10 * time.Millisecond)
	//     m.resultChan <- workflow.WorkflowResult{Success: true, CompletedAt: time.Now()}
	// }()
}

func (m *MockWorkflow) ResultChan() <-chan workflow.WorkflowResult { return m.resultChan }

// Helper method to let the test inject results.
func (m *MockWorkflow) SendResult(result workflow.WorkflowResult) { m.resultChan <- result }

func TestServiceCreate(t *testing.T) {
	ctx := context.Background()
	validParams := tenant.CreateParams{
		Name:   "my-tenant",
		Region: tenantDomain.RegionEU1,
		Tier:   tenantDomain.TierPro,
	}

	testCases := []struct {
		desc                string
		mockTenantRepoFn    func(*MockTenantRepo)
		mockOperationRepoFn func(*MockOperationRepo)
		inputParams         tenant.CreateParams
		expectError         bool
		expectErrorContains string
		expectTenantID      int64
		expectOperationID   int64
		expectErrIs         error
	}{
		{
			desc: "error on FindByName",
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByName", mock.Anything, "my-tenant").
					Return((*tenantDomain.Tenant)(nil), errors.New("DB error"))
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {},
			inputParams:         validParams,
			expectError:         true,
			expectErrorContains: "DB error",
		},
		{
			desc: "tenant already exists",
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByName", mock.Anything, "my-tenant").
					Return(&tenantDomain.Tenant{ID: 123, Name: "my-tenant"}, nil)
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {},
			inputParams:         validParams,
			expectError:         true,
			expectErrIs:         tenantDomain.ErrTenantAlreadyExists,
		},
		{
			desc: "error on tenantRepo.Create",
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByName", mock.Anything, "my-tenant").
					Return((*tenantDomain.Tenant)(nil), tenantDomain.ErrTenantNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*tenant.Tenant")).
					Return(int64(0), errors.New("create error"))
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {},
			inputParams:         validParams,
			expectError:         true,
			expectErrorContains: "failed to persist tenant",
		},
		{
			desc: "error on operationRepo.Create",
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByName", mock.Anything, "my-tenant").
					Return((*tenantDomain.Tenant)(nil), tenantDomain.ErrTenantNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*tenant.Tenant")).
					Return(int64(123), nil)
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*operation.Operation")).
					Return(int64(0), errors.New("op creation error"))
			},
			inputParams:         validParams,
			expectError:         true,
			expectErrorContains: "failed to persist operation",
		},
		{
			desc: "successful create",
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByName", mock.Anything, "my-tenant").
					Return((*tenantDomain.Tenant)(nil), tenantDomain.ErrTenantNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*tenant.Tenant")).
					Return(int64(123), nil)
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*operation.Operation")).
					Return(int64(456), nil)
			},
			inputParams:       validParams,
			expectError:       false,
			expectTenantID:    123,
			expectOperationID: 456,
		},
	}

	for _, tc := range testCases {
		mockTenantRepo := new(MockTenantRepo)
		mockOperationRepo := new(MockOperationRepo)

		tc.mockTenantRepoFn(mockTenantRepo)
		tc.mockOperationRepoFn(mockOperationRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := tenant.NewService(mockTenantRepo, mockOperationRepo, logger, tracer)
		res, err := svc.Create(ctx, tc.inputParams)

		if tc.expectError {
			assert.Error(t, err, "expected an error but got none")

			if tc.expectErrorContains != "" {
				assert.Contains(t, err.Error(), tc.expectErrorContains)
			}
			if tc.expectErrIs != nil {
				assert.ErrorIs(t, err, tc.expectErrIs)
			}
			assert.Nil(t, res)
		} else {
			assert.NoError(t, err, "didn't expect an error, but got one")
			assert.NotNil(t, res)
			assert.EqualValues(t, tc.expectTenantID, res.TenantID)
			assert.EqualValues(t, tc.expectOperationID, res.OperationID)
		}

		mockTenantRepo.AssertExpectations(t)
		mockOperationRepo.AssertExpectations(t)
	}
}

func TestServiceDelete(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		desc                string
		tenantID            int64
		mockTenantRepoFn    func(*MockTenantRepo)
		mockOperationRepoFn func(*MockOperationRepo)
		expectError         bool
		expectErrorContains string
		expectErrIs         error
		expectOperationID   int64
	}{
		{
			desc:     "error finding tenant",
			tenantID: 999,
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByID", mock.Anything, int64(999)).
					Return((*tenantDomain.Tenant)(nil), errors.New("DB error"))
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {},
			expectError:         true,
			expectErrorContains: "error finding tenant",
		},
		{
			desc:     "tenant not found",
			tenantID: 999,
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByID", mock.Anything, int64(999)).
					Return((*tenantDomain.Tenant)(nil), nil)
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {},
			expectError:         true,
			expectErrIs:         tenantDomain.ErrTenantNotFound,
		},
		{
			desc:     "error creating operation",
			tenantID: 123,
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByID", mock.Anything, int64(123)).
					Return(&tenantDomain.Tenant{ID: 123}, nil)
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*operation.Operation")).
					Return(int64(0), errors.New("op creation error"))
			},
			expectError:         true,
			expectErrorContains: "failed to persist operation",
		},
		{
			desc:     "successful delete",
			tenantID: 123,
			mockTenantRepoFn: func(m *MockTenantRepo) {
				m.On("FindByID", mock.Anything, int64(123)).
					Return(&tenantDomain.Tenant{ID: 123, Name: "my-tenant"}, nil)
			},
			mockOperationRepoFn: func(m *MockOperationRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*operation.Operation")).
					Return(int64(456), nil)
			},
			expectError:       false,
			expectOperationID: 456,
		},
	}

	for _, tc := range testCases {
		mockTenantRepo := new(MockTenantRepo)
		mockOperationRepo := new(MockOperationRepo)

		tc.mockTenantRepoFn(mockTenantRepo)
		tc.mockOperationRepoFn(mockOperationRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := tenant.NewService(mockTenantRepo, mockOperationRepo, logger, tracer)
		res, err := svc.Delete(ctx, tc.tenantID)
		if tc.expectError {
			assert.Error(t, err)
			if tc.expectErrorContains != "" {
				assert.Contains(t, err.Error(), tc.expectErrorContains)
			}
			if tc.expectErrIs != nil {
				assert.ErrorIs(t, err, tc.expectErrIs)
			}
			assert.Nil(t, res)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, res)
			assert.EqualValues(t, tc.expectOperationID, res.OperationID)
		}

		mockTenantRepo.AssertExpectations(t)
		mockOperationRepo.AssertExpectations(t)
	}
}

func TestServiceGetOperationStatus(t *testing.T) {
	ctx := context.Background()
	tenantID := int64(55)

	testCases := []struct {
		desc                string
		operationID         int64
		mockOperationRepoFn func(*MockOperationRepo)
		expectError         bool
		expectErrIs         error
		expectOp            *operation.Operation
	}{
		{
			desc:        "operation not found",
			operationID: 999,
			mockOperationRepoFn: func(m *MockOperationRepo) {
				m.On("FindByID", mock.Anything, int64(999)).
					Return((*operation.Operation)(nil), nil)
			},
			expectError: true,
			expectErrIs: operation.ErrOperationNotFound,
		},
		{
			desc:        "operation found",
			operationID: 123,
			mockOperationRepoFn: func(m *MockOperationRepo) {
				op := &operation.Operation{
					ID:       123,
					TenantID: &tenantID,
				}
				m.On("FindByID", mock.Anything, int64(123)).
					Return(op, nil)
			},
			expectError: false,
			expectOp: &operation.Operation{
				ID:       123,
				TenantID: &tenantID,
			},
		},
	}

	for _, tc := range testCases {
		mockTenantRepo := new(MockTenantRepo)
		mockOperationRepo := new(MockOperationRepo)

		tc.mockOperationRepoFn(mockOperationRepo)

		logger := logger.Noop()
		tracer := noop.NewTracerProvider().Tracer("test")
		svc := tenant.NewService(mockTenantRepo, mockOperationRepo, logger, tracer)
		op, err := svc.GetOperationStatus(ctx, tc.operationID)
		if tc.expectError {
			assert.Error(t, err)
			if tc.expectErrIs != nil {
				assert.ErrorIs(t, err, tc.expectErrIs)
			}
			assert.Nil(t, op)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, op)
			assert.Equal(t, tc.expectOp, op)
		}

		mockTenantRepo.AssertExpectations(t)
		mockOperationRepo.AssertExpectations(t)
	}
}
