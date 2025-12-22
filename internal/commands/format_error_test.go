package commands

import (
	"strings"
	"testing"

	apierrors "github.com/diogo/geminiweb/internal/errors"
)

func TestFormatErrorMessage_Nil(t *testing.T) {
	if got := formatErrorMessage(nil, "ctx"); got != "" {
		t.Fatalf("expected empty for nil error, got %s", got)
	}
}

func TestFormatErrorMessage_APIError(t *testing.T) {
	e := apierrors.NewAPIErrorWithBody(500, "/endpoint", "failure", "detailed body")
	out := formatErrorMessage(e, "Failed")
	if out == "" {
		t.Fatalf("expected non-empty message")
	}
	if !strings.Contains(out, "HTTP Status") && !strings.Contains(out, "Endpoint") {
		t.Fatalf("expected HTTP Status or Endpoint in message, got: %s", out)
	}
}

func TestFormatErrorMessage_OtherErrors(t *testing.T) {
	// Auth error
	auth := apierrors.NewAuthErrorWithEndpoint("auth failed", "/auth")
	if out := formatErrorMessage(auth, "Auth"); out == "" {
		t.Fatalf("expected non-empty for auth error")
	}

	// Usage limit error
	usage := apierrors.NewUsageLimitError("model-x")
	if out := formatErrorMessage(usage, "Usage"); out == "" {
		t.Fatalf("expected non-empty for usage limit error")
	}

	// Network error
	netErr := apierrors.NewNetworkErrorWithEndpoint("fetch", "/endpoint", nil)
	if out := formatErrorMessage(netErr, "Net"); out == "" {
		t.Fatalf("expected non-empty for network error")
	}

	// Timeout error
	timeout := apierrors.NewTimeoutErrorWithEndpoint("/endpoint", nil)
	if out := formatErrorMessage(timeout, "Timeout"); out == "" {
		t.Fatalf("expected non-empty for timeout error")
	}

	// Upload error (APIError with status 400)
	upload := apierrors.NewAPIErrorWithBody(400, "/upload", "bad", "upload failed")
	if out := formatErrorMessage(upload, "Upload"); out == "" {
		t.Fatalf("expected non-empty for upload error")
	}

	// Ensure the output contains hints for known error types when body is absent
	noBodyAuth := apierrors.NewAuthError("auth")
	if out := formatErrorMessage(noBodyAuth, "Auth"); !strings.Contains(out, "Hint") {
		t.Fatalf("expected hint in auth error message, got: %s", out)
	}
}
