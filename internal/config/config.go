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
	DefaultModel string `json:"default_model"`
	// AutoClose controls automatic client shutdown after inactivity.
	// When enabled, the GeminiClient will automatically close and stop
	// background cookie rotation after CloseDelay seconds of inactivity.
	AutoClose bool `json:"auto_close"`
	// CloseDelay is the number of seconds of inactivity before auto-close triggers.
	// Default is 300 (5 minutes). Minimum recommended is 30 seconds.
	CloseDelay int `json:"close_delay"`
	// AutoReInit enables automatic re-initialization when a request is made
	// after the client was auto-closed due to inactivity.
	AutoReInit bool `json:"auto_reinit"`
	// Verbose enables detailed logging output during operations.
	// When enabled, shows model info, request timing, and response metadata.
	Verbose         bool           `json:"verbose"`
	CopyToClipboard bool           `json:"copy_to_clipboard"`
	TUITheme        string         `json:"tui_theme,omitempty"`    // TUI color theme
	DownloadDir     string         `json:"download_dir,omitempty"` // Directory for saving images
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
	homeDir, _ := os.UserHomeDir()
	return Config{
		DefaultModel:    "fast",
		AutoClose:       true,
		CloseDelay:      300, // 5 minutes
		AutoReInit:      true,
		Verbose:         false,
		CopyToClipboard: false,
		TUITheme:        "tokyonight",
		DownloadDir:     filepath.Join(homeDir, ".geminiweb", "images"),
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

	// Use 0o700 for sensitive directories (contains cookies and config)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
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

// GetDownloadDir returns the download directory from config, creating it if necessary
func GetDownloadDir(cfg Config) (string, error) {
	dir := cfg.DownloadDir
	if dir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(homeDir, ".geminiweb", "images")
	}

	// Ensure directory exists (0o700 for privacy - downloaded images may be sensitive)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	return dir, nil
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

	// Use 0o600 for sensitive files (config may contain preferences)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AvailableModels returns a list of available model names
func AvailableModels() []string {
	return []string{
		"fast",
		"thinking",
		"pro",
	}
}
