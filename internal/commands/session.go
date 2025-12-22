package commands

import (
	"fmt"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/models"
)

// createChatSession creates a configured chat session with optional gem
func createChatSession(client api.GeminiClientInterface, gemID string, model models.Model) *api.ChatSession {
	opts := []api.ChatOption{
		api.WithChatModel(model),
	}
	if gemID != "" {
		opts = append(opts, api.WithGemID(gemID))
	}
	return client.StartChatWithOptions(opts...)
}

// ResolvedGem contains the resolved gem ID and name
type ResolvedGem struct {
	ID   string
	Name string
}

// resolveGemFlag resolves the --gem flag to a gem ID and name
// Returns empty ResolvedGem if no gem specified
func resolveGemFlag(client api.GeminiClientInterface, gemFlag string) (ResolvedGem, error) {
	if gemFlag == "" {
		return ResolvedGem{}, nil
	}

	gem, err := resolveGem(client, gemFlag)
	if err != nil {
		return ResolvedGem{}, fmt.Errorf("failed to resolve gem '%s': %w", gemFlag, err)
	}

	return ResolvedGem{ID: gem.ID, Name: gem.Name}, nil
}
