package tenant_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/ahrav/hoglet-hub/internal/application/tenant"
	"github.com/ahrav/hoglet-hub/internal/application/workflow"
	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	tenantDomain "github.com/ahrav/hoglet-hub/internal/domain/tenant"
	"github.com/ahrav/hoglet-hub/pkg/common/logger"
)

// DATA RACE PREVENTION PATTERN
//
// This test file implements a specific pattern to prevent data races when testing
// the tenant service, which uses asynchronous workflows:
//
// The Problem:
// - The tenant service launches workflows that run in background goroutines
// - These workflows modify shared state like Operation objects
// - In tests, there's a race between these goroutines and test verification code
// - The race detector flags this as an error even though it's not an issue in production
//
// The Solution:
// 1. Factory Pattern: We use a WorkflowFactory interface to create workflows
// 2. MockWorkflowFactory: In tests, we inject a mock factory that returns controlled workflows
// 3. MockWorkflow.TestMode(): This configures mock workflows to complete synchronously
// 4. Synchronous Execution: The workflow completes in the same goroutine as the test
//
// This pattern preserves the asynchronous design in production while making
// workflow execution deterministic and synchronous during testing.

// TESTING THE WORKFLOW FACTORY PATTERN
//
// The WorkflowFactory pattern is used in our architecture to enable:
// - Separation of concerns between service orchestration and workflow implementation
// - Adaptability to different environments and tenant requirements
// - Extension points for different provisioning strategies based on regions or tiers
// - Dependency management and initialization isolation
//
// Our API design follows the Go principle of "accept interfaces, return concrete types":
// - We accept WorkflowFactory as an interface for flexibility
// - But we return only concrete data (IDs) to clients, not workflow interfaces
// - This decouples clients from our implementation details
//
// When testing this pattern, we face a challenge with asynchronous workflows:
// - In production: Workflows run asynchronously in separate goroutines
// - In tests: This asynchrony causes data races with test verification code
//
// Our testing approach uses the same architectural pattern but with controlled workflows:
// 1. We inject a MockWorkflowFactory that returns MockWorkflow instances
// 2. MockWorkflow.TestMode() configures workflows to complete synchronously
// 3. This allows us to test the service behavior without race conditions
//
// This testing strategy verifies the business logic while maintaining the
// architectural benefits of the factory pattern.

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

// MockWorkflow is a testify mock implementation of the Workflow interface.
//
// While our architectural design uses asynchronous workflows for production,
// this mock allows us to test the tenant service's interaction with workflows
// in a controlled manner. It preserves the interface contract while providing
// test-specific capabilities for controlling execution flow.
//
// The mock is useful for verifying:
// - The service correctly creates and passes parameters to workflows
// - Error handling behavior when workflow operations fail
// - Service behavior throughout the workflow lifecycle
type MockWorkflow struct {
	mock.Mock
	resultChan chan workflow.WorkflowResult
}

// Ensure MockWorkflow satisfies workflow.Workflow
var _ workflow.Workflow = (*MockWorkflow)(nil)

func NewMockWorkflow() *MockWorkflow {
	return &MockWorkflow{resultChan: make(chan workflow.WorkflowResult, 1)}
}

// TestMode configures the mock workflow for synchronous execution in tests.
//
// When testing the tenant service, we need to verify its behavior without
// dealing with asynchronous race conditions. This method:
//
// 1. Configures the Start() method to be called with any context
// 2. Makes Start() immediately provide a successful result
// 3. Ensures workflow completion actions happen in the test goroutine
//
// This approach maintains the architectural integrity of the workflow
// factory pattern while making it practical to test within the unit test framework.
func (m *MockWorkflow) TestMode() {
	// When Start is called in test mode, immediately send a successful result
	m.On("Start", mock.Anything).Run(func(args mock.Arguments) {
		// Immediately send a result to simulate completion
		m.resultChan <- workflow.WorkflowResult{
			Success:     true,
			CompletedAt: time.Now(),
			Result:      map[string]interface{}{},
		}
	})
}

func (m *MockWorkflow) Start(ctx context.Context) {
	m.Called(ctx)
	// In a test without TestMode, we could push a result to the channel here to simulate
	// immediate completion, but for most tests we want to control this behavior with TestMode
}

func (m *MockWorkflow) ResultChan() <-chan workflow.WorkflowResult { return m.resultChan }

// Helper method to let the test inject results.
func (m *MockWorkflow) SendResult(result workflow.WorkflowResult) { m.resultChan <- result }

// MockWorkflowFactory is a testify mock for WorkflowFactory.
//
// This mock factory is a key component in our testing strategy:
// 1. It allows us to control workflow creation in tests
// 2. We can return MockWorkflow instances configured with TestMode()
// 3. This gives us full control over the asynchronous behavior
//
// By using this factory in tests, we transform the asynchronous workflow
// into a synchronous one, eliminating race conditions between the service
// code and test verification.

// MockWorkflowFactory is a test implementation of the WorkflowFactory interface.
//
// This mock lets us verify that:
// - The tenant service correctly requests appropriate workflow types
// - Workflows are created with proper parameters for each tenant operation
// - The service handles the workflow creation and execution pattern correctly
//
// By injecting this factory in tests, we maintain the architectural separation
// between service orchestration and workflow implementation while keeping
// tests deterministic and reliable.
type MockWorkflowFactory struct {
	mock.Mock
}

func (m *MockWorkflowFactory) NewTenantCreationWorkflow(
	t *tenantDomain.Tenant,
	tenantID int64,
	op *operation.Operation,
) workflow.Workflow {
	args := m.Called(t, tenantID, op)
	return args.Get(0).(workflow.Workflow)
}

func (m *MockWorkflowFactory) NewTenantDeletionWorkflow(
	t *tenantDomain.Tenant,
	tenantID int64,
	op *operation.Operation,
) workflow.Workflow {
	args := m.Called(t, tenantID, op)
	return args.Get(0).(workflow.Workflow)
}

type MockProvisioningMetrics struct{ mock.Mock }

func (m *MockProvisioningMetrics) IncProvisioningSuccess(ctx context.Context, tenantTier string, region string) {
	m.Called(ctx, tenantTier, region)
}

func (m *MockProvisioningMetrics) IncProvisioningFailure(ctx context.Context, tenantTier string, region string, reason string) {
	m.Called(ctx, tenantTier, region, reason)
}

func (m *MockProvisioningMetrics) ObserveProvisioningDuration(
	ctx context.Context,
	tenantTier string,
	region string,
	duration time.Duration,
) {
	m.Called(ctx, tenantTier, region, duration)
}

func (m *MockProvisioningMetrics) ObserveProvisioningStageDuration(
	ctx context.Context,
	stage string,
	tenantTier string,
	region string,
	duration time.Duration,
) {
	m.Called(ctx, stage, tenantTier, region, duration)
}

func (m *MockProvisioningMetrics) IncTenantDeletionSuccess(ctx context.Context, tenantTier string, region string) {
	m.Called(ctx, tenantTier, region)
}

func (m *MockProvisioningMetrics) IncTenantDeletionFailure(ctx context.Context, tenantTier string, region string, reason string) {
	m.Called(ctx, tenantTier, region, reason)
}

func (m *MockProvisioningMetrics) ObserveTenantDeletionDuration(
	ctx context.Context,
	tenantTier string,
	region string,
	duration time.Duration,
) {
	m.Called(ctx, tenantTier, region, duration)
}

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
		t.Run(tc.desc, func(t *testing.T) {
			mockTenantRepo := new(MockTenantRepo)
			mockOperationRepo := new(MockOperationRepo)
			mockWorkflow := NewMockWorkflow()
			mockWorkflowFactory := new(MockWorkflowFactory)

			// Set workflow expectations ONLY for successful cases.
			if !tc.expectError {
				// Set up the workflow to complete immediately to avoid race conditions.
				mockWorkflow.TestMode()

				// Set up the factory to return our controlled workflow.
				mockWorkflowFactory.On("NewTenantCreationWorkflow",
					mock.AnythingOfType("*tenant.Tenant"),
					mock.AnythingOfType("int64"),
					mock.AnythingOfType("*operation.Operation")).
					Return(mockWorkflow)
			}

			tc.mockTenantRepoFn(mockTenantRepo)
			tc.mockOperationRepoFn(mockOperationRepo)

			logger := logger.Noop()
			tracer := noop.NewTracerProvider().Tracer("test")

			// Use the service constructor that accepts a workflow factory.
			svc := tenant.NewServiceWithWorkflowFactory(
				mockTenantRepo,
				mockOperationRepo,
				mockWorkflowFactory,
				logger,
				tracer,
				new(MockProvisioningMetrics),
			)

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
			mockWorkflowFactory.AssertExpectations(t)
			mockWorkflow.AssertExpectations(t)
		})
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
		t.Run(tc.desc, func(t *testing.T) {
			mockTenantRepo := new(MockTenantRepo)
			mockOperationRepo := new(MockOperationRepo)
			mockWorkflow := NewMockWorkflow()
			mockWorkflowFactory := new(MockWorkflowFactory)

			// Set workflow expectations ONLY for successful cases.
			if !tc.expectError {
				// Set up the workflow to complete immediately to avoid race conditions.
				mockWorkflow.TestMode()

				// Set up the factory to return our controlled workflow.
				mockWorkflowFactory.On("NewTenantDeletionWorkflow",
					mock.AnythingOfType("*tenant.Tenant"),
					mock.AnythingOfType("int64"),
					mock.AnythingOfType("*operation.Operation")).
					Return(mockWorkflow)
			}

			tc.mockTenantRepoFn(mockTenantRepo)
			tc.mockOperationRepoFn(mockOperationRepo)

			logger := logger.Noop()
			tracer := noop.NewTracerProvider().Tracer("test")

			// Use the service constructor that accepts a workflow factory.
			svc := tenant.NewServiceWithWorkflowFactory(
				mockTenantRepo,
				mockOperationRepo,
				mockWorkflowFactory,
				logger,
				tracer,
				new(MockProvisioningMetrics),
			)

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
			mockWorkflowFactory.AssertExpectations(t)
			mockWorkflow.AssertExpectations(t)
		})
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
		svc := tenant.NewService(mockTenantRepo, mockOperationRepo, logger, tracer, new(MockProvisioningMetrics))
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
