package workflow_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ahrav/hoglet-hub/internal/application/workflow"
)

func TestNewBaseWorkflow(t *testing.T) {
	steps := []workflow.Step{
		{
			Name:        "step1",
			Description: "First step",
			Execute:     func(ctx context.Context) error { return nil },
		},
	}

	wf := workflow.NewBaseWorkflow(steps)
	assert.NotNil(t, wf)
	assert.NotNil(t, wf.ResultChan())
}

func TestNewBaseWorkflowWithTimeout(t *testing.T) {
	steps := []workflow.Step{
		{
			Name:        "step1",
			Description: "First step",
			Execute:     func(ctx context.Context) error { return nil },
		},
	}

	customTimeout := 10 * time.Second
	wf := workflow.NewBaseWorkflowWithTimeout(steps, customTimeout)
	assert.NotNil(t, wf)

	// Test with invalid timeout (should use default).
	wf = workflow.NewBaseWorkflowWithTimeout(steps, -1)
	assert.NotNil(t, wf)
}

func TestWorkflow_Start_Success(t *testing.T) {
	var executionOrder []string

	steps := []workflow.Step{
		{
			Name:        "step1",
			Description: "First step",
			Execute: func(ctx context.Context) error {
				executionOrder = append(executionOrder, "step1")
				return nil
			},
		},
		{
			Name:        "step2",
			Description: "Second step",
			Execute: func(ctx context.Context) error {
				executionOrder = append(executionOrder, "step2")
				return nil
			},
		},
	}

	wf := workflow.NewBaseWorkflow(steps)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	wf.Start(ctx)

	result := <-wf.ResultChan()
	assert.True(t, result.Success)
	assert.Nil(t, result.Error)
	assert.Len(t, result.StepResults, 2)

	for _, stepResult := range result.StepResults {
		assert.True(t, stepResult.Success)
		assert.Nil(t, stepResult.Error)
		assert.Greater(t, stepResult.Duration, time.Duration(0))
	}

	assert.Len(t, executionOrder, 2)
	assert.Equal(t, "step1", executionOrder[0])
	assert.Equal(t, "step2", executionOrder[1])
}

func TestWorkflow_Start_Error(t *testing.T) {
	expectedErr := errors.New("test error")
	executionOrder := []string{}

	steps := []workflow.Step{
		{
			Name:        "step1",
			Description: "First step",
			Execute: func(ctx context.Context) error {
				executionOrder = append(executionOrder, "step1")
				return nil
			},
		},
		{
			Name:        "step2",
			Description: "Second step",
			Execute: func(ctx context.Context) error {
				executionOrder = append(executionOrder, "step2")
				return expectedErr
			},
		},
		{
			Name:        "step3",
			Description: "Third step",
			Execute: func(ctx context.Context) error {
				executionOrder = append(executionOrder, "step3")
				return nil
			},
		},
	}

	wf := workflow.NewBaseWorkflow(steps)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	wf.Start(ctx)

	result := <-wf.ResultChan()

	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
	assert.Len(t, result.StepResults, 2)

	// Verify step1 succeeded and step2 failed.
	assert.True(t, result.StepResults[0].Success)
	assert.Nil(t, result.StepResults[0].Error)

	assert.False(t, result.StepResults[1].Success)
	assert.Equal(t, expectedErr, result.StepResults[1].Error)

	assert.Len(t, executionOrder, 2)
	assert.Equal(t, "step1", executionOrder[0])
	assert.Equal(t, "step2", executionOrder[1])
}

func TestWorkflow_Context_Cancellation(t *testing.T) {
	stepExecuted := false
	waitCh := make(chan struct{})

	steps := []workflow.Step{
		{
			Name:        "slow-step",
			Description: "A step that checks context cancellation",
			Execute: func(ctx context.Context) error {
				// Signal that we've started execution
				close(waitCh)

				select {
				case <-time.After(20 * time.Millisecond):
					stepExecuted = true
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		},
	}

	wf := workflow.NewBaseWorkflow(steps)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	wf.Start(ctx)

	// Cancel the context right away.
	cancel()

	result := <-wf.ResultChan()

	assert.False(t, result.Success)
	assert.Error(t, result.Error)
	assert.False(t, stepExecuted, "Step should not have completed execution")
	assert.Contains(t, result.Error.Error(), "context canceled")
}

func TestWorkflow_CustomTimeout(t *testing.T) {
	synctest.Run(func() {
		// Create a workflow with a very short timeout.
		shortTimeout := 5 * time.Millisecond

		stepExecuted := false
		steps := []workflow.Step{
			{
				Name:        "long-running-step",
				Description: "A step that runs longer than the timeout",
				Execute: func(ctx context.Context) error {
					select {
					case <-time.After(20 * time.Millisecond):
						stepExecuted = true
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				},
			},
		}

		wf := workflow.NewBaseWorkflowWithTimeout(steps, shortTimeout)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond) // longer than the timeout
		defer cancel()
		wf.Start(ctx)

		result := <-wf.ResultChan()

		assert.False(t, result.Success)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "context deadline exceeded")
		assert.False(t, stepExecuted, "Step should not have completed execution")
	})
}

func TestWorkflow_StepHangs_WithTimeout(t *testing.T) {
	synctest.Run(func() {
		steps := []workflow.Step{
			{
				Name:        "hanging-step",
				Description: "A step that hangs indefinitely",
				Execute: func(ctx context.Context) error {
					// This step deliberately ignores the context and would hang forever
					// if not for the workflow timeout
					time.Sleep(10 * time.Second)
					return nil
				},
			},
		}

		// Create a workflow with a specific timeout.
		timeout := 50 * time.Millisecond
		wf := workflow.NewBaseWorkflowWithTimeout(steps, timeout)
		startTime := time.Now()
		wf.Start(context.Background())

		result := <-wf.ResultChan()

		// Verify the workflow terminated because of timeout.
		assert.False(t, result.Success)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "context deadline exceeded")

		// Verify that the workflow terminated close to the specified timeout.
		duration := time.Since(startTime)
		assert.True(t, duration >= timeout)
		assert.True(t, duration < timeout+100*time.Millisecond)
	})
}
