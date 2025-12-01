// Package config handles configuration and cookie management for geminiweb.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MarkdownConfig configures markdown rendering options
type MarkdownConfig struct {
	Style            string `json:"style"`              // "dark", "light", or path to JSON theme
	EnableEmoji      bool   `json:"enable_emoji"`       // Convert :emoji: to unicode
	PreserveNewLines bool   `json:"preserve_newlines"`  // Preserve original line breaks
	TableWrap        bool   `json:"table_wrap"`         // Enable word wrap in table cells
	InlineTableLinks bool   `json:"inline_table_links"` // Render links inline in tables
}

// Config represents the user configuration
type Config struct {
	DefaultModel    string         `json:"default_model"`
	AutoClose       bool           `json:"auto_close"`
	Verbose         bool           `json:"verbose"`
	CopyToClipboard bool           `json:"copy_to_clipboard"`
	Markdown        MarkdownConfig `json:"markdown,omitempty"`
}

// DefaultMarkdownConfig returns the default markdown configuration
func DefaultMarkdownConfig() MarkdownConfig {
	return MarkdownConfig{
		Style:            "dark",
		EnableEmoji:      true,
		PreserveNewLines: true,
		TableWrap:        true,
		InlineTableLinks: false,
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		DefaultModel:    "gemini-2.5-flash",
		AutoClose:       true,
		Verbose:         false,
		CopyToClipboard: false,
		Markdown:        DefaultMarkdownConfig(),
	}
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".geminiweb")
	return configDir, nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// GetCookiesPath returns the path to the cookies file
func GetCookiesPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "cookies.json"), nil
}

// LoadConfig loads the configuration from disk
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()

	configPath, err := GetConfigPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if config doesn't exist
		}
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(cfg Config) error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AvailableModels returns a list of available model names
func AvailableModels() []string {
	return []string{
		"gemini-2.5-flash",
		"gemini-2.5-pro",
		"gemini-3.0-pro",
		"unspecified",
	}
}
