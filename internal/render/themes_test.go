package render

import (
	"os"
	"strings"
	"testing"
)

func TestGetBuiltinTheme_TokyoNight(t *testing.T) {
	theme, ok := GetBuiltinTheme(ThemeTokyoNight)
	if !ok {
		t.Fatal("expected TokyoNight theme to be found")
	}
	if len(theme) == 0 {
		t.Error("expected non-empty theme content")
	}

	// Verify it's valid JSON with expected structure
	if !strings.Contains(string(theme), "document") {
		t.Error("theme should contain 'document' key")
	}
	if !strings.Contains(string(theme), "#c0caf5") {
		t.Error("theme should contain Tokyo Night colors")
	}
}

func TestGetBuiltinTheme_Unknown(t *testing.T) {
	theme, ok := GetBuiltinTheme("unknown_theme")
	if ok {
		t.Error("expected unknown theme to not be found")
	}
	if theme != nil {
		t.Error("expected nil theme for unknown name")
	}
}

func TestIsBuiltinStyle(t *testing.T) {
	tests := []struct {
		style    string
		expected bool
	}{
		{"dark", true},
		{"light", true},
		{"dracula", true},
		{"tokyonight", true},
		{"catppuccin", true},
		{"custom_path.json", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			result := IsBuiltinStyle(tt.style)
			if result != tt.expected {
				t.Errorf("IsBuiltinStyle(%q) = %v, want %v", tt.style, result, tt.expected)
			}
		})
	}
}

func TestWriteThemeToTempFile(t *testing.T) {
	path, err := WriteThemeToTempFile(ThemeTokyoNight)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}

	// Clean up
	defer os.Remove(path)

	// Verify file exists and has content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty file content")
	}
}

func TestWriteThemeToTempFile_UnknownTheme(t *testing.T) {
	path, err := WriteThemeToTempFile("unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Error("expected empty path for unknown theme")
	}
}

func TestMarkdownWithTokyoNight(t *testing.T) {
	opts := DefaultOptions().WithStyle(ThemeTokyoNight)
	input := "# Hello World\n\nThis is **bold** and `code`."

	output, err := Markdown(input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
	// The output should be styled (contain ANSI codes or the text)
	if !strings.Contains(output, "Hello") {
		t.Error("output should contain 'Hello'")
	}
}

func TestGetBuiltinTheme_Dark(t *testing.T) {
	theme, ok := GetBuiltinTheme(ThemeDark)
	if !ok {
		t.Fatal("expected Dark theme to be found")
	}
	if len(theme) == 0 {
		t.Error("expected non-empty theme content")
	}

	// Verify it contains code block separators
	if !strings.Contains(string(theme), "block_prefix") {
		t.Error("theme should contain 'block_prefix' for code blocks")
	}
	if !strings.Contains(string(theme), "block_suffix") {
		t.Error("theme should contain 'block_suffix' for code blocks")
	}
}

func TestMarkdownWithDarkTheme_CodeBlockSeparators(t *testing.T) {
	ClearCache() // Ensure fresh renderer
	opts := DefaultOptions().WithStyle(ThemeDark)
	input := "Text before\n\n```go\nfunc main() {}\n```\n\nText after"

	output, err := Markdown(input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
	// Verify code block separators are present
	if !strings.Contains(output, "----------") {
		t.Error("output should contain code block separator")
	}
}


func TestGetBuiltinTheme_Catppuccin(t *testing.T) {
	theme, ok := GetBuiltinTheme(ThemeCatppuccin)
	if !ok {
		t.Fatal("expected Catppuccin theme to be found")
	}
	if len(theme) == 0 {
		t.Error("expected non-empty theme content")
	}

	// Verify it's valid JSON with expected structure
	if !strings.Contains(string(theme), "document") {
		t.Error("theme should contain 'document' key")
	}
	if !strings.Contains(string(theme), "#cdd6f4") {
		t.Error("theme should contain Catppuccin Mocha text color")
	}
	if !strings.Contains(string(theme), "#1e1e2e") {
		t.Error("theme should contain Catppuccin Mocha background color")
	}
}

func TestMarkdownWithCatppuccin(t *testing.T) {
	opts := DefaultOptions().WithStyle(ThemeCatppuccin)
	input := "# Hello World\n\nThis is **bold** and `code`."

	output, err := Markdown(input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
	// The output should be styled (contain ANSI codes or the text)
	if !strings.Contains(output, "Hello") {
		t.Error("output should contain 'Hello'")
	}
}

func TestMarkdownWithCatppuccin_CodeBlockSeparators(t *testing.T) {
	ClearCache() // Ensure fresh renderer
	opts := DefaultOptions().WithStyle(ThemeCatppuccin)
	input := "Text before\n\n```go\nfunc main() {}\n```\n\nText after"

	output, err := Markdown(input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
	// Verify code block separators are present
	if !strings.Contains(output, "----------") {
		t.Error("output should contain code block separator")
	}
}

func TestMarkdownWithTokyoNight_CodeBlockSeparators(t *testing.T) {
	ClearCache() // Ensure fresh renderer
	opts := DefaultOptions().WithStyle(ThemeTokyoNight)
	input := "Text before\n\n```go\nfunc main() {}\n```\n\nText after"

	output, err := Markdown(input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
	// Verify code block separators are present
	if !strings.Contains(output, "----------") {
		t.Error("output should contain code block separator")
	}
}


func TestAvailableThemes(t *testing.T) {
	themes := AvailableThemes()

	// Should have at least 7 themes
	if len(themes) < 7 {
		t.Errorf("expected at least 7 themes, got %d", len(themes))
	}

	// Verify expected themes are present
	expectedThemes := map[string]bool{
		ThemeDark:       false,
		ThemeTokyoNight: false,
		ThemeCatppuccin: false,
		ThemeLight:      false,
		"dracula":       false,
		"notty":         false,
		"ascii":         false,
	}

	for _, theme := range themes {
		if theme.Name == "" {
			t.Error("theme name should not be empty")
		}
		if theme.Description == "" {
			t.Error("theme description should not be empty")
		}
		if _, exists := expectedThemes[theme.Name]; exists {
			expectedThemes[theme.Name] = true
		}
	}

	for name, found := range expectedThemes {
		if !found {
			t.Errorf("expected theme %q to be present", name)
		}
	}
}

func TestThemeNames(t *testing.T) {
	names := ThemeNames()

	// Should have same count as AvailableThemes
	themes := AvailableThemes()
	if len(names) != len(themes) {
		t.Errorf("expected %d theme names, got %d", len(themes), len(names))
	}

	// Verify names match
	for i, theme := range themes {
		if names[i] != theme.Name {
			t.Errorf("name mismatch at index %d: got %q, want %q", i, names[i], theme.Name)
		}
	}

	// Verify dark is first (default)
	if names[0] != ThemeDark {
		t.Errorf("expected first theme to be %q, got %q", ThemeDark, names[0])
	}
}
