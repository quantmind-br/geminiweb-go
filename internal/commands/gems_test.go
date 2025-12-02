package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGemStdInReader(t *testing.T) {
	input := "test line\n"
	reader := NewGemStdInReader(strings.NewReader(input))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	if line != "test line\n" {
		t.Errorf("ReadString() = %q, want %q", line, "test line\n")
	}
}

func TestParseGemPromptFromStdin(t *testing.T) {
	input := "Line 1\nLine 2\nLine 3\n\n"
	reader := strings.NewReader(input)

	prompt, err := parseGemPromptFromStdin(reader)
	if err != nil {
		t.Fatalf("parseGemPromptFromStdin() error = %v", err)
	}

	expected := "Line 1\nLine 2\nLine 3"
	if prompt != expected {
		t.Errorf("parseGemPromptFromStdin() = %q, want %q", prompt, expected)
	}
}

func TestParseGemPromptFromStdinEmpty(t *testing.T) {
	input := "\n"
	reader := strings.NewReader(input)

	prompt, err := parseGemPromptFromStdin(reader)
	if err != nil {
		t.Fatalf("parseGemPromptFromStdin() error = %v", err)
	}

	if prompt != "" {
		t.Errorf("parseGemPromptFromStdin() = %q, want empty string", prompt)
	}
}

func TestGemsCmdStructure(t *testing.T) {
	// Test that gemsCmd has the expected structure
	if gemsCmd.Use != "gems" {
		t.Errorf("gemsCmd.Use = %q, want %q", gemsCmd.Use, "gems")
	}

	// Test subcommands exist
	subcommands := map[string]bool{
		"list":   false,
		"create": false,
		"update": false,
		"delete": false,
		"show":   false,
	}

	for _, cmd := range gemsCmd.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("gemsCmd missing subcommand: %s", name)
		}
	}
}

func TestGemsListCmdFlags(t *testing.T) {
	flag := gemsListCmd.Flags().Lookup("hidden")
	if flag == nil {
		t.Error("gemsListCmd missing --hidden flag")
	}
}

func TestGemsCreateCmdFlags(t *testing.T) {
	expectedFlags := []string{"prompt", "description", "file"}
	for _, name := range expectedFlags {
		flag := gemsCreateCmd.Flags().Lookup(name)
		if flag == nil {
			t.Errorf("gemsCreateCmd missing --%s flag", name)
		}
	}
}

func TestGemsUpdateCmdFlags(t *testing.T) {
	expectedFlags := []string{"prompt", "description", "file", "name"}
	for _, name := range expectedFlags {
		flag := gemsUpdateCmd.Flags().Lookup(name)
		if flag == nil {
			t.Errorf("gemsUpdateCmd missing --%s flag", name)
		}
	}
}

func TestGemsCreateCmdArgs(t *testing.T) {
	if gemsCreateCmd.Args == nil {
		t.Error("gemsCreateCmd.Args is nil")
	}
}

func TestGemsUpdateCmdArgs(t *testing.T) {
	if gemsUpdateCmd.Args == nil {
		t.Error("gemsUpdateCmd.Args is nil")
	}
}

func TestGemsDeleteCmdArgs(t *testing.T) {
	if gemsDeleteCmd.Args == nil {
		t.Error("gemsDeleteCmd.Args is nil")
	}
}

func TestGemsShowCmdArgs(t *testing.T) {
	if gemsShowCmd.Args == nil {
		t.Error("gemsShowCmd.Args is nil")
	}
}

func TestGemsCommands(t *testing.T) {
	tests := []struct {
		cmd      string
		use      string
		shortLen int
	}{
		{"list", "list", 5},
		{"create", "create <name>", 5},
		{"update", "update <id-or-name>", 5},
		{"delete", "delete <id-or-name>", 5},
		{"show", "show <id-or-name>", 5},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			var found *cobra.Command
			for _, cmd := range gemsCmd.Commands() {
				if cmd.Name() == tt.cmd {
					found = cmd
					break
				}
			}

			if found == nil {
				t.Fatalf("command %q not found", tt.cmd)
			}

			if found.Use != tt.use {
				t.Errorf("Use = %q, want %q", found.Use, tt.use)
			}

			if len(found.Short) < tt.shortLen {
				t.Errorf("Short description too short: %q", found.Short)
			}
		})
	}
}
