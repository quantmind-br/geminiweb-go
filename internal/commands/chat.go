package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/history"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/tui"
)

// chatGemFlag is the --gem flag for the chat command
var chatGemFlag string

// chatNewFlag bypasses history selector and starts a new conversation
var chatNewFlag bool

// chatPersonaFlag is the --persona flag for the chat command
var chatPersonaFlag string

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with Gemini.

The chat maintains conversation context across messages.
Type 'exit', 'quit', or press Ctrl+C to end the session.

HISTORY:
  By default, a history selector lets you resume previous conversations
  or start a new one. Use --new to skip the selector and start fresh.
  Conversations are automatically saved to ~/.geminiweb/history/

GEMS (Server-side Personas):
  Use --gem to start the chat with a specific gem:
    geminiweb chat --gem "Code Helper"
    geminiweb chat -g code

  During chat, type /gems to switch gems without leaving the chat.
  The active gem is shown in the header.

LOCAL PERSONAS:
  Use --persona to apply a local system prompt:
    geminiweb chat --persona coder
    geminiweb chat -p writer

  Local personas are defined in ~/.geminiweb/personas.json
  Use 'geminiweb persona list' to see available personas.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChat()
	},
}

func init() {
	chatCmd.Flags().StringVarP(&chatGemFlag, "gem", "g", "", "Use a gem (by ID or name) - server-side persona")
	chatCmd.Flags().BoolVarP(&chatNewFlag, "new", "n", false, "Start a new conversation (skip history selector)")
	chatCmd.Flags().StringVarP(&chatPersonaFlag, "persona", "p", "", "Use a local persona (system prompt)")
}

func runChat() error {
	modelName := getModel()
	model := models.ModelFromName(modelName)

	// Initialize history store
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to initialize history: %w", err)
	}

	// Select conversation (new or existing)
	var selectedConv *history.Conversation
	if !chatNewFlag {
		result, err := tui.RunHistorySelector(store, modelName)
		if err != nil {
			return fmt.Errorf("history selector error: %w", err)
		}

		// User quit without selecting
		if !result.Confirmed {
			return nil
		}

		selectedConv = result.Conversation
	}

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

	// Resolve gem if specified
	resolvedGem, err := resolveGemFlag(client, chatGemFlag)
	if err != nil {
		return err
	}

	// Resolve persona if specified
	var persona *config.Persona
	if chatPersonaFlag != "" {
		persona, err = config.GetPersona(chatPersonaFlag)
		if err != nil {
			return fmt.Errorf("failed to load persona '%s': %w", chatPersonaFlag, err)
		}
	} else {
		// Check for default persona (if not "default")
		defaultPersona, err := config.GetDefaultPersona()
		if err == nil && defaultPersona != nil && defaultPersona.Name != "default" && defaultPersona.SystemPrompt != "" {
			persona = defaultPersona
		}
	}

	// Create or resume conversation
	if selectedConv == nil {
		// New conversation - create in store
		selectedConv, err = store.CreateConversation(modelName)
		if err != nil {
			return fmt.Errorf("failed to create conversation: %w", err)
		}
	}

	// Create session with conversation context
	session := createChatSessionWithConversation(client, resolvedGem.ID, model, selectedConv)

	// Run chat TUI with conversation, gem name, and persona
	return tui.RunChatWithPersona(client, session, modelName, selectedConv, store, resolvedGem.Name, persona)
}

// createChatSessionWithConversation creates a chat session, optionally resuming from a conversation
func createChatSessionWithConversation(client api.GeminiClientInterface, gemID string, model models.Model, conv *history.Conversation) tui.ChatSessionInterface {
	session := client.StartChat()
	session.SetModel(model)

	// Set gem if specified
	if gemID != "" {
		session.SetGem(gemID)
	}

	// Resume conversation context if we have metadata
	if conv != nil && conv.CID != "" {
		session.SetMetadata(conv.CID, conv.RID, conv.RCID)
	}

	return session
}
