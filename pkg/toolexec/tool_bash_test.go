package toolexec

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestBashTool_Execute(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available in PATH")
	}

	tool := NewBashTool()
	output, err := tool.Execute(context.Background(), NewInput().WithParam("command", "echo hello"))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output == nil {
		t.Fatal("Execute() output is nil")
	}
	if strings.TrimSpace(string(output.Data)) != "hello" {
		t.Fatalf("unexpected output: %q", string(output.Data))
	}
}

func TestBashTool_MissingCommand(t *testing.T) {
	tool := NewBashTool()
	_, err := tool.Execute(context.Background(), NewInput())
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestBashTool_RequiresConfirmation(t *testing.T) {
	tool := NewBashTool()
	if !tool.RequiresConfirmation(nil) {
		t.Fatal("RequiresConfirmation() = false, want true")
	}
}
