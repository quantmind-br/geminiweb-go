package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/models"
)

// createGemsClientFunc is a variable that can be overridden for testing
var createGemsClientFunc = func() (api.GeminiClientInterface, error) {
	return createGemsClient()
}

// GemReaderInterface defines the interface for reading gem input
type GemReaderInterface interface {
	ReadString(delim byte) (string, error)
}

// GemStdInReader is the default implementation of GemReaderInterface
type GemStdInReader struct {
	reader *bufio.Reader
}

// NewGemStdInReader creates a new GemStdInReader
func NewGemStdInReader(reader io.Reader) GemReaderInterface {
	return &GemStdInReader{
		reader: bufio.NewReader(reader),
	}
}

// ReadString implements GemReaderInterface
func (r *GemStdInReader) ReadString(delim byte) (string, error) {
	return r.reader.ReadString(delim)
}

// NewGemsCmd creates a new gems command
func NewGemsCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gems",
		Short: "Manage Gemini Gems (server-side personas)",
		Long: `Gems are custom personas stored on Google's servers.
Unlike local personas, gems sync across devices with your Google account.

INTERACTIVE MODE (default):
  Run 'geminiweb gems' to open the interactive gems manager where you can:
  - Browse and search all gems
  - Create new custom gems
  - Edit existing gems
  - Delete gems
  - Start a chat with any gem

KEYBOARD SHORTCUTS (interactive mode):
  ↑↓       Navigate gems list
  /        Search/filter gems
  n        Create new gem
  e        Edit selected gem
  d        Delete selected gem
  c        Start chat with selected gem
  y        Copy gem ID to clipboard
  Enter    View gem details
  q        Quit

QUICK START:
  geminiweb gems             # Open interactive gems manager
  geminiweb chat --gem code  # Start chat with a gem directly`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGemsInteractive(deps, args)
		},
	}

	cmd.AddCommand(NewGemsListCmd(deps))
	cmd.AddCommand(NewGemsCreateCmd(deps))
	cmd.AddCommand(NewGemsUpdateCmd(deps))
	cmd.AddCommand(NewGemsDeleteCmd(deps))
	cmd.AddCommand(NewGemsShowCmd(deps))

	return cmd
}

// NewGemsListCmd creates a new gems list command
func NewGemsListCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all gems",
		Long: `Browse all gems in an interactive TUI.

KEYBOARD SHORTCUTS:
  c        Start chat with selected gem
  y        Copy gem ID to clipboard
  Enter    View gem details
  /        Search gems
  q        Quit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGemsList(deps, args)
		},
	}

	cmd.Flags().BoolVar(&gemsIncludeHidden, "hidden", false, "Include hidden system gems")
	return cmd
}

// NewGemsCreateCmd creates a new gems create command
func NewGemsCreateCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new gem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGemsCreate(deps, args)
		},
	}

	cmd.Flags().StringVarP(&gemPrompt, "prompt", "p", "", "System prompt for the gem")
	cmd.Flags().StringVarP(&gemDescription, "description", "d", "", "Description")
	cmd.Flags().StringVarP(&gemPromptFile, "file", "f", "", "Read prompt from file")

	return cmd
}

// NewGemsUpdateCmd creates a new gems update command
func NewGemsUpdateCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id-or-name>",
		Short: "Update an existing gem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGemsUpdate(deps, args)
		},
	}

	cmd.Flags().StringVarP(&gemPrompt, "prompt", "p", "", "New system prompt")
	cmd.Flags().StringVarP(&gemDescription, "description", "d", "", "New description")
	cmd.Flags().StringVarP(&gemPromptFile, "file", "f", "", "Read prompt from file")
	cmd.Flags().StringVarP(&gemName, "name", "n", "", "New name for the gem")

	return cmd
}

// NewGemsDeleteCmd creates a new gems delete command
func NewGemsDeleteCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id-or-name>",
		Short: "Delete a gem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGemsDelete(deps, args)
		},
	}
}

// NewGemsShowCmd creates a new gems show command
func NewGemsShowCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id-or-name>",
		Short: "Show gem details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGemsShow(deps, args)
		},
	}
}

// Backward compatibility globals
var gemsCmd = NewGemsCmd(nil)
var gemsListCmd = NewGemsListCmd(nil)
var gemsCreateCmd = NewGemsCreateCmd(nil)
var gemsUpdateCmd = NewGemsUpdateCmd(nil)
var gemsDeleteCmd = NewGemsDeleteCmd(nil)
var gemsShowCmd = NewGemsShowCmd(nil)

// Flags
var (
	gemsIncludeHidden bool
	gemPrompt         string
	gemDescription    string
	gemPromptFile     string
	gemName           string
)

func init() {
	// Root flags and commands are handled in NewRootCmd
}

// runGemsInteractive runs the interactive gems menu (default when no subcommand)
func runGemsInteractive(deps *Dependencies, args []string) error {
	var client api.GeminiClientInterface
	var err error
	if deps != nil && deps.Client != nil {
		client = deps.Client
	} else {
		client, err = createGemsClientFunc()
		if err != nil {
			return err
		}
		defer client.Close()
	}

	// Determine which TUI implementation to use
	var tuiImpl TUIInterface = &DefaultTUI{}
	if deps != nil && deps.TUI != nil {
		tuiImpl = deps.TUI
	}

	// Launch the interactive TUI for gems with full functionality
	result, err := tuiImpl.RunGemsTUI(client, gemsIncludeHidden)
	if err != nil {
		return err
	}

	// Check if user wants to start chat with a gem
	if result.GemID != "" {
		modelName := getModel()
		model := models.ModelFromName(modelName)
		session := createChatSession(client, result.GemID, model)
		return tuiImpl.RunChatWithSession(client, session, modelName)
	}

	return nil
}

func runGemsList(deps *Dependencies, args []string) error {
	var client api.GeminiClientInterface
	var err error
	if deps != nil && deps.Client != nil {
		client = deps.Client
	} else {
		client, err = createGemsClientFunc()
		if err != nil {
			return err
		}
		defer client.Close()
	}

	// Determine which TUI implementation to use
	var tuiImpl TUIInterface = &DefaultTUI{}
	if deps != nil && deps.TUI != nil {
		tuiImpl = deps.TUI
	}

	// Launch the interactive TUI for gems
	result, err := tuiImpl.RunGemsTUI(client, gemsIncludeHidden)
	if err != nil {
		return err
	}

	// Check if user wants to start chat with a gem
	if result.GemID != "" {
		// Create session with the selected gem
		modelName := getModel()
		model := models.ModelFromName(modelName)
		session := createChatSession(client, result.GemID, model)

		// Run chat TUI
		return tuiImpl.RunChatWithSession(client, session, modelName)
	}

	return nil
}

func runGemsCreate(deps *Dependencies, args []string) error {
	name := args[0]

	prompt := gemPrompt
	if gemPromptFile != "" {
		data, err := os.ReadFile(gemPromptFile)
		if err != nil {
			return fmt.Errorf("failed to read prompt file: %w", err)
		}
		prompt = string(data)
	}

	if prompt == "" {
		return fmt.Errorf("prompt is required (use -p or -f)")
	}

	var client api.GeminiClientInterface
	var err error
	if deps != nil && deps.Client != nil {
		client = deps.Client
	} else {
		client, err = createGemsClientFunc()
		if err != nil {
			return err
		}
		defer client.Close()
	}

	gem, err := client.CreateGem(name, prompt, gemDescription)
	if err != nil {
		return fmt.Errorf("failed to create gem: %w", err)
	}

	fmt.Printf("Created gem '%s' with ID: %s\n", gem.Name, gem.ID)
	return nil
}

func runGemsUpdate(deps *Dependencies, args []string) error {
	idOrName := args[0]

	var client api.GeminiClientInterface
	var err error
	if deps != nil && deps.Client != nil {
		client = deps.Client
	} else {
		client, err = createGemsClientFunc()
		if err != nil {
			return err
		}
		defer client.Close()
	}

	gems, err := client.FetchGems(false)
	if err != nil {
		return fmt.Errorf("failed to fetch gems: %w", err)
	}

	gem := gems.Get(idOrName, idOrName)
	if gem == nil {
		return fmt.Errorf("gem '%s' not found", idOrName)
	}

	if gem.Predefined {
		return fmt.Errorf("cannot update system gems")
	}

	// Use existing values if not provided
	newPrompt := gem.Prompt
	newDesc := gem.Description
	newName := gem.Name

	if gemPromptFile != "" {
		data, err := os.ReadFile(gemPromptFile)
		if err != nil {
			return fmt.Errorf("failed to read prompt file: %w", err)
		}
		newPrompt = string(data)
	} else if gemPrompt != "" {
		newPrompt = gemPrompt
	}

	if gemDescription != "" {
		newDesc = gemDescription
	}

	if gemName != "" {
		newName = gemName
	}

	updated, err := client.UpdateGem(gem.ID, newName, newPrompt, newDesc)
	if err != nil {
		return fmt.Errorf("failed to update gem: %w", err)
	}

	fmt.Printf("Updated gem '%s'\n", updated.Name)
	return nil
}

func runGemsDelete(deps *Dependencies, args []string) error {
	idOrName := args[0]

	var client api.GeminiClientInterface
	var err error
	if deps != nil && deps.Client != nil {
		client = deps.Client
	} else {
		client, err = createGemsClientFunc()
		if err != nil {
			return err
		}
		defer client.Close()
	}

	gems, err := client.FetchGems(false)
	if err != nil {
		return fmt.Errorf("failed to fetch gems: %w", err)
	}

	gem := gems.Get(idOrName, idOrName)
	if gem == nil {
		return fmt.Errorf("gem '%s' not found", idOrName)
	}

	if gem.Predefined {
		return fmt.Errorf("cannot delete system gems")
	}

	if err := client.DeleteGem(gem.ID); err != nil {
		return fmt.Errorf("failed to delete gem: %w", err)
	}

	fmt.Printf("Deleted gem '%s'\n", gem.Name)
	return nil
}

func runGemsShow(deps *Dependencies, args []string) error {
	idOrName := args[0]

	var client api.GeminiClientInterface
	var err error
	if deps != nil && deps.Client != nil {
		client = deps.Client
	} else {
		client, err = createGemsClientFunc()
		if err != nil {
			return err
		}
		defer client.Close()
	}

	gems, err := client.FetchGems(true)
	if err != nil {
		return fmt.Errorf("failed to fetch gems: %w", err)
	}

	gem := gems.Get(idOrName, idOrName)
	if gem == nil {
		return fmt.Errorf("gem '%s' not found", idOrName)
	}

	fmt.Printf("ID:          %s\n", gem.ID)
	fmt.Printf("Name:        %s\n", gem.Name)
	fmt.Printf("Description: %s\n", gem.Description)
	gemType := "custom"
	if gem.Predefined {
		gemType = "system"
	}
	fmt.Printf("Type:        %s\n", gemType)
	fmt.Printf("\nPrompt:\n%s\n", gem.Prompt)

	return nil
}

// createGemsClient creates a GeminiClient configured for gems operations
func createGemsClient() (*api.GeminiClient, error) {
	// Build client options
	clientOpts := []api.ClientOption{
		api.WithAutoRefresh(false),
	}

	// Add browser refresh if enabled
	if browserType, enabled := getBrowserRefresh(); enabled {
		clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
	}

	// Create client with nil cookies - Init() will load from disk or browser
	client, err := api.NewClient(nil, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Initialize client
	if err := client.Init(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return client, nil
}

// resolveGem resolves a gem by ID or name using the provided client
// Returns the gem ID if found, empty string otherwise
func resolveGem(client api.GeminiClientInterface, idOrName string) (*models.Gem, error) {
	gems, err := client.FetchGems(false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gems: %w", err)
	}

	if gems == nil {
		return nil, fmt.Errorf("gem '%s' not found (no gems available)", idOrName)
	}

	gem := gems.Get(idOrName, idOrName)

	if gem == nil {
		return nil, fmt.Errorf("gem '%s' not found", idOrName)
	}

	return gem, nil
}

// parseGemPromptFromStdin reads a multi-line prompt from stdin
func parseGemPromptFromStdin(reader io.Reader) (string, error) {
	gemReader := NewGemStdInReader(reader)
	fmt.Println("Enter system prompt (end with an empty line):")
	var promptLines []string
	for {
		line, err := gemReader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			break
		}
		promptLines = append(promptLines, line)
	}
	return strings.Join(promptLines, "\n"), nil
}
