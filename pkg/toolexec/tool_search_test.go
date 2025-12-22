package toolexec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchTool_Literal(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "a.txt")
	path2 := filepath.Join(dir, "b.txt")

	if err := os.WriteFile(path1, []byte("alpha\nneedle\nbeta\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(path2, []byte("needle again\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tool := NewSearchTool()
	input := NewInput().
		WithParam("pattern", "needle").
		WithParam("path", dir).
		WithParam("type", "literal")
	output, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	data := string(output.Data)
	if !strings.Contains(data, "a.txt:2:needle") || !strings.Contains(data, "b.txt:1:needle again") {
		t.Fatalf("unexpected output: %q", data)
	}
}

func TestSearchTool_Regex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("needle\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tool := NewSearchTool()
	input := NewInput().
		WithParam("pattern", "n.*e").
		WithParam("path", dir).
		WithParam("type", "regex")
	output, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(string(output.Data), "a.txt:1:needle") {
		t.Fatalf("unexpected output: %q", string(output.Data))
	}
}

func TestSearchTool_InvalidType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("needle\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tool := NewSearchTool()
	input := NewInput().
		WithParam("pattern", "needle").
		WithParam("path", dir).
		WithParam("type", "unknown")
	_, err := tool.Execute(context.Background(), input)
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
