package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/diogo/geminiweb/internal/config"
)

func TestImportCookiesCmd_Structure(t *testing.T) {
	// Test command structure
	if importCookiesCmd.Use != "import-cookies <path>" {
		t.Errorf("Expected use 'import-cookies <path>', got %s", importCookiesCmd.Use)
	}

	if importCookiesCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if importCookiesCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Verify Args validation is configured
	if importCookiesCmd.Args == nil {
		t.Error("Args validation should be configured")
	}
}

func TestRunImportCookies_FileNotFound(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Try to import from non-existent file
	err := runImportCookies("/nonexistent/file.json")

	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	if !contains(err.Error(), "source file not found") {
		t.Errorf("Expected 'source file not found' error, got: %v", err)
	}
}

func TestRunImportCookies_ValidSource(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test cookies file with dict format
	cookiesFile := filepath.Join(tmpDir, "cookies.json")
	cookies := map[string]string{
		"__Secure-1PSID":   "test_psid_value",
		"__Secure-1PSIDTS": "test_psidts_value",
	}

	data, err := json.Marshal(cookies)
	if err != nil {
		t.Fatalf("Failed to marshal cookies: %v", err)
	}

	err = os.WriteFile(cookiesFile, data, 0o644)
	if err != nil {
		t.Fatalf("Failed to write cookies file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run import
	err = runImportCookies(cookiesFile)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runImportCookies failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain success message
	if !contains(output, "imported successfully") {
		t.Errorf("Output should contain success message: %s", output)
	}

	// Verify the cookies were saved
	loadedCookies, err := config.LoadCookies()
	if err != nil {
		t.Fatalf("Failed to load cookies: %v", err)
	}

	if loadedCookies.Secure1PSID != "test_psid_value" {
		t.Errorf("Expected PSID 'test_psid_value', got %s", loadedCookies.Secure1PSID)
	}

	if loadedCookies.Secure1PSIDTS != "test_psidts_value" {
		t.Errorf("Expected PSIDTS 'test_psidts_value', got %s", loadedCookies.Secure1PSIDTS)
	}
}

func TestRunImportCookies_ListFormat(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test cookies file with list format
	cookiesFile := filepath.Join(tmpDir, "cookies_list.json")
	cookies := []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}{
		{Name: "__Secure-1PSID", Value: "test_psid"},
		{Name: "__Secure-1PSIDTS", Value: "test_psidts"},
	}

	data, err := json.Marshal(cookies)
	if err != nil {
		t.Fatalf("Failed to marshal cookies: %v", err)
	}

	err = os.WriteFile(cookiesFile, data, 0o644)
	if err != nil {
		t.Fatalf("Failed to write cookies file: %v", err)
	}

	// Run import
	err = runImportCookies(cookiesFile)

	if err != nil {
		t.Errorf("runImportCookies failed: %v", err)
	}

	// Verify the cookies were saved
	loadedCookies, err := config.LoadCookies()
	if err != nil {
		t.Fatalf("Failed to load cookies: %v", err)
	}

	if loadedCookies.Secure1PSID != "test_psid" {
		t.Errorf("Expected PSID 'test_psid', got %s", loadedCookies.Secure1PSID)
	}
}

func TestRunImportCookies_OnlyPSID(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test cookies file with only PSID
	cookiesFile := filepath.Join(tmpDir, "cookies_psid_only.json")
	cookies := map[string]string{
		"__Secure-1PSID": "test_psid_only",
	}

	data, err := json.Marshal(cookies)
	if err != nil {
		t.Fatalf("Failed to marshal cookies: %v", err)
	}

	err = os.WriteFile(cookiesFile, data, 0o644)
	if err != nil {
		t.Fatalf("Failed to write cookies file: %v", err)
	}

	// Run import
	err = runImportCookies(cookiesFile)

	if err != nil {
		t.Errorf("runImportCookies failed: %v", err)
	}

	// Verify the cookies were saved
	loadedCookies, err := config.LoadCookies()
	if err != nil {
		t.Fatalf("Failed to load cookies: %v", err)
	}

	if loadedCookies.Secure1PSID != "test_psid_only" {
		t.Errorf("Expected PSID 'test_psid_only', got %s", loadedCookies.Secure1PSID)
	}

	// PSIDTS should be empty
	if loadedCookies.Secure1PSIDTS != "" {
		t.Errorf("Expected empty PSIDTS, got %s", loadedCookies.Secure1PSIDTS)
	}
}

func TestRunImportCookies_InvalidJSON(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test cookies file with invalid JSON
	cookiesFile := filepath.Join(tmpDir, "cookies_invalid.json")

	err := os.WriteFile(cookiesFile, []byte("invalid json {"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write cookies file: %v", err)
	}

	// Run import - should handle error gracefully
	err = runImportCookies(cookiesFile)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// Simple substring search implementation
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
