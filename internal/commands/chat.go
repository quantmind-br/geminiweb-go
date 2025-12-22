package commands

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

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

// chatFileFlag is the --file flag for providing initial prompt from file
var chatFileFlag string

// NewChatCmd creates a new chat command
func NewChatCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Start an interactive chat session",
		Long: `Start an interactive chat session with Gemini.

The chat maintains conversation context across messages.
Type 'exit', 'quit', or press Ctrl+C to end the session.

HISTORY:
  By default, a history selector lets you resume previous conversations
  or start a new one. Use --new to skip the selector and start fresh.
  Conversations are automatically saved to ~/.geminiweb/history/

INITIAL PROMPT FROM FILE:
  Use --file to start the chat with content from a file:
    geminiweb chat --file context.md
    geminiweb chat -f prompt.txt --new

  The file content is sent as the first message, and the chat
  continues interactively. Useful for:
    - Loading project context
    - Starting with predefined prompts
    - Code review sessions

  Combine with other flags:
    geminiweb chat -f task.md --gem "Code Helper"
    geminiweb chat -f context.md --persona coder

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
			return runChat(deps)
		},
	}

	cmd.Flags().StringVarP(&chatGemFlag, "gem", "g", "", "Use a gem (by ID or name) - server-side persona")
	cmd.Flags().BoolVarP(&chatNewFlag, "new", "n", false, "Start a new conversation (skip history selector)")
	cmd.Flags().StringVarP(&chatPersonaFlag, "persona", "p", "", "Use a local persona (system prompt)")
	cmd.Flags().StringVarP(&chatFileFlag, "file", "f", "", "Read initial prompt from file")

	return cmd
}

// Backward compatibility global
var chatCmd = NewChatCmd(nil)

func init() {
	// Flags and command structure are now handled in NewChatCmd and NewRootCmd
}

// maxFileSize is the maximum file size for initial prompt (1MB)
const maxFileSize = 1 * 1024 * 1024

func runChat(deps *Dependencies) error {
	modelName := getModel()
	model := models.ModelFromName(modelName)

	// Determine TUI and Client implementations
	var tuiImpl TUIInterface = &DefaultTUI{}
	if deps != nil && deps.TUI != nil {
		tuiImpl = deps.TUI
	}

	// Read initial prompt from file if specified
	var initialPrompt string
	if chatFileFlag != "" {
		// Check file size first
		info, err := os.Stat(chatFileFlag)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", chatFileFlag)
			}
			return fmt.Errorf("failed to access file '%s': %w", chatFileFlag, err)
		}
		if info.Size() > maxFileSize {
			return fmt.Errorf("file '%s' is too large (max 1MB)", chatFileFlag)
		}

		// Read file content
		data, err := os.ReadFile(chatFileFlag)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", chatFileFlag, err)
		}

		// Validate content
		if !utf8.Valid(data) {
			return fmt.Errorf("file '%s' appears to be binary, not text", chatFileFlag)
		}

		initialPrompt = strings.TrimSpace(string(data))
		if initialPrompt == "" {
			return fmt.Errorf("file '%s' is empty", chatFileFlag)
		}
	}

	// Initialize history store
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to initialize history: %w", err)
	}

	// Select conversation (new or existing)
	var selectedConv *history.Conversation
	if !chatNewFlag {
		result, err := tuiImpl.RunHistorySelector(store, modelName)
		if err != nil {
			return fmt.Errorf("history selector error: %w", err)
		}

		// User quit without selecting
		if !result.Confirmed {
			return nil
		}

		selectedConv = result.Conversation
	}

	var client api.GeminiClientInterface
	if deps != nil && deps.Client != nil {
		client = deps.Client
	} else {
		// Load config for auto-close settings
		cfg, _ := config.LoadConfig()

		// Build client options
		clientOpts := []api.ClientOption{
			api.WithModel(model),
			api.WithAutoRefresh(true),
		}

		// Add browser refresh if enabled (also enables silent auto-login fallback)
		if browserType, enabled := getBrowserRefresh(); enabled {
			clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
		}

		// Add auto-close options from config
		if cfg.AutoClose {
			clientOpts = append(clientOpts,
				api.WithAutoClose(true),
				api.WithCloseDelay(time.Duration(cfg.CloseDelay)*time.Second),
				api.WithAutoReInit(cfg.AutoReInit),
			)
		}

		// Create client with nil cookies - Init() will load from disk or browser
		client, err = api.NewClient(nil, clientOpts...)
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
	}

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

	// Run chat TUI with conversation, gem name, persona, and initial prompt
	return tuiImpl.RunChatWithInitialPrompt(client, session, modelName, selectedConv, store, resolvedGem.Name, persona, initialPrompt)
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