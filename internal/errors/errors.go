// Package errors provides custom error types for the Gemini Web API client.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common cases
var (
	ErrAuthFailed      = errors.New("authentication failed")
	ErrCookiesExpired  = errors.New("cookies have expired")
	ErrNoCookies       = errors.New("no cookies found")
	ErrInvalidResponse = errors.New("invalid response format")
	ErrNoContent       = errors.New("no content in response")
)

// AuthError represents an authentication failure
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	if e.Message == "" {
		return "authentication failed: cookies may have expired"
	}
	return fmt.Sprintf("authentication failed: %s", e.Message)
}

// Is allows comparison with sentinel errors
func (e *AuthError) Is(target error) bool {
	// Match with ErrAuthFailed sentinel error
	if target == ErrAuthFailed {
		return true
	}
	// Match with another AuthError (for error wrapping/unwrapping)
	_, ok := target.(*AuthError)
	return ok
}

// NewAuthError creates a new AuthError
func NewAuthError(message string) *AuthError {
	return &AuthError{Message: message}
}

// APIError represents an API request failure
type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

func (e *APIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("API error [%d] at %s: %s", e.StatusCode, e.Endpoint, e.Message)
	}
	return fmt.Sprintf("API error at %s: %s", e.Endpoint, e.Message)
}

// NewAPIError creates a new APIError
func NewAPIError(statusCode int, endpoint, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Endpoint:   endpoint,
		Message:    message,
	}
}

// TimeoutError represents a request timeout
type TimeoutError struct {
	Message string
}

func (e *TimeoutError) Error() string {
	if e.Message == "" {
		return "request timed out"
	}
	return fmt.Sprintf("request timed out: %s", e.Message)
}

// NewTimeoutError creates a new TimeoutError
func NewTimeoutError(message string) *TimeoutError {
	return &TimeoutError{Message: message}
}

// UsageLimitError represents a usage limit exceeded error
type UsageLimitError struct {
	Message string
}

func (e *UsageLimitError) Error() string {
	if e.Message == "" {
		return "usage limit exceeded"
	}
	return fmt.Sprintf("usage limit exceeded: %s", e.Message)
}

// NewUsageLimitError creates a new UsageLimitError
func NewUsageLimitError(message string) *UsageLimitError {
	return &UsageLimitError{Message: message}
}

// ModelError represents a model-related error
type ModelError struct {
	Message string
}

func (e *ModelError) Error() string {
	return fmt.Sprintf("model error: %s", e.Message)
}

// NewModelError creates a new ModelError
func NewModelError(message string) *ModelError {
	return &ModelError{Message: message}
}

// BlockedError represents an IP block error
type BlockedError struct {
	Message string
}

func (e *BlockedError) Error() string {
	if e.Message == "" {
		return "content blocked"
	}
	return fmt.Sprintf("content blocked: %s", e.Message)
}

// NewBlockedError creates a new BlockedError
func NewBlockedError(message string) *BlockedError {
	return &BlockedError{Message: message}
}

// ParseError represents a response parsing error
type ParseError struct {
	Message string
	Path    string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error: %s", e.Message)
}

// NewParseError creates a new ParseError
func NewParseError(message, path string) *ParseError {
	return &ParseError{Message: message, Path: path}
}

// Is allows comparison with sentinel errors
func (e *ParseError) Is(target error) bool {
	// Match with ErrInvalidResponse sentinel error
	if target == ErrInvalidResponse {
		return true
	}
	// Match with another ParseError (for error wrapping/unwrapping)
	_, ok := target.(*ParseError)
	if ok {
		return true
	}
	// Match with standard errors containing "parse error"
	return target != nil && target.Error() == "parse error"
}
