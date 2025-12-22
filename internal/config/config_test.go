package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultModel != "fast" {
		t.Errorf("Expected default model to be 'fast', got '%s'", cfg.DefaultModel)
	}

	if cfg.AutoClose != true {
		t.Errorf("Expected AutoClose to be true, got %v", cfg.AutoClose)
	}

	if cfg.Verbose != false {
		t.Errorf("Expected Verbose to be false, got %v", cfg.Verbose)
	}

	if cfg.AutoApproveTools != false {
		t.Errorf("Expected AutoApproveTools to be false, got %v", cfg.AutoApproveTools)
	}
}

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() returned error: %v", err)
	}
	if dir == "" {
		t.Error("GetConfigDir() returned empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("GetConfigDir() returned relative path: %s", dir)
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() returned error: %v", err)
	}
	if path == "" {
		t.Error("GetConfigPath() returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("GetConfigPath() returned relative path: %s", path)
	}
}

func TestLoadConfig_FileNotExists(t *testing.T) {
	// Test with current HOME - file may or may not exist
	cfg, err := LoadConfig()
	if err != nil {
		// If there's an error parsing existing config, check it's a parse error
		t.Logf("LoadConfig() returned: %v", err)
	}

	// Check that we got a valid config (either loaded or default)
	if cfg.DefaultModel == "" {
		t.Error("DefaultModel should not be empty")
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		DefaultModel:     "test-model",
		AutoClose:        true,
		Verbose:          true,
		AutoApproveTools: true,
	}

	if cfg.DefaultModel != "test-model" {
		t.Error("DefaultModel mismatch")
	}
	if !cfg.AutoClose {
		t.Error("AutoClose mismatch")
	}
	if !cfg.Verbose {
		t.Error("Verbose mismatch")
	}
	if !cfg.AutoApproveTools {
		t.Error("AutoApproveTools mismatch")
	}
}

func TestAvailableModels(t *testing.T) {
	models := AvailableModels()

	if len(models) == 0 {
		t.Error("AvailableModels() returned empty list")
	}

	// Check for expected models
	expected := []string{"fast", "thinking", "pro"}
	for _, expectedModel := range expected {
		found := false
		for _, model := range models {
			if model == expectedModel {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model '%s' not found in available models", expectedModel)
		}
	}
}

func TestEnsureConfigDir(t *testing.T) {
	// This test uses the real HOME directory
	dir, err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() returned error: %v", err)
	}
	if dir == "" {
		t.Error("EnsureConfigDir() returned empty string")
	}

	// Verify directory exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Path is not a directory")
	}
}

func TestGetCookiesPath(t *testing.T) {
	path, err := GetCookiesPath()
	if err != nil {
		t.Fatalf("GetCookiesPath() returned error: %v", err)
	}
	if path == "" {
		t.Error("GetCookiesPath() returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("GetCookiesPath() returned relative path: %s", path)
	}
	if filepath.Base(path) != "cookies.json" {
		t.Errorf("GetCookiesPath() should end with cookies.json, got %s", filepath.Base(path))
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	cfg := Config{
		DefaultModel:     "gemini-3.0-pro",
		AutoClose:        false,
		Verbose:          true,
		AutoApproveTools: true,
	}

	err := SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig() returned error: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmpDir, ".geminiweb", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Verify content
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to parse saved config: %v", err)
	}

	if saved.DefaultModel != cfg.DefaultModel {
		t.Errorf("DefaultModel = %s, want %s", saved.DefaultModel, cfg.DefaultModel)
	}
	if saved.AutoClose != cfg.AutoClose {
		t.Errorf("AutoClose = %v, want %v", saved.AutoClose, cfg.AutoClose)
	}
	if saved.Verbose != cfg.Verbose {
		t.Errorf("Verbose = %v, want %v", saved.Verbose, cfg.Verbose)
	}
	if saved.AutoApproveTools != cfg.AutoApproveTools {
		t.Errorf("AutoApproveTools = %v, want %v", saved.AutoApproveTools, cfg.AutoApproveTools)
	}

	// Check file permissions (should be 600 for privacy)
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("File permissions = %o, want 600", perm)
	}
}

func TestLoadConfig_WithExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create config directory and file
	configDir := filepath.Join(tmpDir, ".geminiweb")
	_ = os.MkdirAll(configDir, 0o755)

	configPath := filepath.Join(configDir, "config.json")
	originalCfg := Config{
		DefaultModel:     "gemini-2.5-pro",
		AutoClose:        false,
		Verbose:          true,
		AutoApproveTools: true,
	}

	data, _ := json.MarshalIndent(originalCfg, "", "  ")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.DefaultModel != originalCfg.DefaultModel {
		t.Errorf("DefaultModel = %s, want %s", cfg.DefaultModel, originalCfg.DefaultModel)
	}
	if cfg.AutoClose != originalCfg.AutoClose {
		t.Errorf("AutoClose = %v, want %v", cfg.AutoClose, originalCfg.AutoClose)
	}
	if cfg.Verbose != originalCfg.Verbose {
		t.Errorf("Verbose = %v, want %v", cfg.Verbose, originalCfg.Verbose)
	}
	if cfg.AutoApproveTools != originalCfg.AutoApproveTools {
		t.Errorf("AutoApproveTools = %v, want %v", cfg.AutoApproveTools, originalCfg.AutoApproveTools)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create config directory and file with invalid JSON
	configDir := filepath.Join(tmpDir, ".geminiweb")
	_ = os.MkdirAll(configDir, 0o755)

	configPath := filepath.Join(configDir, "config.json")
	invalidJSON := `{"invalid": json content`
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig() with invalid JSON should return error")
	}

	// Should return default config on error
	if cfg.DefaultModel != "fast" {
		t.Errorf("DefaultModel = %s, want fast", cfg.DefaultModel)
	}
}

func TestGetDownloadDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	t.Run("with custom directory", func(t *testing.T) {
		customDir := filepath.Join(tmpDir, "custom_downloads")
		cfg := Config{
			DownloadDir: customDir,
		}

		dir, err := GetDownloadDir(cfg)
		if err != nil {
			t.Fatalf("GetDownloadDir() returned error: %v", err)
		}
		if dir != customDir {
			t.Errorf("GetDownloadDir() = %q, want %q", dir, customDir)
		}

		// Verify directory was created
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("Path is not a directory")
		}

		// Check permissions (0700 for privacy)
		perm := info.Mode().Perm()
		if perm != 0o700 {
			t.Errorf("Directory permissions = %o, want 700", perm)
		}
	})

	t.Run("with empty directory (uses default)", func(t *testing.T) {
		cfg := Config{
			DownloadDir: "",
		}

		dir, err := GetDownloadDir(cfg)
		if err != nil {
			t.Fatalf("GetDownloadDir() returned error: %v", err)
		}

		expectedDir := filepath.Join(tmpDir, ".geminiweb", "images")
		if dir != expectedDir {
			t.Errorf("GetDownloadDir() = %q, want %q", dir, expectedDir)
		}

		// Verify directory was created
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("Path is not a directory")
		}
	})

	t.Run("directory already exists", func(t *testing.T) {
		existingDir := filepath.Join(tmpDir, "existing_dir")
		if err := os.MkdirAll(existingDir, 0o755); err != nil {
			t.Fatalf("Failed to create existing directory: %v", err)
		}

		cfg := Config{
			DownloadDir: existingDir,
		}

		dir, err := GetDownloadDir(cfg)
		if err != nil {
			t.Fatalf("GetDownloadDir() returned error: %v", err)
		}
		if dir != existingDir {
			t.Errorf("GetDownloadDir() = %q, want %q", dir, existingDir)
		}
	})
}

func TestDefaultMarkdownConfig(t *testing.T) {
	cfg := DefaultMarkdownConfig()

	if cfg.Style != "dark" {
		t.Errorf("Style = %q, want 'dark'", cfg.Style)
	}
	if !cfg.EnableEmoji {
		t.Error("EnableEmoji should be true")
	}
	if !cfg.PreserveNewLines {
		t.Error("PreserveNewLines should be true")
	}
	if !cfg.TableWrap {
		t.Error("TableWrap should be true")
	}
	if cfg.InlineTableLinks {
		t.Error("InlineTableLinks should be false")
	}
}

func TestDefaultConfig_AllFields(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultModel != "fast" {
		t.Errorf("DefaultModel = %q, want 'fast'", cfg.DefaultModel)
	}
	if !cfg.AutoClose {
		t.Error("AutoClose should be true")
	}
	if cfg.CloseDelay != 300 {
		t.Errorf("CloseDelay = %d, want 300", cfg.CloseDelay)
	}
	if !cfg.AutoReInit {
		t.Error("AutoReInit should be true")
	}
	if cfg.Verbose {
		t.Error("Verbose should be false")
	}
	if cfg.CopyToClipboard {
		t.Error("CopyToClipboard should be false")
	}
	if cfg.AutoApproveTools {
		t.Error("AutoApproveTools should be false")
	}
	if cfg.TUITheme != "tokyonight" {
		t.Errorf("TUITheme = %q, want 'tokyonight'", cfg.TUITheme)
	}
	if cfg.DownloadDir == "" {
		t.Error("DownloadDir should not be empty")
	}
	if cfg.Markdown.Style != "dark" {
		t.Errorf("Markdown.Style = %q, want 'dark'", cfg.Markdown.Style)
	}
}
