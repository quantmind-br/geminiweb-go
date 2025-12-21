// Package toolexec provides a modular, extensible tool executor architecture.
// This file defines the Result type for async execution results and custom
// error types for structured error handling throughout the package.
package toolexec

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Result represents the outcome of an asynchronous tool execution.
// It combines the tool name, output, and any error that occurred.
// This is used for both single async executions and batch operations.
type Result struct {
	// ToolName is the name of the tool that was executed.
	ToolName string

	// Output contains the tool's output if execution succeeded.
	// Will be nil if an error occurred.
	Output *Output

	// Error contains any error that occurred during execution.
	// Will be nil if execution succeeded.
	Error error

	// StartTime is when the tool execution started.
	StartTime time.Time

	// EndTime is when the tool execution completed.
	EndTime time.Time

	// Duration is the time taken for execution.
	Duration time.Duration
}

// NewResult creates a new Result with the given tool name, output, and error.
func NewResult(toolName string, output *Output, err error) *Result {
	return &Result{
		ToolName: toolName,
		Output:   output,
		Error:    err,
	}
}

// NewSuccessResult creates a new successful Result.
func NewSuccessResult(toolName string, output *Output) *Result {
	return &Result{
		ToolName: toolName,
		Output:   output,
	}
}

// NewErrorResult creates a new failed Result.
func NewErrorResult(toolName string, err error) *Result {
	return &Result{
		ToolName: toolName,
		Error:    err,
	}
}

// IsSuccess returns true if the result represents a successful execution.
func (r *Result) IsSuccess() bool {
	return r.Error == nil
}

// WithTiming sets the timing information on the result.
func (r *Result) WithTiming(start, end time.Time) *Result {
	r.StartTime = start
	r.EndTime = end
	r.Duration = end.Sub(start)
	return r
}

// Sentinel errors for common tool execution cases
var (
	// ErrToolNotFound is returned when a requested tool is not registered.
	ErrToolNotFound = errors.New("tool not found")

	// ErrDuplicateTool is returned when attempting to register a tool with a name
	// that is already registered.
	ErrDuplicateTool = errors.New("tool already registered")

	// ErrNilTool is returned when attempting to register a nil tool.
	ErrNilTool = errors.New("cannot register nil tool")

	// ErrExecutionFailed is returned when tool execution fails.
	ErrExecutionFailed = errors.New("tool execution failed")

	// ErrValidationFailed is returned when input validation fails.
	ErrValidationFailed = errors.New("input validation failed")

	// ErrContextCancelled is returned when the context is cancelled during execution.
	ErrContextCancelled = errors.New("execution cancelled")

	// ErrPanicRecovered is returned when a panic is recovered during execution.
	ErrPanicRecovered = errors.New("panic recovered during execution")

	// ErrMiddlewareFailed is returned when middleware execution fails.
	ErrMiddlewareFailed = errors.New("middleware execution failed")

	// ErrTimeout is returned when execution times out.
	ErrTimeout = errors.New("execution timed out")
)

// ToolError is the base error type for all tool execution errors.
// It implements the error interface and provides rich context for debugging.
type ToolError struct {
	// ToolName is the name of the tool that caused the error.
	ToolName string

	// Operation describes what was being attempted (e.g., "execute", "register", "validate").
	Operation string

	// Message is a human-readable error message.
	Message string

	// Cause is the underlying error that caused this error.
	Cause error
}

// Error implements the error interface.
func (e *ToolError) Error() string {
	var parts []string

	if e.Operation != "" {
		parts = append(parts, e.Operation+" failed")
	}

	if e.ToolName != "" {
		parts = append(parts, fmt.Sprintf("tool=%s", e.ToolName))
	}

	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	if len(parts) == 0 {
		return "tool error"
	}

	return strings.Join(parts, ": ")
}

// Unwrap returns the underlying cause.
func (e *ToolError) Unwrap() error {
	return e.Cause
}

// Is implements error matching for errors.Is().
func (e *ToolError) Is(target error) bool {
	if target == ErrExecutionFailed {
		return e.Operation == "execute"
	}

	if t, ok := target.(*ToolError); ok {
		return e.ToolName == t.ToolName || e.Operation == t.Operation
	}

	return false
}

// NewToolError creates a new ToolError with the given parameters.
func NewToolError(operation, toolName, message string) *ToolError {
	return &ToolError{
		Operation: operation,
		ToolName:  toolName,
		Message:   message,
	}
}

// NewToolErrorWithCause creates a ToolError wrapping another error.
func NewToolErrorWithCause(operation, toolName string, cause error) *ToolError {
	return &ToolError{
		Operation: operation,
		ToolName:  toolName,
		Cause:     cause,
	}
}

// ToolNotFoundError represents an error when a tool is not found in the registry.
type ToolNotFoundError struct {
	*ToolError
}

// NewToolNotFoundError creates a new ToolNotFoundError.
func NewToolNotFoundError(toolName string) *ToolNotFoundError {
	return &ToolNotFoundError{
		ToolError: &ToolError{
			Operation: "get tool",
			ToolName:  toolName,
			Message:   fmt.Sprintf("tool '%s' is not registered", toolName),
		},
	}
}

// Error implements the error interface.
func (e *ToolNotFoundError) Error() string {
	return fmt.Sprintf("tool not found: '%s'", e.ToolName)
}

// Is allows comparison with sentinel errors.
func (e *ToolNotFoundError) Is(target error) bool {
	if target == ErrToolNotFound {
		return true
	}
	if _, ok := target.(*ToolNotFoundError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause.
func (e *ToolNotFoundError) Unwrap() error {
	return e.Cause
}

// DuplicateToolError represents an error when attempting to register a duplicate tool.
type DuplicateToolError struct {
	*ToolError
}

// NewDuplicateToolError creates a new DuplicateToolError.
func NewDuplicateToolError(toolName string) *DuplicateToolError {
	return &DuplicateToolError{
		ToolError: &ToolError{
			Operation: "register",
			ToolName:  toolName,
			Message:   fmt.Sprintf("tool '%s' is already registered", toolName),
		},
	}
}

// Error implements the error interface.
func (e *DuplicateToolError) Error() string {
	return fmt.Sprintf("duplicate tool registration: '%s'", e.ToolName)
}

// Is allows comparison with sentinel errors.
func (e *DuplicateToolError) Is(target error) bool {
	if target == ErrDuplicateTool {
		return true
	}
	if _, ok := target.(*DuplicateToolError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause.
func (e *DuplicateToolError) Unwrap() error {
	return e.Cause
}

// ExecutionError represents an error that occurred during tool execution.
type ExecutionError struct {
	*ToolError
	// Input is the input that was provided to the tool (for debugging).
	Input *Input
}

// NewExecutionError creates a new ExecutionError.
func NewExecutionError(toolName, message string) *ExecutionError {
	return &ExecutionError{
		ToolError: &ToolError{
			Operation: "execute",
			ToolName:  toolName,
			Message:   message,
		},
	}
}

// NewExecutionErrorWithCause creates an ExecutionError wrapping another error.
func NewExecutionErrorWithCause(toolName string, cause error) *ExecutionError {
	return &ExecutionError{
		ToolError: &ToolError{
			Operation: "execute",
			ToolName:  toolName,
			Cause:     cause,
		},
	}
}

// Error implements the error interface.
func (e *ExecutionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool '%s' execution failed: %v", e.ToolName, e.Cause)
	}
	if e.Message != "" {
		return fmt.Sprintf("tool '%s' execution failed: %s", e.ToolName, e.Message)
	}
	return fmt.Sprintf("tool '%s' execution failed", e.ToolName)
}

// Is allows comparison with sentinel errors.
func (e *ExecutionError) Is(target error) bool {
	if target == ErrExecutionFailed {
		return true
	}
	if _, ok := target.(*ExecutionError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause.
func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// WithInput attaches the input to the error for debugging.
func (e *ExecutionError) WithInput(input *Input) *ExecutionError {
	e.Input = input
	return e
}

// ValidationError represents an input validation error.
type ValidationError struct {
	*ToolError
	// Field is the specific field that failed validation (optional).
	Field string
}

// NewValidationError creates a new ValidationError.
func NewValidationError(toolName, message string) *ValidationError {
	return &ValidationError{
		ToolError: &ToolError{
			Operation: "validate input",
			ToolName:  toolName,
			Message:   message,
		},
	}
}

// NewValidationErrorForField creates a ValidationError for a specific field.
func NewValidationErrorForField(toolName, field, message string) *ValidationError {
	return &ValidationError{
		ToolError: &ToolError{
			Operation: "validate input",
			ToolName:  toolName,
			Message:   fmt.Sprintf("field '%s': %s", field, message),
		},
		Field: field,
	}
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for tool '%s' field '%s': %s",
			e.ToolName, e.Field, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("validation failed for tool '%s': %s", e.ToolName, e.Message)
	}
	return fmt.Sprintf("validation failed for tool '%s'", e.ToolName)
}

// Is allows comparison with sentinel errors.
func (e *ValidationError) Is(target error) bool {
	if target == ErrValidationFailed {
		return true
	}
	if _, ok := target.(*ValidationError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause.
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// PanicError represents a recovered panic during tool execution.
type PanicError struct {
	*ToolError
	// PanicValue is the value passed to panic().
	PanicValue any
	// Stack is the stack trace at the time of panic (optional).
	Stack string
}

// NewPanicError creates a new PanicError.
func NewPanicError(toolName string, panicValue any) *PanicError {
	return &PanicError{
		ToolError: &ToolError{
			Operation: "execute",
			ToolName:  toolName,
			Message:   fmt.Sprintf("panic: %v", panicValue),
		},
		PanicValue: panicValue,
	}
}

// NewPanicErrorWithStack creates a PanicError with a stack trace.
func NewPanicErrorWithStack(toolName string, panicValue any, stack string) *PanicError {
	e := NewPanicError(toolName, panicValue)
	e.Stack = stack
	return e
}

// Error implements the error interface.
func (e *PanicError) Error() string {
	if e.Stack != "" {
		return fmt.Sprintf("panic recovered in tool '%s': %v\nStack:\n%s",
			e.ToolName, e.PanicValue, e.Stack)
	}
	return fmt.Sprintf("panic recovered in tool '%s': %v", e.ToolName, e.PanicValue)
}

// Is allows comparison with sentinel errors.
func (e *PanicError) Is(target error) bool {
	if target == ErrPanicRecovered {
		return true
	}
	if _, ok := target.(*PanicError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause.
func (e *PanicError) Unwrap() error {
	return e.Cause
}

// TimeoutError represents an execution timeout.
type TimeoutError struct {
	*ToolError
	// Timeout is the timeout duration that was exceeded.
	Timeout time.Duration
}

// NewTimeoutError creates a new TimeoutError.
func NewTimeoutError(toolName string, timeout time.Duration) *TimeoutError {
	return &TimeoutError{
		ToolError: &ToolError{
			Operation: "execute",
			ToolName:  toolName,
			Message:   fmt.Sprintf("execution exceeded timeout of %v", timeout),
		},
		Timeout: timeout,
	}
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("tool '%s' timed out after %v", e.ToolName, e.Timeout)
}

// Is allows comparison with sentinel errors.
func (e *TimeoutError) Is(target error) bool {
	if target == ErrTimeout {
		return true
	}
	if _, ok := target.(*TimeoutError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause.
func (e *TimeoutError) Unwrap() error {
	return e.Cause
}

// MiddlewareError represents an error in middleware execution.
type MiddlewareError struct {
	*ToolError
	// MiddlewareName is the name of the middleware that failed.
	MiddlewareName string
}

// NewMiddlewareError creates a new MiddlewareError.
func NewMiddlewareError(middlewareName, toolName, message string) *MiddlewareError {
	return &MiddlewareError{
		ToolError: &ToolError{
			Operation: "middleware",
			ToolName:  toolName,
			Message:   message,
		},
		MiddlewareName: middlewareName,
	}
}

// NewMiddlewareErrorWithCause creates a MiddlewareError wrapping another error.
func NewMiddlewareErrorWithCause(middlewareName, toolName string, cause error) *MiddlewareError {
	return &MiddlewareError{
		ToolError: &ToolError{
			Operation: "middleware",
			ToolName:  toolName,
			Cause:     cause,
		},
		MiddlewareName: middlewareName,
	}
}

// Error implements the error interface.
func (e *MiddlewareError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("middleware '%s' failed for tool '%s': %v",
			e.MiddlewareName, e.ToolName, e.Cause)
	}
	if e.Message != "" {
		return fmt.Sprintf("middleware '%s' failed for tool '%s': %s",
			e.MiddlewareName, e.ToolName, e.Message)
	}
	return fmt.Sprintf("middleware '%s' failed for tool '%s'",
		e.MiddlewareName, e.ToolName)
}

// Is allows comparison with sentinel errors.
func (e *MiddlewareError) Is(target error) bool {
	if target == ErrMiddlewareFailed {
		return true
	}
	if _, ok := target.(*MiddlewareError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause.
func (e *MiddlewareError) Unwrap() error {
	return e.Cause
}

// Helper functions for error checking

// IsToolNotFoundError checks if an error is a ToolNotFoundError.
func IsToolNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrToolNotFound) {
		return true
	}
	var notFoundErr *ToolNotFoundError
	return errors.As(err, &notFoundErr)
}

// IsDuplicateToolError checks if an error is a DuplicateToolError.
func IsDuplicateToolError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDuplicateTool) {
		return true
	}
	var dupErr *DuplicateToolError
	return errors.As(err, &dupErr)
}

// IsExecutionError checks if an error is an ExecutionError.
func IsExecutionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrExecutionFailed) {
		return true
	}
	var execErr *ExecutionError
	return errors.As(err, &execErr)
}

// IsValidationError checks if an error is a ValidationError.
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrValidationFailed) {
		return true
	}
	var valErr *ValidationError
	return errors.As(err, &valErr)
}

// IsPanicError checks if an error is a PanicError.
func IsPanicError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrPanicRecovered) {
		return true
	}
	var panicErr *PanicError
	return errors.As(err, &panicErr)
}

// IsTimeoutError checks if an error is a TimeoutError.
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrTimeout) {
		return true
	}
	var timeoutErr *TimeoutError
	return errors.As(err, &timeoutErr)
}

// IsMiddlewareError checks if an error is a MiddlewareError.
func IsMiddlewareError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrMiddlewareFailed) {
		return true
	}
	var mwErr *MiddlewareError
	return errors.As(err, &mwErr)
}

// GetToolName extracts the tool name from an error, if available.
func GetToolName(err error) string {
	if err == nil {
		return ""
	}

	var toolErr *ToolError
	if errors.As(err, &toolErr) {
		return toolErr.ToolName
	}

	var notFoundErr *ToolNotFoundError
	if errors.As(err, &notFoundErr) {
		return notFoundErr.ToolName
	}

	var dupErr *DuplicateToolError
	if errors.As(err, &dupErr) {
		return dupErr.ToolName
	}

	var execErr *ExecutionError
	if errors.As(err, &execErr) {
		return execErr.ToolName
	}

	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return valErr.ToolName
	}

	var panicErr *PanicError
	if errors.As(err, &panicErr) {
		return panicErr.ToolName
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return timeoutErr.ToolName
	}

	var mwErr *MiddlewareError
	if errors.As(err, &mwErr) {
		return mwErr.ToolName
	}

	return ""
}
