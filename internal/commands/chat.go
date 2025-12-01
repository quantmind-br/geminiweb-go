package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/tui"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with Gemini.

The chat maintains conversation context across messages.
Type 'exit', 'quit', or press Ctrl+C to end the session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChat()
	},
}

func runChat() error {
	modelName := getModel()
	model := models.ModelFromName(modelName)

	// Build client options
	clientOpts := []api.ClientOption{
		api.WithModel(model),
		api.WithAutoRefresh(true),
	}

	// Add browser refresh if enabled (also enables silent auto-login fallback)
	if browserType, enabled := getBrowserRefresh(); enabled {
		clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
	}

	// Create client with nil cookies - Init() will load from disk or browser
	client, err := api.NewClient(nil, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Initialize client with animation
	// Init() handles cookie loading from disk and browser fallback
	spin := newSpinner("Connecting to Gemini")
	spin.start()
	if err := client.Init(); err != nil {
		spin.stopWithError()
		return fmt.Errorf("failed to initialize: %w", err)
	}
	spin.stopWithSuccess("Connected")

	// Run chat TUI
	return tui.RunChat(client, modelName)
}
