package workflow

import (
	"context"
	"fmt"
	"time"
)

// Step represents a single executable unit in a workflow.
// Each step has a name, description, and an execution function that will be called
// during workflow execution.
type Step struct {
	Name        string
	Description string
	Execute     func(ctx context.Context) error
}

// WorkflowResult contains the consolidated outcome of a workflow execution.
// It includes success status, timing information, any errors encountered,
// individual step results, and custom result data.
type WorkflowResult struct {
	Success     bool
	CompletedAt time.Time
	Error       error
	StepResults []StepResult
	Result      map[string]any
}

// StepResult tracks the execution result of an individual workflow step.
// It captures performance metrics and error information for reporting and debugging.
type StepResult struct {
	StepName    string
	Success     bool
	Error       error
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
}

// Workflow defines the common interface for all workflow implementations.
// Workflows are executed asynchronously and deliver results through a channel.
//
// Usage example:
//
//	// Create steps for the workflow
//	steps := []workflow.Step{
//		{
//			Name:        "Validate Input",
//			Description: "Ensures the input data is valid",
//			Execute: func(ctx context.Context) error {
//				// Validation logic here
//				return nil
//			},
//		},
//		{
//			Name:        "Process Data",
//			Description: "Processes the validated data",
//			Execute: func(ctx context.Context) error {
//				// Processing logic here
//				return nil
//			},
//		},
//	}
//
//	// Create and start the workflow
//	wf := workflow.NewBaseWorkflow(steps)
//	wf.Start(ctx)
//
//	// Wait for and process the result
//	// Note: All workflow outcomes (success, failure, cancellation, timeout)
//	// are delivered through the result channel
//	result := <-wf.ResultChan()
//	if !result.Success {
//		log.Printf("Workflow failed: %v", result.Error)
//		// Handle failure (could be a regular error or context cancellation)
//		return
//	}
//
//	log.Printf("Workflow completed successfully in %v",
//		result.CompletedAt.Sub(result.StepResults[0].StartedAt))
//	// Process successful result
type Workflow interface {
	Start(ctx context.Context)
	ResultChan() <-chan WorkflowResult
}

// BaseWorkflow provides foundational workflow functionality that can be embedded
// in specific workflow implementations.
type BaseWorkflow struct {
	steps      []Step
	resultChan chan WorkflowResult
	timeout    time.Duration // Default timeout for workflow execution
}

// DefaultTimeout is the default timeout used if none is specified.
const DefaultTimeout = 5 * time.Minute

// NewBaseWorkflow creates a new base workflow with the provided execution steps.
func NewBaseWorkflow(steps []Step) *BaseWorkflow {
	return NewBaseWorkflowWithTimeout(steps, DefaultTimeout)
}

// NewBaseWorkflowWithTimeout creates a new base workflow with the provided execution steps
// and a custom timeout.
func NewBaseWorkflowWithTimeout(steps []Step, timeout time.Duration) *BaseWorkflow {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &BaseWorkflow{
		steps:      steps,
		resultChan: make(chan WorkflowResult, 1),
		timeout:    timeout,
	}
}

// ResultChan returns the channel that will receive the workflow execution result.
// This channel will always receive exactly one WorkflowResult, regardless of whether
// the workflow succeeds, fails, times out, or is cancelled. The Success field and
// Error field of the WorkflowResult indicate the outcome.
func (w *BaseWorkflow) ResultChan() <-chan WorkflowResult { return w.resultChan }

// Start implements the Workflow interface by executing steps asynchronously
// and sending the result to the result channel. All possible outcomes
// (success, error, timeout, cancellation) are delivered through the result channel
// as a WorkflowResult, with appropriate Success and Error fields set.
func (w *BaseWorkflow) Start(ctx context.Context) {
	// Create a derived context with the workflow timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, w.timeout)

	go func() {
		defer cancel() // Ensure context is canceled when done
		result := w.ExecuteSteps(timeoutCtx)
		w.resultChan <- result
	}()
}

// ExecuteSteps runs all workflow steps in sequence and returns a consolidated result.
// It stops execution on the first step failure unless the workflow defines different behavior.
// It also handles context cancellation gracefully by including it in the returned result.
func (w *BaseWorkflow) ExecuteSteps(ctx context.Context) WorkflowResult {
	result := WorkflowResult{
		Success:     true,
		StepResults: make([]StepResult, 0, len(w.steps)),
		Result:      make(map[string]any),
	}

	if ctx.Err() != nil {
		result.Success = false
		result.Error = fmt.Errorf("workflow aborted: %w", ctx.Err())
		result.CompletedAt = time.Now()
		return result
	}

	for _, step := range w.steps {
		stepResult := StepResult{
			StepName:  step.Name,
			StartedAt: time.Now(),
		}

		// We run this in a separate goroutine to avoid blocking the main workflow,
		// and providing us a way to check for context cancellation.
		// This is necessary because we do not control the execution of the steps,
		// and they may take an arbitrary amount of time to complete.
		resultChan := make(chan error, 1)
		go func(s Step) {
			resultChan <- s.Execute(ctx)
		}(step)

		var err error
		select {
		case err = <-resultChan:
			// Step completed.
		case <-ctx.Done():
			// Context canceled - we acknowledge it but don't wait for the step.
			// TODO: Maybe consider giving the step a chance to finish?
			err = ctx.Err()
		}

		stepResult.CompletedAt = time.Now()
		stepResult.Duration = stepResult.CompletedAt.Sub(stepResult.StartedAt)

		if err != nil {
			stepResult.Success = false
			stepResult.Error = err
			result.Success = false
			result.Error = fmt.Errorf("step %s: %w", step.Name, err)
			result.StepResults = append(result.StepResults, stepResult)
			break
		}

		stepResult.Success = true
		result.StepResults = append(result.StepResults, stepResult)
	}
	result.CompletedAt = time.Now()

	return result
}
