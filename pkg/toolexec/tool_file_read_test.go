package toolexec

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileReadTool_ReadAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello\nworld\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tool := NewFileReadTool()
	output, err := tool.Execute(context.Background(), NewInput().WithParam("path", path))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if string(output.Data) != content {
		t.Fatalf("unexpected output: %q", string(output.Data))
	}
}

func TestFileReadTool_ReadLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tool := NewFileReadTool()
	input := NewInput().WithParam("path", path).WithParam("lines", 1)
	output, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if string(output.Data) != "line1\n" {
		t.Fatalf("unexpected output: %q", string(output.Data))
	}
}

func TestFileReadTool_MaxBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	if err := os.WriteFile(path, []byte("1234567890"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tool := NewFileReadTool(WithFileReadMaxBytes(5))
	_, err := tool.Execute(context.Background(), NewInput().WithParam("path", path))
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestFileReadTool_InvalidLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("line1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tool := NewFileReadTool()
	input := NewInput().WithParam("path", path).WithParam("lines", -1)
	_, err := tool.Execute(context.Background(), input)
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
