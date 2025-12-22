// Package toolexec provides a modular, extensible tool executor architecture.
// This file defines the functional options pattern for Executor configuration,
// allowing flexible, backward-compatible configuration of executor behavior.
package toolexec

import "time"

// ExecutorOption is a function that configures an executorConfig.
// Use these options with NewExecutor to customize executor behavior.
//
// Example:
//
//	executor := NewExecutor(
//	    registry,
//	    WithTimeout(60*time.Second),
//	    WithMaxConcurrent(4),
//	    WithMiddleware(NewTimingMiddleware()),
//	)
type ExecutorOption func(*executorConfig)

// WithTimeout sets the default timeout for tool execution.
// If the context passed to Execute does not have a deadline, this timeout
// will be applied. A zero or negative timeout disables the default timeout.
//
// Default: 30 seconds
//
// Example:
//
//	executor := NewExecutor(registry, WithTimeout(60*time.Second))
func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(c *executorConfig) {
		c.timeout = timeout
	}
}

// WithMaxConcurrent sets the maximum number of concurrent tool executions
// for batch operations (ExecuteMany). This limits the number of goroutines
// that can execute tools simultaneously.
//
// Values:
//   - n > 0: limit to n concurrent executions
//   - n <= 0: unlimited concurrency
//
// Default: 1 (sequential execution for safety)
//
// Example:
//
//	executor := NewExecutor(registry, WithMaxConcurrent(4))
func WithMaxConcurrent(n int) ExecutorOption {
	return func(c *executorConfig) {
		c.maxConcurrent = n
	}
}

// WithRecoverPanics sets whether the executor should recover from panics
// during tool execution. When enabled, panics are converted to PanicError
// with stack traces instead of propagating up the call stack.
//
// Default: true (recover panics for stability)
//
// Example:
//
//	// Disable panic recovery for debugging
//	executor := NewExecutor(registry, WithRecoverPanics(false))
func WithRecoverPanics(enabled bool) ExecutorOption {
	return func(c *executorConfig) {
		c.recoverPanics = enabled
	}
}

// WithMiddleware adds a middleware to the executor's middleware chain.
// Middlewares are applied in the order they are added, with the first
// middleware being the outermost wrapper (executed first for pre-processing,
// last for post-processing).
//
// Multiple WithMiddleware options can be provided to build up the chain:
//
//	executor := NewExecutor(registry,
//	    WithMiddleware(NewRecoveryMiddleware(true)),
//	    WithMiddleware(NewTimingMiddleware()),
//	    WithMiddleware(NewInputValidationMiddleware()),
//	)
//
// This creates a chain where recovery wraps timing, which wraps validation.
func WithMiddleware(mw Middleware) ExecutorOption {
	return func(c *executorConfig) {
		if mw != nil {
			if c.middlewareChain == nil {
				c.middlewareChain = NewMiddlewareChain()
			}
			c.middlewareChain.Add(mw)
		}
	}
}

// WithMiddlewareChain sets the entire middleware chain for the executor.
// This replaces any previously configured middlewares.
//
// Use this when you have a pre-configured chain:
//
//	chain := NewMiddlewareChain(
//	    NewRecoveryMiddleware(true),
//	    NewTimingMiddleware(),
//	)
//	executor := NewExecutor(registry, WithMiddlewareChain(chain))
//
// If you want to add to an existing chain, use WithMiddleware instead.
func WithMiddlewareChain(chain *MiddlewareChain) ExecutorOption {
	return func(c *executorConfig) {
		c.middlewareChain = chain
	}
}

// WithDefaultMiddleware configures the executor with the default middleware
// chain, which includes:
//   - RecoveryMiddleware (with stack traces)
//   - ContextCheckMiddleware
//   - InputValidationMiddleware
//   - TimingMiddleware
//
// This is a convenience option for common use cases where you want
// sensible middleware defaults.
//
// Example:
//
//	executor := NewExecutor(registry, WithDefaultMiddleware())
func WithDefaultMiddleware() ExecutorOption {
	return func(c *executorConfig) {
		c.middlewareChain = DefaultMiddlewareChain()
	}
}

// WithUnlimitedConcurrency is a convenience option that sets maxConcurrent
// to unlimited (-1), allowing as many concurrent executions as there are
// tasks in a batch operation.
//
// Use with caution: unlimited concurrency can lead to resource exhaustion.
//
// Example:
//
//	executor := NewExecutor(registry, WithUnlimitedConcurrency())
func WithUnlimitedConcurrency() ExecutorOption {
	return func(c *executorConfig) {
		c.maxConcurrent = -1
	}
}

// WithNoTimeout disables the default timeout for tool execution.
// Executions will only be limited by the context passed to Execute.
//
// Use with caution: without a timeout, executions may hang indefinitely.
//
// Example:
//
//	executor := NewExecutor(registry, WithNoTimeout())
func WithNoTimeout() ExecutorOption {
	return func(c *executorConfig) {
		c.timeout = 0
	}
}

// WithSecurityPolicy sets the security policy for validating tool executions.
// The security policy is checked before each tool execution. If validation
// fails, the execution is blocked and a SecurityViolationError is returned.
//
// If policy is nil, security validation is disabled.
//
// Example:
//
//	executor := NewExecutor(registry,
//	    WithSecurityPolicy(DefaultSecurityPolicy()),
//	)
func WithSecurityPolicy(policy SecurityPolicy) ExecutorOption {
	return func(c *executorConfig) {
		c.securityPolicy = policy
	}
}

// WithConfirmationHandler sets the handler for requesting user confirmation.
// The handler is called when a tool's RequiresConfirmation() returns true.
// If the user denies, a UserDeniedError is returned.
//
// If handler is nil, confirmation is disabled (tools execute without asking).
//
// Example:
//
//	executor := NewExecutor(registry,
//	    WithConfirmationHandler(&AutoApproveHandler{}),
//	)
func WithConfirmationHandler(handler ConfirmationHandler) ExecutorOption {
	return func(c *executorConfig) {
		c.confirmHandler = handler
	}
}

// WithDefaultSecurityPolicy sets the executor to use the default security
// policy which includes blacklist and path validation.
//
// Example:
//
//	executor := NewExecutor(registry, WithDefaultSecurityPolicy())
func WithDefaultSecurityPolicy() ExecutorOption {
	return func(c *executorConfig) {
		c.securityPolicy = DefaultSecurityPolicy()
	}
}

// applyOptions applies all options to the config.
// This is an internal helper function.
func applyOptions(config *executorConfig, opts ...ExecutorOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(config)
		}
	}
}

// ExecutorConfig exposes executor configuration for inspection.
// This is useful for testing and debugging.
type ExecutorConfig struct {
	// Timeout is the default timeout for tool execution.
	Timeout time.Duration

	// MaxConcurrent is the maximum number of concurrent executions.
	MaxConcurrent int

	// RecoverPanics indicates whether panics are recovered.
	RecoverPanics bool

	// HasMiddleware indicates whether middleware is configured.
	HasMiddleware bool

	// MiddlewareCount is the number of middlewares in the chain.
	MiddlewareCount int

	// HasSecurityPolicy indicates whether a security policy is configured.
	HasSecurityPolicy bool

	// HasConfirmationHandler indicates whether a confirmation handler is configured.
	HasConfirmationHandler bool
}

// Config returns the executor's configuration for inspection.
// The returned struct is a copy; modifications do not affect the executor.
func (e *executor) Config() ExecutorConfig {
	config := ExecutorConfig{
		Timeout:                e.config.timeout,
		MaxConcurrent:          e.config.maxConcurrent,
		RecoverPanics:          e.config.recoverPanics,
		HasSecurityPolicy:      e.config.securityPolicy != nil,
		HasConfirmationHandler: e.config.confirmHandler != nil,
	}

	if e.config.middlewareChain != nil {
		config.HasMiddleware = true
		config.MiddlewareCount = e.config.middlewareChain.Len()
	}

	return config
}

// DefaultExecutorOptions returns a slice of options that configure
// an executor with recommended defaults:
//   - 30 second timeout
//   - 1 concurrent execution (sequential for safety)
//   - Panic recovery enabled
//   - Default middleware chain
//
// This is useful when you want to start with defaults and override specific
// options:
//
//	opts := append(DefaultExecutorOptions(), WithTimeout(60*time.Second))
//	executor := NewExecutor(registry, opts...)
func DefaultExecutorOptions() []ExecutorOption {
	return []ExecutorOption{
		WithTimeout(30 * time.Second),
		WithMaxConcurrent(1),
		WithRecoverPanics(true),
	}
}

// CombineOptions combines multiple ExecutorOption slices into one.
// This is useful for merging default options with custom options.
//
// Example:
//
//	defaults := DefaultExecutorOptions()
//	custom := []ExecutorOption{WithTimeout(60*time.Second)}
//	allOptions := CombineOptions(defaults, custom)
func CombineOptions(optionSets ...[]ExecutorOption) []ExecutorOption {
	var combined []ExecutorOption
	for _, opts := range optionSets {
		combined = append(combined, opts...)
	}
	return combined
}
