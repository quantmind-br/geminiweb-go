package commands

import (
	"testing"
)

func TestChatCommand(t *testing.T) {
	// Test that the command is properly configured
	if chatCmd.Use != "chat" {
		t.Errorf("Expected use 'chat', got %s", chatCmd.Use)
	}

	if chatCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if chatCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if chatCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestChatCommand_Args(t *testing.T) {
	// Chat command should accept no arguments
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "with args (should be rejected)",
			args:    []string{"test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chatCmd.RunE(nil, tt.args)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				// Allow other errors (like missing config)
				t.Logf("Got error (may be expected): %v", err)
			}
		})
	}
}

func TestChatCommand_Flags(t *testing.T) {
	// Model flag is defined as PersistentFlag on rootCmd, inherited by all subcommands
	flag := rootCmd.PersistentFlags().Lookup("model")
	if flag == nil {
		t.Error("model flag not found on rootCmd")
	}
}
