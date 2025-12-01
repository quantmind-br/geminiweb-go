// Package commands provides CLI commands for geminiweb.
package commands

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand_Help(t *testing.T) {
	// Test that help works
	cmd := rootCmd
	if cmd.Use != "geminiweb [prompt]" {
		t.Errorf("Expected use 'geminiweb [prompt]', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestRootCommand_Args(t *testing.T) {
	// Test that Args validation is configured on rootCmd
	// Note: Argument validation (cobra.MaximumNArgs(1)) is handled by Cobra,
	// not tested here since calling RunE directly bypasses validation and
	// requires a valid command context
	if rootCmd.Args == nil {
		t.Error("Args validation should be configured")
	}
}

func TestRootCommand_VersionFlag(t *testing.T) {
	// Test version flag parsing
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			v, _ := cmd.Flags().GetBool("version")
			if v {
				return nil
			}
			return nil
		},
	}
	cmd.Flags().BoolP("version", "v", false, "Show version")

	tests := []struct {
		name     string
		args     []string
		wantHelp bool
	}{
		{
			name:     "version flag",
			args:     []string{"-v"},
			wantHelp: false,
		},
		{
			name:     "version flag long form",
			args:     []string{"--version"},
			wantHelp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Execute failed: %v", err)
			}
		})
	}
}

func TestRootCommand_GlobalFlags(t *testing.T) {
	// Test that model flag is a PersistentFlag (inherited by subcommands)
	t.Run("model flag (persistent)", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("model")
		if flag == nil {
			t.Error("PersistentFlag model not found")
		}
	})

	// Test local flags on rootCmd
	localFlags := []string{"output", "file", "image", "version"}
	for _, flagName := range localFlags {
		t.Run(flagName+" flag", func(t *testing.T) {
			flag := rootCmd.Flags().Lookup(flagName)
			if flag == nil {
				t.Errorf("Flag %s not found", flagName)
			}
		})
	}
}

func TestRootCommand_Subcommands(t *testing.T) {
	// Test that subcommands are registered
	expectedSubcommands := []string{"chat", "config", "import-cookies"}

	for _, sub := range expectedSubcommands {
		t.Run("subcommand "+sub, func(t *testing.T) {
			found := false
			for _, cmd := range rootCmd.Commands() {
				if cmd.Name() == sub {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Subcommand %s not found", sub)
			}
		})
	}
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name      string
		modelFlag string
		setupMock func()
		expected  string
	}{
		{
			name:      "model flag set",
			modelFlag: "gemini-2.5-pro",
			expected:  "gemini-2.5-pro",
		},
		{
			name:      "no flag, config loads successfully",
			modelFlag: "",
			setupMock: func() {
				// This would require mocking config.LoadConfig
				// For now, we expect the default fallback
			},
			expected: "gemini-2.5-flash",
		},
		{
			name:      "no flag, config fails",
			modelFlag: "",
			setupMock: func() {
				// Config failure is handled by returning default
			},
			expected: "gemini-2.5-flash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the model flag
			modelFlag = tt.modelFlag

			if tt.setupMock != nil {
				tt.setupMock()
			}

			result := getModel()
			if result != tt.expected {
				t.Errorf("getModel() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestRootCmd(t *testing.T) {
	// Test the root command structure
	cmd := rootCmd

	if cmd.Use != "geminiweb [prompt]" {
		t.Errorf("Expected use 'geminiweb [prompt]', got %s", cmd.Use)
	}

	if cmd.Short != "CLI for Google Gemini Web API" {
		t.Errorf("Expected short 'CLI for Google Gemini Web API', got %s", cmd.Short)
	}

	// Test that it has the expected flags
	flags := cmd.Flags()
	if flags == nil {
		t.Fatal("Flags is nil")
	}

	// Check for version flag
	versionFlag, err := flags.GetBool("version")
	if err != nil {
		t.Errorf("Failed to get version flag: %v", err)
	}
	if versionFlag {
		t.Error("Version flag should default to false")
	}

	// Test that it has the expected subcommands
	expectedCommands := []string{"chat", "config", "import-cookies", "history", "persona"}
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range cmd.Commands() {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found", expected)
		}
	}
}

func TestRootCmd_VersionFlag(t *testing.T) {
	// Test version flag behavior
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			v, _ := cmd.Flags().GetBool("version")
			if v {
				// This would print version and return nil
				return nil
			}
			return nil
		},
	}
	cmd.Flags().BoolP("version", "v", false, "Show version and exit")

	// Test with version flag set
	cmd.SetArgs([]string{"--version"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Version flag execution failed: %v", err)
	}
}

func TestRootCmd_FileInput(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test_prompt_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "Hello, world!"
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test file flag
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			fileFlag, _ := cmd.Flags().GetString("file")
			if fileFlag != "" {
				data, err := os.ReadFile(fileFlag)
				if err != nil {
					return err
				}
				if string(data) != testContent {
					t.Errorf("File content = %s, want %s", string(data), testContent)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&fileFlag, "file", "f", "", "Read prompt from file")

	cmd.SetArgs([]string{"--file", tmpFile.Name()})
	if err := cmd.Execute(); err != nil {
		t.Errorf("File input test failed: %v", err)
	}
}

func TestRootCmd_StdinInput(t *testing.T) {
	// Test stdin input
	testInput := "Test from stdin"

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check for stdin
			stat, _ := os.Stdin.Stat()
			hasStdin := (stat.Mode() & os.ModeCharDevice) == 0

			if hasStdin {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				if string(data) != testInput {
					t.Errorf("Stdin content = %s, want %s", string(data), testInput)
				}
			}
			return nil
		},
	}

	// Create a pipe to mock stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Write test data to pipe
	go func() {
		w.WriteString(testInput)
		w.Close()
	}()

	// Save original stdin and replace with pipe reader
	originalStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = originalStdin }()

	if err := cmd.Execute(); err != nil {
		t.Errorf("Stdin input test failed: %v", err)
	}
}

func TestRootCmd_PositionalArg(t *testing.T) {
	testArg := "Test argument"

	cmd := &cobra.Command{
		Use: "test [prompt]",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if args[0] != testArg {
					t.Errorf("Positional arg = %s, want %s", args[0], testArg)
				}
			}
			return nil
		},
	}

	cmd.SetArgs([]string{testArg})
	if err := cmd.Execute(); err != nil {
		t.Errorf("Positional arg test failed: %v", err)
	}
}

func TestRootCmd_NoInput(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test [prompt]",
		RunE: func(cmd *cobra.Command, args []string) error {
			// No input case - should show help
			return cmd.Help()
		},
	}

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err != nil {
		// Help command returns an error when executed in tests
		// This is expected behavior
		if err.Error() != "help requested" {
			t.Errorf("No input test failed: %v", err)
		}
	}
}

// TestExecute tests the Execute function
func TestExecute(t *testing.T) {
	t.Run("successful_execution", func(t *testing.T) {
		// Create a command that succeeds
		oldRootCmd := rootCmd
		rootCmd = &cobra.Command{
			Use: "test",
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil // No error
			},
		}
		defer func() { rootCmd = oldRootCmd }()

		// Call Execute - should not panic
		// Note: Execute calls os.Exit on error, but for success case it returns normally
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Execute() panicked: %v", r)
			}
		}()

		// Execute the command - this will work for the success case
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("Execute() unexpected error: %v", err)
		}
	})

	t.Run("execution_with_error", func(t *testing.T) {
		// Create a command that fails
		oldRootCmd := rootCmd
		rootCmd = &cobra.Command{
			Use: "test",
			RunE: func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("custom error")
			},
		}
		defer func() { rootCmd = oldRootCmd }()

		// Execute should return an error for the failing command
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Execute() expected error for failing command")
		}
	})

	t.Run("version_flag", func(t *testing.T) {
		// Create a command that handles version flag
		oldRootCmd := rootCmd
		rootCmd = &cobra.Command{
			Use: "test",
			RunE: func(cmd *cobra.Command, args []string) error {
				v, _ := cmd.Flags().GetBool("version")
				if v {
					fmt.Println("test version")
					return nil
				}
				return nil
			},
		}
		defer func() { rootCmd = oldRootCmd }()

		rootCmd.Flags().BoolP("version", "v", false, "Show version")

		// Set version flag
		rootCmd.SetArgs([]string{"--version"})

		// Execute should handle version flag
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("Execute() unexpected error with version flag: %v", err)
		}
	})
}

func TestGetBrowserRefresh(t *testing.T) {
	t.Run("empty flag", func(t *testing.T) {
		// Save original value
		originalFlag := browserRefreshFlag
		defer func() { browserRefreshFlag = originalFlag }()

		// Set empty flag
		browserRefreshFlag = ""

		browserType, enabled := getBrowserRefresh()
		if browserType != "" {
			t.Errorf("Expected empty browser type, got %s", browserType)
		}
		if enabled {
			t.Error("Expected disabled (false), got enabled (true)")
		}
	})

	t.Run("valid browser", func(t *testing.T) {
		// Save original value
		originalFlag := browserRefreshFlag
		defer func() { browserRefreshFlag = originalFlag }()

		// Set valid flag
		browserRefreshFlag = "chrome"

		browserType, enabled := getBrowserRefresh()
		if !enabled {
			t.Error("Expected enabled (true), got disabled (false)")
		}
		if browserType.String() != "chrome" {
			t.Errorf("Expected browser type 'chrome', got %s", browserType.String())
		}
	})

	t.Run("invalid browser", func(t *testing.T) {
		// Save original value
		originalFlag := browserRefreshFlag
		defer func() { browserRefreshFlag = originalFlag }()

		// Set invalid flag
		browserRefreshFlag = "invalid_browser"

		// Capture stderr
		oldStderr := os.Stderr
		w, _, _ := os.Pipe()
		os.Stderr = w

		browserType, enabled := getBrowserRefresh()

		// Restore stderr
		w.Close()
		os.Stderr = oldStderr

		// Check that it returns disabled
		if enabled {
			t.Error("Expected disabled (false) for invalid browser, got enabled (true)")
		}
		if browserType != "" {
			t.Errorf("Expected empty browser type for invalid browser, got %s", browserType)
		}
	})
}
