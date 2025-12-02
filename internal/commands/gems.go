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
	"github.com/diogo/geminiweb/internal/tui"
)

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

var gemsCmd = &cobra.Command{
	Use:   "gems",
	Short: "Manage Gemini Gems (server-side personas)",
	Long: `Gems are custom personas stored on Google's servers.
Unlike local personas, gems sync across devices with your Google account.

Use 'geminiweb gems list' to see available gems.
Use 'geminiweb gems create' to create a new gem.`,
}

var gemsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all gems",
	RunE:  runGemsList,
}

var gemsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new gem",
	Args:  cobra.ExactArgs(1),
	RunE:  runGemsCreate,
}

var gemsUpdateCmd = &cobra.Command{
	Use:   "update <id-or-name>",
	Short: "Update an existing gem",
	Args:  cobra.ExactArgs(1),
	RunE:  runGemsUpdate,
}

var gemsDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-name>",
	Short: "Delete a gem",
	Args:  cobra.ExactArgs(1),
	RunE:  runGemsDelete,
}

var gemsShowCmd = &cobra.Command{
	Use:   "show <id-or-name>",
	Short: "Show gem details",
	Args:  cobra.ExactArgs(1),
	RunE:  runGemsShow,
}

// Flags
var (
	gemsIncludeHidden bool
	gemPrompt         string
	gemDescription    string
	gemPromptFile     string
	gemName           string
)

func init() {
	gemsCmd.AddCommand(gemsListCmd)
	gemsCmd.AddCommand(gemsCreateCmd)
	gemsCmd.AddCommand(gemsUpdateCmd)
	gemsCmd.AddCommand(gemsDeleteCmd)
	gemsCmd.AddCommand(gemsShowCmd)

	// Flags
	gemsListCmd.Flags().BoolVar(&gemsIncludeHidden, "hidden", false, "Include hidden system gems")

	gemsCreateCmd.Flags().StringVarP(&gemPrompt, "prompt", "p", "", "System prompt for the gem")
	gemsCreateCmd.Flags().StringVarP(&gemDescription, "description", "d", "", "Description")
	gemsCreateCmd.Flags().StringVarP(&gemPromptFile, "file", "f", "", "Read prompt from file")

	gemsUpdateCmd.Flags().StringVarP(&gemPrompt, "prompt", "p", "", "New system prompt")
	gemsUpdateCmd.Flags().StringVarP(&gemDescription, "description", "d", "", "New description")
	gemsUpdateCmd.Flags().StringVarP(&gemPromptFile, "file", "f", "", "Read prompt from file")
	gemsUpdateCmd.Flags().StringVarP(&gemName, "name", "n", "", "New name for the gem")
}

func runGemsList(cmd *cobra.Command, args []string) error {
	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Launch the interactive TUI for gems
	return tui.RunGemsTUI(client, gemsIncludeHidden)
}

func runGemsCreate(cmd *cobra.Command, args []string) error {
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

	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

	gem, err := client.CreateGem(name, prompt, gemDescription)
	if err != nil {
		return fmt.Errorf("failed to create gem: %w", err)
	}

	fmt.Printf("Created gem '%s' with ID: %s\n", gem.Name, gem.ID)
	return nil
}

func runGemsUpdate(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

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

func runGemsDelete(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

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

func runGemsShow(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

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
func resolveGem(client *api.GeminiClient, idOrName string) (*models.Gem, error) {
	gems, err := client.FetchGems(false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gems: %w", err)
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
