// Package toolexec provides a modular, extensible tool executor architecture.
// This file defines the Middleware interface and chain implementation for
// adding cross-cutting concerns (logging, validation, metrics) to tool execution.
package toolexec

import (
	"context"
	"runtime/debug"
	"time"
)

// ToolFunc is the function signature for tool execution.
// It matches the core execution pattern: context, tool name, input -> output, error.
// Middleware wraps this function to add pre/post execution logic.
type ToolFunc func(ctx context.Context, toolName string, input *Input) (*Output, error)

// Middleware defines the interface for tool execution middleware.
// Middleware can wrap tool execution to add cross-cutting concerns such as:
//   - Logging (before/after execution)
//   - Metrics (timing, success/failure rates)
//   - Validation (input/output validation)
//   - Error handling (panic recovery, error wrapping)
//   - Caching (memoization of results)
//   - Rate limiting (throttling requests)
//
// Middleware is applied in order: the first middleware added is the outermost wrapper.
// For example, if middlewares are added in order [A, B, C], execution flows as:
// A.before -> B.before -> C.before -> tool -> C.after -> B.after -> A.after
type Middleware interface {
	// Name returns the middleware name for debugging and error messages.
	// This is used in MiddlewareError to identify which middleware failed.
	Name() string

	// Wrap wraps a ToolFunc to add pre/post execution logic.
	// The middleware should call 'next' to continue the chain.
	// Returning without calling 'next' short-circuits the chain.
	//
	// Example implementation:
	//   func (m *LoggingMiddleware) Wrap(next ToolFunc) ToolFunc {
	//       return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
	//           log.Printf("Starting execution of %s", toolName)
	//           output, err := next(ctx, toolName, input)
	//           log.Printf("Finished execution of %s", toolName)
	//           return output, err
	//       }
	//   }
	Wrap(next ToolFunc) ToolFunc
}

// MiddlewareChain chains multiple middlewares together.
// Middlewares are applied in the order they are added, with the first
// middleware being the outermost wrapper.
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewMiddlewareChain creates a new middleware chain with the given middlewares.
// Middlewares are applied in order: first middleware is outermost.
func NewMiddlewareChain(middlewares ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: middlewares,
	}
}

// Add appends a middleware to the chain.
// Returns the chain for method chaining.
func (c *MiddlewareChain) Add(mw Middleware) *MiddlewareChain {
	c.middlewares = append(c.middlewares, mw)
	return c
}

// Prepend adds a middleware to the beginning of the chain.
// This middleware will be the outermost wrapper.
// Returns the chain for method chaining.
func (c *MiddlewareChain) Prepend(mw Middleware) *MiddlewareChain {
	c.middlewares = append([]Middleware{mw}, c.middlewares...)
	return c
}

// Len returns the number of middlewares in the chain.
func (c *MiddlewareChain) Len() int {
	return len(c.middlewares)
}

// Middlewares returns a copy of the middlewares in the chain.
// The returned slice is a copy, so modifications do not affect the chain.
func (c *MiddlewareChain) Middlewares() []Middleware {
	result := make([]Middleware, len(c.middlewares))
	copy(result, c.middlewares)
	return result
}

// Wrap applies all middlewares to a ToolFunc.
// Middlewares are applied in reverse order so that the first middleware
// in the chain is the outermost wrapper (executed first/last).
//
// Example:
//
//	chain := NewMiddlewareChain(loggingMw, metricsMw, validationMw)
//	wrapped := chain.Wrap(originalFunc)
//	// Execution order: logging.before -> metrics.before -> validation.before
//	//                  -> originalFunc
//	//                  -> validation.after -> metrics.after -> logging.after
func (c *MiddlewareChain) Wrap(fn ToolFunc) ToolFunc {
	if len(c.middlewares) == 0 {
		return fn
	}

	// Apply middlewares in reverse order
	// So that the first middleware is the outermost wrapper
	wrapped := fn
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		wrapped = c.middlewares[i].Wrap(wrapped)
	}

	return wrapped
}

// MiddlewareFunc is a function adapter for creating simple middlewares.
// It implements the Middleware interface, allowing functions to be used
// as middlewares without creating a full struct.
type MiddlewareFunc struct {
	// name is the middleware name for debugging.
	name string

	// fn is the wrapper function.
	fn func(next ToolFunc) ToolFunc
}

// NewMiddlewareFunc creates a Middleware from a function.
// This is useful for creating simple inline middlewares.
//
// Example:
//
//	mw := NewMiddlewareFunc("timing", func(next ToolFunc) ToolFunc {
//	    return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
//	        start := time.Now()
//	        output, err := next(ctx, toolName, input)
//	        log.Printf("%s took %v", toolName, time.Since(start))
//	        return output, err
//	    }
//	})
func NewMiddlewareFunc(name string, fn func(next ToolFunc) ToolFunc) *MiddlewareFunc {
	return &MiddlewareFunc{
		name: name,
		fn:   fn,
	}
}

// Name returns the middleware name.
func (m *MiddlewareFunc) Name() string {
	return m.name
}

// Wrap applies the middleware function.
func (m *MiddlewareFunc) Wrap(next ToolFunc) ToolFunc {
	if m.fn == nil {
		return next
	}
	return m.fn(next)
}

// Compile-time verification that MiddlewareFunc implements Middleware.
var _ Middleware = (*MiddlewareFunc)(nil)

// ===========================================================================
// Built-in Middlewares
// ===========================================================================

// RecoveryMiddleware recovers from panics in tool execution.
// It converts panics to PanicError with stack traces.
type RecoveryMiddleware struct {
	// includeStack determines whether to include stack traces in errors.
	includeStack bool
}

// NewRecoveryMiddleware creates a new panic recovery middleware.
// If includeStack is true, the stack trace is included in the error.
func NewRecoveryMiddleware(includeStack bool) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		includeStack: includeStack,
	}
}

// Name returns the middleware name.
func (m *RecoveryMiddleware) Name() string {
	return "recovery"
}

// Wrap wraps the ToolFunc with panic recovery.
func (m *RecoveryMiddleware) Wrap(next ToolFunc) ToolFunc {
	return func(ctx context.Context, toolName string, input *Input) (output *Output, err error) {
		defer func() {
			if r := recover(); r != nil {
				if m.includeStack {
					stack := string(debug.Stack())
					err = NewPanicErrorWithStack(toolName, r, stack)
				} else {
					err = NewPanicError(toolName, r)
				}
				output = nil
			}
		}()

		return next(ctx, toolName, input)
	}
}

// Compile-time verification that RecoveryMiddleware implements Middleware.
var _ Middleware = (*RecoveryMiddleware)(nil)

// TimingMiddleware records execution timing in the output metadata.
// It adds "execution_time_ms" and "execution_start" metadata fields.
type TimingMiddleware struct{}

// NewTimingMiddleware creates a new timing middleware.
func NewTimingMiddleware() *TimingMiddleware {
	return &TimingMiddleware{}
}

// Name returns the middleware name.
func (m *TimingMiddleware) Name() string {
	return "timing"
}

// Wrap wraps the ToolFunc to record timing information.
// On success, it adds timing metadata to the output.
func (m *TimingMiddleware) Wrap(next ToolFunc) ToolFunc {
	return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
		start := time.Now()

		output, err := next(ctx, toolName, input)

		duration := time.Since(start)

		// Add timing to output metadata if successful
		if output != nil {
			if output.Metadata == nil {
				output.Metadata = make(map[string]string)
			}
			output.Metadata["execution_time_ms"] = formatDurationMs(duration)
			output.Metadata["execution_start"] = start.Format(time.RFC3339Nano)
		}

		return output, err
	}
}

// formatDurationMs formats a duration as milliseconds with 3 decimal places.
func formatDurationMs(d time.Duration) string {
	ms := float64(d.Nanoseconds()) / float64(time.Millisecond)
	return formatFloat(ms, 3)
}

// formatFloat formats a float with the given precision.
// Avoids importing fmt just for formatting.
func formatFloat(f float64, precision int) string {
	// Simple implementation for our use case
	// For ms values, we want something like "123.456"
	intPart := int64(f)
	fracPart := f - float64(intPart)

	// Build the fractional part
	frac := ""
	for i := 0; i < precision; i++ {
		fracPart *= 10
		digit := int(fracPart) % 10
		frac += string(rune('0' + digit))
	}

	// Use strconv-like approach
	result := formatInt64(intPart) + "." + frac
	return result
}

// formatInt64 formats an int64 as a string.
func formatInt64(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// Compile-time verification that TimingMiddleware implements Middleware.
var _ Middleware = (*TimingMiddleware)(nil)

// ContextCheckMiddleware checks that the context is not cancelled before execution.
// This provides early detection of cancelled contexts.
type ContextCheckMiddleware struct{}

// NewContextCheckMiddleware creates a new context check middleware.
func NewContextCheckMiddleware() *ContextCheckMiddleware {
	return &ContextCheckMiddleware{}
}

// Name returns the middleware name.
func (m *ContextCheckMiddleware) Name() string {
	return "context-check"
}

// Wrap wraps the ToolFunc to check context before execution.
func (m *ContextCheckMiddleware) Wrap(next ToolFunc) ToolFunc {
	return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
		// Check context before proceeding
		select {
		case <-ctx.Done():
			return nil, &ToolError{
				Operation: "middleware",
				ToolName:  toolName,
				Message:   "context cancelled before execution",
				Cause:     ctx.Err(),
			}
		default:
		}

		return next(ctx, toolName, input)
	}
}

// Compile-time verification that ContextCheckMiddleware implements Middleware.
var _ Middleware = (*ContextCheckMiddleware)(nil)

// InputValidationMiddleware validates that input is not nil.
// This catches nil input errors early in the middleware chain.
type InputValidationMiddleware struct{}

// NewInputValidationMiddleware creates a new input validation middleware.
func NewInputValidationMiddleware() *InputValidationMiddleware {
	return &InputValidationMiddleware{}
}

// Name returns the middleware name.
func (m *InputValidationMiddleware) Name() string {
	return "input-validation"
}

// Wrap wraps the ToolFunc to validate input.
func (m *InputValidationMiddleware) Wrap(next ToolFunc) ToolFunc {
	return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
		if input == nil {
			return nil, NewValidationError(toolName, "input cannot be nil")
		}

		return next(ctx, toolName, input)
	}
}

// Compile-time verification that InputValidationMiddleware implements Middleware.
var _ Middleware = (*InputValidationMiddleware)(nil)

// LoggingMiddleware provides hooks for logging before and after tool execution.
// It does not perform actual logging (to avoid import dependencies) but provides
// callbacks that can be used to integrate with any logging framework.
type LoggingMiddleware struct {
	// beforeFunc is called before tool execution.
	// It receives the tool name and input.
	beforeFunc func(toolName string, input *Input)

	// afterFunc is called after tool execution.
	// It receives the tool name, output, error, and duration.
	afterFunc func(toolName string, output *Output, err error, duration time.Duration)
}

// NewLoggingMiddleware creates a new logging middleware with the given callbacks.
// Either callback can be nil to skip that hook.
func NewLoggingMiddleware(
	beforeFunc func(toolName string, input *Input),
	afterFunc func(toolName string, output *Output, err error, duration time.Duration),
) *LoggingMiddleware {
	return &LoggingMiddleware{
		beforeFunc: beforeFunc,
		afterFunc:  afterFunc,
	}
}

// Name returns the middleware name.
func (m *LoggingMiddleware) Name() string {
	return "logging"
}

// Wrap wraps the ToolFunc to add logging hooks.
func (m *LoggingMiddleware) Wrap(next ToolFunc) ToolFunc {
	return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
		// Before hook
		if m.beforeFunc != nil {
			m.beforeFunc(toolName, input)
		}

		start := time.Now()
		output, err := next(ctx, toolName, input)
		duration := time.Since(start)

		// After hook
		if m.afterFunc != nil {
			m.afterFunc(toolName, output, err, duration)
		}

		return output, err
	}
}

// Compile-time verification that LoggingMiddleware implements Middleware.
var _ Middleware = (*LoggingMiddleware)(nil)

// ===========================================================================
// Utility Functions
// ===========================================================================

// ChainMiddleware creates a MiddlewareChain from the given middlewares.
// This is a convenience function equivalent to NewMiddlewareChain.
func ChainMiddleware(middlewares ...Middleware) *MiddlewareChain {
	return NewMiddlewareChain(middlewares...)
}

// ApplyMiddleware applies a slice of middlewares to a ToolFunc.
// Middlewares are applied in order: first middleware is outermost.
func ApplyMiddleware(fn ToolFunc, middlewares ...Middleware) ToolFunc {
	return NewMiddlewareChain(middlewares...).Wrap(fn)
}

// CombineMiddleware combines multiple MiddlewareChains into one.
// The resulting chain contains all middlewares from all chains in order.
func CombineMiddleware(chains ...*MiddlewareChain) *MiddlewareChain {
	combined := NewMiddlewareChain()
	for _, chain := range chains {
		if chain != nil {
			combined.middlewares = append(combined.middlewares, chain.middlewares...)
		}
	}
	return combined
}

// DefaultMiddlewareChain returns a middleware chain with recommended defaults.
// The chain includes:
//   - RecoveryMiddleware (with stack traces)
//   - ContextCheckMiddleware
//   - InputValidationMiddleware
//   - TimingMiddleware
func DefaultMiddlewareChain() *MiddlewareChain {
	return NewMiddlewareChain(
		NewRecoveryMiddleware(true),
		NewContextCheckMiddleware(),
		NewInputValidationMiddleware(),
		NewTimingMiddleware(),
	)
}
