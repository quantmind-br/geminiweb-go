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
	const maxBodyLen = 1000
	if len(body) > maxBodyLen {
		e.Body = body[:maxBodyLen] + "...(truncated)"
	} else {
		e.Body = body
	}
	return e
}

// GetBody returns the response body stored in the error
func (e *GeminiError) GetBody() string {
	return e.Body
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
	if e.Message == "" {
		return "authentication failed: cookies may have expired"
	}
	return fmt.Sprintf("authentication failed: %s", e.Message)
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
	return e.Cause
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
	_ = e.WithBody(body)
	return e
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.HTTPStatus > 0 {
		if e.Body != "" {
			return fmt.Sprintf("API error [%d] at %s: %s (body: %s)",
				e.HTTPStatus, e.Endpoint, e.Message, e.Body)
		}
		return fmt.Sprintf("API error [%d] at %s: %s",
			e.HTTPStatus, e.Endpoint, e.Message)
	}
	return fmt.Sprintf("API error at %s: %s", e.Endpoint, e.Message)
}

// StatusCode returns the HTTP status code (for backwards compatibility)
func (e *APIError) StatusCode() int {
	return e.HTTPStatus
}

// Is allows comparison with other errors
func (e *APIError) Is(target error) bool {
	if target == ErrAuthFailed && e.HTTPStatus == 401 {
		return true
	}
	if target == ErrRateLimited && (e.HTTPStatus == 429 || e.Code == ErrCodeUsageLimitExceeded) {
		return true
	}
	if t, ok := target.(*APIError); ok {
		return e.HTTPStatus == t.HTTPStatus
	}
	return false
}

// Unwrap returns the underlying error
func (e *APIError) Unwrap() error {
	return e.Cause
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
	if e.Cause != nil {
		return fmt.Sprintf("network error during %s: %v", e.Operation, e.Cause)
	}
	return fmt.Sprintf("network error during %s", e.Operation)
}

// Is allows comparison with sentinel errors
func (e *NetworkError) Is(target error) bool {
	if target == ErrNetworkFailure {
		return true
	}
	if target == ErrTimeout && e.IsTimeout() {
		return true
	}
	if _, ok := target.(*NetworkError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *NetworkError) Unwrap() error {
	return e.Cause
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
	if e.Message == "" {
		return "request timed out"
	}
	return fmt.Sprintf("request timed out: %s", e.Message)
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
	return e.Cause
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
	if e.Message == "" {
		return "usage limit exceeded"
	}
	return fmt.Sprintf("usage limit exceeded: %s", e.Message)
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
	return e.Cause
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
	return fmt.Sprintf("model error: %s", e.Message)
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
	return e.Cause
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
	if e.Message == "" {
		return "content blocked"
	}
	return fmt.Sprintf("blocked: %s", e.Message)
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
	return e.Cause
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
		return fmt.Sprintf("parse error at %s: %s", e.Path, e.Message)
	}
	return fmt.Sprintf("parse error: %s", e.Message)
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
	return e.Cause
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
	return fmt.Sprintf("prompt too long: %s", e.Message)
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
	return e.Cause
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
		return apiErr.HTTPStatus == 401
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
		return apiErr.HTTPStatus
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.HTTPStatus
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
		return apiErr.Code
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
		return apiErr.Endpoint
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.Endpoint
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return netErr.Endpoint
	}

	var uploadErr *UploadError
	if errors.As(err, &uploadErr) {
		return uploadErr.Endpoint
	}

	return ""
}

// GetResponseBody extracts the response body from an error, if available
func GetResponseBody(err error) string {
	if err == nil {
		return ""
	}

	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.Body
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Body
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.Body
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return netErr.Body
	}

	var uploadErr *UploadError
	if errors.As(err, &uploadErr) {
		return uploadErr.Body
	}

	return ""
}

// UploadError represents a file upload failure
type UploadError struct {
	*GeminiError
	FileName string
}

// NewUploadError creates a new UploadError
func NewUploadError(fileName, message string) *UploadError {
	return &UploadError{
		GeminiError: &GeminiError{
			Operation: "upload file",
			Endpoint:  "https://content-push.googleapis.com/upload",
			Message:   message,
		},
		FileName: fileName,
	}
}

// NewUploadErrorWithStatus creates an UploadError with HTTP status
func NewUploadErrorWithStatus(fileName string, statusCode int, body string) *UploadError {
	e := &UploadError{
		GeminiError: &GeminiError{
			HTTPStatus: statusCode,
			Operation:  "upload file",
			Endpoint:   "https://content-push.googleapis.com/upload",
			Message:    fmt.Sprintf("upload failed with status %d", statusCode),
		},
		FileName: fileName,
	}
	_ = e.WithBody(body)
	return e
}

// NewUploadNetworkError creates an UploadError for network failures
func NewUploadNetworkError(fileName string, cause error) *UploadError {
	return &UploadError{
		GeminiError: &GeminiError{
			Operation: "upload file",
			Endpoint:  "https://content-push.googleapis.com/upload",
			Message:   "network request failed",
			Cause:     cause,
		},
		FileName: fileName,
	}
}

// Error implements the error interface
func (e *UploadError) Error() string {
	if e.FileName != "" {
		if e.HTTPStatus > 0 {
			return fmt.Sprintf("upload error for '%s': HTTP %d - %s",
				e.FileName, e.HTTPStatus, e.Message)
		}
		if e.Cause != nil {
			return fmt.Sprintf("upload error for '%s': %v", e.FileName, e.Cause)
		}
		return fmt.Sprintf("upload error for '%s': %s", e.FileName, e.Message)
	}
	return fmt.Sprintf("upload error: %s", e.Message)
}

// Is allows comparison with other errors
func (e *UploadError) Is(target error) bool {
	if _, ok := target.(*UploadError); ok {
		return true
	}
	if target == ErrNetworkFailure && e.Cause != nil {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *UploadError) Unwrap() error {
	return e.Cause
}

// IsUploadError checks if an error is an upload error
func IsUploadError(err error) bool {
	if err == nil {
		return false
	}

	var uploadErr *UploadError
	return errors.As(err, &uploadErr)
}

// DownloadError represents an error during image download
type DownloadError struct {
	*GeminiError
	URL string
}

// NewDownloadError creates a new DownloadError
func NewDownloadError(message, url string) *DownloadError {
	return &DownloadError{
		GeminiError: &GeminiError{
			Operation: "download image",
			Message:   message,
		},
		URL: url,
	}
}

// NewDownloadErrorWithStatus creates a DownloadError with HTTP status
func NewDownloadErrorWithStatus(url string, statusCode int) *DownloadError {
	return &DownloadError{
		GeminiError: &GeminiError{
			HTTPStatus: statusCode,
			Operation:  "download image",
			Message:    fmt.Sprintf("download failed with status %d", statusCode),
		},
		URL: url,
	}
}

// NewDownloadNetworkError creates a DownloadError for network failures
func NewDownloadNetworkError(url string, cause error) *DownloadError {
	return &DownloadError{
		GeminiError: &GeminiError{
			Operation: "download image",
			Message:   "network request failed",
			Cause:     cause,
		},
		URL: url,
	}
}

// Error implements the error interface
func (e *DownloadError) Error() string {
	if e.URL != "" {
		if e.HTTPStatus > 0 {
			return fmt.Sprintf("download error for '%s': HTTP %d - %s",
				e.URL, e.HTTPStatus, e.Message)
		}
		if e.Cause != nil {
			return fmt.Sprintf("download error for '%s': %v", e.URL, e.Cause)
		}
		return fmt.Sprintf("download error for '%s': %s", e.URL, e.Message)
	}
	return fmt.Sprintf("download error: %s", e.Message)
}

// Is allows comparison with other errors
func (e *DownloadError) Is(target error) bool {
	if _, ok := target.(*DownloadError); ok {
		return true
	}
	if target == ErrNetworkFailure && e.Cause != nil {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *DownloadError) Unwrap() error {
	return e.Cause
}

// IsDownloadError checks if an error is a download error
func IsDownloadError(err error) bool {
	if err == nil {
		return false
	}

	var downloadErr *DownloadError
	return errors.As(err, &downloadErr)
}

// GemError represents errors specific to gem operations
type GemError struct {
	*GeminiError
	GemID   string
	GemName string
}

// NewGemError creates a new generic gem error
func NewGemError(gemID, gemName, message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "gem operation",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
		GemID:   gemID,
		GemName: gemName,
	}
}

// NewGemNotFoundError creates an error for gem not found
func NewGemNotFoundError(idOrName string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "get gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   fmt.Sprintf("gem '%s' not found", idOrName),
		},
	}
}

// NewGemReadOnlyError creates an error for attempting to modify a system gem
func NewGemReadOnlyError(gemName string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "modify gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   fmt.Sprintf("cannot modify system gem '%s'", gemName),
		},
		GemName: gemName,
	}
}

// NewGemCreateError creates an error for gem creation failure
func NewGemCreateError(name, message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "create gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
		GemName: name,
	}
}

// NewGemUpdateError creates an error for gem update failure
func NewGemUpdateError(gemID, message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "update gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
		GemID: gemID,
	}
}

// NewGemDeleteError creates an error for gem deletion failure
func NewGemDeleteError(gemID, message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "delete gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
		GemID: gemID,
	}
}

// NewGemFetchError creates an error for gem fetch failure
func NewGemFetchError(message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "fetch gems",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
	}
}

// Error implements the error interface
func (e *GemError) Error() string {
	if e.GemName != "" {
		return fmt.Sprintf("gem error (%s): %s", e.GemName, e.Message)
	}
	if e.GemID != "" {
		return fmt.Sprintf("gem error (ID: %s): %s", e.GemID, e.Message)
	}
	return fmt.Sprintf("gem error: %s", e.Message)
}

// Is allows comparison with other errors
func (e *GemError) Is(target error) bool {
	if _, ok := target.(*GemError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *GemError) Unwrap() error {
	return e.Cause
}

// IsGemError checks if an error is a gem error
func IsGemError(err error) bool {
	if err == nil {
		return false
	}

	var gemErr *GemError
	return errors.As(err, &gemErr)
}
