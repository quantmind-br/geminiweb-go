package errors

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

// mockNetError implements net.Error for testing
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock network error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

// Ensure mockNetError implements net.Error
var _ net.Error = (*mockNetError)(nil)

func TestErrorCodeString(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrCodeUnknown, "unknown error"},
		{ErrCodeUsageLimitExceeded, "usage limit exceeded"},
		{ErrCodeModelInconsistent, "model inconsistent with chat history"},
		{ErrCodeModelHeaderInvalid, "model header invalid or model unavailable"},
		{ErrCodeIPBlocked, "IP temporarily blocked"},
		{ErrorCode(9999), "unknown error"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			if got := tt.code.String(); got != tt.expected {
				t.Errorf("ErrorCode.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGeminiError(t *testing.T) {
	t.Run("basic error", func(t *testing.T) {
		err := NewGeminiError("test operation", "test message")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		if !containsString(err.Error(), "test operation") {
			t.Errorf("Error() should contain operation, got %q", err.Error())
		}
		if !containsString(err.Error(), "test message") {
			t.Errorf("Error() should contain message, got %q", err.Error())
		}
	})

	t.Run("error with status", func(t *testing.T) {
		err := NewGeminiErrorWithStatus("op", 401, "https://example.com", "unauthorized")
		if err.HTTPStatus != 401 {
			t.Errorf("HTTPStatus = %d, want 401", err.HTTPStatus)
		}
		if err.Endpoint != "https://example.com" {
			t.Errorf("Endpoint = %q, want %q", err.Endpoint, "https://example.com")
		}
	})

	t.Run("error with code", func(t *testing.T) {
		err := NewGeminiErrorWithCode("op", ErrCodeUsageLimitExceeded, "https://example.com")
		if err.Code != ErrCodeUsageLimitExceeded {
			t.Errorf("Code = %d, want %d", err.Code, ErrCodeUsageLimitExceeded)
		}
	})

	t.Run("error with cause", func(t *testing.T) {
		cause := errors.New("underlying cause")
		err := NewGeminiErrorWithCause("op", cause)
		if err.Unwrap() != cause {
			t.Error("Unwrap() should return the cause")
		}
	})

	t.Run("WithBody truncates long body", func(t *testing.T) {
		err := NewGeminiError("op", "msg")
		// Create body longer than maxBodyLen (1000) to trigger truncation
		longBody := make([]byte, 1500)
		for i := range longBody {
			longBody[i] = 'a'
		}
		_ = err.WithBody(string(longBody))
		// maxBodyLen is 1000, so truncated body should be ~1000 + truncation marker
		if len(err.Body) <= 1000 {
			t.Errorf("Body should be truncated around 1000 chars, got %d", len(err.Body))
		}
		if !containsString(err.Body, "truncated") {
			t.Error("Body should contain truncation marker")
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := &GeminiError{HTTPStatus: 401}
		if !err.Is(ErrAuthFailed) {
			t.Error("401 error should match ErrAuthFailed")
		}

		err = &GeminiError{Code: ErrCodeUsageLimitExceeded}
		if !err.Is(ErrRateLimited) {
			t.Error("UsageLimitExceeded error should match ErrRateLimited")
		}

		err = &GeminiError{HTTPStatus: 429}
		if !err.Is(ErrRateLimited) {
			t.Error("429 error should match ErrRateLimited")
		}
	})

	t.Run("IsAuth", func(t *testing.T) {
		err := &GeminiError{HTTPStatus: 401}
		if !err.IsAuth() {
			t.Error("401 error should be auth error")
		}
		err = &GeminiError{HTTPStatus: 200}
		if err.IsAuth() {
			t.Error("200 error should not be auth error")
		}
	})

	t.Run("IsRateLimit", func(t *testing.T) {
		err := &GeminiError{Code: ErrCodeUsageLimitExceeded}
		if !err.IsRateLimit() {
			t.Error("UsageLimitExceeded should be rate limit error")
		}
		err = &GeminiError{HTTPStatus: 429}
		if !err.IsRateLimit() {
			t.Error("429 should be rate limit error")
		}
	})

	t.Run("IsNetwork", func(t *testing.T) {
		netErr := &mockNetError{}
		err := &GeminiError{Cause: netErr}
		if !err.IsNetwork() {
			t.Error("Error with net.Error cause should be network error")
		}
		err = &GeminiError{Cause: errors.New("not a net error")}
		if err.IsNetwork() {
			t.Error("Error with non-net.Error cause should not be network error")
		}
	})

	t.Run("IsTimeout", func(t *testing.T) {
		netErr := &mockNetError{timeout: true}
		err := &GeminiError{Cause: netErr}
		if !err.IsTimeout() {
			t.Error("Error with timeout net.Error should be timeout error")
		}
		netErr = &mockNetError{timeout: false}
		err = &GeminiError{Cause: netErr}
		if err.IsTimeout() {
			t.Error("Error with non-timeout net.Error should not be timeout error")
		}
	})

	t.Run("IsBlocked", func(t *testing.T) {
		err := &GeminiError{Code: ErrCodeIPBlocked}
		if !err.IsBlocked() {
			t.Error("IPBlocked error should be blocked")
		}
	})

	t.Run("IsModelError", func(t *testing.T) {
		err := &GeminiError{Code: ErrCodeModelInconsistent}
		if !err.IsModelError() {
			t.Error("ModelInconsistent should be model error")
		}
		err = &GeminiError{Code: ErrCodeModelHeaderInvalid}
		if !err.IsModelError() {
			t.Error("ModelHeaderInvalid should be model error")
		}
	})
}

func TestAuthError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewAuthError("test auth error")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		expected := "authentication failed: test auth error"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("empty message", func(t *testing.T) {
		err := NewAuthError("")
		expected := "authentication failed: cookies may have expired"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("with endpoint", func(t *testing.T) {
		err := NewAuthErrorWithEndpoint("test", "https://example.com")
		if err.Endpoint != "https://example.com" {
			t.Errorf("Endpoint = %q, want %q", err.Endpoint, "https://example.com")
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewAuthError("test")
		if !err.Is(ErrAuthFailed) {
			t.Error("AuthError should match ErrAuthFailed")
		}
		if !err.Is(NewAuthError("other")) {
			t.Error("AuthError should match other AuthError")
		}
		if err.Is(NewAPIError(400, "", "")) {
			t.Error("AuthError should not match APIError")
		}
	})
}

func TestAPIError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewAPIError(400, "test-endpoint", "test API error")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		expected := "API error [400] at test-endpoint: test API error"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("with body", func(t *testing.T) {
		err := NewAPIErrorWithBody(500, "endpoint", "error", "response body")
		if !containsString(err.Error(), "response body") {
			t.Errorf("Error() should contain body, got %q", err.Error())
		}
	})

	t.Run("with code", func(t *testing.T) {
		err := NewAPIErrorWithCode(ErrCodeUsageLimitExceeded, "endpoint")
		if err.Code != ErrCodeUsageLimitExceeded {
			t.Errorf("Code = %d, want %d", err.Code, ErrCodeUsageLimitExceeded)
		}
	})

	t.Run("StatusCode method", func(t *testing.T) {
		err := NewAPIError(404, "", "")
		if err.StatusCode() != 404 {
			t.Errorf("StatusCode() = %d, want 404", err.StatusCode())
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewAPIError(401, "", "")
		if !err.Is(ErrAuthFailed) {
			t.Error("401 APIError should match ErrAuthFailed")
		}
		err = NewAPIError(429, "", "")
		if !err.Is(ErrRateLimited) {
			t.Error("429 APIError should match ErrRateLimited")
		}
	})
}

func TestNetworkError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := NewNetworkError("test op", cause)
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		if !containsString(err.Error(), "network error") {
			t.Errorf("Error() should contain 'network error', got %q", err.Error())
		}
	})

	t.Run("with endpoint", func(t *testing.T) {
		err := NewNetworkErrorWithEndpoint("op", "https://example.com", nil)
		if err.Endpoint != "https://example.com" {
			t.Errorf("Endpoint = %q, want %q", err.Endpoint, "https://example.com")
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewNetworkError("op", nil)
		if !err.Is(ErrNetworkFailure) {
			t.Error("NetworkError should match ErrNetworkFailure")
		}
		if !err.Is(NewNetworkError("other", nil)) {
			t.Error("NetworkError should match other NetworkError")
		}
	})

	t.Run("Is with timeout", func(t *testing.T) {
		netErr := &mockNetError{timeout: true}
		err := NewNetworkError("op", netErr)
		if !err.Is(ErrTimeout) {
			t.Error("NetworkError with timeout cause should match ErrTimeout")
		}
	})
}

func TestTimeoutError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewTimeoutError("test timeout")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		expected := "request timed out: test timeout"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("empty message", func(t *testing.T) {
		err := NewTimeoutError("")
		expected := "request timed out"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("with endpoint", func(t *testing.T) {
		cause := &mockNetError{timeout: true}
		err := NewTimeoutErrorWithEndpoint("https://example.com", cause)
		if err.Endpoint != "https://example.com" {
			t.Errorf("Endpoint = %q, want %q", err.Endpoint, "https://example.com")
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewTimeoutError("test")
		if !err.Is(ErrTimeout) {
			t.Error("TimeoutError should match ErrTimeout")
		}
		if !err.Is(NewTimeoutError("other")) {
			t.Error("TimeoutError should match other TimeoutError")
		}
	})
}

func TestUsageLimitError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewUsageLimitError("gemini-pro")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		if !containsString(err.Error(), "usage limit exceeded") {
			t.Errorf("Error() should contain 'usage limit exceeded', got %q", err.Error())
		}
		if !containsString(err.Error(), "gemini-pro") {
			t.Errorf("Error() should contain model name, got %q", err.Error())
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewUsageLimitError("model")
		if !err.Is(ErrRateLimited) {
			t.Error("UsageLimitError should match ErrRateLimited")
		}
	})
}

func TestModelError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewModelError("model not found")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		expected := "model error: model not found"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("with code", func(t *testing.T) {
		err := NewModelErrorWithCode(ErrCodeModelInconsistent)
		if err.Code != ErrCodeModelInconsistent {
			t.Errorf("Code = %d, want %d", err.Code, ErrCodeModelInconsistent)
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewModelError("test")
		if !err.Is(NewModelError("other")) {
			t.Error("ModelError should match other ModelError")
		}
	})
}

func TestBlockedError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewBlockedError("IP blocked")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		if !containsString(err.Error(), "blocked") {
			t.Errorf("Error() should contain 'blocked', got %q", err.Error())
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewBlockedError("test")
		if !err.Is(NewBlockedError("other")) {
			t.Error("BlockedError should match other BlockedError")
		}
	})
}

func TestParseError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewParseError("invalid JSON", "response.body")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		if !containsString(err.Error(), "parse error") {
			t.Errorf("Error() should contain 'parse error', got %q", err.Error())
		}
		if !containsString(err.Error(), "response.body") {
			t.Errorf("Error() should contain path, got %q", err.Error())
		}
	})

	t.Run("empty path", func(t *testing.T) {
		err := NewParseError("invalid", "")
		if containsString(err.Error(), "at") {
			t.Errorf("Error() without path should not contain 'at', got %q", err.Error())
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewParseError("test", "path")
		if !err.Is(ErrInvalidResponse) {
			t.Error("ParseError should match ErrInvalidResponse")
		}
		if !err.Is(NewParseError("other", "other")) {
			t.Error("ParseError should match other ParseError")
		}
	})
}

func TestHandleErrorCode(t *testing.T) {
	tests := []struct {
		code      ErrorCode
		endpoint  string
		modelName string
		checkType func(error) bool
	}{
		{
			code:      ErrCodeUsageLimitExceeded,
			modelName: "test-model",
			checkType: func(e error) bool {
				var usageErr *UsageLimitError
				return errors.As(e, &usageErr)
			},
		},
		{
			code: ErrCodeModelInconsistent,
			checkType: func(e error) bool {
				var modelErr *ModelError
				return errors.As(e, &modelErr)
			},
		},
		{
			code: ErrCodeModelHeaderInvalid,
			checkType: func(e error) bool {
				var modelErr *ModelError
				return errors.As(e, &modelErr)
			},
		},
		{
			code: ErrCodeIPBlocked,
			checkType: func(e error) bool {
				var blockedErr *BlockedError
				return errors.As(e, &blockedErr)
			},
		},
		{
			code:     ErrorCode(9999),
			endpoint: "test-endpoint",
			checkType: func(e error) bool {
				var apiErr *APIError
				return errors.As(e, &apiErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			err := HandleErrorCode(tt.code, tt.endpoint, tt.modelName)
			if err == nil {
				t.Fatal("Expected non-nil error")
			}
			if !tt.checkType(err) {
				t.Errorf("HandleErrorCode(%d) returned wrong error type", tt.code)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"AuthError", NewAuthError("test"), true},
		{"APIError 401", NewAPIError(401, "", ""), true},
		{"APIError 500", NewAPIError(500, "", ""), false},
		{"GeminiError 401", &GeminiError{HTTPStatus: 401}, true},
		{"ErrAuthFailed", ErrAuthFailed, true},
		{"wrapped AuthError", fmt.Errorf("wrapped: %w", NewAuthError("test")), true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"NetworkError", NewNetworkError("op", nil), true},
		{"ErrNetworkFailure", ErrNetworkFailure, true},
		{"wrapped NetworkError", fmt.Errorf("wrapped: %w", NewNetworkError("op", nil)), true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkError(tt.err); got != tt.expected {
				t.Errorf("IsNetworkError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"TimeoutError", NewTimeoutError("test"), true},
		{"ErrTimeout", ErrTimeout, true},
		{"net.Error timeout", &mockNetError{timeout: true}, true},
		{"net.Error no timeout", &mockNetError{timeout: false}, false},
		{"wrapped TimeoutError", fmt.Errorf("wrapped: %w", NewTimeoutError("test")), true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTimeoutError(tt.err); got != tt.expected {
				t.Errorf("IsTimeoutError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"UsageLimitError", NewUsageLimitError("model"), true},
		{"ErrRateLimited", ErrRateLimited, true},
		{"wrapped UsageLimitError", fmt.Errorf("wrapped: %w", NewUsageLimitError("model")), true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimitError(tt.err); got != tt.expected {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil", nil, 0},
		{"GeminiError", &GeminiError{HTTPStatus: 401}, 401},
		{"APIError", NewAPIError(500, "", ""), 500},
		{"AuthError", NewAuthError("test"), 401},
		{"wrapped GeminiError", fmt.Errorf("wrapped: %w", &GeminiError{HTTPStatus: 403}), 403},
		{"other error", errors.New("other"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHTTPStatus(tt.err); got != tt.expected {
				t.Errorf("GetHTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{"nil", nil, ErrCodeUnknown},
		{"GeminiError", &GeminiError{Code: ErrCodeUsageLimitExceeded}, ErrCodeUsageLimitExceeded},
		{"APIError with code", NewAPIErrorWithCode(ErrCodeIPBlocked, ""), ErrCodeIPBlocked},
		{"wrapped GeminiError", fmt.Errorf("wrapped: %w", &GeminiError{Code: ErrCodeModelInconsistent}), ErrCodeModelInconsistent},
		{"other error", errors.New("other"), ErrCodeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorCode(tt.err); got != tt.expected {
				t.Errorf("GetErrorCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestGetEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil", nil, ""},
		{"GeminiError", &GeminiError{Endpoint: "https://test.com"}, "https://test.com"},
		{"APIError", NewAPIError(500, "https://api.test.com", ""), "https://api.test.com"},
		{"AuthError", NewAuthErrorWithEndpoint("test", "https://auth.test.com"), "https://auth.test.com"},
		{"NetworkError", NewNetworkErrorWithEndpoint("op", "https://net.test.com", nil), "https://net.test.com"},
		{"wrapped GeminiError", fmt.Errorf("wrapped: %w", &GeminiError{Endpoint: "https://wrapped.com"}), "https://wrapped.com"},
		{"other error", errors.New("other"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEndpoint(tt.err); got != tt.expected {
				t.Errorf("GetEndpoint() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	t.Run("AuthError embeds GeminiError", func(t *testing.T) {
		err := NewAuthError("test")
		// AuthError embeds *GeminiError, accessible via field
		if err.GeminiError == nil {
			t.Error("AuthError.GeminiError should not be nil")
		}
		if err.HTTPStatus != 401 {
			t.Errorf("AuthError.HTTPStatus = %d, want 401", err.HTTPStatus)
		}
	})

	t.Run("APIError embeds GeminiError", func(t *testing.T) {
		err := NewAPIError(500, "endpoint", "message")
		// APIError embeds *GeminiError, accessible via field
		if err.GeminiError == nil {
			t.Error("APIError.GeminiError should not be nil")
		}
		if err.HTTPStatus != 500 {
			t.Errorf("APIError.HTTPStatus = %d, want 500", err.HTTPStatus)
		}
	})

	t.Run("errors.Is works through wrapping", func(t *testing.T) {
		authErr := NewAuthError("test")
		wrapped := fmt.Errorf("context: %w", authErr)
		if !errors.Is(wrapped, ErrAuthFailed) {
			t.Error("wrapped AuthError should match ErrAuthFailed via errors.Is")
		}
	})

	t.Run("errors.As finds AuthError through wrapping", func(t *testing.T) {
		authErr := NewAuthError("test")
		wrapped := fmt.Errorf("context: %w", authErr)
		var foundErr *AuthError
		if !errors.As(wrapped, &foundErr) {
			t.Error("wrapped AuthError should be findable via errors.As")
		}
	})

	t.Run("errors.As finds APIError through wrapping", func(t *testing.T) {
		apiErr := NewAPIError(500, "", "")
		wrapped := fmt.Errorf("context: %w", apiErr)
		var foundErr *APIError
		if !errors.As(wrapped, &foundErr) {
			t.Error("wrapped APIError should be findable via errors.As")
		}
	})
}

func TestSentinelErrors(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrAuthFailed", ErrAuthFailed},
		{"ErrCookiesExpired", ErrCookiesExpired},
		{"ErrNoCookies", ErrNoCookies},
		{"ErrInvalidResponse", ErrInvalidResponse},
		{"ErrNoContent", ErrNoContent},
		{"ErrNetworkFailure", ErrNetworkFailure},
		{"ErrTimeout", ErrTimeout},
		{"ErrRateLimited", ErrRateLimited},
	}

	for _, s := range sentinels {
		t.Run(s.name, func(t *testing.T) {
			if s.err == nil {
				t.Errorf("%s should not be nil", s.name)
			}
			if s.err.Error() == "" {
				t.Errorf("%s.Error() should not be empty", s.name)
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestUploadError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewUploadError("test.txt", "file not found")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		if !containsString(err.Error(), "upload error") {
			t.Errorf("Error() should contain 'upload error', got %q", err.Error())
		}
		if !containsString(err.Error(), "test.txt") {
			t.Errorf("Error() should contain filename, got %q", err.Error())
		}
		if err.FileName != "test.txt" {
			t.Errorf("FileName = %q, want 'test.txt'", err.FileName)
		}
	})

	t.Run("with status", func(t *testing.T) {
		err := NewUploadErrorWithStatus("test.png", 404, "not found")
		if err.HTTPStatus != 404 {
			t.Errorf("HTTPStatus = %d, want 404", err.HTTPStatus)
		}
		if !containsString(err.Error(), "HTTP 404") {
			t.Errorf("Error() should contain 'HTTP 404', got %q", err.Error())
		}
	})

	t.Run("network error", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := NewUploadNetworkError("large.md", cause)
		if err.Cause != cause {
			t.Error("Cause should be set")
		}
		if !containsString(err.Error(), "connection refused") {
			t.Errorf("Error() should contain cause message, got %q", err.Error())
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewUploadError("test.txt", "error")
		if !err.Is(NewUploadError("other.txt", "other")) {
			t.Error("UploadError should match other UploadError")
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("underlying")
		err := NewUploadNetworkError("test.txt", cause)
		if err.Unwrap() != cause {
			t.Error("Unwrap should return the cause")
		}
	})
}

func TestIsUploadError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"UploadError", NewUploadError("test.txt", "error"), true},
		{"UploadErrorWithStatus", NewUploadErrorWithStatus("test.txt", 500, "error"), true},
		{"UploadNetworkError", NewUploadNetworkError("test.txt", errors.New("net")), true},
		{"wrapped UploadError", fmt.Errorf("wrapped: %w", NewUploadError("test.txt", "error")), true},
		{"other error", errors.New("other"), false},
		{"APIError", NewAPIError(500, "", ""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUploadError(tt.err); got != tt.expected {
				t.Errorf("IsUploadError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetEndpointWithUploadError(t *testing.T) {
	err := NewUploadError("test.txt", "error")
	endpoint := GetEndpoint(err)
	if endpoint != "https://content-push.googleapis.com/upload" {
		t.Errorf("GetEndpoint() = %q, want upload endpoint", endpoint)
	}
}

func TestGemError(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := NewGemError("gem-id", "Test Gem", "operation failed")
		if err == nil {
			t.Fatal("Expected non-nil error")
		}
		if !containsString(err.Error(), "gem error") {
			t.Errorf("Error() should contain 'gem error', got %q", err.Error())
		}
		if !containsString(err.Error(), "Test Gem") {
			t.Errorf("Error() should contain gem name, got %q", err.Error())
		}
		if err.GemID != "gem-id" {
			t.Errorf("GemID = %q, want 'gem-id'", err.GemID)
		}
		if err.GemName != "Test Gem" {
			t.Errorf("GemName = %q, want 'Test Gem'", err.GemName)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := NewGemNotFoundError("my-gem")
		if !containsString(err.Error(), "not found") {
			t.Errorf("Error() should contain 'not found', got %q", err.Error())
		}
		if !containsString(err.Error(), "my-gem") {
			t.Errorf("Error() should contain gem identifier, got %q", err.Error())
		}
	})

	t.Run("read only", func(t *testing.T) {
		err := NewGemReadOnlyError("System Gem")
		if !containsString(err.Error(), "cannot modify") {
			t.Errorf("Error() should contain 'cannot modify', got %q", err.Error())
		}
		if !containsString(err.Error(), "System Gem") {
			t.Errorf("Error() should contain gem name, got %q", err.Error())
		}
		if err.GemName != "System Gem" {
			t.Errorf("GemName = %q, want 'System Gem'", err.GemName)
		}
	})

	t.Run("create error", func(t *testing.T) {
		err := NewGemCreateError("New Gem", "validation failed")
		if !containsString(err.Error(), "validation failed") {
			t.Errorf("Error() should contain message, got %q", err.Error())
		}
		if err.GemName != "New Gem" {
			t.Errorf("GemName = %q, want 'New Gem'", err.GemName)
		}
		if err.Operation != "create gem" {
			t.Errorf("Operation = %q, want 'create gem'", err.Operation)
		}
	})

	t.Run("update error", func(t *testing.T) {
		err := NewGemUpdateError("gem-123", "not found")
		if err.GemID != "gem-123" {
			t.Errorf("GemID = %q, want 'gem-123'", err.GemID)
		}
		if err.Operation != "update gem" {
			t.Errorf("Operation = %q, want 'update gem'", err.Operation)
		}
	})

	t.Run("delete error", func(t *testing.T) {
		err := NewGemDeleteError("gem-456", "permission denied")
		if err.GemID != "gem-456" {
			t.Errorf("GemID = %q, want 'gem-456'", err.GemID)
		}
		if err.Operation != "delete gem" {
			t.Errorf("Operation = %q, want 'delete gem'", err.Operation)
		}
	})

	t.Run("fetch error", func(t *testing.T) {
		err := NewGemFetchError("network error")
		if !containsString(err.Error(), "network error") {
			t.Errorf("Error() should contain message, got %q", err.Error())
		}
		if err.Operation != "fetch gems" {
			t.Errorf("Operation = %q, want 'fetch gems'", err.Operation)
		}
	})

	t.Run("error with only ID", func(t *testing.T) {
		err := &GemError{
			GeminiError: &GeminiError{Message: "test"},
			GemID:       "only-id",
		}
		if !containsString(err.Error(), "ID: only-id") {
			t.Errorf("Error() should show ID when no name, got %q", err.Error())
		}
	})

	t.Run("error without ID or name", func(t *testing.T) {
		err := &GemError{
			GeminiError: &GeminiError{Message: "generic error"},
		}
		expected := "gem error: generic error"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Is method", func(t *testing.T) {
		err := NewGemError("id", "name", "msg")
		if !err.Is(NewGemError("other", "other", "other")) {
			t.Error("GemError should match other GemError")
		}
		if err.Is(NewAPIError(500, "", "")) {
			t.Error("GemError should not match APIError")
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("underlying")
		err := &GemError{
			GeminiError: &GeminiError{Cause: cause},
		}
		if err.Unwrap() != cause {
			t.Error("Unwrap should return the cause")
		}
	})

	t.Run("endpoint is set to batch execute", func(t *testing.T) {
		err := NewGemError("id", "name", "msg")
		expected := "https://gemini.google.com/_/BardChatUi/data/batchexecute"
		if err.Endpoint != expected {
			t.Errorf("Endpoint = %q, want %q", err.Endpoint, expected)
		}
	})
}

func TestIsGemError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"GemError", NewGemError("id", "name", "msg"), true},
		{"GemNotFoundError", NewGemNotFoundError("name"), true},
		{"GemReadOnlyError", NewGemReadOnlyError("name"), true},
		{"GemCreateError", NewGemCreateError("name", "msg"), true},
		{"GemUpdateError", NewGemUpdateError("id", "msg"), true},
		{"GemDeleteError", NewGemDeleteError("id", "msg"), true},
		{"GemFetchError", NewGemFetchError("msg"), true},
		{"wrapped GemError", fmt.Errorf("wrapped: %w", NewGemError("id", "name", "msg")), true},
		{"other error", errors.New("other"), false},
		{"APIError", NewAPIError(500, "", ""), false},
		{"UploadError", NewUploadError("file", "msg"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGemError(tt.err); got != tt.expected {
				t.Errorf("IsGemError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Ensure interfaces are implemented
var _ error = (*GeminiError)(nil)
var _ error = (*AuthError)(nil)
var _ error = (*APIError)(nil)
var _ error = (*NetworkError)(nil)
var _ error = (*TimeoutError)(nil)
var _ error = (*UsageLimitError)(nil)
var _ error = (*ModelError)(nil)
var _ error = (*BlockedError)(nil)
var _ error = (*ParseError)(nil)
var _ error = (*UploadError)(nil)
var _ error = (*GemError)(nil)

// Benchmark tests
func BenchmarkIsAuthError(b *testing.B) {
	err := NewAuthError("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsAuthError(err)
	}
}

func BenchmarkGetHTTPStatus(b *testing.B) {
	err := NewAPIError(500, "endpoint", "message")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetHTTPStatus(err)
	}
}

// Ensure time package is used
var _ = time.Second
