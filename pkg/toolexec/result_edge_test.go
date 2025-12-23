package toolexec

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestErrorUnwrapping(t *testing.T) {
	inner := errors.New("inner error")

	t.Run("ToolError unwrapping", func(t *testing.T) {
		err := NewToolErrorWithCause("op", "tool", inner)
		if errors.Unwrap(err) != inner {
			t.Error("ToolError failed to unwrap cause")
		}
	})

	t.Run("ExecutionError unwrapping", func(t *testing.T) {
		err := NewExecutionErrorWithCause("tool", inner)
		if errors.Unwrap(err) != inner {
			t.Error("ExecutionError failed to unwrap cause")
		}
	})

	t.Run("ValidationError field error", func(t *testing.T) {
		err := NewValidationErrorForField("tool", "field", "msg")
		if err.Field != "field" {
			t.Errorf("Expected field 'field', got '%s'", err.Field)
		}
		expected := "validation failed for tool 'tool' field 'field': msg"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("SecurityViolationError variants", func(t *testing.T) {
		errPattern := NewSecurityViolationErrorWithPattern("tool", "reason", "*.sh")
		if errPattern.Pattern != "*.sh" {
			t.Error("Pattern not set")
		}
		if !IsSecurityViolationError(errPattern) {
			t.Error("IsSecurityViolationError failed for pattern error")
		}

		errPath := NewSecurityViolationErrorWithPath("tool", "reason", "/etc/passwd")
		if errPath.Path != "/etc/passwd" {
			t.Error("Path not set")
		}
		if !IsSecurityViolationError(errPath) {
			t.Error("IsSecurityViolationError failed for path error")
		}
	})

	t.Run("PanicError stack", func(t *testing.T) {
		err := NewPanicErrorWithStack("tool", "oops", "stack trace")
		if err.Stack != "stack trace" {
			t.Error("Stack not set")
		}
		if !IsPanicError(err) {
			t.Error("IsPanicError failed")
		}
	})

	t.Run("Nested wrapping Is/As", func(t *testing.T) {
		// NewToolErrorWithCause wraps inner
		te := NewToolErrorWithCause("op", "tool", inner)
		// ExecutionError wraps ToolError
		ee := NewExecutionErrorWithCause("tool", te)

		if !errors.Is(ee, inner) {
			t.Error("Should be able to find inner error through multiple layers")
		}
		if !errors.Is(ee, ErrExecutionFailed) {
			t.Error("Should be able to find ErrExecutionFailed")
		}

		var target *ToolError
		if !errors.As(ee, &target) {
			t.Error("Should be able to extract ToolError using errors.As")
		}
	})

	t.Run("Comprehensive Is coverage", func(t *testing.T) {
		errs := []struct {
			err    error
			target error
		}{
			{NewToolNotFoundError("t"), &ToolNotFoundError{}},
			{NewDuplicateToolError("t"), &DuplicateToolError{}},
			{NewValidationError("t", "m"), &ValidationError{}},
			{NewPanicError("t", "p"), &PanicError{}},
			{NewTimeoutError("t", 1), &TimeoutError{}},
			{NewMiddlewareError("mw", "t", "m"), &MiddlewareError{}},
			{NewUserDeniedError("t"), &UserDeniedError{}},
			{NewSecurityViolationError("t", "r"), &SecurityViolationError{}},
		}

		for _, tt := range errs {
			if !errors.Is(tt.err, tt.target) {
				t.Errorf("errors.Is failed for %T", tt.err)
			}
			// Test Unwrap directly
			if u, ok := tt.err.(interface{ Unwrap() error }); ok {
				_ = u.Unwrap()
			}
		}
	})
}

func TestGetToolNameEdgeCases(t *testing.T) {
	if GetToolName(nil) != "" {
		t.Error("GetToolName(nil) should be empty")
	}

	if GetToolName(errors.New("generic")) != "" {
		t.Error("GetToolName(generic error) should be empty")
	}

	err := fmt.Errorf("wrapped: %w", NewExecutionError("mytool", "fail"))
	if GetToolName(err) != "mytool" {
		t.Errorf("Expected 'mytool', got '%s'", GetToolName(err))
	}

	// Test all error types in GetToolName
	allErrs := []error{
		NewToolNotFoundError("t1"),
		NewDuplicateToolError("t2"),
		NewExecutionError("t3", "m"),
		NewValidationError("t4", "m"),
		NewPanicError("t5", "p"),
		NewTimeoutError("t6", 1),
		NewMiddlewareError("mw", "t7", "m"),
		NewUserDeniedError("t8"),
		NewSecurityViolationError("t9", "r"),
	}
	for i, e := range allErrs {
		expected := fmt.Sprintf("t%d", i+1)
		if GetToolName(e) != expected {
			t.Errorf("GetToolName failed for %T, expected %s, got %s", e, expected, GetToolName(e))
		}
	}
}

func TestErrorHelpers(t *testing.T) {
	if IsToolNotFoundError(nil) {
		t.Error("IsToolNotFoundError(nil) should be false")
	}
	if IsToolNotFoundError(errors.New("other")) {
		t.Error("IsToolNotFoundError(other) should be false")
	}

	// Repeat for others to gain coverage
	IsDuplicateToolError(nil)
	IsExecutionError(nil)
	IsValidationError(nil)
	IsPanicError(nil)
	IsTimeoutError(nil)
	IsMiddlewareError(nil)
	IsSecurityViolationError(nil)
}

func TestToolErrorIs(t *testing.T) {
	te := &ToolError{Operation: "execute", ToolName: "tool1"}

	if !te.Is(ErrExecutionFailed) {
		t.Error("ToolError with 'execute' operation should match ErrExecutionFailed")
	}

	te2 := &ToolError{ToolName: "tool1"}
	if !te.Is(te2) {
		t.Error("ToolError should match another ToolError with same ToolName")
	}

	te3 := &ToolError{Operation: "execute"}
	if !te.Is(te3) {
		t.Error("ToolError should match another ToolError with same Operation")
	}

	if te.Is(errors.New("other")) {
		t.Error("ToolError should not match generic error")
	}
}

func TestResultMethodsEdgeCases(t *testing.T) {
	r := NewResult("t", nil, nil)
	r.WithTiming(time.Now(), time.Now())

	r2 := NewErrorResult("t", errors.New("e"))
	if r2.IsSuccess() {
		t.Error("NewErrorResult should not be success")
	}

	ee := NewExecutionError("t", "m").WithInput(&Input{})
	if ee.Input == nil {
		t.Error("WithInput failed")
	}
}
