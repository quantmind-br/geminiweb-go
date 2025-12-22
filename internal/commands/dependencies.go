package commands

import (
	"github.com/diogo/geminiweb/internal/api"
)

// Dependencies holds the external dependencies for the commands.
// This allows for dependency injection and easier testing.
type Dependencies struct {
	// Client is the Gemini API client.
	Client api.GeminiClientInterface
}

// NewDependencies creates a new Dependencies struct with default implementations.
func NewDependencies() *Dependencies {
	return &Dependencies{
		// Dependencies will be lazily initialized or set by the caller
	}
}
