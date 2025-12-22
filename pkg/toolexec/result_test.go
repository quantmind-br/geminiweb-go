package toolexec

import (
	"errors"
	"testing"
)

// TestOutputTruncation tests the output truncation functionality.
func TestOutputTruncation(t *testing.T) {
	t.Run("truncate large data", func(t *testing.T) {
		// Create data larger than default max size
		largeData := make([]byte, DefaultMaxOutputSize+1000)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		output := NewOutput().WithData(largeData).TruncateDefault()

		if len(output.Data) != DefaultMaxOutputSize {
			t.Errorf("Data length = %d, want %d", len(output.Data), DefaultMaxOutputSize)
		}

		if !output.Truncated {
			t.Error("Truncated should be true")
		}
	})

	t.Run("do not truncate small data", func(t *testing.T) {
		smallData := []byte("small data")
		output := NewOutput().WithData(smallData).TruncateDefault()

		if len(output.Data) != len(smallData) {
			t.Errorf("Data length = %d, want %d", len(output.Data), len(smallData))
		}

		if output.Truncated {
			t.Error("Truncated should be false for small data")
		}
	})

	t.Run("truncate at exact boundary", func(t *testing.T) {
		// Data exactly at limit should not be truncated
		exactData := make([]byte, DefaultMaxOutputSize)
		output := NewOutput().WithData(exactData).TruncateDefault()

		if len(output.Data) != DefaultMaxOutputSize {
			t.Errorf("Data length = %d, want %d", len(output.Data), DefaultMaxOutputSize)
		}

		if output.Truncated {
			t.Error("Truncated should be false for data at exact boundary")
		}
	})

	t.Run("custom truncation size", func(t *testing.T) {
		customSize := 100
		data := make([]byte, 200)
		output := NewOutput().WithData(data).Truncate(customSize)

		if len(output.Data) != customSize {
			t.Errorf("Data length = %d, want %d", len(output.Data), customSize)
		}

		if !output.Truncated {
			t.Error("Truncated should be true")
		}
	})

	t.Run("zero max size does not truncate", func(t *testing.T) {
		data := []byte("test data")
		output := NewOutput().WithData(data).Truncate(0)

		if len(output.Data) != len(data) {
			t.Errorf("Data length changed with zero maxSize")
		}

		if output.Truncated {
			t.Error("Truncated should be false with zero maxSize")
		}
	})

	t.Run("negative max size does not truncate", func(t *testing.T) {
		data := []byte("test data")
		output := NewOutput().WithData(data).Truncate(-1)

		if len(output.Data) != len(data) {
			t.Errorf("Data length changed with negative maxSize")
		}

		if output.Truncated {
			t.Error("Truncated should be false with negative maxSize")
		}
	})

	t.Run("WithTruncatedData convenience method", func(t *testing.T) {
		data := make([]byte, 200)
		output := NewOutput().WithTruncatedData(data, 100)

		if len(output.Data) != 100 {
			t.Errorf("Data length = %d, want 100", len(output.Data))
		}

		if !output.Truncated {
			t.Error("Truncated should be true")
		}
	})

	t.Run("TruncateOutput helper function", func(t *testing.T) {
		data := make([]byte, 200)
		output := NewOutput().WithData(data)
		result := TruncateOutput(output, 100)

		if result != output {
			t.Error("TruncateOutput should return same output")
		}

		if len(output.Data) != 100 {
			t.Errorf("Data length = %d, want 100", len(output.Data))
		}
	})

	t.Run("TruncateOutput with nil output", func(t *testing.T) {
		result := TruncateOutput(nil, 100)

		if result != nil {
			t.Error("TruncateOutput(nil) should return nil")
		}
	})

	t.Run("TruncateOutput with zero maxSize uses default", func(t *testing.T) {
		data := make([]byte, DefaultMaxOutputSize+1000)
		output := NewOutput().WithData(data)
		TruncateOutput(output, 0)

		if len(output.Data) != DefaultMaxOutputSize {
			t.Errorf("Data length = %d, want %d (default)", len(output.Data), DefaultMaxOutputSize)
		}
	})

	t.Run("preserves data content", func(t *testing.T) {
		data := []byte("Hello, World! This is a test.")
		maxSize := 13
		output := NewOutput().WithData(data).Truncate(maxSize)

		expected := "Hello, World!"
		if string(output.Data) != expected {
			t.Errorf("Data = %s, want %s", string(output.Data), expected)
		}
	})

	t.Run("default max output size is 100KB", func(t *testing.T) {
		expectedSize := 100 * 1024
		if DefaultMaxOutputSize != expectedSize {
			t.Errorf("DefaultMaxOutputSize = %d, want %d", DefaultMaxOutputSize, expectedSize)
		}
	})
}

// TestErrorWrapping tests that errors wrap correctly with %w.
func TestErrorWrapping(t *testing.T) {
	t.Run("ToolNotFoundError wraps correctly", func(t *testing.T) {
		err := NewToolNotFoundError("test-tool")

		if !errors.Is(err, ErrToolNotFound) {
			t.Error("ToolNotFoundError should wrap ErrToolNotFound")
		}

		toolName := GetToolName(err)
		if toolName != "test-tool" {
			t.Errorf("GetToolName() = %s, want test-tool", toolName)
		}
	})

	t.Run("DuplicateToolError wraps correctly", func(t *testing.T) {
		err := NewDuplicateToolError("test-tool")

		if !errors.Is(err, ErrDuplicateTool) {
			t.Error("DuplicateToolError should wrap ErrDuplicateTool")
		}
	})

	t.Run("ExecutionError wraps correctly", func(t *testing.T) {
		err := NewExecutionErrorWithCause("test-tool", errors.New("inner error"))

		if !errors.Is(err, ErrExecutionFailed) {
			t.Error("ExecutionError should wrap ErrExecutionFailed")
		}

		if !IsExecutionError(err) {
			t.Error("IsExecutionError should return true")
		}
	})

	t.Run("TimeoutError wraps correctly", func(t *testing.T) {
		err := NewTimeoutError("test-tool", 30)

		if !errors.Is(err, ErrTimeout) {
			t.Error("TimeoutError should wrap ErrTimeout")
		}

		if !IsTimeoutError(err) {
			t.Error("IsTimeoutError should return true")
		}
	})

	t.Run("PanicError wraps correctly", func(t *testing.T) {
		err := NewPanicError("test-tool", "panic message")

		if !errors.Is(err, ErrPanicRecovered) {
			t.Error("PanicError should wrap ErrPanicRecovered")
		}

		if !IsPanicError(err) {
			t.Error("IsPanicError should return true")
		}
	})

	t.Run("ValidationError wraps correctly", func(t *testing.T) {
		err := NewValidationError("test-tool", "invalid input")

		if !errors.Is(err, ErrValidationFailed) {
			t.Error("ValidationError should wrap ErrValidationFailed")
		}

		if !IsValidationError(err) {
			t.Error("IsValidationError should return true")
		}
	})

	t.Run("UserDeniedError wraps correctly", func(t *testing.T) {
		err := NewUserDeniedError("test-tool")

		if !errors.Is(err, ErrUserDenied) {
			t.Error("UserDeniedError should wrap ErrUserDenied")
		}

		if !IsUserDeniedError(err) {
			t.Error("IsUserDeniedError should return true")
		}
	})

	t.Run("SecurityViolationError wraps correctly", func(t *testing.T) {
		err := NewSecurityViolationError("test-tool", "blocked command")

		if !errors.Is(err, ErrSecurityViolation) {
			t.Error("SecurityViolationError should wrap ErrSecurityViolation")
		}

		if !IsSecurityViolationError(err) {
			t.Error("IsSecurityViolationError should return true")
		}
	})

	t.Run("MiddlewareError wraps correctly", func(t *testing.T) {
		err := NewMiddlewareError("timing", "test-tool", "failed")

		if !errors.Is(err, ErrMiddlewareFailed) {
			t.Error("MiddlewareError should wrap ErrMiddlewareFailed")
		}

		if !IsMiddlewareError(err) {
			t.Error("IsMiddlewareError should return true")
		}
	})
}

// TestErrorTypes tests that all custom error types define Error(), Unwrap(), Is().
func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
	}{
		{"ErrToolNotFound", NewToolNotFoundError("t"), ErrToolNotFound},
		{"ErrDuplicateTool", NewDuplicateToolError("t"), ErrDuplicateTool},
		{"ErrExecutionFailed", NewExecutionError("t", "msg"), ErrExecutionFailed},
		{"ErrTimeout", NewTimeoutError("t", 30), ErrTimeout},
		{"ErrPanicRecovered", NewPanicError("t", "p"), ErrPanicRecovered},
		{"ErrValidationFailed", NewValidationError("t", "msg"), ErrValidationFailed},
		{"ErrUserDenied", NewUserDeniedError("t"), ErrUserDenied},
		{"ErrSecurityViolation", NewSecurityViolationError("t", "r"), ErrSecurityViolation},
		{"ErrMiddlewareFailed", NewMiddlewareError("mw", "t", "msg"), ErrMiddlewareFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Error() returns non-empty string
			if tt.err.Error() == "" {
				t.Error("Error() should return non-empty string")
			}

			// Test Is() matches target
			if !errors.Is(tt.err, tt.target) {
				t.Errorf("errors.Is() should match %v", tt.target)
			}
		})
	}
}

// TestResult tests the Result struct methods.
func TestResult(t *testing.T) {
	t.Run("NewResult", func(t *testing.T) {
		output := NewOutput().WithMessage("success")
		result := NewResult("test-tool", output, nil)

		if result.ToolName != "test-tool" {
			t.Errorf("ToolName = %s, want test-tool", result.ToolName)
		}

		if result.Output != output {
			t.Error("Output should match")
		}

		if result.Error != nil {
			t.Error("Error should be nil")
		}
	})

	t.Run("NewSuccessResult", func(t *testing.T) {
		output := NewOutput()
		result := NewSuccessResult("test-tool", output)

		if !result.IsSuccess() {
			t.Error("IsSuccess() should return true")
		}
	})

	t.Run("NewErrorResult", func(t *testing.T) {
		err := errors.New("test error")
		result := NewErrorResult("test-tool", err)

		if result.IsSuccess() {
			t.Error("IsSuccess() should return false")
		}

		if result.Error != err {
			t.Error("Error should match")
		}
	})

	t.Run("WithTiming", func(t *testing.T) {
		result := NewSuccessResult("test-tool", NewOutput())

		// Simulate some timing
		start := result.StartTime
		result.WithTiming(start, start.Add(100))

		if result.Duration != 100 {
			t.Errorf("Duration = %v, want 100ns", result.Duration)
		}
	})
}
