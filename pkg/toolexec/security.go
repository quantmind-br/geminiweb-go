// Package toolexec provides a modular, extensible tool executor architecture.
// This file implements the security layer for validating tool executions,
// including blacklist checking, path validation, and composite security policies.
package toolexec

import (
	"context"
	"path/filepath"
	"strings"
)

// SecurityPolicy defines the interface for validating tool executions.
// Implementations can check for dangerous commands, blocked paths, or
// any other security constraints before allowing a tool to execute.
type SecurityPolicy interface {
	// Validate checks if the tool execution is allowed.
	// Returns nil if the execution is permitted, or an error (typically
	// SecurityViolationError) if the execution should be blocked.
	// The context can be used for cancellation or deadline checking.
	Validate(ctx context.Context, toolName string, args map[string]any) error
}

// BlacklistValidator blocks dangerous command patterns.
// It is primarily used for bash/shell commands to prevent destructive operations.
type BlacklistValidator struct {
	// blockedPatterns are command substrings that should be blocked.
	// Any command containing one of these patterns will be rejected.
	blockedPatterns []string
}

// NewBlacklistValidator creates a new BlacklistValidator with the given blocked patterns.
// Common patterns to block include "rm -rf /", "dd", "mkfs", etc.
func NewBlacklistValidator(patterns ...string) *BlacklistValidator {
	return &BlacklistValidator{
		blockedPatterns: patterns,
	}
}

// DefaultBlacklistValidator creates a BlacklistValidator with common dangerous patterns.
// This includes:
//   - rm -rf / (recursive delete root)
//   - dd (disk destroyer)
//   - mkfs (make filesystem)
//   - :(){:|:&};: (fork bomb)
//   - > /dev/sda (write to disk)
//   - chmod -R 777 / (make everything world-writable)
func DefaultBlacklistValidator() *BlacklistValidator {
	return NewBlacklistValidator(
		"rm -rf /",
		"rm -rf /*",
		"rm -rf ~",
		"dd if=",
		"mkfs",
		":(){:|:&};:",
		"> /dev/sda",
		"> /dev/hda",
		"chmod -R 777 /",
		"chown -R",
		"wget | sh",
		"curl | sh",
		"wget | bash",
		"curl | bash",
	)
}

// Validate implements SecurityPolicy.Validate.
// It only validates "bash" tools and checks if the command contains any blocked patterns.
func (v *BlacklistValidator) Validate(ctx context.Context, toolName string, args map[string]any) error {
	// Only validate bash commands
	if toolName != "bash" {
		return nil
	}

	// Get the command argument
	cmd, ok := args["command"].(string)
	if !ok {
		// No command argument or wrong type - let other validators handle this
		return nil
	}

	// Check each blocked pattern
	for _, pattern := range v.blockedPatterns {
		if strings.Contains(cmd, pattern) {
			return NewSecurityViolationErrorWithPattern(
				toolName,
				"blocked command pattern detected",
				pattern,
			)
		}
	}

	return nil
}

// PathValidator blocks access to sensitive file paths.
// It is primarily used for file_read and file_write tools to prevent
// access to sensitive files like .env, .ssh/, or *.pem files.
type PathValidator struct {
	// blockedPaths are glob patterns for paths that should be blocked.
	blockedPaths []string

	// toolNames are the tool names this validator applies to.
	// If empty, it applies to "file_read" and "file_write" by default.
	toolNames []string
}

// NewPathValidator creates a new PathValidator with the given blocked paths.
// The paths are glob patterns (e.g., "*.pem", ".env", ".ssh/*").
func NewPathValidator(paths ...string) *PathValidator {
	return &PathValidator{
		blockedPaths: paths,
		toolNames:    []string{"file_read", "file_write"},
	}
}

// DefaultPathValidator creates a PathValidator with common sensitive paths blocked.
// This includes:
//   - .env files (environment variables with secrets)
//   - .ssh/ directory (SSH keys)
//   - *.pem files (certificates/keys)
//   - *.key files (private keys)
//   - /etc/passwd, /etc/shadow (system user files)
//   - ~/.aws/* (AWS credentials)
//   - ~/.config/gcloud/* (GCP credentials)
func DefaultPathValidator() *PathValidator {
	return NewPathValidator(
		".env",
		".env.*",
		"*.env",
		".ssh/*",
		"*.pem",
		"*.key",
		"*/.ssh/*",
		"/etc/passwd",
		"/etc/shadow",
		"~/.aws/*",
		"~/.config/gcloud/*",
		"*credentials*",
		"*secret*",
	)
}

// WithToolNames sets the tool names this validator applies to.
func (v *PathValidator) WithToolNames(names ...string) *PathValidator {
	v.toolNames = names
	return v
}

// Validate implements SecurityPolicy.Validate.
// It validates file access tools and checks if the path matches any blocked patterns.
func (v *PathValidator) Validate(ctx context.Context, toolName string, args map[string]any) error {
	// Check if this validator applies to this tool
	applies := false
	for _, name := range v.toolNames {
		if toolName == name {
			applies = true
			break
		}
	}
	if !applies {
		return nil
	}

	// Get the path argument
	path, ok := args["path"].(string)
	if !ok {
		// No path argument or wrong type - let other validators handle this
		return nil
	}

	// Clean the path for consistent matching
	cleanPath := filepath.Clean(path)
	baseName := filepath.Base(cleanPath)

	// Check each blocked pattern
	for _, pattern := range v.blockedPaths {
		// Try matching against full path
		if matched, _ := filepath.Match(pattern, cleanPath); matched {
			return NewSecurityViolationErrorWithPath(
				toolName,
				"access denied to sensitive path",
				path,
			)
		}

		// Try matching against base name
		if matched, _ := filepath.Match(pattern, baseName); matched {
			return NewSecurityViolationErrorWithPath(
				toolName,
				"access denied to sensitive path",
				path,
			)
		}

		// Try matching if the pattern is a prefix (for directory patterns)
		if strings.HasSuffix(pattern, "/*") {
			dir := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(cleanPath, dir+"/") || cleanPath == dir {
				return NewSecurityViolationErrorWithPath(
					toolName,
					"access denied to sensitive directory",
					path,
				)
			}
		}

		// Handle "*/dirname/*" patterns (block directory anywhere in path)
		if strings.HasPrefix(pattern, "*/") && strings.HasSuffix(pattern, "/*") {
			dirName := pattern[2 : len(pattern)-2]
			parts := strings.Split(cleanPath, string(filepath.Separator))
			for _, part := range parts {
				if matched, _ := filepath.Match(dirName, part); matched {
					return NewSecurityViolationErrorWithPath(
						toolName,
						"access denied to sensitive directory component",
						path,
					)
				}
			}
		}
	}

	return nil
}

// CompositeSecurityPolicy chains multiple SecurityPolicy validators together.
// All validators must pass for the execution to be allowed.
// Validation stops at the first failure (short-circuit evaluation).
type CompositeSecurityPolicy struct {
	validators []SecurityPolicy
}

// NewCompositeSecurityPolicy creates a new CompositeSecurityPolicy with the given validators.
func NewCompositeSecurityPolicy(validators ...SecurityPolicy) *CompositeSecurityPolicy {
	return &CompositeSecurityPolicy{
		validators: validators,
	}
}

// DefaultSecurityPolicy creates a CompositeSecurityPolicy with the default
// blacklist and path validators configured.
func DefaultSecurityPolicy() *CompositeSecurityPolicy {
	return NewCompositeSecurityPolicy(
		DefaultBlacklistValidator(),
		DefaultPathValidator(),
	)
}

// Add adds a validator to the policy.
func (p *CompositeSecurityPolicy) Add(validator SecurityPolicy) *CompositeSecurityPolicy {
	if validator != nil {
		p.validators = append(p.validators, validator)
	}
	return p
}

// Validate implements SecurityPolicy.Validate.
// It runs all validators in order and returns the first error encountered.
func (p *CompositeSecurityPolicy) Validate(ctx context.Context, toolName string, args map[string]any) error {
	for _, validator := range p.validators {
		// Check context cancellation between validators
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := validator.Validate(ctx, toolName, args); err != nil {
			return err
		}
	}
	return nil
}

// Len returns the number of validators in this composite policy.
func (p *CompositeSecurityPolicy) Len() int {
	return len(p.validators)
}

// NoOpSecurityPolicy is a security policy that allows all executions.
// Use this when you want to explicitly disable security validation.
type NoOpSecurityPolicy struct{}

// Validate always returns nil (allows all executions).
func (p *NoOpSecurityPolicy) Validate(ctx context.Context, toolName string, args map[string]any) error {
	return nil
}

// Ensure all validators implement SecurityPolicy.
var (
	_ SecurityPolicy = (*BlacklistValidator)(nil)
	_ SecurityPolicy = (*PathValidator)(nil)
	_ SecurityPolicy = (*CompositeSecurityPolicy)(nil)
	_ SecurityPolicy = (*NoOpSecurityPolicy)(nil)
)
