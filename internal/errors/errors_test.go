package errors

import (
	"errors"
	"testing"
)

func TestAuthError(t *testing.T) {
	err := NewAuthError("test auth error")

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expected := "authentication failed: test auth error"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}

	// Test Is method
	target := NewAuthError("target")
	if !err.Is(target) {
		t.Error("Expected error to be auth error type")
	}

	// Test Is with different type
	other := NewAPIError(400, "test", "other error")
	if err.Is(other) {
		t.Error("Expected error not to match different type")
	}

	// Test Is with standard errors
	stdErr := errors.New("standard error")
	if err.Is(stdErr) {
		t.Error("Expected error not to match standard error")
	}
}

func TestAPIError(t *testing.T) {
	err := NewAPIError(400, "test-endpoint", "test API error")

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expected := "API error [400] at test-endpoint: test API error"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}

	// APIError doesn't have Is method, so we skip that test
}

func TestTimeoutError(t *testing.T) {
	err := NewTimeoutError("test timeout error")

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expected := "request timed out: test timeout error"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}

	// TimeoutError doesn't have Is method, so we skip that test
}

func TestUsageLimitError(t *testing.T) {
	err := NewUsageLimitError("test usage limit error")

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expected := "usage limit exceeded: test usage limit error"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}

	// UsageLimitError doesn't have Is method, so we skip that test
}

func TestModelError(t *testing.T) {
	err := NewModelError("test model error")

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expected := "model error: test model error"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}

	// ModelError doesn't have Is method, so we skip that test
}

func TestBlockedError(t *testing.T) {
	err := NewBlockedError("test blocked error")

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expected := "content blocked: test blocked error"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}

	// BlockedError doesn't have Is method, so we skip that test
}

func TestParseError(t *testing.T) {
	err := NewParseError("test parse error", "test/path")

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expected := "parse error: test parse error"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}

	// Test Is method
	target := NewParseError("target", "target/path")
	if !err.Is(target) {
		t.Error("Expected error to be parse error type")
	}

	// Test Is with different type
	blockedErr := NewBlockedError("blocked")
	if err.Is(blockedErr) {
		t.Error("Expected error not to match different type")
	}

	// Test Is with standard errors
	stdErr := errors.New("parse error")
	if !err.Is(stdErr) {
		t.Error("Expected parse error to match standard parse error")
	}
}

func TestErrorIs(t *testing.T) {
	// Test that AuthError and ParseError implement the Is method correctly
	authErr := NewAuthError("auth")
	parseErr := NewParseError("parse", "path")

	// Test AuthError.Is
	if !authErr.Is(ErrAuthFailed) {
		t.Error("AuthError should match ErrAuthFailed")
	}

	// Test ParseError.Is
	if !parseErr.Is(ErrInvalidResponse) {
		t.Error("ParseError should match ErrInvalidResponse")
	}

	// Test with standard errors
	stdErr := errors.New("parse error")
	if !parseErr.Is(stdErr) {
		t.Error("ParseError should match standard parse error")
	}
}
