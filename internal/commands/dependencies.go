package commands

import (
	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/tui"
)

// TUIInterface defines the methods required from the TUI package.
type TUIInterface interface {
	RunGemsTUI(client api.GeminiClientInterface, includeHidden bool) (tui.GemsTUIResult, error)
	RunChatWithSession(client api.GeminiClientInterface, session tui.ChatSessionInterface, modelName string) error
}

// Dependencies holds the external dependencies for the commands.
// This allows for dependency injection and easier testing.
type Dependencies struct {
	// Client is the Gemini API client.
	Client api.GeminiClientInterface

	// TUI is the terminal user interface.
	TUI TUIInterface
}

// DefaultTUI is the production implementation of TUIInterface.
type DefaultTUI struct{}

func (d *DefaultTUI) RunGemsTUI(client api.GeminiClientInterface, includeHidden bool) (tui.GemsTUIResult, error) {
	return tui.RunGemsTUI(client, includeHidden)
}

func (d *DefaultTUI) RunChatWithSession(client api.GeminiClientInterface, session tui.ChatSessionInterface, modelName string) error {
	return tui.RunChatWithSession(client, session, modelName)
}

// NewDependencies creates a new Dependencies struct with default implementations.
func NewDependencies() *Dependencies {
	return &Dependencies{
		TUI: &DefaultTUI{},
	}
}
