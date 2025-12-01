// Package models contains data types and constants for the Gemini Web API.
package models

// Endpoints for Gemini Web API
const (
	EndpointGoogle        = "https://www.google.com"
	EndpointInit          = "https://gemini.google.com/app"
	EndpointGenerate      = "https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate"
	EndpointRotateCookies = "https://accounts.google.com/RotateCookies"
	EndpointUpload        = "https://content-push.googleapis.com/upload"
	EndpointBatchExec     = "https://gemini.google.com/_/BardChatUi/data/batchexecute"
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
type ErrorCode int

const (
	ErrUsageLimitExceeded ErrorCode = 1037
	ErrModelInconsistent  ErrorCode = 1050
	ErrModelHeaderInvalid ErrorCode = 1052
	ErrIPBlocked          ErrorCode = 1060
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
