package toolexec

import (
	"context"
	"testing"
)

func TestBlacklistValidator(t *testing.T) {
	v := DefaultBlacklistValidator()
	ctx := context.Background()

	tests := []struct {
		name      string
		toolName  string
		command   string
		shouldErr bool
	}{
		{"Safe Command", "bash", "echo hello", false},
		{"Dangerous Command", "bash", "rm -rf /", true},
		{"Dangerous Command Substring", "bash", "echo start && rm -rf / && echo end", true},
		{"Ignored Tool", "python", "rm -rf /", false},
		{"Empty Command", "bash", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{"command": tt.command}
			err := v.Validate(ctx, tt.toolName, args)
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected nil, got error: %v", err)
			}
			if tt.shouldErr && err != nil {
				if !IsSecurityViolationError(err) {
					t.Errorf("Expected SecurityViolationError, got %T", err)
				}
			}
		})
	}
}

func TestPathValidator(t *testing.T) {
	v := DefaultPathValidator()
	ctx := context.Background()

	tests := []struct {
		name      string
		toolName  string
		path      string
		shouldErr bool
	}{
		{"Safe File", "file_read", "data.txt", false},
		{"Sensitive File .env", "file_read", ".env", true},
		{"Sensitive File in Dir", "file_read", "config/.env", true},
		{"Sensitive Dir .ssh", "file_read", ".ssh/id_rsa", true},
		{"Sensitive Dir Absolute", "file_read", "/home/user/.ssh/id_rsa", true},
		{"Ignored Tool", "bash", ".env", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{"path": tt.path}
			err := v.Validate(ctx, tt.toolName, args)
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected nil, got error: %v", err)
			}
		})
	}
}

func TestSecurityHelpers(t *testing.T) {
	// Test PathValidator builder
	pv := NewPathValidator().WithToolNames("custom_tool")
	if len(pv.toolNames) != 1 || pv.toolNames[0] != "custom_tool" {
		t.Error("WithToolNames failed")
	}

	// Test Composite builder
	cp := NewCompositeSecurityPolicy()
	if cp.Len() != 0 {
		t.Error("Expected empty policy")
	}
	cp.Add(pv)
	if cp.Len() != 1 {
		t.Error("Add failed")
	}

	// Test NoOp
	noop := &NoOpSecurityPolicy{}
	if err := noop.Validate(context.Background(), "any", nil); err != nil {
		t.Error("NoOp should not error")
	}
}

func TestCompositeSecurityPolicy(t *testing.T) {
	p := DefaultSecurityPolicy()
	ctx := context.Background()

	// Test blacklist part
	if err := p.Validate(ctx, "bash", map[string]any{"command": "rm -rf /"}); err == nil {
		t.Error("Composite policy failed to block blacklisted command")
	}

	// Test path part
	if err := p.Validate(ctx, "file_read", map[string]any{"path": ".env"}); err == nil {
		t.Error("Composite policy failed to block sensitive path")
	}

	// Test safe
	if err := p.Validate(ctx, "bash", map[string]any{"command": "ls"}); err != nil {
		t.Error("Composite policy blocked safe command")
	}
}
