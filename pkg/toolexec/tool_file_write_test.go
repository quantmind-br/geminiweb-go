package toolexec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriteTool_Write(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "test.txt")

	tool := NewFileWriteTool()
	output, err := tool.Execute(context.Background(),
		NewInput().
			WithParam("path", path).
			WithParam("content", "hello"),
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output == nil || !strings.Contains(output.Message, "wrote") {
		t.Fatalf("unexpected output message: %v", output)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestFileWriteTool_ContentLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	tool := NewFileWriteTool(WithFileWriteMaxBytes(3))
	_, err := tool.Execute(context.Background(),
		NewInput().
			WithParam("path", path).
			WithParam("content", "hello"),
	)
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestFileWriteTool_RequiresConfirmation(t *testing.T) {
	tool := NewFileWriteTool()
	if !tool.RequiresConfirmation(nil) {
		t.Fatal("RequiresConfirmation() = false, want true")
	}
}
