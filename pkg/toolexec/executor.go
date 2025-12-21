// Package toolexec provides a modular, extensible tool executor architecture.
// This file implements the Executor for tool execution with context support,
// timeouts, and proper error handling.
package toolexec

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

// Executor defines the interface for executing tools.
// It provides synchronous, asynchronous, and batch execution methods.
// All methods accept context for cancellation and timeout support.
type Executor interface {
	// Execute runs a tool synchronously with the given input.
	// It blocks until the tool completes or the context is cancelled.
	// Returns the result and any error that occurred.
	Execute(ctx context.Context, toolName string, input *Input) (*Output, error)

	// ExecuteAsync runs a tool asynchronously and returns a channel for the result.
	// The result channel will receive exactly one Result and then close.
	// The caller should read from the channel to get the result.
	ExecuteAsync(ctx context.Context, toolName string, input *Input) <-chan *Result

	// ExecuteMany runs multiple tools concurrently and returns all results.
	// Execution uses fail-fast behavior: the first error cancels remaining executions.
	// Partial results are returned even on error.
	ExecuteMany(ctx context.Context, executions []ToolExecution) ([]*Result, error)
}

// ToolExecution represents a single tool execution request for batch operations.
type ToolExecution struct {
	// ToolName is the name of the tool to execute.
	ToolName string

	// Input is the input data for the tool.
	Input *Input
}

// executorConfig holds the configuration for an executor.
// It is populated by functional options during construction.
type executorConfig struct {
	// timeout is the default timeout for tool execution.
	// Zero means no timeout (rely on context).
	timeout time.Duration

	// maxConcurrent is the maximum number of concurrent tool executions.
	// Zero or negative means unlimited.
	maxConcurrent int

	// recoverPanics determines whether to recover from panics in tool execution.
	// When true, panics are converted to PanicError.
	recoverPanics bool
}

// defaultConfig returns the default executor configuration.
func defaultConfig() *executorConfig {
	return &executorConfig{
		timeout:       30 * time.Second, // Default 30 second timeout per spec
		maxConcurrent: 1,                // Conservative default for safety
		recoverPanics: true,             // Recover panics by default for stability
	}
}

// executor is the default implementation of the Executor interface.
// It uses a Registry to look up tools and executes them with proper
// context handling, timeout enforcement, and panic recovery.
type executor struct {
	registry Registry
	config   *executorConfig
}

// NewExecutor creates a new Executor with the given registry.
// If registry is nil, the default global registry is used.
// Additional configuration can be provided via ExecutorOption functions
// (which will be added in a future subtask).
func NewExecutor(registry Registry) *executor {
	if registry == nil {
		registry = DefaultRegistry()
	}

	return &executor{
		registry: registry,
		config:   defaultConfig(),
	}
}

// Execute runs a tool synchronously with the given input.
// It performs the following steps:
//  1. Look up the tool in the registry
//  2. Apply timeout if configured
//  3. Check context before execution
//  4. Execute the tool with panic recovery
//  5. Return the output or error
//
// The context is used for cancellation and can have a timeout applied.
// If the executor has a default timeout configured and the context has no
// deadline, a timeout will be applied.
func (e *executor) Execute(ctx context.Context, toolName string, input *Input) (*Output, error) {
	// Step 1: Look up the tool in the registry
	tool, err := e.registry.Get(toolName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool '%s': %w", toolName, err)
	}

	// Step 2: Apply timeout if configured and context has no deadline
	if e.config.timeout > 0 {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, e.config.timeout)
			defer cancel()
		}
	}

	// Step 3: Check context before execution
	select {
	case <-ctx.Done():
		return nil, e.wrapContextError(ctx, toolName)
	default:
	}

	// Step 4: Execute the tool with optional panic recovery
	if e.config.recoverPanics {
		return e.executeWithRecovery(ctx, tool, toolName, input)
	}

	return e.executeDirectly(ctx, tool, toolName, input)
}

// executeWithRecovery executes a tool with panic recovery.
// If a panic occurs, it is converted to a PanicError with stack trace.
func (e *executor) executeWithRecovery(ctx context.Context, tool Tool, toolName string, input *Input) (output *Output, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err = NewPanicErrorWithStack(toolName, r, stack)
			output = nil
		}
	}()

	return e.executeDirectly(ctx, tool, toolName, input)
}

// executeDirectly executes a tool without panic recovery.
// It wraps any errors from the tool execution.
func (e *executor) executeDirectly(ctx context.Context, tool Tool, toolName string, input *Input) (*Output, error) {
	output, err := tool.Execute(ctx, input)
	if err != nil {
		// Check if this was a context error
		if ctx.Err() != nil {
			return nil, e.wrapContextError(ctx, toolName)
		}
		// Wrap the execution error
		return nil, NewExecutionErrorWithCause(toolName, err)
	}

	return output, nil
}

// wrapContextError wraps a context error into the appropriate error type.
// context.DeadlineExceeded becomes TimeoutError.
// context.Canceled becomes ErrContextCancelled wrapped in a ToolError.
func (e *executor) wrapContextError(ctx context.Context, toolName string) error {
	switch ctx.Err() {
	case context.DeadlineExceeded:
		// Determine the timeout duration if available
		if deadline, ok := ctx.Deadline(); ok {
			// Calculate approximate timeout from deadline
			timeout := time.Until(deadline)
			if timeout < 0 {
				// Deadline has passed, use config timeout as approximation
				timeout = e.config.timeout
			}
			return NewTimeoutError(toolName, timeout)
		}
		return NewTimeoutError(toolName, e.config.timeout)
	case context.Canceled:
		return &ToolError{
			Operation: "execute",
			ToolName:  toolName,
			Message:   "execution cancelled",
			Cause:     ErrContextCancelled,
		}
	default:
		// Unknown context error
		return &ToolError{
			Operation: "execute",
			ToolName:  toolName,
			Message:   "context error",
			Cause:     ctx.Err(),
		}
	}
}

// GetRegistry returns the registry used by this executor.
// This is useful for testing and debugging.
func (e *executor) GetRegistry() Registry {
	return e.registry
}

// GetTimeout returns the configured timeout for this executor.
func (e *executor) GetTimeout() time.Duration {
	return e.config.timeout
}

// GetMaxConcurrent returns the configured maximum concurrent executions.
func (e *executor) GetMaxConcurrent() int {
	return e.config.maxConcurrent
}

// RecoversPanics returns whether this executor recovers from panics.
func (e *executor) RecoversPanics() bool {
	return e.config.recoverPanics
}

// ExecuteAsync runs a tool asynchronously and returns a channel for the result.
// The result channel will receive exactly one Result and then close.
// This allows callers to start execution and retrieve results when needed.
//
// The implementation:
//   - Uses a buffered channel (size 1) to prevent goroutine leaks
//   - Closes the channel when done to signal completion
//   - Includes timing information in the result (start, end, duration)
//   - Respects context cancellation through the underlying Execute call
//
// Usage:
//
//	resultCh := executor.ExecuteAsync(ctx, "mytool", input)
//	result := <-resultCh
//	if result.Error != nil {
//	    // Handle error
//	}
func (e *executor) ExecuteAsync(ctx context.Context, toolName string, input *Input) <-chan *Result {
	resultCh := make(chan *Result, 1)

	go func() {
		defer close(resultCh)

		start := time.Now()
		output, err := e.Execute(ctx, toolName, input)
		end := time.Now()

		result := &Result{
			ToolName:  toolName,
			Output:    output,
			Error:     err,
			StartTime: start,
			EndTime:   end,
			Duration:  end.Sub(start),
		}

		resultCh <- result
	}()

	return resultCh
}

// ExecuteMany runs multiple tools concurrently and returns all results.
// This is a placeholder implementation that will be expanded in subtask 3-3.
// Currently executes tools sequentially for simplicity.
func (e *executor) ExecuteMany(ctx context.Context, executions []ToolExecution) ([]*Result, error) {
	results := make([]*Result, len(executions))
	var firstErr error

	for i, exec := range executions {
		// Check context before each execution
		select {
		case <-ctx.Done():
			// Fill remaining results with cancelled errors
			for j := i; j < len(executions); j++ {
				results[j] = &Result{
					ToolName: executions[j].ToolName,
					Error:    e.wrapContextError(ctx, executions[j].ToolName),
				}
			}
			if firstErr == nil {
				firstErr = e.wrapContextError(ctx, exec.ToolName)
			}
			return results, firstErr
		default:
		}

		start := time.Now()
		output, err := e.Execute(ctx, exec.ToolName, exec.Input)
		end := time.Now()

		results[i] = &Result{
			ToolName:  exec.ToolName,
			Output:    output,
			Error:     err,
			StartTime: start,
			EndTime:   end,
			Duration:  end.Sub(start),
		}

		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

// Ensure executor implements the Executor interface.
var _ Executor = (*executor)(nil)
