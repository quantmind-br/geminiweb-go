package toolexec

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewMiddlewareChain tests the NewMiddlewareChain function.
func TestNewMiddlewareChain(t *testing.T) {
	t.Run("empty chain", func(t *testing.T) {
		chain := NewMiddlewareChain()

		if chain == nil {
			t.Error("NewMiddlewareChain() returned nil")
		}

		if chain.Len() != 0 {
			t.Errorf("NewMiddlewareChain().Len() = %d, want 0", chain.Len())
		}
	})

	t.Run("with middlewares", func(t *testing.T) {
		mw1 := NewMiddlewareFunc("mw1", nil)
		mw2 := NewMiddlewareFunc("mw2", nil)

		chain := NewMiddlewareChain(mw1, mw2)

		if chain.Len() != 2 {
			t.Errorf("NewMiddlewareChain().Len() = %d, want 2", chain.Len())
		}
	})
}

// TestMiddlewareChain_Add tests the Add method.
func TestMiddlewareChain_Add(t *testing.T) {
	chain := NewMiddlewareChain()
	mw := NewMiddlewareFunc("test", nil)

	result := chain.Add(mw)

	// Should return chain for chaining
	if result != chain {
		t.Error("Add() should return the chain for method chaining")
	}

	if chain.Len() != 1 {
		t.Errorf("Add() Len() = %d, want 1", chain.Len())
	}
}

// TestMiddlewareChain_Prepend tests the Prepend method.
func TestMiddlewareChain_Prepend(t *testing.T) {
	chain := NewMiddlewareChain()
	mw1 := NewMiddlewareFunc("mw1", nil)
	mw2 := NewMiddlewareFunc("mw2", nil)

	chain.Add(mw1)
	chain.Prepend(mw2)

	middlewares := chain.Middlewares()

	if len(middlewares) != 2 {
		t.Fatalf("Prepend() Len() = %d, want 2", len(middlewares))
	}

	// mw2 should be first (prepended)
	if middlewares[0].Name() != "mw2" {
		t.Errorf("Prepend() first middleware = %s, want mw2", middlewares[0].Name())
	}

	if middlewares[1].Name() != "mw1" {
		t.Errorf("Prepend() second middleware = %s, want mw1", middlewares[1].Name())
	}
}

// TestMiddlewareChain_Middlewares tests the Middlewares method returns a copy.
func TestMiddlewareChain_Middlewares(t *testing.T) {
	mw := NewMiddlewareFunc("test", nil)
	chain := NewMiddlewareChain(mw)

	middlewares := chain.Middlewares()

	// Modify returned slice
	middlewares[0] = NewMiddlewareFunc("modified", nil)

	// Original chain should be unchanged
	original := chain.Middlewares()
	if original[0].Name() != "test" {
		t.Error("Middlewares() should return a copy, not the original slice")
	}
}

// TestMiddlewareChain_Wrap tests the Wrap method.
func TestMiddlewareChain_Wrap(t *testing.T) {
	t.Run("empty chain passes through", func(t *testing.T) {
		chain := NewMiddlewareChain()

		executed := false
		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			executed = true
			return NewOutput().WithMessage("base"), nil
		}

		wrapped := chain.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if err != nil {
			t.Errorf("Wrapped() error: %v", err)
		}

		if !executed {
			t.Error("Base function was not executed")
		}

		if output.Message != "base" {
			t.Errorf("Output.Message = %s, want 'base'", output.Message)
		}
	})

	t.Run("middlewares wrap in order", func(t *testing.T) {
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

		chain := NewMiddlewareChain(mw1, mw2)

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			order = append(order, "base")
			return NewOutput(), nil
		}

		wrapped := chain.Wrap(baseFn)
		_, _ = wrapped(context.Background(), "test", NewInput())

		// mw1 is outermost: mw1-before -> mw2-before -> base -> mw2-after -> mw1-after
		expected := []string{"mw1-before", "mw2-before", "base", "mw2-after", "mw1-after"}
		if len(order) != len(expected) {
			t.Fatalf("Execution order length = %d, want %d", len(order), len(expected))
		}

		for i, v := range order {
			if v != expected[i] {
				t.Errorf("Execution order[%d] = %s, want %s", i, v, expected[i])
			}
		}
	})

	t.Run("middleware can short-circuit", func(t *testing.T) {
		baseExecuted := false

		shortCircuit := NewMiddlewareFunc("short-circuit", func(next ToolFunc) ToolFunc {
			return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
				return nil, errors.New("short-circuited")
			}
		})

		chain := NewMiddlewareChain(shortCircuit)

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			baseExecuted = true
			return NewOutput(), nil
		}

		wrapped := chain.Wrap(baseFn)
		_, err := wrapped(context.Background(), "test", NewInput())

		if err == nil {
			t.Error("Expected error from short-circuit")
		}

		if baseExecuted {
			t.Error("Base function should not be executed when middleware short-circuits")
		}
	})
}

// TestNewMiddlewareFunc tests the NewMiddlewareFunc function.
func TestNewMiddlewareFunc(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		mw := NewMiddlewareFunc("test-mw", func(next ToolFunc) ToolFunc {
			return next
		})

		if mw.Name() != "test-mw" {
			t.Errorf("Name() = %s, want test-mw", mw.Name())
		}
	})

	t.Run("nil function passes through", func(t *testing.T) {
		mw := NewMiddlewareFunc("nil-fn", nil)

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			return NewOutput().WithMessage("base"), nil
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if err != nil {
			t.Errorf("Wrapped() error: %v", err)
		}

		if output.Message != "base" {
			t.Error("Nil function middleware should pass through")
		}
	})
}

// TestRecoveryMiddleware tests the RecoveryMiddleware.
func TestRecoveryMiddleware(t *testing.T) {
	t.Run("no panic passes through", func(t *testing.T) {
		mw := NewRecoveryMiddleware(true)

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			return NewOutput().WithMessage("success"), nil
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if err != nil {
			t.Errorf("Wrapped() unexpected error: %v", err)
		}

		if output.Message != "success" {
			t.Errorf("Output.Message = %s, want 'success'", output.Message)
		}
	})

	t.Run("recovers panic with stack trace", func(t *testing.T) {
		mw := NewRecoveryMiddleware(true)

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			panic("test panic")
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if output != nil {
			t.Error("Output should be nil on panic")
		}

		if err == nil {
			t.Fatal("Expected error on panic")
		}

		if !IsPanicError(err) {
			t.Errorf("Error should be PanicError, got: %T", err)
		}

		var panicErr *PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("Cannot extract PanicError")
		}

		if panicErr.Stack == "" {
			t.Error("Stack trace should be included when includeStack is true")
		}
	})

	t.Run("recovers panic without stack trace", func(t *testing.T) {
		mw := NewRecoveryMiddleware(false)

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			panic("test panic")
		}

		wrapped := mw.Wrap(baseFn)
		_, err := wrapped(context.Background(), "test", NewInput())

		if err == nil {
			t.Fatal("Expected error on panic")
		}

		var panicErr *PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("Cannot extract PanicError")
		}

		if panicErr.Stack != "" {
			t.Error("Stack trace should be empty when includeStack is false")
		}
	})

	t.Run("name returns recovery", func(t *testing.T) {
		mw := NewRecoveryMiddleware(true)
		if mw.Name() != "recovery" {
			t.Errorf("Name() = %s, want 'recovery'", mw.Name())
		}
	})
}

// TestTimingMiddleware tests the TimingMiddleware.
func TestTimingMiddleware(t *testing.T) {
	t.Run("adds timing metadata", func(t *testing.T) {
		mw := NewTimingMiddleware()

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			time.Sleep(10 * time.Millisecond) // Ensure measurable time
			return NewOutput(), nil
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if err != nil {
			t.Fatalf("Wrapped() error: %v", err)
		}

		if output.Metadata == nil {
			t.Fatal("Output.Metadata is nil")
		}

		timeMs, ok := output.Metadata["execution_time_ms"]
		if !ok {
			t.Error("Missing 'execution_time_ms' in metadata")
		}

		// Verify it's a reasonable number
		if len(timeMs) == 0 {
			t.Error("execution_time_ms is empty")
		}

		startTime, ok := output.Metadata["execution_start"]
		if !ok {
			t.Error("Missing 'execution_start' in metadata")
		}

		// Verify it's a valid timestamp format
		_, parseErr := time.Parse(time.RFC3339Nano, startTime)
		if parseErr != nil {
			t.Errorf("Invalid timestamp format in execution_start: %v", parseErr)
		}
	})

	t.Run("handles nil output", func(t *testing.T) {
		mw := NewTimingMiddleware()

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			return nil, errors.New("failed")
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if err == nil {
			t.Error("Expected error from base function")
		}

		if output != nil {
			t.Error("Output should remain nil when base returns nil")
		}
	})

	t.Run("initializes nil metadata", func(t *testing.T) {
		mw := NewTimingMiddleware()

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			// Return output with nil Metadata
			return &Output{Success: true}, nil
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if err != nil {
			t.Fatalf("Wrapped() error: %v", err)
		}

		if output.Metadata == nil {
			t.Error("Metadata should be initialized")
		}

		if _, ok := output.Metadata["execution_time_ms"]; !ok {
			t.Error("Timing metadata should be added")
		}
	})

	t.Run("name returns timing", func(t *testing.T) {
		mw := NewTimingMiddleware()
		if mw.Name() != "timing" {
			t.Errorf("Name() = %s, want 'timing'", mw.Name())
		}
	})
}

// TestContextCheckMiddleware tests the ContextCheckMiddleware.
func TestContextCheckMiddleware(t *testing.T) {
	t.Run("passes through on valid context", func(t *testing.T) {
		mw := NewContextCheckMiddleware()

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			return NewOutput().WithMessage("success"), nil
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if err != nil {
			t.Errorf("Wrapped() unexpected error: %v", err)
		}

		if output.Message != "success" {
			t.Error("Should pass through on valid context")
		}
	})

	t.Run("returns error on cancelled context", func(t *testing.T) {
		mw := NewContextCheckMiddleware()
		baseExecuted := false

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			baseExecuted = true
			return NewOutput(), nil
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(ctx, "test", NewInput())

		if err == nil {
			t.Error("Expected error on cancelled context")
		}

		if output != nil {
			t.Error("Output should be nil on cancelled context")
		}

		if baseExecuted {
			t.Error("Base function should not be executed on cancelled context")
		}

		// Verify error contains context cancelled info
		if !strings.Contains(err.Error(), "context cancelled") {
			t.Errorf("Error should mention context cancelled, got: %v", err)
		}
	})

	t.Run("name returns context-check", func(t *testing.T) {
		mw := NewContextCheckMiddleware()
		if mw.Name() != "context-check" {
			t.Errorf("Name() = %s, want 'context-check'", mw.Name())
		}
	})
}

// TestInputValidationMiddleware tests the InputValidationMiddleware.
func TestInputValidationMiddleware(t *testing.T) {
	t.Run("passes through on valid input", func(t *testing.T) {
		mw := NewInputValidationMiddleware()

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			return NewOutput().WithMessage("success"), nil
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		if err != nil {
			t.Errorf("Wrapped() unexpected error: %v", err)
		}

		if output.Message != "success" {
			t.Error("Should pass through on valid input")
		}
	})

	t.Run("returns error on nil input", func(t *testing.T) {
		mw := NewInputValidationMiddleware()
		baseExecuted := false

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			baseExecuted = true
			return NewOutput(), nil
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", nil)

		if err == nil {
			t.Error("Expected error on nil input")
		}

		if output != nil {
			t.Error("Output should be nil on validation error")
		}

		if baseExecuted {
			t.Error("Base function should not be executed on nil input")
		}

		if !IsValidationError(err) {
			t.Errorf("Error should be ValidationError, got: %T", err)
		}
	})

	t.Run("name returns input-validation", func(t *testing.T) {
		mw := NewInputValidationMiddleware()
		if mw.Name() != "input-validation" {
			t.Errorf("Name() = %s, want 'input-validation'", mw.Name())
		}
	})
}

// TestLoggingMiddleware tests the LoggingMiddleware.
func TestLoggingMiddleware(t *testing.T) {
	t.Run("calls before and after hooks", func(t *testing.T) {
		var beforeCalled, afterCalled bool
		var beforeToolName, afterToolName string
		var afterDuration time.Duration

		mw := NewLoggingMiddleware(
			func(toolName string, input *Input) {
				beforeCalled = true
				beforeToolName = toolName
			},
			func(toolName string, output *Output, err error, duration time.Duration) {
				afterCalled = true
				afterToolName = toolName
				afterDuration = duration
			},
		)

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			time.Sleep(5 * time.Millisecond)
			return NewOutput(), nil
		}

		wrapped := mw.Wrap(baseFn)
		_, _ = wrapped(context.Background(), "my-tool", NewInput())

		if !beforeCalled {
			t.Error("Before hook was not called")
		}

		if beforeToolName != "my-tool" {
			t.Errorf("Before hook toolName = %s, want my-tool", beforeToolName)
		}

		if !afterCalled {
			t.Error("After hook was not called")
		}

		if afterToolName != "my-tool" {
			t.Errorf("After hook toolName = %s, want my-tool", afterToolName)
		}

		if afterDuration < 5*time.Millisecond {
			t.Errorf("After hook duration = %v, want >= 5ms", afterDuration)
		}
	})

	t.Run("handles nil hooks", func(t *testing.T) {
		mw := NewLoggingMiddleware(nil, nil)

		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			return NewOutput().WithMessage("success"), nil
		}

		wrapped := mw.Wrap(baseFn)
		output, err := wrapped(context.Background(), "test", NewInput())

		// Should not panic with nil hooks
		if err != nil {
			t.Errorf("Wrapped() error: %v", err)
		}

		if output.Message != "success" {
			t.Error("Should pass through with nil hooks")
		}
	})

	t.Run("after hook receives error", func(t *testing.T) {
		var receivedErr error

		mw := NewLoggingMiddleware(
			nil,
			func(toolName string, output *Output, err error, duration time.Duration) {
				receivedErr = err
			},
		)

		expectedErr := errors.New("test error")
		baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			return nil, expectedErr
		}

		wrapped := mw.Wrap(baseFn)
		_, _ = wrapped(context.Background(), "test", NewInput())

		if receivedErr != expectedErr {
			t.Errorf("After hook received err = %v, want %v", receivedErr, expectedErr)
		}
	})

	t.Run("name returns logging", func(t *testing.T) {
		mw := NewLoggingMiddleware(nil, nil)
		if mw.Name() != "logging" {
			t.Errorf("Name() = %s, want 'logging'", mw.Name())
		}
	})
}

// TestChainMiddleware tests the ChainMiddleware utility function.
func TestChainMiddleware(t *testing.T) {
	mw1 := NewMiddlewareFunc("mw1", nil)
	mw2 := NewMiddlewareFunc("mw2", nil)

	chain := ChainMiddleware(mw1, mw2)

	if chain.Len() != 2 {
		t.Errorf("ChainMiddleware().Len() = %d, want 2", chain.Len())
	}
}

// TestApplyMiddleware tests the ApplyMiddleware utility function.
func TestApplyMiddleware(t *testing.T) {
	var order []string

	mw := NewMiddlewareFunc("mw", func(next ToolFunc) ToolFunc {
		return func(ctx context.Context, toolName string, input *Input) (*Output, error) {
			order = append(order, "mw")
			return next(ctx, toolName, input)
		}
	})

	baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
		order = append(order, "base")
		return NewOutput(), nil
	}

	wrapped := ApplyMiddleware(baseFn, mw)
	_, _ = wrapped(context.Background(), "test", NewInput())

	expected := []string{"mw", "base"}
	if len(order) != len(expected) {
		t.Fatalf("Order length = %d, want %d", len(order), len(expected))
	}

	for i, v := range order {
		if v != expected[i] {
			t.Errorf("Order[%d] = %s, want %s", i, v, expected[i])
		}
	}
}

// TestCombineMiddleware tests the CombineMiddleware utility function.
func TestCombineMiddleware(t *testing.T) {
	chain1 := NewMiddlewareChain(NewMiddlewareFunc("mw1", nil))
	chain2 := NewMiddlewareChain(NewMiddlewareFunc("mw2", nil), NewMiddlewareFunc("mw3", nil))

	combined := CombineMiddleware(chain1, chain2)

	if combined.Len() != 3 {
		t.Errorf("CombineMiddleware().Len() = %d, want 3", combined.Len())
	}

	middlewares := combined.Middlewares()
	expectedNames := []string{"mw1", "mw2", "mw3"}
	for i, mw := range middlewares {
		if mw.Name() != expectedNames[i] {
			t.Errorf("Combined[%d].Name() = %s, want %s", i, mw.Name(), expectedNames[i])
		}
	}
}

// TestCombineMiddleware_NilChains tests CombineMiddleware with nil chains.
func TestCombineMiddleware_NilChains(t *testing.T) {
	chain1 := NewMiddlewareChain(NewMiddlewareFunc("mw1", nil))

	combined := CombineMiddleware(chain1, nil)

	if combined.Len() != 1 {
		t.Errorf("CombineMiddleware() with nil should ignore nil, Len() = %d, want 1", combined.Len())
	}
}

// TestDefaultMiddlewareChain tests the DefaultMiddlewareChain utility function.
func TestDefaultMiddlewareChain(t *testing.T) {
	chain := DefaultMiddlewareChain()

	if chain == nil {
		t.Fatal("DefaultMiddlewareChain() returned nil")
	}

	if chain.Len() != 4 {
		t.Errorf("DefaultMiddlewareChain().Len() = %d, want 4", chain.Len())
	}

	middlewares := chain.Middlewares()
	expectedNames := []string{"recovery", "context-check", "input-validation", "timing"}

	for i, mw := range middlewares {
		if mw.Name() != expectedNames[i] {
			t.Errorf("DefaultMiddlewareChain()[%d].Name() = %s, want %s", i, mw.Name(), expectedNames[i])
		}
	}
}

// TestMiddleware_ConcurrentAccess tests thread-safety of middleware chain.
func TestMiddleware_ConcurrentAccess(t *testing.T) {
	chain := DefaultMiddlewareChain()

	baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
		return NewOutput(), nil
	}

	wrapped := chain.Wrap(baseFn)

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := wrapped(context.Background(), "test", NewInput())
			if err != nil {
				t.Errorf("Concurrent execution error: %v", err)
			}
		}()
	}

	wg.Wait()
}

// TestFormatDurationMs tests the formatDurationMs helper function.
func TestFormatDurationMs(t *testing.T) {
	tests := []struct {
		duration   time.Duration
		wantPrefix string
	}{
		{0, "0."},
		{time.Millisecond, "1."},
		{100 * time.Millisecond, "100."},
		{1500 * time.Millisecond, "1500."},
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			result := formatDurationMs(tt.duration)
			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("formatDurationMs(%v) = %s, want prefix %s", tt.duration, result, tt.wantPrefix)
			}
		})
	}
}

// TestFormatInt64 tests the formatInt64 helper function.
func TestFormatInt64(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{-1, "-1"},
		{-123, "-123"},
		{1234567890, "1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := formatInt64(tt.input)
			if result != tt.want {
				t.Errorf("formatInt64(%d) = %s, want %s", tt.input, result, tt.want)
			}
		})
	}
}

// TestMiddlewareChain_MethodChaining tests method chaining.
func TestMiddlewareChain_MethodChaining(t *testing.T) {
	mw1 := NewMiddlewareFunc("mw1", nil)
	mw2 := NewMiddlewareFunc("mw2", nil)
	mw3 := NewMiddlewareFunc("mw3", nil)

	chain := NewMiddlewareChain().
		Add(mw1).
		Add(mw2).
		Prepend(mw3)

	if chain.Len() != 3 {
		t.Errorf("Chained methods Len() = %d, want 3", chain.Len())
	}

	middlewares := chain.Middlewares()
	if middlewares[0].Name() != "mw3" {
		t.Error("Prepended middleware should be first")
	}
}

// TestRecoveryMiddleware_DifferentPanicTypes tests recovery from different panic values.
func TestRecoveryMiddleware_DifferentPanicTypes(t *testing.T) {
	tests := []struct {
		name       string
		panicValue interface{}
	}{
		{"string panic", "string error"},
		{"error panic", errors.New("error value")},
		{"int panic", 42},
		{"struct panic", struct{ msg string }{"test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := NewRecoveryMiddleware(true)

			baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
				panic(tt.panicValue)
			}

			wrapped := mw.Wrap(baseFn)
			_, err := wrapped(context.Background(), "test", NewInput())

			if err == nil {
				t.Error("Expected error from panic")
			}

			if !IsPanicError(err) {
				t.Errorf("Error should be PanicError, got: %T", err)
			}
		})
	}
}

// TestTimingMiddleware_Precision tests timing precision.
func TestTimingMiddleware_Precision(t *testing.T) {
	mw := NewTimingMiddleware()

	sleepDuration := 50 * time.Millisecond

	baseFn := func(ctx context.Context, toolName string, input *Input) (*Output, error) {
		time.Sleep(sleepDuration)
		return NewOutput(), nil
	}

	wrapped := mw.Wrap(baseFn)
	output, err := wrapped(context.Background(), "test", NewInput())

	if err != nil {
		t.Fatalf("Wrapped() error: %v", err)
	}

	timeMs := output.Metadata["execution_time_ms"]

	// Parse the time value (should be >= 50)
	if len(timeMs) < 2 {
		t.Errorf("execution_time_ms too short: %s", timeMs)
	}

	// Check it starts with at least "5" (50+ ms)
	if timeMs[0] < '5' {
		t.Errorf("execution_time_ms should be >= 50ms, got: %s", timeMs)
	}
}
