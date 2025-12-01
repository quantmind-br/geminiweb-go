package render

import (
	"os"
	"testing"
)

func TestLoadOptionsFromConfig(t *testing.T) {
	// Clear any environment variable
	origStyle := os.Getenv("GLAMOUR_STYLE")
	os.Unsetenv("GLAMOUR_STYLE")
	defer func() {
		if origStyle != "" {
			os.Setenv("GLAMOUR_STYLE", origStyle)
		}
	}()

	opts := LoadOptionsFromConfig()

	// Should return valid options (either from config or defaults)
	if opts.Style == "" {
		t.Error("expected non-empty style")
	}
	if opts.Width != 80 {
		t.Errorf("expected default width 80, got %d", opts.Width)
	}
}

func TestLoadOptionsFromConfig_EnvOverride(t *testing.T) {
	// Set environment variable
	os.Setenv("GLAMOUR_STYLE", "light")
	defer os.Unsetenv("GLAMOUR_STYLE")

	opts := LoadOptionsFromConfig()

	if opts.Style != "light" {
		t.Errorf("expected Style='light' from env, got %s", opts.Style)
	}
}

func TestLoadOptionsFromConfigWithWidth(t *testing.T) {
	origStyle := os.Getenv("GLAMOUR_STYLE")
	os.Unsetenv("GLAMOUR_STYLE")
	defer func() {
		if origStyle != "" {
			os.Setenv("GLAMOUR_STYLE", origStyle)
		}
	}()

	opts := LoadOptionsFromConfigWithWidth(120)

	if opts.Width != 120 {
		t.Errorf("expected width 120, got %d", opts.Width)
	}
}

func TestLoadOptionsFromConfig_ValidOptions(t *testing.T) {
	origStyle := os.Getenv("GLAMOUR_STYLE")
	os.Unsetenv("GLAMOUR_STYLE")
	defer func() {
		if origStyle != "" {
			os.Setenv("GLAMOUR_STYLE", origStyle)
		}
	}()

	opts := LoadOptionsFromConfig()

	// Test that we can render with the loaded options
	output, err := Markdown("# Test", opts)
	if err != nil {
		t.Fatalf("Markdown render failed with loaded options: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}
