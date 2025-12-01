package commands

import (
	"strings"
	"testing"
)

func TestRunAutoLogin(t *testing.T) {
	// Note: This test only verifies the function exists and doesn't panic
	// Full integration testing of runAutoLogin would require complex mocking
	// or integration tests that actually interact with the browser

	// Test with invalid browser (should fail at parse stage)
	err := runAutoLogin("invalid-browser")
	if err == nil {
		t.Error("runAutoLogin with invalid browser should return error")
	}
	if err != nil && !strings.Contains(err.Error(), "unsupported browser") {
		t.Errorf("Expected 'unsupported browser' in error, got: %v", err)
	}
}

func TestRunListBrowsers(t *testing.T) {
	// Note: This test only verifies the function exists and doesn't panic
	// Actual functionality depends on installed browsers

	err := runListBrowsers()
	if err != nil {
		t.Errorf("runListBrowsers() unexpected error: %v", err)
	}
}

func TestTruncateValue(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactlyten", 10, "exactlyten"}, // 10 chars, should fit exactly
		{"this-is-longer", 5, "this-"},
		{"", 10, ""},
		{"test", 0, ""}, // Edge case: maxLen = 0 returns empty string
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateValue(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateValue(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestGetAutoLoginCmd(t *testing.T) {
	cmd := GetAutoLoginCmd()
	if cmd == nil {
		t.Error("GetAutoLoginCmd() returned nil")
	}
	if cmd.Use != "auto-login" {
		t.Errorf("GetAutoLoginCmd() use = %q, want %q", cmd.Use, "auto-login")
	}
}

func TestSupportedBrowsersHelp(t *testing.T) {
	help := SupportedBrowsersHelp()
	if help == "" {
		t.Error("SupportedBrowsersHelp() returned empty string")
	}

	// Check that it contains known browsers
	expectedBrowsers := []string{"chrome", "firefox", "edge"}
	for _, browser := range expectedBrowsers {
		if !strings.Contains(strings.ToLower(help), browser) {
			t.Errorf("SupportedBrowsersHelp() expected to contain %q, got %q", browser, help)
		}
	}
}

// TestAutoLoginCmdFlags tests the auto-login command flags
func TestAutoLoginCmdFlags(t *testing.T) {
	cmd := GetAutoLoginCmd()
	if cmd == nil {
		t.Fatal("GetAutoLoginCmd() returned nil")
	}

	// Test that required flags exist
	flags := cmd.Flags()
	if flags == nil {
		t.Error("Flags should not be nil")
	}

	// Test browser flag
	browserFlag := flags.Lookup("browser")
	if browserFlag == nil {
		t.Error("Browser flag should exist")
	}

	// Test list flag
	listFlag := flags.Lookup("list")
	if listFlag == nil {
		t.Error("List flag should exist")
	}
}

// TestAutoLoginCommandStructure tests the command structure
func TestAutoLoginCommandStructure(t *testing.T) {
	cmd := GetAutoLoginCmd()
	if cmd == nil {
		t.Fatal("GetAutoLoginCmd() returned nil")
	}

	// Test command metadata
	if cmd.Use != "auto-login" {
		t.Errorf("Expected use 'auto-login', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

// TestTruncateValueEdgeCases tests edge cases for truncateValue
func TestTruncateValueEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "empty string with zero max",
			input:    "",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "string shorter than max",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "string exactly max length",
			input:    "exactlylen",
			maxLen:   10,
			expected: "exactlylen",
		},
		{
			name:     "string longer than max",
			input:    "this-is-very-long",
			maxLen:   5,
			expected: "this-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateValue(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateValue(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
