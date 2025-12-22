package commands

import (
	"testing"
)

// TestNewConfigCmd tests the config command constructor
func TestNewConfigCmd(t *testing.T) {
	deps := &Dependencies{}
	cmd := NewConfigCmd(deps)

	if cmd == nil {
		t.Fatal("NewConfigCmd() returned nil")
	}

	if cmd.Use != "config" {
		t.Errorf("expected Use 'config', got '%s'", cmd.Use)
	}

	if cmd.Short != "Open configuration menu" {
		t.Errorf("expected Short 'Open configuration menu', got '%s'", cmd.Short)
	}

	if cmd.Long != `Interactive menu to configure geminiweb settings.` {
		t.Errorf("unexpected Long description: %s", cmd.Long)
	}

	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Test with nil deps (backward compatibility)
	cmd2 := NewConfigCmd(nil)
	if cmd2 == nil {
		t.Fatal("NewConfigCmd(nil) returned nil")
	}

	if cmd2.Use != "config" {
		t.Errorf("expected Use 'config', got '%s'", cmd2.Use)
	}
}

// TestNewConfigCmd_CommandProperties tests various command properties
func TestNewConfigCmd_CommandProperties(t *testing.T) {
	cmd := NewConfigCmd(nil)

	// Test that RunE is set
	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Verify the command doesn't have Args validation (default is nil, meaning no args validation)
	if cmd.Args != nil {
		t.Error("config command should not have argument validation")
	}
}

// TestNewConfigCmd_GlobalVariable tests the backward compatibility global
func TestNewConfigCmd_GlobalVariable(t *testing.T) {
	// The global configCmd should be initialized
	if configCmd == nil {
		t.Error("global configCmd should not be nil")
	}

	if configCmd.Use != "config" {
		t.Errorf("expected global configCmd.Use 'config', got '%s'", configCmd.Use)
	}
}
