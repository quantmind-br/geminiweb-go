// Package models contains data types and constants for the Gemini Web API.
package models

import apierrors "github.com/diogo/geminiweb/internal/errors"

// Endpoints for Gemini Web API
const (
	EndpointGoogle        = "https://www.google.com"
	EndpointInit          = "https://gemini.google.com/app"
	EndpointGenerate      = "https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate"
	EndpointRotateCookies = "https://accounts.google.com/RotateCookies"
	EndpointUpload        = "https://content-push.googleapis.com/upload"
	EndpointBatchExec     = "https://gemini.google.com/_/BardChatUi/data/batchexecute"
)

// RPC IDs para operações de Gems (batch execute)
const (
	RPCListGems   = "CNgdBe"
	RPCCreateGem  = "oMH3Zd"
	RPCUpdateGem  = "kHv0Vd"
	RPCDeleteGem  = "UXcSJb"
)

// Parâmetros para ListGems
const (
	ListGemsNormal        = 3 // Gems normais (visíveis na UI)
	ListGemsIncludeHidden = 4 // Incluir gems ocultos de sistema
	ListGemsCustom        = 2 // Gems customizados do usuário
)

// Model represents available Gemini models with their headers
type Model struct {
	Name   string
	Header map[string]string
}

// Available models
var (
	ModelUnspecified = Model{
		Name:   "unspecified",
		Header: nil,
	}

	Model25Flash = Model{
		Name: "gemini-2.5-flash",
		Header: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"9ec249fc9ad08861",null,null,0,[4]]`,
		},
	}

	Model25Pro = Model{
		Name: "gemini-2.5-pro",
		Header: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"4af6c7f5da75d65d",null,null,0,[4]]`,
		},
	}

	Model30Pro = Model{
		Name: "gemini-3.0-pro",
		Header: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"9d8ca3786ebdfbea",null,null,0,[4]]`,
		},
	}
)

// AllModels returns a list of all available models
func AllModels() []Model {
	return []Model{Model25Flash, Model25Pro, Model30Pro}
}

// ModelFromName returns a Model by its name
func ModelFromName(name string) Model {
	switch name {
	case "gemini-2.5-flash":
		return Model25Flash
	case "gemini-2.5-pro":
		return Model25Pro
	case "gemini-3.0-pro":
		return Model30Pro
	default:
		return ModelUnspecified
	}
}

// ErrorCode represents known API error codes
// Deprecated: Use errors.ErrorCode instead. These are kept for backward compatibility.
type ErrorCode = apierrors.ErrorCode

// Error code constants - aliased from errors package for backward compatibility
const (
	ErrUsageLimitExceeded = apierrors.ErrCodeUsageLimitExceeded
	ErrModelInconsistent  = apierrors.ErrCodeModelInconsistent
	ErrModelHeaderInvalid = apierrors.ErrCodeModelHeaderInvalid
	ErrIPBlocked          = apierrors.ErrCodeIPBlocked
)

// DefaultHeaders returns the default headers for Gemini requests
func DefaultHeaders() map[string]string {
	return map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded;charset=utf-8",
		"Host":          "gemini.google.com",
		"Origin":        "https://gemini.google.com",
		"Referer":       "https://gemini.google.com/",
		"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"X-Same-Domain": "1",
	}
}

// RotateCookiesHeaders returns headers for the cookie rotation endpoint
func RotateCookiesHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// UploadHeaders returns headers for the file upload endpoint
func UploadHeaders() map[string]string {
	return map[string]string{
		"Push-ID": "feeds/mcudyrk2a4khkz",
	}
}
