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
	RPCListGems  = "CNgdBe"
	RPCCreateGem = "oMH3Zd"
	RPCUpdateGem = "kHv0Vd"
	RPCDeleteGem = "UXcSJb"
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
	// ModelUnspecified uses the server's default model (no model header sent)
	ModelUnspecified = Model{
		Name:   "unspecified",
		Header: nil,
	}

	Model25Flash = Model{
		Name: "gemini-2.5-flash",
		Header: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"71c2d248d3b102ff",null,null,0,[4],null,null,2]`,
		},
	}

	Model30Pro = Model{
		Name: "gemini-3.0-pro",
		Header: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"e6fa609c3fa255c0",null,null,0,[4],null,null,2]`,
		},
	}

	// DefaultModel is the recommended default
	DefaultModel = Model30Pro

	// GeminiGenericHeader is the header used for generic Gem operations
	GeminiGenericHeader = map[string]string{
		"x-goog-ext-525001261-jspb": `[1,null,null,null,null,null,null,null,[4]]`,
	}
)

// AllModels returns a list of all available models
func AllModels() []Model {
	return []Model{Model25Flash, Model30Pro}
}

// ModelFromName returns a Model by its name
func ModelFromName(name string) Model {
	switch name {
	case "gemini-2.5-flash":
		return Model25Flash
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
		"Content-Type":              "application/x-www-form-urlencoded;charset=utf-8",
		"Host":                      "gemini.google.com",
		"Origin":                    "https://gemini.google.com",
		"Referer":                   "https://gemini.google.com/",
		"User-Agent":                "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "en-US,en;q=0.9",
		"Accept-Encoding":           "gzip, deflate, br, zstd",
		"Sec-CH-UA":                 `"Google Chrome";v="133", "Chromium";v="133", "Not_A Brand";v="24"`,
		"Sec-CH-UA-Mobile":          "?0",
		"Sec-CH-UA-Platform":        `"Linux"`,
		"Sec-Fetch-Site":            "same-origin",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-User":            "?1",
		"Sec-Fetch-Dest":            "document",
		"Upgrade-Insecure-Requests": "1",
		"X-Same-Domain":             "1",
		"x-goog-ext-73010989-jspb":  "[0]", // Required safety/feature flag header
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
