// Package errors provides custom error types for the Gemini Web API client.
package errors

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// ErrorCode represents known API error codes from Gemini
type ErrorCode int

const (
	ErrCodeUnknown            ErrorCode = 0
	ErrCodePromptTooLong      ErrorCode = 3
	ErrCodeUsageLimitExceeded ErrorCode = 1037
	ErrCodeModelInconsistent  ErrorCode = 1050
	ErrCodeModelHeaderInvalid ErrorCode = 1052
	ErrCodeIPBlocked          ErrorCode = 1060
)

// String returns a human-readable description of the error code
func (c ErrorCode) String() string {
	switch c {
	case ErrCodePromptTooLong:
		return "prompt too long - reduce input size"
	case ErrCodeUsageLimitExceeded:
		return "usage limit exceeded"
	case ErrCodeModelInconsistent:
		return "model inconsistent with chat history"
	case ErrCodeModelHeaderInvalid:
		return "model header invalid or model unavailable"
	case ErrCodeIPBlocked:
		return "IP temporarily blocked"
	default:
		return "unknown error"
	}
}

// Sentinel errors for common cases
var (
	ErrAuthFailed      = errors.New("authentication failed")
	ErrCookiesExpired  = errors.New("cookies have expired")
	ErrNoCookies       = errors.New("no cookies found")
	ErrInvalidResponse = errors.New("invalid response format")
	ErrNoContent       = errors.New("no content in response")
	ErrNetworkFailure  = errors.New("network failure")
	ErrTimeout         = errors.New("request timed out")
	ErrRateLimited     = errors.New("rate limited")
)

// GeminiError is the base error type for all Gemini API errors.
// It implements the error interface and provides rich context for debugging.
type GeminiError struct {
	// Code is the internal error code from the Gemini API (e.g., 1037, 1050)
	Code ErrorCode

	// HTTPStatus is the HTTP status code (e.g., 401, 500)
	HTTPStatus int

	// Endpoint is the API endpoint that was called
	Endpoint string

	// Operation describes what was being attempted (e.g., "generate content", "get access token")
	Operation string

	// Message is a human-readable error message
	Message string

	// Body contains the response body (truncated) for diagnostic purposes
	Body string

	// Cause is the underlying error that caused this error
	Cause error
}

// Error implements the error interface
func (e *GeminiError) Error() string {
	var parts []string

	if e.Operation != "" {
		parts = append(parts, e.Operation+" failed")
	}

	if e.HTTPStatus > 0 {
		parts = append(parts, fmt.Sprintf("HTTP %d", e.HTTPStatus))
	}

	if e.Code != ErrCodeUnknown {
		parts = append(parts, fmt.Sprintf("code=%d (%s)", e.Code, e.Code.String()))
	}

	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	if e.Endpoint != "" {
		parts = append(parts, fmt.Sprintf("endpoint=%s", e.Endpoint))
	}

	if len(parts) == 0 {
		return "gemini error"
	}

	return strings.Join(parts, ": ")
}

// Unwrap returns the underlying cause
func (e *GeminiError) Unwrap() error {
	return e.Cause
}

// Is implements error matching for errors.Is()
func (e *GeminiError) Is(target error) bool {
	switch target {
	case ErrAuthFailed:
		return e.HTTPStatus == 401 || e.IsAuth()
	case ErrRateLimited:
		return e.Code == ErrCodeUsageLimitExceeded || e.HTTPStatus == 429
	case ErrNetworkFailure:
		return e.IsNetwork()
	case ErrTimeout:
		return e.IsTimeout()
	}

	if t, ok := target.(*GeminiError); ok {
		return e.Code == t.Code || (e.HTTPStatus != 0 && e.HTTPStatus == t.HTTPStatus)
	}

	return false
}

// IsAuth returns true if this is an authentication error
func (e *GeminiError) IsAuth() bool {
	return e.HTTPStatus == 401
}

// IsRateLimit returns true if this is a rate limit error
func (e *GeminiError) IsRateLimit() bool {
	return e.Code == ErrCodeUsageLimitExceeded || e.HTTPStatus == 429
}

// IsNetwork returns true if this is a network error
func (e *GeminiError) IsNetwork() bool {
	if e.Cause == nil {
		return false
	}

	var netErr net.Error
	return errors.As(e.Cause, &netErr)
}

// IsTimeout returns true if this is a timeout error
func (e *GeminiError) IsTimeout() bool {
	if e.Cause == nil {
		return false
	}

	var netErr net.Error
	if errors.As(e.Cause, &netErr) {
		return netErr.Timeout()
	}

	return false
}

// IsBlocked returns true if the IP is blocked
func (e *GeminiError) IsBlocked() bool {
	return e.Code == ErrCodeIPBlocked
}

// IsModelError returns true if this is a model-related error
func (e *GeminiError) IsModelError() bool {
	return e.Code == ErrCodeModelInconsistent || e.Code == ErrCodeModelHeaderInvalid
}

// WithBody adds the response body to the error (truncated for safety)
func (e *GeminiError) WithBody(body string) *GeminiError {
	const maxBodyLen = 500
	if len(body) > maxBodyLen {
		e.Body = body[:maxBodyLen] + "...(truncated)"
	} else {
		e.Body = body
	}
	return e
}

// NewGeminiError creates a new GeminiError with the given parameters
func NewGeminiError(operation string, message string) *GeminiError {
	return &GeminiError{
		Operation: operation,
		Message:   message,
	}
}

// NewGeminiErrorWithStatus creates a GeminiError with HTTP status
func NewGeminiErrorWithStatus(operation string, httpStatus int, endpoint string, message string) *GeminiError {
	return &GeminiError{
		Operation:  operation,
		HTTPStatus: httpStatus,
		Endpoint:   endpoint,
		Message:    message,
	}
}

// NewGeminiErrorWithCode creates a GeminiError with an API error code
func NewGeminiErrorWithCode(operation string, code ErrorCode, endpoint string) *GeminiError {
	return &GeminiError{
		Operation: operation,
		Code:      code,
		Endpoint:  endpoint,
		Message:   code.String(),
	}
}

// NewGeminiErrorWithCause creates a GeminiError wrapping another error
func NewGeminiErrorWithCause(operation string, cause error) *GeminiError {
	return &GeminiError{
		Operation: operation,
		Cause:     cause,
	}
}

// AuthError represents an authentication failure
type AuthError struct {
	*GeminiError
}

// NewAuthError creates a new AuthError
func NewAuthError(message string) *AuthError {
	return &AuthError{
		GeminiError: &GeminiError{
			HTTPStatus: 401,
			Operation:  "authentication",
			Message:    message,
		},
	}
}

// NewAuthErrorWithEndpoint creates a new AuthError with endpoint info
func NewAuthErrorWithEndpoint(message string, endpoint string) *AuthError {
	return &AuthError{
		GeminiError: &GeminiError{
			HTTPStatus: 401,
			Operation:  "authentication",
			Endpoint:   endpoint,
			Message:    message,
		},
	}
}

// Error implements the error interface
func (e *AuthError) Error() string {
	if e.GeminiError.Message == "" {
		return "authentication failed: cookies may have expired"
	}
	return fmt.Sprintf("authentication failed: %s", e.GeminiError.Message)
}

// Is allows comparison with sentinel errors
func (e *AuthError) Is(target error) bool {
	if target == ErrAuthFailed {
		return true
	}
	if _, ok := target.(*AuthError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying GeminiError
func (e *AuthError) Unwrap() error {
	return e.GeminiError.Cause
}

// APIError represents an API request failure
type APIError struct {
	*GeminiError
}

// NewAPIError creates a new APIError
func NewAPIError(statusCode int, endpoint, message string) *APIError {
	return &APIError{
		GeminiError: &GeminiError{
			HTTPStatus: statusCode,
			Endpoint:   endpoint,
			Operation:  "API request",
			Message:    message,
		},
	}
}

// NewAPIErrorWithCode creates an APIError with an internal error code
func NewAPIErrorWithCode(code ErrorCode, endpoint string) *APIError {
	return &APIError{
		GeminiError: &GeminiError{
			Code:      code,
			Endpoint:  endpoint,
			Operation: "API request",
			Message:   code.String(),
		},
	}
}

// NewAPIErrorWithBody creates an APIError with the response body
func NewAPIErrorWithBody(statusCode int, endpoint, message, body string) *APIError {
	e := NewAPIError(statusCode, endpoint, message)
	e.GeminiError.WithBody(body)
	return e
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.GeminiError.HTTPStatus > 0 {
		if e.GeminiError.Body != "" {
			return fmt.Sprintf("API error [%d] at %s: %s (body: %s)",
				e.GeminiError.HTTPStatus, e.GeminiError.Endpoint, e.GeminiError.Message, e.GeminiError.Body)
		}
		return fmt.Sprintf("API error [%d] at %s: %s",
			e.GeminiError.HTTPStatus, e.GeminiError.Endpoint, e.GeminiError.Message)
	}
	return fmt.Sprintf("API error at %s: %s", e.GeminiError.Endpoint, e.GeminiError.Message)
}

// StatusCode returns the HTTP status code (for backwards compatibility)
func (e *APIError) StatusCode() int {
	return e.GeminiError.HTTPStatus
}

// Is allows comparison with other errors
func (e *APIError) Is(target error) bool {
	if target == ErrAuthFailed && e.GeminiError.HTTPStatus == 401 {
		return true
	}
	if target == ErrRateLimited && (e.GeminiError.HTTPStatus == 429 || e.GeminiError.Code == ErrCodeUsageLimitExceeded) {
		return true
	}
	if t, ok := target.(*APIError); ok {
		return e.GeminiError.HTTPStatus == t.GeminiError.HTTPStatus
	}
	return false
}

// Unwrap returns the underlying error
func (e *APIError) Unwrap() error {
	return e.GeminiError.Cause
}

// NetworkError represents a network-level failure
type NetworkError struct {
	*GeminiError
}

// NewNetworkError creates a new NetworkError
func NewNetworkError(operation string, cause error) *NetworkError {
	return &NetworkError{
		GeminiError: &GeminiError{
			Operation: operation,
			Message:   "network request failed",
			Cause:     cause,
		},
	}
}

// NewNetworkErrorWithEndpoint creates a NetworkError with endpoint info
func NewNetworkErrorWithEndpoint(operation string, endpoint string, cause error) *NetworkError {
	return &NetworkError{
		GeminiError: &GeminiError{
			Operation: operation,
			Endpoint:  endpoint,
			Message:   "network request failed",
			Cause:     cause,
		},
	}
}

// Error implements the error interface
func (e *NetworkError) Error() string {
	if e.GeminiError.Cause != nil {
		return fmt.Sprintf("network error during %s: %v", e.GeminiError.Operation, e.GeminiError.Cause)
	}
	return fmt.Sprintf("network error during %s", e.GeminiError.Operation)
}

// Is allows comparison with sentinel errors
func (e *NetworkError) Is(target error) bool {
	if target == ErrNetworkFailure {
		return true
	}
	if target == ErrTimeout && e.GeminiError.IsTimeout() {
		return true
	}
	if _, ok := target.(*NetworkError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *NetworkError) Unwrap() error {
	return e.GeminiError.Cause
}

// TimeoutError represents a request timeout
type TimeoutError struct {
	*GeminiError
}

// NewTimeoutError creates a new TimeoutError
func NewTimeoutError(message string) *TimeoutError {
	return &TimeoutError{
		GeminiError: &GeminiError{
			Operation: "request",
			Message:   message,
		},
	}
}

// NewTimeoutErrorWithEndpoint creates a TimeoutError with endpoint info
func NewTimeoutErrorWithEndpoint(endpoint string, cause error) *TimeoutError {
	return &TimeoutError{
		GeminiError: &GeminiError{
			Operation: "request",
			Endpoint:  endpoint,
			Message:   "request timed out",
			Cause:     cause,
		},
	}
}

// Error implements the error interface
func (e *TimeoutError) Error() string {
	if e.GeminiError.Message == "" {
		return "request timed out"
	}
	return fmt.Sprintf("request timed out: %s", e.GeminiError.Message)
}

// Is allows comparison with sentinel errors
func (e *TimeoutError) Is(target error) bool {
	if target == ErrTimeout {
		return true
	}
	if _, ok := target.(*TimeoutError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *TimeoutError) Unwrap() error {
	return e.GeminiError.Cause
}

// UsageLimitError represents a usage limit exceeded error
type UsageLimitError struct {
	*GeminiError
}

// NewUsageLimitError creates a new UsageLimitError
func NewUsageLimitError(modelName string) *UsageLimitError {
	return &UsageLimitError{
		GeminiError: &GeminiError{
			Code:      ErrCodeUsageLimitExceeded,
			Operation: "generate content",
			Message:   fmt.Sprintf("usage limit exceeded for model %s", modelName),
		},
	}
}

// Error implements the error interface
func (e *UsageLimitError) Error() string {
	if e.GeminiError.Message == "" {
		return "usage limit exceeded"
	}
	return fmt.Sprintf("usage limit exceeded: %s", e.GeminiError.Message)
}

// Is allows comparison with sentinel errors
func (e *UsageLimitError) Is(target error) bool {
	if target == ErrRateLimited {
		return true
	}
	if _, ok := target.(*UsageLimitError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *UsageLimitError) Unwrap() error {
	return e.GeminiError.Cause
}

// ModelError represents a model-related error
type ModelError struct {
	*GeminiError
}

// NewModelError creates a new ModelError
func NewModelError(message string) *ModelError {
	return &ModelError{
		GeminiError: &GeminiError{
			Operation: "model selection",
			Message:   message,
		},
	}
}

// NewModelErrorWithCode creates a ModelError with an error code
func NewModelErrorWithCode(code ErrorCode) *ModelError {
	return &ModelError{
		GeminiError: &GeminiError{
			Code:      code,
			Operation: "model selection",
			Message:   code.String(),
		},
	}
}

// Error implements the error interface
func (e *ModelError) Error() string {
	return fmt.Sprintf("model error: %s", e.GeminiError.Message)
}

// Is allows comparison with other ModelErrors
func (e *ModelError) Is(target error) bool {
	if _, ok := target.(*ModelError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *ModelError) Unwrap() error {
	return e.GeminiError.Cause
}

// BlockedError represents an IP block error
type BlockedError struct {
	*GeminiError
}

// NewBlockedError creates a new BlockedError
func NewBlockedError(message string) *BlockedError {
	return &BlockedError{
		GeminiError: &GeminiError{
			Code:      ErrCodeIPBlocked,
			Operation: "API request",
			Message:   message,
		},
	}
}

// Error implements the error interface
func (e *BlockedError) Error() string {
	if e.GeminiError.Message == "" {
		return "content blocked"
	}
	return fmt.Sprintf("blocked: %s", e.GeminiError.Message)
}

// Is allows comparison with other BlockedErrors
func (e *BlockedError) Is(target error) bool {
	if _, ok := target.(*BlockedError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *BlockedError) Unwrap() error {
	return e.GeminiError.Cause
}

// ParseError represents a response parsing error
type ParseError struct {
	*GeminiError
	Path string // JSON path where parsing failed
}

// NewParseError creates a new ParseError
func NewParseError(message, path string) *ParseError {
	return &ParseError{
		GeminiError: &GeminiError{
			Operation: "parse response",
			Message:   message,
		},
		Path: path,
	}
}

// Error implements the error interface
func (e *ParseError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("parse error at %s: %s", e.Path, e.GeminiError.Message)
	}
	return fmt.Sprintf("parse error: %s", e.GeminiError.Message)
}

// Is allows comparison with sentinel errors
func (e *ParseError) Is(target error) bool {
	if target == ErrInvalidResponse {
		return true
	}
	if _, ok := target.(*ParseError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *ParseError) Unwrap() error {
	return e.GeminiError.Cause
}

// PromptTooLongError represents an error when the prompt exceeds the model's context limit
type PromptTooLongError struct {
	*GeminiError
}

// NewPromptTooLongError creates a new PromptTooLongError
func NewPromptTooLongError(modelName string) *PromptTooLongError {
	return &PromptTooLongError{
		GeminiError: &GeminiError{
			Code:      ErrCodePromptTooLong,
			Operation: "generate content",
			Message:   fmt.Sprintf("prompt too long for model %s - try reducing input size", modelName),
		},
	}
}

// Error implements the error interface
func (e *PromptTooLongError) Error() string {
	return fmt.Sprintf("prompt too long: %s", e.GeminiError.Message)
}

// Is allows comparison with other PromptTooLongErrors
func (e *PromptTooLongError) Is(target error) bool {
	if _, ok := target.(*PromptTooLongError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *PromptTooLongError) Unwrap() error {
	return e.GeminiError.Cause
}

// HandleErrorCode converts API error codes to appropriate errors
func HandleErrorCode(code ErrorCode, endpoint, modelName string) error {
	switch code {
	case ErrCodePromptTooLong:
		return NewPromptTooLongError(modelName)
	case ErrCodeUsageLimitExceeded:
		return NewUsageLimitError(modelName)
	case ErrCodeModelInconsistent:
		return NewModelErrorWithCode(code)
	case ErrCodeModelHeaderInvalid:
		return NewModelErrorWithCode(code)
	case ErrCodeIPBlocked:
		return NewBlockedError("IP temporarily blocked by Google")
	default:
		return NewAPIErrorWithCode(code, endpoint)
	}
}

// IsAuthError checks if an error is an authentication error
// using errors.Is and errors.As for proper error wrapping support
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	// Check using errors.Is for sentinel error
	if errors.Is(err, ErrAuthFailed) {
		return true
	}

	// Check for AuthError type
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return true
	}

	// Check for APIError with 401 status
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GeminiError.HTTPStatus == 401
	}

	// Check for GeminiError with 401 status
	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.HTTPStatus == 401
	}

	return false
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrNetworkFailure) {
		return true
	}

	var netErr *NetworkError
	return errors.As(err, &netErr)
}

// IsTimeoutError checks if an error is a timeout error
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrTimeout) {
		return true
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return true
	}

	// Check for net.Error timeout
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	return false
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrRateLimited) {
		return true
	}

	var usageErr *UsageLimitError
	return errors.As(err, &usageErr)
}

// GetHTTPStatus extracts the HTTP status code from an error, if available
func GetHTTPStatus(err error) int {
	if err == nil {
		return 0
	}

	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.HTTPStatus
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GeminiError.HTTPStatus
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.GeminiError.HTTPStatus
	}

	return 0
}

// GetErrorCode extracts the Gemini error code from an error, if available
func GetErrorCode(err error) ErrorCode {
	if err == nil {
		return ErrCodeUnknown
	}

	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.Code
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GeminiError.Code
	}

	return ErrCodeUnknown
}

// GetEndpoint extracts the endpoint from an error, if available
func GetEndpoint(err error) string {
	if err == nil {
		return ""
	}

	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.Endpoint
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GeminiError.Endpoint
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.GeminiError.Endpoint
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return netErr.GeminiError.Endpoint
	}

	return ""
}
