package commands

import (
	"fmt"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/models"
)

// createChatSession creates a configured chat session with optional gem
func createChatSession(client *api.GeminiClient, gemID string, model models.Model) *api.ChatSession {
	opts := []api.ChatOption{
		api.WithChatModel(model),
	}
	if gemID != "" {
		opts = append(opts, api.WithGemID(gemID))
	}
	return client.StartChatWithOptions(opts...)
}

// resolveGemFlag resolves the --gem flag to a gem ID
// Returns empty string if no gem specified
func resolveGemFlag(client *api.GeminiClient, gemFlag string) (string, error) {
	if gemFlag == "" {
		return "", nil
	}

	gem, err := resolveGem(client, gemFlag)
	if err != nil {
		return "", fmt.Errorf("failed to resolve gem '%s': %w", gemFlag, err)
	}

	return gem.ID, nil
}
