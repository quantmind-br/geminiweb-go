package toolexec

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockTool is a mock implementation of the Tool interface for testing.
type MockTool struct {
	name                 string
	description          string
	executeFunc          func(ctx context.Context, input *Input) (*Output, error)
	requiresConfirmation bool
}

// NewMockTool creates a new MockTool with the given name and description.
func NewMockTool(name, description string) *MockTool {
	return &MockTool{
		name:        name,
		description: description,
	}
}

// Name implements Tool.Name.
func (m *MockTool) Name() string {
	return m.name
}

// Description implements Tool.Description.
func (m *MockTool) Description() string {
	return m.description
}

// Execute implements Tool.Execute.
func (m *MockTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return NewOutput().WithMessage("mock executed"), nil
}

// RequiresConfirmation implements Tool.RequiresConfirmation.
func (m *MockTool) RequiresConfirmation(args map[string]any) bool {
	return m.requiresConfirmation
}

// WithExecuteFunc sets a custom execute function for the mock tool.
func (m *MockTool) WithExecuteFunc(fn func(ctx context.Context, input *Input) (*Output, error)) *MockTool {
	m.executeFunc = fn
	return m
}

// WithRequiresConfirmation sets whether the mock tool requires confirmation.
func (m *MockTool) WithRequiresConfirmation(requires bool) *MockTool {
	m.requiresConfirmation = requires
	return m
}

// TestNewExecutor tests the NewExecutor function.
func TestNewExecutor(t *testing.T) {
	tests := []struct {
		name           string
		registry       Registry
		opts           []ExecutorOption
		wantTimeout    time.Duration
		wantConcurrent int
		wantRecovers   bool
	}{
		{
			name:           "default configuration",
			registry:       NewRegistry(),
			wantTimeout:    30 * time.Second,
			wantConcurrent: 1,
			wantRecovers:   true,
		},
		{
			name:           "nil registry uses default",
			registry:       nil,
			wantTimeout:    30 * time.Second,
			wantConcurrent: 1,
			wantRecovers:   true,
		},
		{
			name:           "custom timeout",
			registry:       NewRegistry(),
			opts:           []ExecutorOption{WithTimeout(60 * time.Second)},
			wantTimeout:    60 * time.Second,
			wantConcurrent: 1,
			wantRecovers:   true,
		},
		{
			name:           "custom concurrency",
			registry:       NewRegistry(),
			opts:           []ExecutorOption{WithMaxConcurrent(4)},
			wantTimeout:    30 * time.Second,
			wantConcurrent: 4,
			wantRecovers:   true,
		},
		{
			name:           "disable panic recovery",
			registry:       NewRegistry(),
			opts:           []ExecutorOption{WithRecoverPanics(false)},
			wantTimeout:    30 * time.Second,
			wantConcurrent: 1,
			wantRecovers:   false,
		},
		{
			name:           "no timeout",
			registry:       NewRegistry(),
			opts:           []ExecutorOption{WithNoTimeout()},
			wantTimeout:    0,
			wantConcurrent: 1,
			wantRecovers:   true,
		},
		{
			name:           "unlimited concurrency",
			registry:       NewRegistry(),
			opts:           []ExecutorOption{WithUnlimitedConcurrency()},
			wantTimeout:    30 * time.Second,
			wantConcurrent: -1,
			wantRecovers:   true,
		},
		{
			name:     "multiple options",
			registry: NewRegistry(),
			opts: []ExecutorOption{
				WithTimeout(45 * time.Second),
				WithMaxConcurrent(8),
				WithRecoverPanics(false),
			},
			wantTimeout:    45 * time.Second,
			wantConcurrent: 8,
			wantRecovers:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewExecutor(tt.registry, tt.opts...)

			if exec == nil {
				t.Error("NewExecutor() returned nil")
				return
			}

			if exec.GetTimeout() != tt.wantTimeout {
				t.Errorf("GetTimeout() = %v, want %v", exec.GetTimeout(), tt.wantTimeout)
			}

			if exec.GetMaxConcurrent() != tt.wantConcurrent {
				t.Errorf("GetMaxConcurrent() = %v, want %v", exec.GetMaxConcurrent(), tt.wantConcurrent)
			}

			if exec.RecoversPanics() != tt.wantRecovers {
				t.Errorf("RecoversPanics() = %v, want %v", exec.RecoversPanics(), tt.wantRecovers)
			}
		})
	}
}

// TestExecutor_Execute tests the Execute method.
func TestExecutor_Execute(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		setupTool  func() *MockTool
		input      *Input
		wantOutput bool
		wantErr    bool
		errCheck   func(error) bool
	}{
		{
			name:     "successful execution",
			toolName: "test-tool",
			setupTool: func() *MockTool {
				return NewMockTool("test-tool", "A test tool").WithExecuteFunc(
					func(ctx context.Context, input *Input) (*Output, error) {
						return NewOutput().WithMessage("success"), nil
					},
				)
			},
			input:      NewInput(),
			wantOutput: true,
			wantErr:    false,
		},
		{
			name:     "tool returns error",
			toolName: "error-tool",
			setupTool: func() *MockTool {
				return NewMockTool("error-tool", "A tool that errors").WithExecuteFunc(
					func(ctx context.Context, input *Input) (*Output, error) {
						return nil, errors.New("execution error")
					},
				)
			},
			input:      NewInput(),
			wantOutput: false,
			wantErr:    true,
			errCheck:   IsExecutionError,
		},
		{
			name:       "tool not found",
			toolName:   "nonexistent",
			setupTool:  nil,
			input:      NewInput(),
			wantOutput: false,
			wantErr:    true,
			errCheck:   IsToolNotFoundError,
		},
		{
			name:     "tool with input parameters",
			toolName: "param-tool",
			setupTool: func() *MockTool {
				return NewMockTool("param-tool", "A tool that uses params").WithExecuteFunc(
					func(ctx context.Context, input *Input) (*Output, error) {
						name := input.GetParamString("name")
						return NewOutput().WithMessage("Hello, " + name), nil
					},
				)
			},
			input:      NewInput().WithParam("name", "World"),
			wantOutput: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			if tt.setupTool != nil {
				tool := tt.setupTool()
				if err := registry.Register(tool); err != nil {
					t.Fatalf("Failed to register tool: %v", err)
				}
			}

			exec := NewExecutor(registry)
			output, err := exec.Execute(context.Background(), tt.toolName, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Execute() expected error but got none")
					return
				}
				if tt.errCheck != nil && !tt.errCheck(err) {
					t.Errorf("Execute() error type check failed, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Execute() unexpected error: %v", err)
					return
				}
			}

			if tt.wantOutput && output == nil {
				t.Error("Execute() expected output but got nil")
			}
			if !tt.wantOutput && output != nil {
				t.Error("Execute() expected nil output but got value")
			}
		})
	}
}

// TestExecutor_Execute_PanicRecovery tests panic recovery in Execute.
func TestExecutor_Execute_PanicRecovery(t *testing.T) {
	t.Run("recovers from panic when enabled", func(t *testing.T) {
		registry := NewRegistry()
		panicTool := NewMockTool("panic-tool", "A tool that panics").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				panic("test panic")
			},
		)
		if err := registry.Register(panicTool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry, WithRecoverPanics(true))
		output, err := exec.Execute(context.Background(), "panic-tool", NewInput())

		if output != nil {
			t.Error("Execute() should return nil output on panic")
		}

		if err == nil {
			t.Error("Execute() should return error on panic")
			return
		}

		if !IsPanicError(err) {
			t.Errorf("Execute() error should be PanicError, got: %T", err)
		}
	})

	t.Run("propagates panic when recovery disabled", func(t *testing.T) {
		registry := NewRegistry()
		panicTool := NewMockTool("panic-tool", "A tool that panics").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				panic("test panic")
			},
		)
		if err := registry.Register(panicTool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry, WithRecoverPanics(false))

		defer func() {
			if r := recover(); r == nil {
				t.Error("Execute() should have panicked")
			}
		}()

		_, _ = exec.Execute(context.Background(), "panic-tool", NewInput())
	})
}

// TestExecutor_Execute_Timeout tests timeout handling in Execute.
func TestExecutor_Execute_Timeout(t *testing.T) {
	t.Run("times out with default timeout", func(t *testing.T) {
		registry := NewRegistry()
		slowTool := NewMockTool("slow-tool", "A slow tool").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				select {
				case <-time.After(2 * time.Second):
					return NewOutput(), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		)
		if err := registry.Register(slowTool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry, WithTimeout(100*time.Millisecond))
		_, err := exec.Execute(context.Background(), "slow-tool", NewInput())

		if err == nil {
			t.Error("Execute() should return error on timeout")
			return
		}

		if !IsTimeoutError(err) {
			t.Errorf("Execute() error should be TimeoutError, got: %T (%v)", err, err)
		}
	})

	t.Run("respects context deadline", func(t *testing.T) {
		registry := NewRegistry()
		slowTool := NewMockTool("slow-tool", "A slow tool").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				select {
				case <-time.After(2 * time.Second):
					return NewOutput(), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		)
		if err := registry.Register(slowTool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry, WithNoTimeout()) // No default timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := exec.Execute(ctx, "slow-tool", NewInput())

		if err == nil {
			t.Error("Execute() should return error on context timeout")
			return
		}

		if !IsTimeoutError(err) {
			t.Errorf("Execute() error should be TimeoutError, got: %T (%v)", err, err)
		}
	})
}

// TestExecutor_Execute_ContextCancellation tests context cancellation handling.
func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	t.Run("cancelled context before execution", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("test-tool", "A test tool")
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := exec.Execute(ctx, "test-tool", NewInput())

		if err == nil {
			t.Error("Execute() should return error on cancelled context")
			return
		}

		if !errors.Is(err, ErrContextCancelled) {
			t.Errorf("Execute() error should contain ErrContextCancelled, got: %v", err)
		}
	})

	t.Run("cancelled context during execution", func(t *testing.T) {
		registry := NewRegistry()
		executionStarted := make(chan struct{})
		slowTool := NewMockTool("slow-tool", "A slow tool").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				close(executionStarted)
				select {
				case <-time.After(5 * time.Second):
					return NewOutput(), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		)
		if err := registry.Register(slowTool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry, WithNoTimeout())
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			<-executionStarted
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := exec.Execute(ctx, "slow-tool", NewInput())

		if err == nil {
			t.Error("Execute() should return error on cancelled context")
			return
		}

		if !errors.Is(err, ErrContextCancelled) {
			t.Errorf("Execute() error should contain ErrContextCancelled, got: %v", err)
		}
	})
}

// TestExecutor_Execute_WithMiddleware tests middleware integration.
func TestExecutor_Execute_WithMiddleware(t *testing.T) {
	t.Run("middleware chain is applied", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("test-tool", "A test tool")
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		var order []string
		mw1 := NewMiddlewareFunc("mw1", func(next ToolFunc) ToolFunc {
			return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
				order = append(order, "mw1-before")
				out, err := next(ctx, toolName, input)
				order = append(order, "mw1-after")
				return out, err
			}
		})

		mw2 := NewMiddlewareFunc("mw2", func(next ToolFunc) ToolFunc {
			return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
				order = append(order, "mw2-before")
				out, err := next(ctx, toolName, input)
				order = append(order, "mw2-after")
				return out, err
			}
		})

		exec := NewExecutor(registry, WithMiddleware(mw1), WithMiddleware(mw2))
		_, err := exec.Execute(context.Background(), "test-tool", NewInput())

		if err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
		}

		expectedOrder := []string{"mw1-before", "mw2-before", "mw2-after", "mw1-after"}
		if len(order) != len(expectedOrder) {
			t.Errorf("Middleware order length = %v, want %v", len(order), len(expectedOrder))
		}
		for i, v := range order {
			if i < len(expectedOrder) && v != expectedOrder[i] {
				t.Errorf("Middleware order[%d] = %v, want %v", i, v, expectedOrder[i])
			}
		}
	})

	t.Run("timing middleware adds metadata", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("test-tool", "A test tool")
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry, WithMiddleware(NewTimingMiddleware()))
		output, err := exec.Execute(context.Background(), "test-tool", NewInput())

		if err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
		}

		if output == nil {
			t.Fatal("Execute() returned nil output")
		}

		if output.Metadata == nil {
			t.Fatal("Output.Metadata is nil")
		}

		if _, ok := output.Metadata["execution_time_ms"]; !ok {
			t.Error("Output.Metadata missing 'execution_time_ms'")
		}

		if _, ok := output.Metadata["execution_start"]; !ok {
			t.Error("Output.Metadata missing 'execution_start'")
		}
	})
}

// TestExecutor_ExecuteAsync tests the ExecuteAsync method.
func TestExecutor_ExecuteAsync(t *testing.T) {
	t.Run("successful async execution", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("test-tool", "A test tool").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("async success"), nil
			},
		)
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry)
		resultCh := exec.ExecuteAsync(context.Background(), "test-tool", NewInput())

		result := <-resultCh
		if result == nil {
			t.Fatal("ExecuteAsync() returned nil result")
		}

		if result.Error != nil {
			t.Errorf("ExecuteAsync() unexpected error: %v", result.Error)
		}

		if result.Output == nil {
			t.Error("ExecuteAsync() returned nil output")
		}

		if result.ToolName != "test-tool" {
			t.Errorf("Result.ToolName = %s, want test-tool", result.ToolName)
		}

		if result.Duration == 0 {
			t.Error("Result.Duration should be non-zero")
		}
	})

	t.Run("async execution with error", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("error-tool", "A tool that errors").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				return nil, errors.New("async error")
			},
		)
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry)
		resultCh := exec.ExecuteAsync(context.Background(), "error-tool", NewInput())

		result := <-resultCh
		if result == nil {
			t.Fatal("ExecuteAsync() returned nil result")
		}

		if result.Error == nil {
			t.Error("ExecuteAsync() expected error but got none")
		}

		if result.Output != nil {
			t.Error("ExecuteAsync() should return nil output on error")
		}
	})

	t.Run("channel closes after result", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("test-tool", "A test tool")
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		exec := NewExecutor(registry)
		resultCh := exec.ExecuteAsync(context.Background(), "test-tool", NewInput())

		// Read first result
		<-resultCh

		// Second read should return zero value (channel closed)
		result, ok := <-resultCh
		if ok {
			t.Error("Channel should be closed after result")
		}
		if result != nil {
			t.Error("Second read should return nil")
		}
	})
}

// TestExecutor_ExecuteMany tests the ExecuteMany method.
func TestExecutor_ExecuteMany(t *testing.T) {
	t.Run("empty executions", func(t *testing.T) {
		registry := NewRegistry()
		exec := NewExecutor(registry)

		results, err := exec.ExecuteMany(context.Background(), []ToolExecution{})

		if err != nil {
			t.Errorf("ExecuteMany() unexpected error: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("ExecuteMany() returned %d results, want 0", len(results))
		}
	})

	t.Run("all successful executions", func(t *testing.T) {
		registry := NewRegistry()
		tool1 := NewMockTool("tool-1", "Tool 1").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("tool1"), nil
			},
		)
		tool2 := NewMockTool("tool-2", "Tool 2").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("tool2"), nil
			},
		)
		if err := registry.Register(tool1); err != nil {
			t.Fatalf("Failed to register tool1: %v", err)
		}
		if err := registry.Register(tool2); err != nil {
			t.Fatalf("Failed to register tool2: %v", err)
		}

		exec := NewExecutor(registry, WithMaxConcurrent(2))

		executions := []ToolExecution{
			{ToolName: "tool-1", Input: NewInput()},
			{ToolName: "tool-2", Input: NewInput()},
		}

		results, err := exec.ExecuteMany(context.Background(), executions)

		if err != nil {
			t.Errorf("ExecuteMany() unexpected error: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("ExecuteMany() returned %d results, want 2", len(results))
		}

		for i, result := range results {
			if result.Error != nil {
				t.Errorf("Result[%d] unexpected error: %v", i, result.Error)
			}
			if result.Output == nil {
				t.Errorf("Result[%d] has nil output", i)
			}
		}
	})

	t.Run("partial failure with fail-fast", func(t *testing.T) {
		registry := NewRegistry()
		tool1 := NewMockTool("tool-1", "Tool 1").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				time.Sleep(10 * time.Millisecond) // Delay to ensure tool-2 starts first
				return NewOutput(), nil
			},
		)
		tool2 := NewMockTool("tool-2", "Tool 2").WithExecuteFunc(
			func(ctx context.Context, input *Input) (*Output, error) {
				return nil, errors.New("tool2 failed")
			},
		)
		if err := registry.Register(tool1); err != nil {
			t.Fatalf("Failed to register tool1: %v", err)
		}
		if err := registry.Register(tool2); err != nil {
			t.Fatalf("Failed to register tool2: %v", err)
		}

		exec := NewExecutor(registry, WithMaxConcurrent(2))

		executions := []ToolExecution{
			{ToolName: "tool-1", Input: NewInput()},
			{ToolName: "tool-2", Input: NewInput()},
		}

		results, err := exec.ExecuteMany(context.Background(), executions)

		if err == nil {
			t.Error("ExecuteMany() expected error but got none")
		}

		// Should still return partial results
		if len(results) != 2 {
			t.Errorf("ExecuteMany() returned %d results, want 2", len(results))
		}
	})

	t.Run("results in order", func(t *testing.T) {
		registry := NewRegistry()
		for i := 0; i < 5; i++ {
			name := "tool-" + string(rune('a'+i))
			tool := NewMockTool(name, "Tool "+name).WithExecuteFunc(
				func(ctx context.Context, input *Input) (*Output, error) {
					time.Sleep(time.Duration(5-i) * time.Millisecond) // Vary timing
					return NewOutput(), nil
				},
			)
			if err := registry.Register(tool); err != nil {
				t.Fatalf("Failed to register %s: %v", name, err)
			}
		}

		exec := NewExecutor(registry, WithMaxConcurrent(5))

		executions := []ToolExecution{
			{ToolName: "tool-a", Input: NewInput()},
			{ToolName: "tool-b", Input: NewInput()},
			{ToolName: "tool-c", Input: NewInput()},
			{ToolName: "tool-d", Input: NewInput()},
			{ToolName: "tool-e", Input: NewInput()},
		}

		results, err := exec.ExecuteMany(context.Background(), executions)

		if err != nil {
			t.Errorf("ExecuteMany() unexpected error: %v", err)
		}

		// Verify results are in the same order as executions
		for i, result := range results {
			if result.ToolName != executions[i].ToolName {
				t.Errorf("Result[%d].ToolName = %s, want %s", i, result.ToolName, executions[i].ToolName)
			}
		}
	})

	t.Run("respects concurrency limit", func(t *testing.T) {
		registry := NewRegistry()
		var concurrent int32
		var maxConcurrent int32

		for i := 0; i < 10; i++ {
			name := "tool-" + string(rune('0'+i))
			tool := NewMockTool(name, "Tool "+name).WithExecuteFunc(
				func(ctx context.Context, input *Input) (*Output, error) {
					current := atomic.AddInt32(&concurrent, 1)
					defer atomic.AddInt32(&concurrent, -1)

					// Track max concurrent
					for {
						old := atomic.LoadInt32(&maxConcurrent)
						if current <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, current) {
							break
						}
					}

					time.Sleep(50 * time.Millisecond)
					return NewOutput(), nil
				},
			)
			if err := registry.Register(tool); err != nil {
				t.Fatalf("Failed to register %s: %v", name, err)
			}
		}

		exec := NewExecutor(registry, WithMaxConcurrent(3))

		executions := make([]ToolExecution, 10)
		for i := 0; i < 10; i++ {
			executions[i] = ToolExecution{
				ToolName: "tool-" + string(rune('0'+i)),
				Input:    NewInput(),
			}
		}

		_, err := exec.ExecuteMany(context.Background(), executions)

		if err != nil {
			t.Errorf("ExecuteMany() unexpected error: %v", err)
		}

		if maxConcurrent > 3 {
			t.Errorf("Max concurrent = %d, want <= 3", maxConcurrent)
		}
	})
}

// TestExecutor_ConcurrentAccess tests concurrent access to executor methods.
func TestExecutor_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()
	tool := NewMockTool("test-tool", "A test tool")
	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	exec := NewExecutor(registry, WithMaxConcurrent(10))

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := exec.Execute(context.Background(), "test-tool", NewInput())
			if err != nil {
				t.Errorf("Execute() error: %v", err)
			}
		}()
	}

	wg.Wait()
}

// TestExecutor_GetRegistry tests the GetRegistry method.
func TestExecutor_GetRegistry(t *testing.T) {
	registry := NewRegistry()
	exec := NewExecutor(registry)

	if exec.GetRegistry() != registry {
		t.Error("GetRegistry() returned different registry than passed to NewExecutor()")
	}
}

// TestExecutor_Config tests the Config method.
func TestExecutor_Config(t *testing.T) {
	exec := NewExecutor(
		NewRegistry(),
		WithTimeout(45*time.Second),
		WithMaxConcurrent(4),
		WithRecoverPanics(false),
		WithDefaultMiddleware(),
	)

	config := exec.Config()

	if config.Timeout != 45*time.Second {
		t.Errorf("Config.Timeout = %v, want 45s", config.Timeout)
	}

	if config.MaxConcurrent != 4 {
		t.Errorf("Config.MaxConcurrent = %d, want 4", config.MaxConcurrent)
	}

	if config.RecoverPanics != false {
		t.Error("Config.RecoverPanics = true, want false")
	}

	if !config.HasMiddleware {
		t.Error("Config.HasMiddleware = false, want true")
	}

	if config.MiddlewareCount != 4 { // Default chain has 4 middlewares
		t.Errorf("Config.MiddlewareCount = %d, want 4", config.MiddlewareCount)
	}
}

// TestExecutor_HasMiddleware tests the HasMiddleware method.
func TestExecutor_HasMiddleware(t *testing.T) {
	t.Run("no middleware", func(t *testing.T) {
		exec := NewExecutor(NewRegistry())
		if exec.HasMiddleware() {
			t.Error("HasMiddleware() = true, want false")
		}
	})

	t.Run("with middleware", func(t *testing.T) {
		exec := NewExecutor(NewRegistry(), WithDefaultMiddleware())
		if !exec.HasMiddleware() {
			t.Error("HasMiddleware() = false, want true")
		}
	})
}

// TestExecutor_GetMiddlewareChain tests the GetMiddlewareChain method.
func TestExecutor_GetMiddlewareChain(t *testing.T) {
	t.Run("nil when no middleware", func(t *testing.T) {
		exec := NewExecutor(NewRegistry())
		chain := exec.GetMiddlewareChain()
		if chain != nil {
			t.Error("GetMiddlewareChain() should return nil when no middleware configured")
		}
	})

	t.Run("returns copy of chain", func(t *testing.T) {
		exec := NewExecutor(NewRegistry(), WithDefaultMiddleware())
		chain := exec.GetMiddlewareChain()

		if chain == nil {
			t.Fatal("GetMiddlewareChain() returned nil")
		}

		// Modify the returned chain
		chain.Add(NewTimingMiddleware())

		// Original chain should be unchanged
		originalChain := exec.GetMiddlewareChain()
		if originalChain.Len() != 4 { // Default chain has 4
			t.Errorf("Original chain length changed to %d", originalChain.Len())
		}
	})
}

// TestExecutorOption_CombineOptions tests the CombineOptions function.
func TestExecutorOption_CombineOptions(t *testing.T) {
	defaults := DefaultExecutorOptions()
	custom := []ExecutorOption{WithTimeout(60 * time.Second)}

	combined := CombineOptions(defaults, custom)

	exec := NewExecutor(NewRegistry(), combined...)

	// The custom timeout should override the default
	if exec.GetTimeout() != 60*time.Second {
		t.Errorf("Combined timeout = %v, want 60s", exec.GetTimeout())
	}
}

// TestToolExecution tests the ToolExecution struct.
func TestToolExecution(t *testing.T) {
	exec := ToolExecution{
		ToolName: "test-tool",
		Input:    NewInput().WithParam("key", "value"),
	}

	if exec.ToolName != "test-tool" {
		t.Errorf("ToolExecution.ToolName = %s, want test-tool", exec.ToolName)
	}

	if exec.Input.GetParamString("key") != "value" {
		t.Error("ToolExecution.Input param 'key' != 'value'")
	}
}

// TestExecutorSecurityViolation tests that the executor returns ErrSecurityViolation
// when the security policy blocks a tool execution.
func TestExecutorSecurityViolation(t *testing.T) {
	t.Run("blacklist validator blocks dangerous commands", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("bash", "A bash tool")
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Create executor with blacklist validator
		exec := NewExecutor(registry, WithSecurityPolicy(DefaultBlacklistValidator()))

		// Try to execute a blocked command
		input := NewInput().WithParam("command", "rm -rf /")
		_, err := exec.Execute(context.Background(), "bash", input)

		if err == nil {
			t.Error("Execute() should return error for blocked command")
			return
		}

		if !IsSecurityViolationError(err) {
			t.Errorf("Execute() error should be SecurityViolationError, got: %T (%v)", err, err)
		}

		if !errors.Is(err, ErrSecurityViolation) {
			t.Errorf("Execute() error should wrap ErrSecurityViolation")
		}
	})

	t.Run("path validator blocks sensitive files", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("file_read", "A file read tool")
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Create executor with path validator
		exec := NewExecutor(registry, WithSecurityPolicy(DefaultPathValidator()))

		// Try to read a blocked path
		input := NewInput().WithParam("path", ".env")
		_, err := exec.Execute(context.Background(), "file_read", input)

		if err == nil {
			t.Error("Execute() should return error for blocked path")
			return
		}

		if !IsSecurityViolationError(err) {
			t.Errorf("Execute() error should be SecurityViolationError, got: %T (%v)", err, err)
		}
	})

	t.Run("composite security policy chains validators", func(t *testing.T) {
		registry := NewRegistry()
		bashTool := NewMockTool("bash", "A bash tool")
		fileTool := NewMockTool("file_read", "A file read tool")
		if err := registry.Register(bashTool); err != nil {
			t.Fatalf("Failed to register bash tool: %v", err)
		}
		if err := registry.Register(fileTool); err != nil {
			t.Fatalf("Failed to register file tool: %v", err)
		}

		// Create executor with default composite policy
		exec := NewExecutor(registry, WithSecurityPolicy(DefaultSecurityPolicy()))

		// Test blacklist (bash command)
		input1 := NewInput().WithParam("command", "rm -rf /")
		_, err := exec.Execute(context.Background(), "bash", input1)
		if !IsSecurityViolationError(err) {
			t.Error("Expected SecurityViolationError for blocked bash command")
		}

		// Test path validator (file read)
		input2 := NewInput().WithParam("path", ".ssh/id_rsa")
		_, err = exec.Execute(context.Background(), "file_read", input2)
		if !IsSecurityViolationError(err) {
			t.Error("Expected SecurityViolationError for blocked path")
		}
	})
}

// TestExecutorUserDenied tests that the executor returns ErrUserDenied
// when the user denies confirmation for a tool execution.
func TestExecutorUserDenied(t *testing.T) {
	t.Run("user denies confirmation", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("dangerous-tool", "A dangerous tool").
			WithRequiresConfirmation(true)
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Create executor with auto-deny handler
		exec := NewExecutor(registry, WithConfirmationHandler(&AutoDenyHandler{}))

		_, err := exec.Execute(context.Background(), "dangerous-tool", NewInput())

		if err == nil {
			t.Error("Execute() should return error when user denies")
			return
		}

		if !IsUserDeniedError(err) {
			t.Errorf("Execute() error should be UserDeniedError, got: %T (%v)", err, err)
		}

		if !errors.Is(err, ErrUserDenied) {
			t.Errorf("Execute() error should wrap ErrUserDenied")
		}
	})

	t.Run("user approves confirmation", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("dangerous-tool", "A dangerous tool").
			WithRequiresConfirmation(true).
			WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("executed"), nil
			})
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Create executor with auto-approve handler
		exec := NewExecutor(registry, WithConfirmationHandler(&AutoApproveHandler{}))

		output, err := exec.Execute(context.Background(), "dangerous-tool", NewInput())

		if err != nil {
			t.Errorf("Execute() unexpected error: %v", err)
		}

		if output == nil {
			t.Error("Execute() returned nil output")
		}
	})

	t.Run("tool not requiring confirmation skips handler", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("safe-tool", "A safe tool").
			WithRequiresConfirmation(false).
			WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("executed"), nil
			})
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Create executor with auto-deny handler (should not be called)
		exec := NewExecutor(registry, WithConfirmationHandler(&AutoDenyHandler{}))

		output, err := exec.Execute(context.Background(), "safe-tool", NewInput())

		if err != nil {
			t.Errorf("Execute() unexpected error: %v", err)
		}

		if output == nil {
			t.Error("Execute() returned nil output")
		}
	})
}

// TestNilSecurityPolicy tests that the executor handles nil security policy gracefully.
func TestNilSecurityPolicy(t *testing.T) {
	t.Run("nil security policy skips validation", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("bash", "A bash tool").
			WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("executed"), nil
			})
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Create executor without security policy (nil by default)
		exec := NewExecutor(registry)

		// Execute a command that would be blocked by security policy
		input := NewInput().WithParam("command", "rm -rf /")
		output, err := exec.Execute(context.Background(), "bash", input)

		// Should succeed because no security policy
		if err != nil {
			t.Errorf("Execute() unexpected error: %v", err)
		}

		if output == nil {
			t.Error("Execute() returned nil output")
		}
	})

	t.Run("explicit nil security policy", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("bash", "A bash tool").
			WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("executed"), nil
			})
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Explicitly set nil security policy
		exec := NewExecutor(registry, WithSecurityPolicy(nil))

		input := NewInput().WithParam("command", "rm -rf /")
		output, err := exec.Execute(context.Background(), "bash", input)

		if err != nil {
			t.Errorf("Execute() unexpected error: %v", err)
		}

		if output == nil {
			t.Error("Execute() returned nil output")
		}

		// Verify config shows no security policy
		config := exec.Config()
		if config.HasSecurityPolicy {
			t.Error("Config.HasSecurityPolicy should be false")
		}
	})
}

// TestNilConfirmationHandler tests that the executor handles nil confirmation handler gracefully.
func TestNilConfirmationHandler(t *testing.T) {
	t.Run("nil confirmation handler skips confirmation", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("dangerous-tool", "A dangerous tool").
			WithRequiresConfirmation(true).
			WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("executed"), nil
			})
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Create executor without confirmation handler (nil by default)
		exec := NewExecutor(registry)

		output, err := exec.Execute(context.Background(), "dangerous-tool", NewInput())

		// Should succeed because no confirmation handler to block it
		if err != nil {
			t.Errorf("Execute() unexpected error: %v", err)
		}

		if output == nil {
			t.Error("Execute() returned nil output")
		}
	})

	t.Run("explicit nil confirmation handler", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewMockTool("dangerous-tool", "A dangerous tool").
			WithRequiresConfirmation(true).
			WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
				return NewOutput().WithMessage("executed"), nil
			})
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		// Explicitly set nil confirmation handler
		exec := NewExecutor(registry, WithConfirmationHandler(nil))

		output, err := exec.Execute(context.Background(), "dangerous-tool", NewInput())

		if err != nil {
			t.Errorf("Execute() unexpected error: %v", err)
		}

		if output == nil {
			t.Error("Execute() returned nil output")
		}

		// Verify config shows no confirmation handler
		config := exec.Config()
		if config.HasConfirmationHandler {
			t.Error("Config.HasConfirmationHandler should be false")
		}
	})

	t.Run("confirmation handler called before execution", func(t *testing.T) {
		registry := NewRegistry()
		var confirmationCalled bool
		var executionCalled bool

		tool := NewMockTool("test-tool", "A test tool").
			WithRequiresConfirmation(true).
			WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
				executionCalled = true
				return NewOutput(), nil
			})
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Failed to register tool: %v", err)
		}

		handler := ConfirmationFunc(func(ctx context.Context, tool Tool, args map[string]any) (bool, error) {
			confirmationCalled = true
			return true, nil
		})

		exec := NewExecutor(registry, WithConfirmationHandler(handler))

		_, err := exec.Execute(context.Background(), "test-tool", NewInput())

		if err != nil {
			t.Errorf("Execute() unexpected error: %v", err)
		}

		if !confirmationCalled {
			t.Error("Confirmation handler should have been called")
		}

		if !executionCalled {
			t.Error("Tool execution should have been called")
		}
	})
}
