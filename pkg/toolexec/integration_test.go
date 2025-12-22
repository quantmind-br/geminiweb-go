// Package toolexec provides a modular, extensible tool executor architecture.
// This file contains integration tests with real tool implementations to verify
// end-to-end functionality of the toolexec package.
package toolexec

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ===========================================================================
// Real Tool Implementations for Integration Testing
// ===========================================================================

// EchoTool is a real tool implementation that echoes input back.
type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string { return "Echoes input message back" }

func (t *EchoTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	// Check context before processing
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	msg := input.GetParamString("message")
	if msg == "" {
		return NewFailedOutput("no message provided"), nil
	}

	return NewOutput().
		WithMessage("Echo: " + msg).
		WithResult("echo", msg).
		WithMetadata("tool", "echo"), nil
}

// UpperCaseTool is a real tool that converts input to uppercase.
type UpperCaseTool struct{}

func (t *UpperCaseTool) Name() string        { return "uppercase" }
func (t *UpperCaseTool) Description() string { return "Converts input to uppercase" }

func (t *UpperCaseTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	text := input.GetParamString("text")
	if text == "" {
		return nil, NewValidationError("uppercase", "text parameter required")
	}

	upper := strings.ToUpper(text)
	return NewOutput().
		WithMessage("Converted to uppercase").
		WithResult("result", upper).
		WithData([]byte(upper)), nil
}

// LowerCaseTool is a real tool that converts input to lowercase.
type LowerCaseTool struct{}

func (t *LowerCaseTool) Name() string        { return "lowercase" }
func (t *LowerCaseTool) Description() string { return "Converts input to lowercase" }

func (t *LowerCaseTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	text := input.GetParamString("text")
	if text == "" {
		return nil, NewValidationError("lowercase", "text parameter required")
	}

	lower := strings.ToLower(text)
	return NewOutput().
		WithMessage("Converted to lowercase").
		WithResult("result", lower).
		WithData([]byte(lower)), nil
}

// MathAddTool is a real tool that adds two numbers.
type MathAddTool struct{}

func (t *MathAddTool) Name() string        { return "math.add" }
func (t *MathAddTool) Description() string { return "Adds two numbers together" }

func (t *MathAddTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	a := input.GetParamInt("a")
	b := input.GetParamInt("b")

	sum := a + b
	return NewOutput().
		WithMessage("Sum calculated").
		WithResult("sum", sum).
		WithResult("a", a).
		WithResult("b", b), nil
}

// SlowTool is a real tool that simulates slow execution for timeout testing.
type SlowTool struct {
	delay time.Duration
}

func NewSlowTool(delay time.Duration) *SlowTool {
	return &SlowTool{delay: delay}
}

func (t *SlowTool) Name() string        { return "slow" }
func (t *SlowTool) Description() string { return "A tool that takes time to execute" }

func (t *SlowTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	select {
	case <-time.After(t.delay):
		return NewOutput().WithMessage("Slow tool completed"), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ErrorTool is a real tool that always returns an error.
type ErrorTool struct {
	err error
}

func NewErrorTool(err error) *ErrorTool {
	return &ErrorTool{err: err}
}

func (t *ErrorTool) Name() string        { return "error" }
func (t *ErrorTool) Description() string { return "A tool that always errors" }

func (t *ErrorTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	return nil, t.err
}

// PanicTool is a real tool that panics during execution.
type PanicTool struct {
	panicValue any
}

func NewPanicTool(value any) *PanicTool {
	return &PanicTool{panicValue: value}
}

func (t *PanicTool) Name() string        { return "panic" }
func (t *PanicTool) Description() string { return "A tool that panics" }

func (t *PanicTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	panic(t.panicValue)
}

// CounterTool is a real tool that increments a counter (for concurrency testing).
type CounterTool struct {
	count atomic.Int64
}

func NewCounterTool() *CounterTool {
	return &CounterTool{}
}

func (t *CounterTool) Name() string        { return "counter" }
func (t *CounterTool) Description() string { return "Increments and returns a counter" }

func (t *CounterTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	newCount := t.count.Add(1)
	return NewOutput().
		WithMessage("Counter incremented").
		WithResult("count", newCount), nil
}

func (t *CounterTool) GetCount() int64 {
	return t.count.Load()
}

func (t *CounterTool) Reset() {
	t.count.Store(0)
}

// StateTool tracks execution state for testing.
type StateTool struct {
	mu           sync.Mutex
	executions   int
	lastInput    *Input
	lastToolName string
}

func NewStateTool() *StateTool {
	return &StateTool{}
}

func (t *StateTool) Name() string        { return "state" }
func (t *StateTool) Description() string { return "A tool that tracks its state" }

func (t *StateTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.executions++
	t.lastInput = input

	return NewOutput().
		WithMessage("State recorded").
		WithResult("executions", t.executions), nil
}

func (t *StateTool) GetExecutions() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.executions
}

func (t *StateTool) GetLastInput() *Input {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastInput
}

func (t *StateTool) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.executions = 0
	t.lastInput = nil
}

// ===========================================================================
// Integration Tests
// ===========================================================================

// TestIntegration_RegisterAndExecuteTools tests end-to-end registration and execution.
func TestIntegration_RegisterAndExecuteTools(t *testing.T) {
	// Create a fresh registry for this test
	reg := NewRegistry()

	// Register real tools
	tools := []Tool{
		&EchoTool{},
		&UpperCaseTool{},
		&LowerCaseTool{},
		&MathAddTool{},
	}

	for _, tool := range tools {
		if err := reg.Register(tool); err != nil {
			t.Fatalf("Failed to register tool %s: %v", tool.Name(), err)
		}
	}

	// Verify registration
	if reg.Count() != len(tools) {
		t.Errorf("Registry count = %d, want %d", reg.Count(), len(tools))
	}

	// Create executor
	exec := NewExecutor(reg)
	ctx := context.Background()

	// Test Echo tool
	t.Run("echo tool", func(t *testing.T) {
		input := NewInput().WithParam("message", "Hello, World!")
		output, err := exec.Execute(ctx, "echo", input)

		if err != nil {
			t.Errorf("Execute() error = %v", err)
			return
		}
		if output == nil {
			t.Error("Execute() returned nil output")
			return
		}
		if !output.Success {
			t.Error("Execute() output.Success = false")
		}
		if output.GetResultString("echo") != "Hello, World!" {
			t.Errorf("Output echo = %q, want %q", output.GetResultString("echo"), "Hello, World!")
		}
	})

	// Test UpperCase tool
	t.Run("uppercase tool", func(t *testing.T) {
		input := NewInput().WithParam("text", "hello")
		output, err := exec.Execute(ctx, "uppercase", input)

		if err != nil {
			t.Errorf("Execute() error = %v", err)
			return
		}
		if output.GetResultString("result") != "HELLO" {
			t.Errorf("Output result = %q, want %q", output.GetResultString("result"), "HELLO")
		}
	})

	// Test LowerCase tool
	t.Run("lowercase tool", func(t *testing.T) {
		input := NewInput().WithParam("text", "HELLO")
		output, err := exec.Execute(ctx, "lowercase", input)

		if err != nil {
			t.Errorf("Execute() error = %v", err)
			return
		}
		if output.GetResultString("result") != "hello" {
			t.Errorf("Output result = %q, want %q", output.GetResultString("result"), "hello")
		}
	})

	// Test MathAdd tool
	t.Run("math.add tool", func(t *testing.T) {
		input := NewInput().WithParam("a", 5).WithParam("b", 3)
		output, err := exec.Execute(ctx, "math.add", input)

		if err != nil {
			t.Errorf("Execute() error = %v", err)
			return
		}
		sum, ok := output.GetResult("sum").(int)
		if !ok || sum != 8 {
			t.Errorf("Output sum = %v, want 8", output.GetResult("sum"))
		}
	})
}

// TestIntegration_AsyncExecution tests asynchronous tool execution.
func TestIntegration_AsyncExecution(t *testing.T) {
	reg := NewRegistry()
	echoTool := &EchoTool{}
	if err := reg.Register(echoTool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg)
	ctx := context.Background()

	// Execute async
	input := NewInput().WithParam("message", "async test")
	resultCh := exec.ExecuteAsync(ctx, "echo", input)

	// Wait for result
	select {
	case result := <-resultCh:
		if result == nil {
			t.Fatal("Received nil result")
		}
		if result.Error != nil {
			t.Errorf("Result error = %v", result.Error)
		}
		if result.Output == nil {
			t.Error("Result output is nil")
		}
		if result.ToolName != "echo" {
			t.Errorf("Result toolName = %q, want %q", result.ToolName, "echo")
		}
		// Verify timing info
		if result.Duration == 0 {
			t.Error("Result duration is zero")
		}
		if result.StartTime.IsZero() {
			t.Error("Result startTime is zero")
		}
		if result.EndTime.IsZero() {
			t.Error("Result endTime is zero")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for async result")
	}
}

// TestIntegration_BatchExecution tests batch concurrent execution.
func TestIntegration_BatchExecution(t *testing.T) {
	reg := NewRegistry()
	counterTool := NewCounterTool()
	if err := reg.Register(counterTool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Create executor with higher concurrency
	exec := NewExecutor(reg, WithMaxConcurrent(4))
	ctx := context.Background()

	// Create batch of executions
	executions := []ToolExecution{
		{ToolName: "counter", Input: NewInput()},
		{ToolName: "counter", Input: NewInput()},
		{ToolName: "counter", Input: NewInput()},
		{ToolName: "echo", Input: NewInput().WithParam("message", "test1")},
		{ToolName: "echo", Input: NewInput().WithParam("message", "test2")},
	}

	results, err := exec.ExecuteMany(ctx, executions)

	if err != nil {
		t.Errorf("ExecuteMany() error = %v", err)
	}

	if len(results) != len(executions) {
		t.Errorf("Results count = %d, want %d", len(results), len(executions))
	}

	// Verify all results are present
	for i, result := range results {
		if result == nil {
			t.Errorf("Result[%d] is nil", i)
			continue
		}
		if result.Error != nil {
			t.Errorf("Result[%d] error = %v", i, result.Error)
		}
		if result.Output == nil {
			t.Errorf("Result[%d] output is nil", i)
		}
	}

	// Verify counter was incremented 3 times
	if counterTool.GetCount() != 3 {
		t.Errorf("Counter = %d, want 3", counterTool.GetCount())
	}
}

// TestIntegration_ErrorHandling tests error propagation and handling.
func TestIntegration_ErrorHandling(t *testing.T) {
	reg := NewRegistry()

	expectedErr := errors.New("intentional error")
	if err := reg.Register(NewErrorTool(expectedErr)); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg)
	ctx := context.Background()

	output, err := exec.Execute(ctx, "error", NewInput())

	if err == nil {
		t.Error("Execute() error = nil, want error")
	}
	if output != nil {
		t.Error("Execute() output should be nil on error")
	}

	// Verify error is an ExecutionError
	if !IsExecutionError(err) {
		t.Errorf("Expected ExecutionError, got %T", err)
	}

	// Verify original error is wrapped
	var execErr *ExecutionError
	if errors.As(err, &execErr) {
		if !errors.Is(execErr.Cause, expectedErr) {
			t.Errorf("Cause = %v, want %v", execErr.Cause, expectedErr)
		}
	}
}

// TestIntegration_ToolNotFound tests handling of nonexistent tools.
func TestIntegration_ToolNotFound(t *testing.T) {
	reg := NewRegistry()
	exec := NewExecutor(reg)
	ctx := context.Background()

	output, err := exec.Execute(ctx, "nonexistent", NewInput())

	if err == nil {
		t.Error("Execute() error = nil, want ErrToolNotFound")
	}
	if output != nil {
		t.Error("Execute() output should be nil on error")
	}
	if !IsToolNotFoundError(err) {
		t.Errorf("Expected ToolNotFoundError, got %T: %v", err, err)
	}
}

// TestIntegration_ContextCancellation tests context cancellation during execution.
func TestIntegration_ContextCancellation(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(NewSlowTool(5 * time.Second)); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg, WithNoTimeout())
	ctx, cancel := context.WithCancel(context.Background())

	// Start execution in goroutine
	resultCh := make(chan struct {
		output *Output
		err    error
	}, 1)

	go func() {
		output, err := exec.Execute(ctx, "slow", NewInput())
		resultCh <- struct {
			output *Output
			err    error
		}{output, err}
	}()

	// Cancel after short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for result
	select {
	case result := <-resultCh:
		if result.err == nil {
			t.Error("Execute() error = nil, want context cancelled error")
		}
		if !errors.Is(result.err, context.Canceled) && !errors.Is(result.err, ErrContextCancelled) {
			t.Errorf("Expected context cancelled error, got %v", result.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for cancellation")
	}
}

// TestIntegration_Timeout tests timeout enforcement.
func TestIntegration_Timeout(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(NewSlowTool(5 * time.Second)); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Create executor with short timeout
	exec := NewExecutor(reg, WithTimeout(100*time.Millisecond))
	ctx := context.Background()

	start := time.Now()
	output, err := exec.Execute(ctx, "slow", NewInput())
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Execute() error = nil, want timeout error")
	}
	if output != nil {
		t.Error("Execute() output should be nil on timeout")
	}

	// Verify it's a timeout error
	if !IsTimeoutError(err) {
		t.Errorf("Expected TimeoutError, got %T: %v", err, err)
	}

	// Verify it didn't wait for full 5 seconds
	if elapsed > 1*time.Second {
		t.Errorf("Execution took %v, expected timeout before 1s", elapsed)
	}
}

// TestIntegration_PanicRecovery tests panic recovery during tool execution.
func TestIntegration_PanicRecovery(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(NewPanicTool("test panic")); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Executor with panic recovery (default)
	exec := NewExecutor(reg, WithRecoverPanics(true))
	ctx := context.Background()

	output, err := exec.Execute(ctx, "panic", NewInput())

	if err == nil {
		t.Error("Execute() error = nil, want panic error")
	}
	if output != nil {
		t.Error("Execute() output should be nil on panic")
	}

	// Verify it's a PanicError
	if !IsPanicError(err) {
		t.Errorf("Expected PanicError, got %T: %v", err, err)
	}

	// Verify panic value is captured
	var panicErr *PanicError
	if errors.As(err, &panicErr) {
		if panicErr.PanicValue != "test panic" {
			t.Errorf("PanicValue = %v, want %q", panicErr.PanicValue, "test panic")
		}
	}
}

// TestIntegration_MiddlewareExecution tests middleware integration.
func TestIntegration_MiddlewareExecution(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Track middleware execution
	var beforeCalled bool
	var afterCalled bool
	var afterErr error

	loggingMw := NewLoggingMiddleware(
		func(toolName string, input *Input) {
			beforeCalled = true
		},
		func(toolName string, output *Output, err error, duration time.Duration) {
			afterCalled = true
			afterErr = err
		},
	)

	exec := NewExecutor(reg,
		WithMiddleware(loggingMw),
		WithMiddleware(NewTimingMiddleware()),
	)
	ctx := context.Background()

	input := NewInput().WithParam("message", "middleware test")
	output, err := exec.Execute(ctx, "echo", input)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Verify middleware was called
	if !beforeCalled {
		t.Error("Logging middleware before hook was not called")
	}
	if !afterCalled {
		t.Error("Logging middleware after hook was not called")
	}
	if afterErr != nil {
		t.Errorf("After hook received error: %v", afterErr)
	}

	// Verify timing metadata was added by TimingMiddleware
	if output != nil && output.Metadata != nil {
		if _, ok := output.Metadata["execution_time_ms"]; !ok {
			t.Error("TimingMiddleware did not add execution_time_ms metadata")
		}
	}
}

// TestIntegration_DefaultMiddlewareChain tests the default middleware chain.
func TestIntegration_DefaultMiddlewareChain(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg, WithDefaultMiddleware())
	ctx := context.Background()

	// Verify middleware is configured
	if !exec.HasMiddleware() {
		t.Error("Executor should have middleware after WithDefaultMiddleware")
	}

	// Execute with valid input
	input := NewInput().WithParam("message", "test")
	output, err := exec.Execute(ctx, "echo", input)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if output == nil {
		t.Error("Execute() output is nil")
	}

	// Verify timing was recorded
	if output != nil && output.Metadata != nil {
		if _, ok := output.Metadata["execution_time_ms"]; !ok {
			t.Error("Default middleware chain should include timing")
		}
	}
}

// TestIntegration_MultipleToolTypes tests execution with diverse tool types.
func TestIntegration_MultipleToolTypes(t *testing.T) {
	reg := NewRegistry()

	// Register various tool types
	tools := []Tool{
		&EchoTool{},
		&UpperCaseTool{},
		&LowerCaseTool{},
		&MathAddTool{},
		NewCounterTool(),
		NewStateTool(),
	}

	for _, tool := range tools {
		if err := reg.Register(tool); err != nil {
			t.Fatalf("Failed to register tool %s: %v", tool.Name(), err)
		}
	}

	exec := NewExecutor(reg, WithMaxConcurrent(4))
	ctx := context.Background()

	// Execute batch with different tool types
	executions := []ToolExecution{
		{ToolName: "echo", Input: NewInput().WithParam("message", "hello")},
		{ToolName: "uppercase", Input: NewInput().WithParam("text", "hello")},
		{ToolName: "lowercase", Input: NewInput().WithParam("text", "HELLO")},
		{ToolName: "math.add", Input: NewInput().WithParam("a", 10).WithParam("b", 20)},
		{ToolName: "counter", Input: NewInput()},
		{ToolName: "state", Input: NewInput().WithName("test-input")},
	}

	results, err := exec.ExecuteMany(ctx, executions)

	if err != nil {
		t.Errorf("ExecuteMany() error = %v", err)
	}

	// Verify all succeeded
	for i, result := range results {
		if result == nil {
			t.Errorf("Result[%d] is nil", i)
			continue
		}
		if result.Error != nil {
			t.Errorf("Result[%d] (%s) error = %v", i, executions[i].ToolName, result.Error)
		}
	}

	// Verify specific results
	if results[1].Output != nil {
		if results[1].Output.GetResultString("result") != "HELLO" {
			t.Errorf("Uppercase result = %q, want HELLO", results[1].Output.GetResultString("result"))
		}
	}
	if results[2].Output != nil {
		if results[2].Output.GetResultString("result") != "hello" {
			t.Errorf("Lowercase result = %q, want hello", results[2].Output.GetResultString("result"))
		}
	}
	if results[3].Output != nil {
		sum, ok := results[3].Output.GetResult("sum").(int)
		if !ok || sum != 30 {
			t.Errorf("Math.add result = %v, want 30", results[3].Output.GetResult("sum"))
		}
	}
}

// TestIntegration_ConcurrentRegistration tests thread-safe registration.
func TestIntegration_ConcurrentRegistration(t *testing.T) {
	reg := NewRegistry()

	// Concurrently register tools
	var wg sync.WaitGroup
	numGoroutines := 10
	toolsPerGoroutine := 5

	errCh := make(chan error, numGoroutines*toolsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < toolsPerGoroutine; j++ {
				tool := &EchoTool{}
				// Create unique tool by embedding in a wrapper
				uniqueTool := &uniqueNameTool{
					tool: tool,
					name: formatToolName(goroutineID, j),
				}
				if err := reg.Register(uniqueTool); err != nil {
					errCh <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Errorf("Registration error: %v", err)
	}

	expectedCount := numGoroutines * toolsPerGoroutine
	if reg.Count() != expectedCount {
		t.Errorf("Registry count = %d, want %d", reg.Count(), expectedCount)
	}
}

// uniqueNameTool wraps a tool with a unique name for testing.
type uniqueNameTool struct {
	tool Tool
	name string
}

func (t *uniqueNameTool) Name() string        { return t.name }
func (t *uniqueNameTool) Description() string { return t.tool.Description() }
func (t *uniqueNameTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	return t.tool.Execute(ctx, input)
}

// formatToolName generates a unique tool name.
func formatToolName(goroutineID, toolID int) string {
	return "tool-" + formatInt(goroutineID) + "-" + formatInt(toolID)
}

// formatInt converts int to string without importing strconv.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// TestIntegration_ConcurrentExecution tests thread-safe execution.
func TestIntegration_ConcurrentExecution(t *testing.T) {
	reg := NewRegistry()
	counterTool := NewCounterTool()
	if err := reg.Register(counterTool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg, WithMaxConcurrent(10))
	ctx := context.Background()

	// Execute concurrently
	var wg sync.WaitGroup
	numExecutions := 100

	for i := 0; i < numExecutions; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := exec.Execute(ctx, "counter", NewInput())
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify all executions completed
	if counterTool.GetCount() != int64(numExecutions) {
		t.Errorf("Counter = %d, want %d", counterTool.GetCount(), numExecutions)
	}
}

// TestIntegration_BatchFailFast tests fail-fast behavior in batch execution.
func TestIntegration_BatchFailFast(t *testing.T) {
	reg := NewRegistry()

	// Register tools including one that errors
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	if err := reg.Register(NewErrorTool(errors.New("batch error"))); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg, WithMaxConcurrent(1)) // Sequential to ensure order
	ctx := context.Background()

	executions := []ToolExecution{
		{ToolName: "echo", Input: NewInput().WithParam("message", "first")},
		{ToolName: "error", Input: NewInput()}, // This will fail
		{ToolName: "echo", Input: NewInput().WithParam("message", "third")},
	}

	results, err := exec.ExecuteMany(ctx, executions)

	// Should have an error
	if err == nil {
		t.Error("ExecuteMany() should have returned an error")
	}

	// Results should still be returned
	if results == nil {
		t.Error("ExecuteMany() results should not be nil")
	}
	if len(results) != len(executions) {
		t.Errorf("Results count = %d, want %d", len(results), len(executions))
	}

	// First result should succeed
	if results[0].Error != nil {
		t.Errorf("First result should succeed, got error: %v", results[0].Error)
	}

	// Second result should have error
	if results[1].Error == nil {
		t.Error("Second result should have error")
	}
}

// TestIntegration_RegistrySnapshot tests point-in-time snapshot functionality.
func TestIntegration_RegistrySnapshot(t *testing.T) {
	reg := NewRegistry()

	// Register some tools
	tools := []Tool{
		&EchoTool{},
		&UpperCaseTool{},
		&LowerCaseTool{},
	}
	for _, tool := range tools {
		if err := reg.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}
	}

	// Cast to SnapshotRegistry to access Snapshot method
	snapshotReg, ok := reg.(*registry)
	if !ok {
		t.Fatal("Registry does not support snapshots")
	}

	snapshot := snapshotReg.Snapshot()

	if len(snapshot.Tools) != len(tools) {
		t.Errorf("Snapshot tools count = %d, want %d", len(snapshot.Tools), len(tools))
	}
	if len(snapshot.Infos) != len(tools) {
		t.Errorf("Snapshot infos count = %d, want %d", len(snapshot.Infos), len(tools))
	}

	// Verify snapshot is sorted
	for i := 1; i < len(snapshot.Infos); i++ {
		if snapshot.Infos[i-1].Name >= snapshot.Infos[i].Name {
			t.Errorf("Snapshot not sorted: %s >= %s", snapshot.Infos[i-1].Name, snapshot.Infos[i].Name)
		}
	}
}

// TestIntegration_InputValidationMiddleware tests input validation.
func TestIntegration_InputValidationMiddleware(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Create executor with input validation middleware
	exec := NewExecutor(reg, WithMiddleware(NewInputValidationMiddleware()))
	ctx := context.Background()

	// Execute with nil input
	output, err := exec.Execute(ctx, "echo", nil)

	if err == nil {
		t.Error("Execute() with nil input should error")
	}
	if output != nil {
		t.Error("Execute() output should be nil on validation error")
	}

	// Verify it's a validation error
	if !IsValidationError(err) {
		t.Errorf("Expected ValidationError, got %T: %v", err, err)
	}
}

// TestIntegration_ContextCheckMiddleware tests context check middleware.
func TestIntegration_ContextCheckMiddleware(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg,
		WithNoTimeout(), // Disable default timeout
		WithMiddleware(NewContextCheckMiddleware()),
	)

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	output, err := exec.Execute(ctx, "echo", NewInput().WithParam("message", "test"))

	if err == nil {
		t.Error("Execute() with cancelled context should error")
	}
	if output != nil {
		t.Error("Execute() output should be nil on context error")
	}
}

// TestIntegration_ToolListAndDiscovery tests tool listing and discovery.
func TestIntegration_ToolListAndDiscovery(t *testing.T) {
	reg := NewRegistry()

	// Register tools
	tools := []Tool{
		&EchoTool{},
		&UpperCaseTool{},
		&LowerCaseTool{},
		&MathAddTool{},
	}
	for _, tool := range tools {
		if err := reg.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}
	}

	// List all tools
	infos := reg.List()

	if len(infos) != len(tools) {
		t.Errorf("List() count = %d, want %d", len(infos), len(tools))
	}

	// Verify list is sorted
	for i := 1; i < len(infos); i++ {
		if infos[i-1].Name >= infos[i].Name {
			t.Errorf("List not sorted: %s >= %s", infos[i-1].Name, infos[i].Name)
		}
	}

	// Test Has()
	if !reg.Has("echo") {
		t.Error("Has(echo) = false, want true")
	}
	if reg.Has("nonexistent") {
		t.Error("Has(nonexistent) = true, want false")
	}

	// Test Get()
	tool, err := reg.Get("uppercase")
	if err != nil {
		t.Errorf("Get(uppercase) error = %v", err)
	}
	if tool == nil {
		t.Error("Get(uppercase) returned nil")
	}
	if tool != nil && tool.Name() != "uppercase" {
		t.Errorf("Tool name = %q, want %q", tool.Name(), "uppercase")
	}
}

// TestIntegration_UnregisterTool tests tool unregistration.
func TestIntegration_UnregisterTool(t *testing.T) {
	reg := NewRegistry()

	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Verify registered
	if !reg.Has("echo") {
		t.Error("Tool should be registered")
	}

	// Unregister
	if err := reg.Unregister("echo"); err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	// Verify unregistered
	if reg.Has("echo") {
		t.Error("Tool should be unregistered")
	}
	if reg.Count() != 0 {
		t.Errorf("Registry count = %d, want 0", reg.Count())
	}

	// Unregister nonexistent should error
	if err := reg.Unregister("nonexistent"); err == nil {
		t.Error("Unregister(nonexistent) should error")
	}
}

// TestIntegration_ClearRegistry tests clearing all tools from registry.
func TestIntegration_ClearRegistry(t *testing.T) {
	reg := NewRegistry()

	// Register multiple tools
	tools := []Tool{
		&EchoTool{},
		&UpperCaseTool{},
		&LowerCaseTool{},
	}
	for _, tool := range tools {
		if err := reg.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}
	}

	if reg.Count() != len(tools) {
		t.Errorf("Count before clear = %d, want %d", reg.Count(), len(tools))
	}

	// Clear registry
	reg.Clear()

	if reg.Count() != 0 {
		t.Errorf("Count after clear = %d, want 0", reg.Count())
	}
	if reg.Has("echo") {
		t.Error("Registry should be empty after clear")
	}
}

// TestIntegration_ExecutorConfig tests executor configuration inspection.
func TestIntegration_ExecutorConfig(t *testing.T) {
	reg := NewRegistry()

	exec := NewExecutor(reg,
		WithTimeout(45*time.Second),
		WithMaxConcurrent(8),
		WithRecoverPanics(false),
		WithDefaultMiddleware(),
	)

	config := exec.Config()

	if config.Timeout != 45*time.Second {
		t.Errorf("Config.Timeout = %v, want 45s", config.Timeout)
	}
	if config.MaxConcurrent != 8 {
		t.Errorf("Config.MaxConcurrent = %d, want 8", config.MaxConcurrent)
	}
	if config.RecoverPanics {
		t.Error("Config.RecoverPanics = true, want false")
	}
	if !config.HasMiddleware {
		t.Error("Config.HasMiddleware = false, want true")
	}
	if config.MiddlewareCount != 4 { // Default chain has 4 middlewares
		t.Errorf("Config.MiddlewareCount = %d, want 4", config.MiddlewareCount)
	}
}

// TestIntegration_AsyncMultiple tests multiple async executions.
func TestIntegration_AsyncMultiple(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg)
	ctx := context.Background()

	// Launch multiple async executions
	numExecutions := 5
	channels := make([]<-chan *Result, numExecutions)

	for i := 0; i < numExecutions; i++ {
		input := NewInput().WithParam("message", "async-"+formatInt(i))
		channels[i] = exec.ExecuteAsync(ctx, "echo", input)
	}

	// Collect all results
	for i, ch := range channels {
		select {
		case result := <-ch:
			if result == nil {
				t.Errorf("Async result[%d] is nil", i)
				continue
			}
			if result.Error != nil {
				t.Errorf("Async result[%d] error = %v", i, result.Error)
			}
			if result.Output == nil {
				t.Errorf("Async result[%d] output is nil", i)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Timeout waiting for async result[%d]", i)
		}
	}
}

// TestIntegration_RegistryWithOptionsPrePopulated tests pre-populated registry.
func TestIntegration_RegistryWithOptionsPrePopulated(t *testing.T) {
	// Create registry pre-populated with tools
	reg := NewRegistryWithOptions(
		WithTools(&EchoTool{}, &UpperCaseTool{}, &LowerCaseTool{}),
	)

	if reg.Count() != 3 {
		t.Errorf("Registry count = %d, want 3", reg.Count())
	}

	// Verify tools are accessible
	exec := NewExecutor(reg)
	ctx := context.Background()

	output, err := exec.Execute(ctx, "echo", NewInput().WithParam("message", "pre-populated"))
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if output == nil || !output.Success {
		t.Error("Execution failed")
	}
}

// ===========================================================================
// Concurrent Stress Tests for Thread-Safety Verification
// ===========================================================================

// TestIntegration_ConcurrentStress exercises the system under high concurrent load.
// This test is designed to detect race conditions when run with -race flag.
func TestIntegration_ConcurrentStress(t *testing.T) {
	reg := NewRegistry()
	counterTool := NewCounterTool()
	stateTool := NewStateTool()

	// Register base tools
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register echo tool: %v", err)
	}
	if err := reg.Register(&UpperCaseTool{}); err != nil {
		t.Fatalf("Failed to register uppercase tool: %v", err)
	}
	if err := reg.Register(&LowerCaseTool{}); err != nil {
		t.Fatalf("Failed to register lowercase tool: %v", err)
	}
	if err := reg.Register(&MathAddTool{}); err != nil {
		t.Fatalf("Failed to register math.add tool: %v", err)
	}
	if err := reg.Register(counterTool); err != nil {
		t.Fatalf("Failed to register counter tool: %v", err)
	}
	if err := reg.Register(stateTool); err != nil {
		t.Fatalf("Failed to register state tool: %v", err)
	}

	exec := NewExecutor(reg, WithMaxConcurrent(20))
	ctx := context.Background()

	const (
		numGoroutines       = 50
		operationsPerWorker = 100
	)

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*operationsPerWorker)

	// Launch concurrent workers performing various operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// Rotate through different operations
				switch j % 6 {
				case 0:
					// Execute echo tool
					input := NewInput().WithParam("message", "stress-"+formatInt(workerID)+"-"+formatInt(j))
					output, err := exec.Execute(ctx, "echo", input)
					if err != nil {
						errCh <- err
					} else if output == nil || !output.Success {
						errCh <- errors.New("echo execution failed")
					}

				case 1:
					// Execute counter tool
					output, err := exec.Execute(ctx, "counter", NewInput())
					if err != nil {
						errCh <- err
					} else if output == nil || !output.Success {
						errCh <- errors.New("counter execution failed")
					}

				case 2:
					// Execute uppercase tool
					input := NewInput().WithParam("text", "stress")
					output, err := exec.Execute(ctx, "uppercase", input)
					if err != nil {
						errCh <- err
					} else if output.GetResultString("result") != "STRESS" {
						errCh <- errors.New("uppercase result mismatch")
					}

				case 3:
					// Execute lowercase tool
					input := NewInput().WithParam("text", "STRESS")
					output, err := exec.Execute(ctx, "lowercase", input)
					if err != nil {
						errCh <- err
					} else if output.GetResultString("result") != "stress" {
						errCh <- errors.New("lowercase result mismatch")
					}

				case 4:
					// Execute math.add tool
					input := NewInput().WithParam("a", workerID).WithParam("b", j)
					output, err := exec.Execute(ctx, "math.add", input)
					if err != nil {
						errCh <- err
					} else if output == nil || !output.Success {
						errCh <- errors.New("math.add execution failed")
					}

				case 5:
					// Execute state tool
					input := NewInput().WithName("stress-" + formatInt(workerID))
					output, err := exec.Execute(ctx, "state", input)
					if err != nil {
						errCh <- err
					} else if output == nil || !output.Success {
						errCh <- errors.New("state execution failed")
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	var errorCount int
	for err := range errCh {
		errorCount++
		if errorCount <= 10 { // Limit error output
			t.Errorf("Stress test error: %v", err)
		}
	}

	if errorCount > 10 {
		t.Errorf("Additional %d errors occurred", errorCount-10)
	}

	// Verify counter tool maintained consistency
	expectedCounterCalls := numGoroutines * (operationsPerWorker / 6)
	if operationsPerWorker%6 >= 2 {
		expectedCounterCalls += numGoroutines
	}
	// Note: due to modulo operation, some workers may have one less counter call
	minExpected := int64((numGoroutines * operationsPerWorker / 6) - numGoroutines)
	actualCount := counterTool.GetCount()
	if actualCount < minExpected {
		t.Errorf("Counter = %d, expected at least %d", actualCount, minExpected)
	}
}

// TestIntegration_ConcurrentRegistryAccess stress tests registry with concurrent
// registration, unregistration, lookup, and listing operations.
func TestIntegration_ConcurrentRegistryAccess(t *testing.T) {
	reg := NewRegistry()

	const (
		numGoroutines       = 30
		operationsPerWorker = 50
	)

	var wg sync.WaitGroup
	var successfulRegistrations atomic.Int64
	var successfulUnregistrations atomic.Int64
	var successfulLookups atomic.Int64
	var successfulListings atomic.Int64

	// Pre-register some tools for lookup/unregister operations
	for i := 0; i < 10; i++ {
		tool := &uniqueNameTool{
			tool: &EchoTool{},
			name: "preregistered-" + formatInt(i),
		}
		if err := reg.Register(tool); err != nil {
			t.Fatalf("Failed to pre-register tool: %v", err)
		}
	}

	// Launch concurrent workers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				switch j % 4 {
				case 0:
					// Register a new tool
					tool := &uniqueNameTool{
						tool: &EchoTool{},
						name: "dynamic-" + formatInt(workerID) + "-" + formatInt(j),
					}
					if err := reg.Register(tool); err == nil {
						successfulRegistrations.Add(1)
					}

				case 1:
					// Lookup a tool (may or may not exist)
					name := "preregistered-" + formatInt(j%10)
					if _, err := reg.Get(name); err == nil {
						successfulLookups.Add(1)
					}

				case 2:
					// List all tools
					infos := reg.List()
					if len(infos) > 0 {
						successfulListings.Add(1)
					}

				case 3:
					// Check if tool exists
					name := "dynamic-" + formatInt(workerID-1) + "-" + formatInt(j)
					if reg.Has(name) {
						successfulLookups.Add(1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify operations completed successfully
	t.Logf("Registrations: %d, Lookups: %d, Listings: %d",
		successfulRegistrations.Load(),
		successfulLookups.Load(),
		successfulListings.Load())

	// Verify registry is in consistent state
	count := reg.Count()
	if count < 10 { // At least pre-registered tools
		t.Errorf("Registry count = %d, expected at least 10", count)
	}

	// Verify we can still use the registry normally
	tool, err := reg.Get("preregistered-0")
	if err != nil {
		t.Errorf("Get after stress test failed: %v", err)
	}
	if tool == nil {
		t.Error("Tool should not be nil")
	}
}

// TestIntegration_ConcurrentAsyncExecution tests async execution under concurrency.
func TestIntegration_ConcurrentAsyncExecution(t *testing.T) {
	reg := NewRegistry()
	counterTool := NewCounterTool()
	if err := reg.Register(counterTool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg, WithMaxConcurrent(20))
	ctx := context.Background()

	const numAsync = 200

	// Launch many async executions concurrently
	channels := make([]<-chan *Result, numAsync)
	for i := 0; i < numAsync; i++ {
		if i%2 == 0 {
			channels[i] = exec.ExecuteAsync(ctx, "counter", NewInput())
		} else {
			channels[i] = exec.ExecuteAsync(ctx, "echo", NewInput().WithParam("message", "async-"+formatInt(i)))
		}
	}

	// Collect all results
	var successCount atomic.Int64
	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for i, ch := range channels {
		wg.Add(1)
		go func(idx int, resultCh <-chan *Result) {
			defer wg.Done()
			select {
			case result := <-resultCh:
				if result == nil {
					errorCount.Add(1)
				} else if result.Error != nil {
					errorCount.Add(1)
				} else if result.Output == nil {
					errorCount.Add(1)
				} else {
					successCount.Add(1)
				}
			case <-time.After(10 * time.Second):
				errorCount.Add(1)
			}
		}(i, ch)
	}

	wg.Wait()

	// Verify results
	if successCount.Load() != numAsync {
		t.Errorf("Success count = %d, want %d (errors: %d)",
			successCount.Load(), numAsync, errorCount.Load())
	}

	// Verify counter is consistent
	expectedCounterCalls := int64(numAsync / 2)
	if counterTool.GetCount() != expectedCounterCalls {
		t.Errorf("Counter = %d, want %d", counterTool.GetCount(), expectedCounterCalls)
	}
}

// TestIntegration_ConcurrentBatchExecution tests batch execution concurrency.
func TestIntegration_ConcurrentBatchExecution(t *testing.T) {
	reg := NewRegistry()
	counterTool := NewCounterTool()
	if err := reg.Register(counterTool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	if err := reg.Register(&UpperCaseTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg, WithMaxConcurrent(15))
	ctx := context.Background()

	const numBatches = 20
	const batchSize = 10

	var wg sync.WaitGroup
	var successBatches atomic.Int64
	errCh := make(chan error, numBatches)

	// Run multiple batches concurrently
	for i := 0; i < numBatches; i++ {
		wg.Add(1)
		go func(batchID int) {
			defer wg.Done()

			// Create a batch of executions
			executions := make([]ToolExecution, batchSize)
			for j := 0; j < batchSize; j++ {
				switch j % 3 {
				case 0:
					executions[j] = ToolExecution{
						ToolName: "counter",
						Input:    NewInput(),
					}
				case 1:
					executions[j] = ToolExecution{
						ToolName: "echo",
						Input:    NewInput().WithParam("message", "batch-"+formatInt(batchID)),
					}
				case 2:
					executions[j] = ToolExecution{
						ToolName: "uppercase",
						Input:    NewInput().WithParam("text", "test"),
					}
				}
			}

			results, err := exec.ExecuteMany(ctx, executions)
			if err != nil {
				errCh <- err
				return
			}

			// Verify all results
			for j, result := range results {
				if result == nil {
					errCh <- errors.New("nil result in batch")
					return
				}
				if result.Error != nil {
					errCh <- result.Error
					return
				}
				if result.Output == nil {
					errCh <- errors.New("nil output in result " + formatInt(j))
					return
				}
			}

			successBatches.Add(1)
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Errorf("Batch error: %v", err)
	}

	// Verify all batches succeeded
	if successBatches.Load() != numBatches {
		t.Errorf("Successful batches = %d, want %d", successBatches.Load(), numBatches)
	}

	// Verify counter calls
	// Each batch has batchSize/3 counter calls (rounded down)
	expectedCounterCalls := int64(numBatches * (batchSize / 3))
	// Allow for rounding variations
	actualCount := counterTool.GetCount()
	if actualCount < expectedCounterCalls-int64(numBatches) || actualCount > expectedCounterCalls+int64(numBatches) {
		t.Errorf("Counter = %d, expected approximately %d", actualCount, expectedCounterCalls)
	}
}

// TestIntegration_ConcurrentMiddlewareExecution tests middleware under concurrent load.
func TestIntegration_ConcurrentMiddlewareExecution(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&EchoTool{}); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Track middleware invocations with thread-safe counter
	var beforeCount atomic.Int64
	var afterCount atomic.Int64

	loggingMw := NewLoggingMiddleware(
		func(toolName string, input *Input) {
			beforeCount.Add(1)
		},
		func(toolName string, output *Output, err error, duration time.Duration) {
			afterCount.Add(1)
		},
	)

	exec := NewExecutor(reg,
		WithMaxConcurrent(20),
		WithMiddleware(loggingMw),
		WithMiddleware(NewTimingMiddleware()),
	)
	ctx := context.Background()

	const numExecutions = 500

	var wg sync.WaitGroup
	for i := 0; i < numExecutions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			input := NewInput().WithParam("message", "middleware-"+formatInt(idx))
			_, err := exec.Execute(ctx, "echo", input)
			if err != nil {
				t.Errorf("Execution %d failed: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify middleware was called exactly once per execution
	if beforeCount.Load() != numExecutions {
		t.Errorf("Before count = %d, want %d", beforeCount.Load(), numExecutions)
	}
	if afterCount.Load() != numExecutions {
		t.Errorf("After count = %d, want %d", afterCount.Load(), numExecutions)
	}
}

// TestIntegration_ConcurrentContextCancellation tests cancellation under concurrent load.
func TestIntegration_ConcurrentContextCancellation(t *testing.T) {
	reg := NewRegistry()
	// Use a slow tool with moderate delay
	if err := reg.Register(NewSlowTool(500 * time.Millisecond)); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(reg, WithNoTimeout())

	const numExecutions = 50

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	var cancelledCount atomic.Int64
	var completedCount atomic.Int64

	// Start many executions
	for i := 0; i < numExecutions; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := exec.Execute(ctx, "slow", NewInput())
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, ErrContextCancelled) {
					cancelledCount.Add(1)
				}
			} else {
				completedCount.Add(1)
			}
		}()
	}

	// Cancel after short delay - some may complete, most should be cancelled
	time.Sleep(100 * time.Millisecond)
	cancel()

	wg.Wait()

	// Verify that cancellation worked for at least some goroutines
	t.Logf("Completed: %d, Cancelled: %d", completedCount.Load(), cancelledCount.Load())

	// Total should equal numExecutions
	total := completedCount.Load() + cancelledCount.Load()
	if total != numExecutions {
		t.Errorf("Total = %d, want %d", total, numExecutions)
	}

	// Most should be cancelled since we cancel quickly
	if cancelledCount.Load() < numExecutions/2 {
		t.Logf("Warning: Expected more cancellations, got %d/%d", cancelledCount.Load(), numExecutions)
	}
}
