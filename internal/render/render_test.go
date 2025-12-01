package render

import (
	"strings"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Width != 80 {
		t.Errorf("expected Width=80, got %d", opts.Width)
	}
	if opts.Style != "dark" {
		t.Errorf("expected Style='dark', got %s", opts.Style)
	}
	if !opts.EnableEmoji {
		t.Error("expected EnableEmoji=true")
	}
	if !opts.PreserveNewLines {
		t.Error("expected PreserveNewLines=true")
	}
	if !opts.TableWrap {
		t.Error("expected TableWrap=true")
	}
	if opts.InlineTableLinks {
		t.Error("expected InlineTableLinks=false")
	}
}

func TestOptionsWithWidth(t *testing.T) {
	opts := DefaultOptions().WithWidth(120)

	if opts.Width != 120 {
		t.Errorf("expected Width=120, got %d", opts.Width)
	}
	// Verify other options are preserved
	if opts.Style != "dark" {
		t.Errorf("expected Style='dark', got %s", opts.Style)
	}
}

func TestOptionsWithStyle(t *testing.T) {
	opts := DefaultOptions().WithStyle("light")

	if opts.Style != "light" {
		t.Errorf("expected Style='light', got %s", opts.Style)
	}
}

func TestOptionsChaining(t *testing.T) {
	opts := DefaultOptions().
		WithWidth(100).
		WithStyle("light").
		WithEmoji(false).
		WithPreserveNewLines(false).
		WithTableWrap(false).
		WithInlineTableLinks(true)

	if opts.Width != 100 {
		t.Errorf("expected Width=100, got %d", opts.Width)
	}
	if opts.Style != "light" {
		t.Errorf("expected Style='light', got %s", opts.Style)
	}
	if opts.EnableEmoji {
		t.Error("expected EnableEmoji=false")
	}
	if opts.PreserveNewLines {
		t.Error("expected PreserveNewLines=false")
	}
	if opts.TableWrap {
		t.Error("expected TableWrap=false")
	}
	if !opts.InlineTableLinks {
		t.Error("expected InlineTableLinks=true")
	}
}

func TestMarkdown(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		width    int
		contains string
	}{
		{
			name:     "heading",
			input:    "# Hello World",
			width:    80,
			contains: "Hello", // Check individual words due to ANSI codes
		},
		{
			name:     "bold",
			input:    "This is **bold** text",
			width:    80,
			contains: "bold",
		},
		{
			name:     "code_block",
			input:    "```go\nfmt.Println(\"hello\")\n```",
			width:    80,
			contains: "Println",
		},
		{
			name:     "link",
			input:    "[Link](https://example.com)",
			width:    80,
			contains: "Link",
		},
		{
			name:     "multiline",
			input:    "Line 1\n\nLine 2\n\nLine 3",
			width:    80,
			contains: "Line",
		},
		{
			name:     "narrow_width",
			input:    "# Long heading that should wrap",
			width:    40,
			contains: "Long",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := DefaultOptions().WithWidth(tc.width)
			output, err := Markdown(tc.input, opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(output, tc.contains) {
				t.Errorf("output should contain %q, got: %s", tc.contains, output)
			}
		})
	}
}

func TestMarkdownWithWidth(t *testing.T) {
	input := "# Hello World\n\nThis is a test."
	output, err := MarkdownWithWidth(input, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check individual words due to ANSI codes in output
	if !strings.Contains(output, "Hello") {
		t.Errorf("output should contain 'Hello', got: %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("output should contain 'test', got: %s", output)
	}
}

func TestMarkdownEmoji(t *testing.T) {
	input := "Hello :smile: world"

	// With emoji enabled (default)
	opts := DefaultOptions()
	output, err := Markdown(input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When emoji is enabled, :smile: should be converted to the emoji character
	if strings.Contains(output, ":smile:") {
		t.Errorf("emoji should have been converted, got: %s", output)
	}

	// With emoji disabled
	opts = DefaultOptions().WithEmoji(false)
	output, err = Markdown(input, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When emoji is disabled, :smile: should remain as text
	if !strings.Contains(output, ":smile:") {
		t.Errorf("emoji should NOT have been converted, got: %s", output)
	}
}

func TestMarkdownTable(t *testing.T) {
	input := "| A | B |\n|---|---|\n| 1 | 2 |"
	output, err := MarkdownWithWidth(input, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "A") || !strings.Contains(output, "B") {
		t.Errorf("table should contain headers, got: %s", output)
	}
}

func TestMarkdownInvalidStyle(t *testing.T) {
	opts := DefaultOptions().WithStyle("nonexistent_style_path")
	_, err := Markdown("# Test", opts)
	// glamour should return an error for invalid style path
	if err == nil {
		t.Error("expected error for invalid style path")
	}
}
