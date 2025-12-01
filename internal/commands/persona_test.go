package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diogo/geminiweb/internal/config"
)

func TestPersonaCommand(t *testing.T) {
	// Test that the command is properly configured
	if personaCmd.Use != "persona" {
		t.Errorf("Expected use 'persona', got %s", personaCmd.Use)
	}

	if personaCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Test that subcommands are registered
	expectedSubcommands := []string{"list", "show", "add", "delete", "default"}
	for _, sub := range expectedSubcommands {
		found := false
		for _, cmd := range personaCmd.Commands() {
			if cmd.Name() == sub {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Subcommand %s not found", sub)
		}
	}
}

func TestPersonaListCommand(t *testing.T) {
	// Test command structure
	if personaListCmd.Use != "list" {
		t.Errorf("Expected use 'list', got %s", personaListCmd.Use)
	}

	if personaListCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if personaListCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Note: Argument validation is handled by Cobra's Args field, not tested here
	// since calling RunE directly bypasses Cobra's validation
}

func TestPersonaShowCommand(t *testing.T) {
	// Test command structure
	if personaShowCmd.Use != "show <name>" {
		t.Errorf("Expected use 'show <name>', got %s", personaShowCmd.Use)
	}

	if personaShowCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Verify Args validation is configured
	if personaShowCmd.Args == nil {
		t.Error("Args validation should be configured")
	}

	// Note: Argument validation (cobra.ExactArgs(1)) is handled by Cobra,
	// not tested here since calling RunE directly bypasses validation
}

func TestPersonaAddCommand(t *testing.T) {
	// Test command structure
	if personaAddCmd.Use != "add <name>" {
		t.Errorf("Expected use 'add <name>', got %s", personaAddCmd.Use)
	}

	if personaAddCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestPersonaDeleteCommand(t *testing.T) {
	// Test command structure
	if personaDeleteCmd.Use != "delete <name>" {
		t.Errorf("Expected use 'delete <name>', got %s", personaDeleteCmd.Use)
	}

	if personaDeleteCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestPersonaSetDefaultCommand(t *testing.T) {
	// Test command structure
	if personaSetDefaultCmd.Use != "default <name>" {
		t.Errorf("Expected use 'default <name>', got %s", personaSetDefaultCmd.Use)
	}

	if personaSetDefaultCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

// Test that persona commands work with a temporary config
func TestPersonaCommands_WithConfig(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create personas directory
	personasDir := filepath.Join(tmpDir, ".geminiweb")
	os.MkdirAll(personasDir, 0o755)

	// Create a test persona config
	personas := &config.PersonaConfig{
		Personas: []config.Persona{
			{
				Name:        "test-persona",
				Description: "Test persona",
				SystemPrompt: "You are a test assistant.",
			},
		},
	}

	err := config.SavePersonas(personas)
	if err != nil {
		t.Fatalf("Failed to save personas: %v", err)
	}

	// Test GetPersona
	persona, err := config.GetPersona("test-persona")
	if err != nil {
		t.Fatalf("Failed to get persona: %v", err)
	}

	if persona.Name != "test-persona" {
		t.Errorf("Persona name mismatch")
	}

	if persona.SystemPrompt != "You are a test assistant." {
		t.Errorf("System prompt mismatch")
	}

	// Test ListPersonaNames
	names, err := config.ListPersonaNames()
	if err != nil {
		t.Fatalf("Failed to list persona names: %v", err)
	}
	if len(names) == 0 {
		t.Error("Expected at least one persona name")
	}

	// Test AddPersona
	newPersona := config.Persona{
		Name:        "another-persona",
		Description: "Another test persona",
		SystemPrompt: "You are another assistant.",
	}
	err = config.AddPersona(newPersona)
	if err != nil {
		t.Fatalf("Failed to add persona: %v", err)
	}

	// Test UpdatePersona
	updatedPersona := *persona
	updatedPersona.Description = "Updated description"
	err = config.UpdatePersona(updatedPersona)
	if err != nil {
		t.Fatalf("Failed to update persona: %v", err)
	}

	// Test DeletePersona
	err = config.DeletePersona("another-persona")
	if err != nil {
		t.Fatalf("Failed to delete persona: %v", err)
	}

	// Test SetDefaultPersona
	err = config.SetDefaultPersona("test-persona")
	if err != nil {
		t.Fatalf("Failed to set default persona: %v", err)
	}

	defaultPersona, err := config.GetDefaultPersona()
	if err != nil {
		t.Fatalf("Failed to get default persona: %v", err)
	}

	if defaultPersona == nil {
		t.Error("Default persona is nil")
	} else if defaultPersona.Name != "test-persona" {
		t.Errorf("Default persona = %s, want test-persona", defaultPersona.Name)
	}
}

func TestRunPersonaList_Integration(t *testing.T) {
	// Create a temporary directory for personas
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test persona config
	cfg := &config.PersonaConfig{
		Personas: []config.Persona{
			{
				Name:        "test-persona",
				Description: "Test persona",
				SystemPrompt: "You are a test assistant.",
			},
		},
		DefaultPersona: "test-persona",
	}

	// Save the config
	if err := config.SavePersonas(cfg); err != nil {
		t.Fatalf("Failed to save persona config: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	err := runPersonaList(personaListCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runPersonaList failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain the persona name
	if !strings.Contains(output, "test-persona") {
		t.Errorf("Output should contain persona name: %s", output)
	}
}

func TestRunPersonaShow_NotFound(t *testing.T) {
	// Create a temporary directory for personas
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Run with non-existent persona
	err := runPersonaShow(personaShowCmd, []string{"nonexistent"})

	if err == nil {
		t.Error("Expected error for non-existent persona")
	}
}

func TestRunPersonaShow_Success(t *testing.T) {
	// Create a temporary directory for personas
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test persona
	persona := config.Persona{
		Name:        "test-show",
		Description: "Test show persona",
		SystemPrompt: "You are a test assistant for showing.",
	}

	if err := config.AddPersona(persona); err != nil {
		t.Fatalf("Failed to add persona: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	err := runPersonaShow(personaShowCmd, []string{"test-show"})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runPersonaShow failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain persona details
	if !strings.Contains(output, "Name:") {
		t.Error("Output should contain 'Name:'")
	}
	if !strings.Contains(output, "test-show") {
		t.Error("Output should contain persona name")
	}
}

func TestRunPersonaAdd_Duplicate(t *testing.T) {
	// Create a temporary directory for personas
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test persona
	persona := config.Persona{
		Name:        "test-duplicate",
		Description: "Test duplicate persona",
		SystemPrompt: "You are a test assistant.",
	}

	if err := config.AddPersona(persona); err != nil {
		t.Fatalf("Failed to add persona: %v", err)
	}

	// Try to add the same persona again
	err := runPersonaAdd(personaAddCmd, []string{"test-duplicate"})

	if err == nil {
		t.Error("Expected error for duplicate persona")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' in error, got: %v", err)
	}
}

func TestRunPersonaDelete_NotFound(t *testing.T) {
	// Create a temporary directory for personas
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Try to delete non-existent persona
	err := runPersonaDelete(personaDeleteCmd, []string{"nonexistent"})

	if err == nil {
		t.Error("Expected error for non-existent persona")
	}
}

func TestRunPersonaDelete_Success(t *testing.T) {
	// Create a temporary directory for personas
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test persona
	persona := config.Persona{
		Name:        "test-delete",
		Description: "Test delete persona",
		SystemPrompt: "You are a test assistant.",
	}

	if err := config.AddPersona(persona); err != nil {
		t.Fatalf("Failed to add persona: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Delete the persona
	err := runPersonaDelete(personaDeleteCmd, []string{"test-delete"})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runPersonaDelete failed: %v", err)
	}

	// Verify the persona was deleted
	_, err = config.GetPersona("test-delete")
	if err == nil {
		t.Error("Persona should be deleted")
	}
}

func TestRunPersonaSetDefault_NotFound(t *testing.T) {
	// Create a temporary directory for personas
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Try to set default for non-existent persona
	err := runPersonaSetDefault(personaSetDefaultCmd, []string{"nonexistent"})

	if err == nil {
		t.Error("Expected error for non-existent persona")
	}
}

func TestRunPersonaSetDefault_Success(t *testing.T) {
	// Create a temporary directory for personas
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test persona
	persona := config.Persona{
		Name:        "test-default",
		Description: "Test default persona",
		SystemPrompt: "You are a test assistant.",
	}

	if err := config.AddPersona(persona); err != nil {
		t.Fatalf("Failed to add persona: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Set default persona
	err := runPersonaSetDefault(personaSetDefaultCmd, []string{"test-default"})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runPersonaSetDefault failed: %v", err)
	}

	// Verify the default was set
	cfg, err := config.LoadPersonas()
	if err != nil {
		t.Fatalf("Failed to load personas: %v", err)
	}

	if cfg.DefaultPersona != "test-default" {
		t.Errorf("Expected default persona to be 'test-default', got %s", cfg.DefaultPersona)
	}
}

func TestPersonaCommands_Basic(t *testing.T) {
	// Simple test to increase coverage by checking function existence
	t.Run("persona list command exists", func(t *testing.T) {
		if personaListCmd == nil {
			t.Error("personaListCmd should not be nil")
		}
	})

	t.Run("persona add command exists", func(t *testing.T) {
		if personaAddCmd == nil {
			t.Error("personaAddCmd should not be nil")
		}
	})

	t.Run("persona delete command exists", func(t *testing.T) {
		if personaDeleteCmd == nil {
			t.Error("personaDeleteCmd should not be nil")
		}
	})

	t.Run("persona show command exists", func(t *testing.T) {
		if personaShowCmd == nil {
			t.Error("personaShowCmd should not be nil")
		}
	})

	t.Run("persona set default command exists", func(t *testing.T) {
		if personaSetDefaultCmd == nil {
			t.Error("personaSetDefaultCmd should not be nil")
		}
	})
}
