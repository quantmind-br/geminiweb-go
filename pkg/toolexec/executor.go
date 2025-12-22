// Package toolexec provides a modular, extensible tool executor architecture.
// This file implements the Executor for tool execution with context support,
// timeouts, and proper error handling.
package toolexec

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
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

	// middlewareChain is the chain of middlewares to apply to tool execution.
	// Middlewares are applied in order, with the first middleware being the
	// outermost wrapper.
	middlewareChain *MiddlewareChain

	// securityPolicy is the security policy for validating tool executions.
	// If nil, no security validation is performed.
	securityPolicy SecurityPolicy

	// confirmHandler is the handler for requesting user confirmation.
	// If nil, no confirmation is requested even if the tool requires it.
	confirmHandler ConfirmationHandler
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

// NewExecutor creates a new Executor with the given registry and options.
// If registry is nil, the default global registry is used.
// Additional configuration can be provided via ExecutorOption functions.
//
// Example:
//
//	executor := NewExecutor(
//	    registry,
//	    WithTimeout(60*time.Second),
//	    WithMaxConcurrent(4),
//	    WithDefaultMiddleware(),
//	)
//
// Default configuration (when no options provided):
//   - 30 second timeout
//   - 1 concurrent execution (sequential for safety)
//   - Panic recovery enabled
//   - No middleware chain (pass-through execution)
func NewExecutor(registry Registry, opts ...ExecutorOption) *executor {
	if registry == nil {
		registry = DefaultRegistry()
	}

	config := defaultConfig()
	applyOptions(config, opts...)

	return &executor{
		registry: registry,
		config:   config,
	}
}

// Execute runs a tool synchronously with the given input.
// It performs the following steps:
//  1. Look up the tool in the registry
//  2. Apply timeout if configured
//  3. Check context before execution
//  4. Validate against security policy (if configured)
//  5. Request confirmation if tool requires it (if handler configured)
//  6. Apply middleware chain (if configured)
//  7. Execute the tool with panic recovery
//  8. Return the output or error
//
// The context is used for cancellation and can have a timeout applied.
// If the executor has a default timeout configured and the context has no
// deadline, a timeout will be applied.
//
// Security validation happens before confirmation, and both happen before
// the actual tool execution.
//
// Middleware chain is applied around the tool execution, allowing pre/post
// execution hooks for logging, validation, metrics, etc.
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

	// Step 4: Validate against security policy if configured
	if e.config.securityPolicy != nil {
		// Convert input params to args for security validation
		args := make(map[string]any)
		if input != nil && input.Params != nil {
			args = input.Params
		}
		if err := e.config.securityPolicy.Validate(ctx, toolName, args); err != nil {
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
	}

	// Step 5: Request confirmation if tool requires it and handler is configured
	if e.config.confirmHandler != nil {
		// Convert input params to args for confirmation check
		args := make(map[string]any)
		if input != nil && input.Params != nil {
			args = input.Params
		}
		if tool.RequiresConfirmation(args) {
			confirmed, err := e.config.confirmHandler.RequestConfirmation(ctx, tool, args)
			if err != nil {
				return nil, fmt.Errorf("confirmation failed: %w", err)
			}
			if !confirmed {
				return nil, NewUserDeniedError(toolName)
			}
		}
	}

	// Step 6: Create the base execution function
	// This function performs the actual tool execution with error wrapping
	baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
		return e.executeToolDirectly(ctx, tool, toolName, input)
	}

	// Step 7: Apply middleware chain if configured
	execFn := baseFn
	if e.config.middlewareChain != nil && e.config.middlewareChain.Len() > 0 {
		execFn = e.config.middlewareChain.Wrap(baseFn)
	}

	// Step 8: Execute with optional panic recovery
	// Note: If middleware chain includes RecoveryMiddleware, this provides
	// a second layer of protection. The executor's panic recovery is always
	// the outermost layer when enabled.
	if e.config.recoverPanics {
		return e.executeWithRecovery(ctx, execFn, toolName, input)
	}

	return execFn(ctx, toolName, input)
}

// executeWithRecovery executes a ToolFunc with panic recovery.
// If a panic occurs, it is converted to a PanicError with stack trace.
// This wraps the entire middleware-wrapped execution chain.
func (e *executor) executeWithRecovery(ctx context.Context, fn ToolFunc, toolName string, input *Input) (output *Output, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err = NewPanicErrorWithStack(toolName, r, stack)
			output = nil
		}
	}()

	return fn(ctx, toolName, input)
}

// executeToolDirectly executes a tool without panic recovery or middleware.
// It wraps any errors from the tool execution.
// This is the innermost execution function that actually calls the tool.
func (e *executor) executeToolDirectly(ctx context.Context, tool Tool, toolName string, input *Input) (*Output, error) {
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

// GetMiddlewareChain returns a copy of the middleware chain used by this executor.
// Returns nil if no middleware chain is configured.
// The returned chain is a copy; modifications do not affect the executor.
func (e *executor) GetMiddlewareChain() *MiddlewareChain {
	if e.config.middlewareChain == nil {
		return nil
	}
	// Return a copy to prevent external modification
	return NewMiddlewareChain(e.config.middlewareChain.Middlewares()...)
}

// HasMiddleware returns whether this executor has a middleware chain configured.
func (e *executor) HasMiddleware() bool {
	return e.config.middlewareChain != nil && e.config.middlewareChain.Len() > 0
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
// It uses errgroup for coordinated concurrent execution with fail-fast behavior.
//
// Behavior:
//   - Executes tools concurrently up to the configured maxConcurrent limit
//   - Fail-fast: the first error cancels all remaining executions via context
//   - Partial results are always returned, even when an error occurs
//   - Each result includes timing information (start, end, duration)
//   - Results are returned in the same order as the input executions
//
// Concurrency control:
//   - If maxConcurrent <= 0, unlimited concurrency is used
//   - If maxConcurrent == 1, executions run sequentially (safe default)
//   - If maxConcurrent > 1, up to that many executions run in parallel
//
// Usage:
//
//	executions := []ToolExecution{
//	    {ToolName: "tool1", Input: input1},
//	    {ToolName: "tool2", Input: input2},
//	}
//	results, err := executor.ExecuteMany(ctx, executions)
//	// results[0] corresponds to tool1, results[1] to tool2
//	// err is the first error that occurred, if any
func (e *executor) ExecuteMany(ctx context.Context, executions []ToolExecution) ([]*Result, error) {
	if len(executions) == 0 {
		return []*Result{}, nil
	}

	// Pre-allocate results slice
	results := make([]*Result, len(executions))

	// Use a mutex to protect results slice from concurrent writes
	// (though each goroutine writes to a distinct index, the slice header
	// could theoretically race on some architectures)
	var mu sync.Mutex

	// Create errgroup with context for coordinated cancellation
	// When one goroutine returns an error, gctx is cancelled,
	// which signals all other goroutines to stop
	g, gctx := errgroup.WithContext(ctx)

	// Apply concurrency limit if configured
	// SetLimit(n) limits the number of active goroutines to n
	// SetLimit(0) or negative means unlimited
	if e.config.maxConcurrent > 0 {
		g.SetLimit(e.config.maxConcurrent)
	}

	// Launch all executions
	for i, exec := range executions {
		// Capture loop variables to avoid closure issues
		// In Go 1.22+ this is handled automatically, but we support older versions
		i, exec := i, exec

		g.Go(func() error {
			// Check if context is already cancelled before starting
			select {
			case <-gctx.Done():
				// Context cancelled (likely due to another execution failing)
				// Record the cancellation in the result
				mu.Lock()
				results[i] = &Result{
					ToolName:  exec.ToolName,
					Output:    nil,
					Error:     e.wrapContextError(gctx, exec.ToolName),
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Duration:  0,
				}
				mu.Unlock()
				return nil // Don't propagate - let the original error be the one returned
			default:
			}

			// Execute the tool
			start := time.Now()
			output, err := e.Execute(gctx, exec.ToolName, exec.Input)
			end := time.Now()

			// Record the result
			mu.Lock()
			results[i] = &Result{
				ToolName:  exec.ToolName,
				Output:    output,
				Error:     err,
				StartTime: start,
				EndTime:   end,
				Duration:  end.Sub(start),
			}
			mu.Unlock()

			// Return error for fail-fast behavior
			// This will cancel gctx and stop other executions
			if err != nil {
				return err
			}

			return nil
		})
	}

	// Wait for all goroutines to complete
	// Returns the first non-nil error (if any)
	err := g.Wait()

	// Fill in any nil results with cancelled errors
	// This handles the case where goroutines were never started due to limit
	for i, result := range results {
		if result == nil {
			results[i] = &Result{
				ToolName:  executions[i].ToolName,
				Output:    nil,
				Error:     e.wrapContextError(ctx, executions[i].ToolName),
				StartTime: time.Time{},
				EndTime:   time.Time{},
				Duration:  0,
			}
		}
	}

	// Return partial results along with the first error
	if err != nil {
		return results, fmt.Errorf("batch execution failed: %w", err)
	}

	return results, nil
}

// Ensure executor implements the Executor interface.
var _ Executor = (*executor)(nil)
