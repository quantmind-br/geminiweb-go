package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/tui"
)

// PersonaReaderInterface defines the interface for reading persona input
type PersonaReaderInterface interface {
	ReadString(delim byte) (string, error)
}

// StdInReader is the default implementation of PersonaReaderInterface
type StdInReader struct {
	reader *bufio.Reader
}

// NewStdInReader creates a new StdInReader
func NewStdInReader(reader io.Reader) PersonaReaderInterface {
	return &StdInReader{
		reader: bufio.NewReader(reader),
	}
}

// ReadString implements PersonaReaderInterface
func (r *StdInReader) ReadString(delim byte) (string, error) {
	return r.reader.ReadString(delim)
}

// NewPersonaCmd creates a new persona command
func NewPersonaCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "persona",
		Short: "Manage chat personas",
		Long:  `View and manage personas (system prompts) for chat sessions.`,
	}

	cmd.AddCommand(NewPersonaListCmd(deps))
	cmd.AddCommand(NewPersonaShowCmd(deps))
	cmd.AddCommand(NewPersonaAddCmd(deps))
	cmd.AddCommand(NewPersonaDeleteCmd(deps))
	cmd.AddCommand(NewPersonaSetDefaultCmd(deps))
	cmd.AddCommand(NewPersonaManageCmd(deps))

	return cmd
}

func NewPersonaListCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available personas",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPersonaList(cmd, args)
		},
	}
}

func NewPersonaShowCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show persona details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPersonaShow(cmd, args)
		},
	}
}

func NewPersonaAddCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new persona",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPersonaAdd(cmd, args)
		},
	}
}

func NewPersonaDeleteCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a persona",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPersonaDelete(cmd, args)
		},
	}
}

func NewPersonaSetDefaultCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "default <name>",
		Short: "Set default persona",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPersonaSetDefault(cmd, args)
		},
	}
}

func NewPersonaManageCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "manage",
		Short: "Launch the persona manager TUI",
		Long:  `Launch an interactive Terminal UI for managing personas with create, edit, delete, and set default operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPersonaManage(cmd, args)
		},
	}
}

// Backward compatibility globals
var personaCmd = NewPersonaCmd(nil)
var personaListCmd = NewPersonaListCmd(nil)
var personaShowCmd = NewPersonaShowCmd(nil)
var personaAddCmd = NewPersonaAddCmd(nil)
var personaDeleteCmd = NewPersonaDeleteCmd(nil)
var personaSetDefaultCmd = NewPersonaSetDefaultCmd(nil)

func init() {
	// Root flags and commands are handled in NewRootCmd
}

func runPersonaList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadPersonas()
	if err != nil {
		return fmt.Errorf("failed to load personas: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDESCRIPTION\tDEFAULT")
	_, _ = fmt.Fprintln(w, "----\t-----------\t-------")

	for _, p := range cfg.Personas {
		isDefault := ""
		if p.Name == cfg.DefaultPersona {
			isDefault = "✓"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Description, isDefault)
	}

	return w.Flush()
}

func runPersonaShow(cmd *cobra.Command, args []string) error {
	persona, err := config.GetPersona(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Name: %s\n", persona.Name)
	fmt.Printf("Description: %s\n", persona.Description)
	if persona.Model != "" {
		fmt.Printf("Preferred Model: %s\n", persona.Model)
	}
	fmt.Printf("\nSystem Prompt:\n%s\n", persona.SystemPrompt)

	return nil
}

func runPersonaAdd(cmd *cobra.Command, args []string) error {
	return runPersonaAddWithReader(os.Stdin, args)
}

// runPersonaAddWithReader is the internal implementation that accepts a reader for testing
func runPersonaAddWithReader(reader io.Reader, args []string) error {
	name := args[0]

	// Check if already exists
	if _, err := config.GetPersona(name); err == nil {
		return fmt.Errorf("persona '%s' already exists", name)
	}

	personaReader := NewStdInReader(reader)

	fmt.Print("Enter description: ")
	desc, err := personaReader.ReadString('\n')
	if err != nil {
		return err
	}
	desc = strings.TrimSpace(desc)

	fmt.Println("Enter system prompt (end with an empty line):")
	var promptLines []string
	for {
		line, err := personaReader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			break
		}
		promptLines = append(promptLines, line)
	}
	prompt := strings.Join(promptLines, "\n")

	persona := config.Persona{
		Name:         name,
		Description:  desc,
		SystemPrompt: prompt,
	}

	if err := config.AddPersona(persona); err != nil {
		return err
	}

	fmt.Printf("Persona '%s' created.\n", name)
	return nil
}

func runPersonaDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := config.DeletePersona(name); err != nil {
		return err
	}

	fmt.Printf("Persona '%s' deleted.\n", name)
	return nil
}

func runPersonaSetDefault(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := config.SetDefaultPersona(name); err != nil {
		return err
	}

	fmt.Printf("Persona '%s' set as default.\n", name)
	return nil
}

func runPersonaManage(cmd *cobra.Command, args []string) error {
	store := tui.NewPersonaStore()
	return tui.RunPersonaManagerTUI(store)
}
