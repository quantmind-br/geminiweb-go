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

var personaCmd = &cobra.Command{
	Use:   "persona",
	Short: "Manage chat personas",
	Long:  `View and manage personas (system prompts) for chat sessions.`,
}

var personaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available personas",
	RunE:  runPersonaList,
}

var personaShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show persona details",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaShow,
}

var personaAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new persona",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaAdd,
}

var personaDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a persona",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaDelete,
}

var personaSetDefaultCmd = &cobra.Command{
	Use:   "default <name>",
	Short: "Set default persona",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaSetDefault,
}

func init() {
	personaCmd.AddCommand(personaListCmd)
	personaCmd.AddCommand(personaShowCmd)
	personaCmd.AddCommand(personaAddCmd)
	personaCmd.AddCommand(personaDeleteCmd)
	personaCmd.AddCommand(personaSetDefaultCmd)
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

	fmt.Printf("Default persona set to '%s'.\n", name)
	return nil
}
