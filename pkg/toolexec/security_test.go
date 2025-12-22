// Package toolexec provides a modular, extensible tool executor architecture.
// This file contains tests for the security layer components.
package toolexec

import (
	"context"
	"errors"
	"testing"
)

// TestBlacklistValidator tests the BlacklistValidator implementation.
func TestBlacklistValidator(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		toolName    string
		args        map[string]any
		expectError bool
	}{
		{
			name:        "allows non-bash tools",
			patterns:    []string{"rm -rf"},
			toolName:    "file_read",
			args:        map[string]any{"path": "/tmp/test"},
			expectError: false,
		},
		{
			name:        "allows safe bash commands",
			patterns:    []string{"rm -rf /"},
			toolName:    "bash",
			args:        map[string]any{"command": "ls -la"},
			expectError: false,
		},
		{
			name:        "blocks dangerous pattern",
			patterns:    []string{"rm -rf /"},
			toolName:    "bash",
			args:        map[string]any{"command": "rm -rf /"},
			expectError: true,
		},
		{
			name:        "blocks pattern anywhere in command",
			patterns:    []string{"rm -rf /"},
			toolName:    "bash",
			args:        map[string]any{"command": "sudo rm -rf / --no-preserve-root"},
			expectError: true,
		},
		{
			name:        "blocks fork bomb",
			patterns:    []string{":(){:|:&};:"},
			toolName:    "bash",
			args:        map[string]any{"command": ":(){:|:&};:"},
			expectError: true,
		},
		{
			name:        "allows when no command arg",
			patterns:    []string{"rm -rf /"},
			toolName:    "bash",
			args:        map[string]any{"path": "/tmp"},
			expectError: false,
		},
		{
			name:        "allows when command is not string",
			patterns:    []string{"rm -rf /"},
			toolName:    "bash",
			args:        map[string]any{"command": 123},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewBlacklistValidator(tt.patterns...)
			err := validator.Validate(context.Background(), tt.toolName, tt.args)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && err != nil {
				if !IsSecurityViolationError(err) {
					t.Errorf("expected SecurityViolationError, got %T", err)
				}
			}
		})
	}
}

// TestDefaultBlacklistValidator tests the default dangerous patterns.
func TestDefaultBlacklistValidator(t *testing.T) {
	validator := DefaultBlacklistValidator()
	ctx := context.Background()

	dangerousCommands := []string{
		"rm -rf /",
		"rm -rf /*",
		"rm -rf ~",
		"dd if=/dev/zero of=/dev/sda",
		"mkfs.ext4 /dev/sda1",
		":(){:|:&};:",
		"echo 'test' > /dev/sda",
		"chmod -R 777 /",
		"chown -R root:root /tmp",
		"wget | sh",       // pipe to shell
		"curl | bash",     // pipe to bash
	}

	for _, cmd := range dangerousCommands {
		testName := cmd
		if len(testName) > 20 {
			testName = testName[:20] + "..."
		}
		t.Run(testName, func(t *testing.T) {
			err := validator.Validate(ctx, "bash", map[string]any{"command": cmd})
			if err == nil {
				t.Errorf("expected command to be blocked: %s", cmd)
			}
		})
	}

	safeCommands := []string{
		"ls -la",
		"cat /tmp/test.txt",
		"echo 'hello world'",
		"git status",
		"npm install",
	}

	for _, cmd := range safeCommands {
		t.Run("safe:"+cmd, func(t *testing.T) {
			err := validator.Validate(ctx, "bash", map[string]any{"command": cmd})
			if err != nil {
				t.Errorf("expected command to be allowed: %s, got error: %v", cmd, err)
			}
		})
	}
}

// TestPathValidator tests the PathValidator implementation.
func TestPathValidator(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		toolName    string
		args        map[string]any
		expectError bool
	}{
		{
			name:        "allows non-file tools",
			patterns:    []string{".env"},
			toolName:    "bash",
			args:        map[string]any{"path": ".env"},
			expectError: false,
		},
		{
			name:        "allows safe paths for file_read",
			patterns:    []string{".env"},
			toolName:    "file_read",
			args:        map[string]any{"path": "/tmp/test.txt"},
			expectError: false,
		},
		{
			name:        "blocks .env file",
			patterns:    []string{".env"},
			toolName:    "file_read",
			args:        map[string]any{"path": ".env"},
			expectError: true,
		},
		{
			name:        "blocks .env in subdirectory",
			patterns:    []string{".env"},
			toolName:    "file_read",
			args:        map[string]any{"path": "config/.env"},
			expectError: true,
		},
		{
			name:        "blocks pem files",
			patterns:    []string{"*.pem"},
			toolName:    "file_write",
			args:        map[string]any{"path": "/home/user/key.pem"},
			expectError: true,
		},
		{
			name:        "allows when no path arg",
			patterns:    []string{".env"},
			toolName:    "file_read",
			args:        map[string]any{"content": "test"},
			expectError: false,
		},
		{
			name:        "allows when path is not string",
			patterns:    []string{".env"},
			toolName:    "file_read",
			args:        map[string]any{"path": 123},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewPathValidator(tt.patterns...)
			err := validator.Validate(context.Background(), tt.toolName, tt.args)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && err != nil {
				if !IsSecurityViolationError(err) {
					t.Errorf("expected SecurityViolationError, got %T", err)
				}
			}
		})
	}
}

// TestDefaultPathValidator tests the default sensitive paths.
func TestDefaultPathValidator(t *testing.T) {
	validator := DefaultPathValidator()
	ctx := context.Background()

	sensitiveFiles := []string{
		".env",
		".env.local",
		"production.env",
		"credentials.json",
		"secrets.yaml",
		"key.pem",
		"private.key",
	}

	for _, path := range sensitiveFiles {
		t.Run("blocked:"+path, func(t *testing.T) {
			err := validator.Validate(ctx, "file_read", map[string]any{"path": path})
			if err == nil {
				t.Errorf("expected path to be blocked: %s", path)
			}
		})
	}

	safeFiles := []string{
		"main.go",
		"README.md",
		"config.json",
		"test.txt",
	}

	for _, path := range safeFiles {
		t.Run("allowed:"+path, func(t *testing.T) {
			err := validator.Validate(ctx, "file_read", map[string]any{"path": path})
			if err != nil {
				t.Errorf("expected path to be allowed: %s, got error: %v", path, err)
			}
		})
	}
}

// TestPathValidator_CustomToolNames tests PathValidator with custom tool names.
func TestPathValidator_CustomToolNames(t *testing.T) {
	validator := NewPathValidator(".env").WithToolNames("custom_read", "custom_write")
	ctx := context.Background()

	// Should block custom tools
	err := validator.Validate(ctx, "custom_read", map[string]any{"path": ".env"})
	if err == nil {
		t.Error("expected error for custom_read with .env")
	}

	// Should allow default file tools since we overrode the list
	err = validator.Validate(ctx, "file_read", map[string]any{"path": ".env"})
	if err != nil {
		t.Errorf("expected file_read to be allowed with custom tool names, got: %v", err)
	}
}

// TestCompositeSecurityPolicy tests the CompositeSecurityPolicy implementation.
func TestCompositeSecurityPolicy(t *testing.T) {
	t.Run("empty policy allows all", func(t *testing.T) {
		policy := NewCompositeSecurityPolicy()
		err := policy.Validate(context.Background(), "bash", map[string]any{"command": "rm -rf /"})
		if err != nil {
			t.Errorf("empty policy should allow all, got: %v", err)
		}
	})

	t.Run("chains validators in order", func(t *testing.T) {
		blacklist := NewBlacklistValidator("dangerous")
		path := NewPathValidator(".secret")

		policy := NewCompositeSecurityPolicy(blacklist, path)

		// Should be blocked by blacklist
		err := policy.Validate(context.Background(), "bash", map[string]any{"command": "dangerous command"})
		if err == nil {
			t.Error("expected blacklist to block")
		}

		// Should be blocked by path validator
		err = policy.Validate(context.Background(), "file_read", map[string]any{"path": ".secret"})
		if err == nil {
			t.Error("expected path validator to block")
		}

		// Should allow safe operations
		err = policy.Validate(context.Background(), "bash", map[string]any{"command": "ls"})
		if err != nil {
			t.Errorf("expected safe command to be allowed, got: %v", err)
		}
	})

	t.Run("stops on first failure", func(t *testing.T) {
		var calledCount int
		countingValidator := &countingSecurityPolicy{onValidate: func() { calledCount++ }}
		failingValidator := &failingSecurityPolicy{}

		// Failing validator first - counting should never be called
		policy := NewCompositeSecurityPolicy(failingValidator, countingValidator)
		err := policy.Validate(context.Background(), "test", nil)

		if err == nil {
			t.Error("expected error from failing validator")
		}
		if calledCount != 0 {
			t.Errorf("counting validator should not be called, was called %d times", calledCount)
		}
	})

	t.Run("Add method works", func(t *testing.T) {
		policy := NewCompositeSecurityPolicy()
		policy.Add(NewBlacklistValidator("blocked"))

		if policy.Len() != 1 {
			t.Errorf("expected 1 validator, got %d", policy.Len())
		}

		err := policy.Validate(context.Background(), "bash", map[string]any{"command": "blocked cmd"})
		if err == nil {
			t.Error("expected error after adding validator")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		slowValidator := &slowSecurityPolicy{}
		policy := NewCompositeSecurityPolicy(slowValidator)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := policy.Validate(ctx, "test", nil)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	})
}

// TestDefaultSecurityPolicy tests the default combined policy.
func TestDefaultSecurityPolicy(t *testing.T) {
	policy := DefaultSecurityPolicy()

	if policy.Len() != 2 {
		t.Errorf("expected 2 validators, got %d", policy.Len())
	}

	ctx := context.Background()

	// Should block dangerous bash commands
	err := policy.Validate(ctx, "bash", map[string]any{"command": "rm -rf /"})
	if err == nil {
		t.Error("expected dangerous command to be blocked")
	}

	// Should block sensitive files
	err = policy.Validate(ctx, "file_read", map[string]any{"path": ".env"})
	if err == nil {
		t.Error("expected .env to be blocked")
	}

	// Should allow safe operations
	err = policy.Validate(ctx, "bash", map[string]any{"command": "echo hello"})
	if err != nil {
		t.Errorf("expected safe command, got: %v", err)
	}
}

// TestNoOpSecurityPolicy tests the no-op policy.
func TestNoOpSecurityPolicy(t *testing.T) {
	policy := &NoOpSecurityPolicy{}

	// Should allow anything
	err := policy.Validate(context.Background(), "bash", map[string]any{"command": "rm -rf /"})
	if err != nil {
		t.Errorf("NoOpSecurityPolicy should allow all, got: %v", err)
	}

	err = policy.Validate(context.Background(), "file_read", map[string]any{"path": ".env"})
	if err != nil {
		t.Errorf("NoOpSecurityPolicy should allow all, got: %v", err)
	}
}

// TestSecurityViolationError tests the error type.
func TestSecurityViolationError(t *testing.T) {
	t.Run("basic error", func(t *testing.T) {
		err := NewSecurityViolationError("bash", "blocked command")
		if err.ToolName != "bash" {
			t.Errorf("expected ToolName 'bash', got %q", err.ToolName)
		}
		if err.Reason != "blocked command" {
			t.Errorf("expected Reason 'blocked command', got %q", err.Reason)
		}
	})

	t.Run("with pattern", func(t *testing.T) {
		err := NewSecurityViolationErrorWithPattern("bash", "blocked", "rm -rf")
		if err.Pattern != "rm -rf" {
			t.Errorf("expected Pattern 'rm -rf', got %q", err.Pattern)
		}
		if err.Error() == "" {
			t.Error("Error() should not be empty")
		}
	})

	t.Run("with path", func(t *testing.T) {
		err := NewSecurityViolationErrorWithPath("file_read", "blocked", ".env")
		if err.Path != ".env" {
			t.Errorf("expected Path '.env', got %q", err.Path)
		}
		if err.Error() == "" {
			t.Error("Error() should not be empty")
		}
	})

	t.Run("Is works with sentinel", func(t *testing.T) {
		err := NewSecurityViolationError("bash", "test")
		if !errors.Is(err, ErrSecurityViolation) {
			t.Error("expected Is(ErrSecurityViolation) to be true")
		}
	})

	t.Run("helper function works", func(t *testing.T) {
		err := NewSecurityViolationError("bash", "test")
		if !IsSecurityViolationError(err) {
			t.Error("expected IsSecurityViolationError to be true")
		}
		if IsSecurityViolationError(nil) {
			t.Error("expected IsSecurityViolationError(nil) to be false")
		}
		if IsSecurityViolationError(errors.New("other error")) {
			t.Error("expected IsSecurityViolationError for other error to be false")
		}
	})
}

// Helper test types

type countingSecurityPolicy struct {
	onValidate func()
}

func (p *countingSecurityPolicy) Validate(ctx context.Context, toolName string, args map[string]any) error {
	if p.onValidate != nil {
		p.onValidate()
	}
	return nil
}

type failingSecurityPolicy struct{}

func (p *failingSecurityPolicy) Validate(ctx context.Context, toolName string, args map[string]any) error {
	return NewSecurityViolationError(toolName, "always fails")
}

type slowSecurityPolicy struct{}

func (p *slowSecurityPolicy) Validate(ctx context.Context, toolName string, args map[string]any) error {
	// This validator is designed to check context before doing work
	// In real use, the composite policy checks context between validators
	return nil
}
