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
type Workflow interface {
	Start(ctx context.Context)
	ResultChan() <-chan WorkflowResult
}

// BaseWorkflow provides foundational workflow functionality that can be embedded
// in specific workflow implementations.
type BaseWorkflow struct {
	steps      []Step
	resultChan chan WorkflowResult
}

// NewBaseWorkflow creates a new base workflow with the provided execution steps.
func NewBaseWorkflow(steps []Step) *BaseWorkflow {
	return &BaseWorkflow{
		steps:      steps,
		resultChan: make(chan WorkflowResult, 1),
	}
}

// ResultChan returns the channel that will receive the workflow execution result.
func (w *BaseWorkflow) ResultChan() <-chan WorkflowResult {
	return w.resultChan
}

// ExecuteSteps runs all workflow steps in sequence and returns a consolidated result.
// It stops execution on the first step failure unless the workflow defines different behavior.
func (w *BaseWorkflow) ExecuteSteps(ctx context.Context) WorkflowResult {
	result := WorkflowResult{
		Success:     true,
		StepResults: make([]StepResult, 0, len(w.steps)),
		Result:      make(map[string]any),
	}

	// Execute each step.
	for _, step := range w.steps {
		stepResult := StepResult{
			StepName:  step.Name,
			StartedAt: time.Now(),
		}

		err := step.Execute(ctx)

		// Record result.
		stepResult.CompletedAt = time.Now()
		stepResult.Duration = stepResult.CompletedAt.Sub(stepResult.StartedAt)

		if err != nil {
			stepResult.Success = false
			stepResult.Error = err
			result.Success = false
			result.Error = fmt.Errorf("step %s failed: %w", step.Name, err)
			result.StepResults = append(result.StepResults, stepResult)
			break
		}

		stepResult.Success = true
		result.StepResults = append(result.StepResults, stepResult)
	}

	result.CompletedAt = time.Now()

	return result
}
