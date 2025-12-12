package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultModel != "gemini-2.5-flash" {
		t.Errorf("Expected default model to be 'gemini-2.5-flash', got '%s'", cfg.DefaultModel)
	}

	if cfg.AutoClose != true {
		t.Errorf("Expected AutoClose to be true, got %v", cfg.AutoClose)
	}

	if cfg.Verbose != false {
		t.Errorf("Expected Verbose to be false, got %v", cfg.Verbose)
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
		DefaultModel: "test-model",
		AutoClose:    true,
		Verbose:      true,
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
}

func TestAvailableModels(t *testing.T) {
	models := AvailableModels()

	if len(models) == 0 {
		t.Error("AvailableModels() returned empty list")
	}

	// Check for expected models
	expected := []string{"gemini-2.5-flash", "gemini-3.0-pro", "unspecified"}
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
		DefaultModel: "gemini-3.0-pro",
		AutoClose:    false,
		Verbose:      true,
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

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o644 {
		t.Errorf("File permissions = %o, want 644", perm)
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
		DefaultModel: "gemini-2.5-pro",
		AutoClose:    false,
		Verbose:      true,
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
	if cfg.DefaultModel != "gemini-2.5-flash" {
		t.Errorf("DefaultModel = %s, want gemini-2.5-flash", cfg.DefaultModel)
	}
}
