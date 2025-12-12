package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCookies_DictFormat(t *testing.T) {
	dictFormat := `{
  "__Secure-1PSID": "test_psid_value",
  "__Secure-1PSIDTS": "test_psidts_value"
}`

	cookies, err := parseCookies([]byte(dictFormat))
	if err != nil {
		t.Fatalf("parseCookies() with dict format returned error: %v", err)
	}

	if cookies.Secure1PSID != "test_psid_value" {
		t.Errorf("Expected Secure1PSID 'test_psid_value', got '%s'", cookies.Secure1PSID)
	}

	if cookies.Secure1PSIDTS != "test_psidts_value" {
		t.Errorf("Expected Secure1PSIDTS 'test_psidts_value', got '%s'", cookies.Secure1PSIDTS)
	}
}

func TestParseCookies_DictFormat_WithoutPSIDTS(t *testing.T) {
	dictFormat := `{
  "__Secure-1PSID": "test_psid_value"
}`

	cookies, err := parseCookies([]byte(dictFormat))
	if err != nil {
		t.Fatalf("parseCookies() with dict format (no PSIDTS) returned error: %v", err)
	}

	if cookies.Secure1PSID != "test_psid_value" {
		t.Errorf("Expected Secure1PSID 'test_psid_value', got '%s'", cookies.Secure1PSID)
	}

	if cookies.Secure1PSIDTS != "" {
		t.Errorf("Expected Secure1PSIDTS to be empty, got '%s'", cookies.Secure1PSIDTS)
	}
}

func TestParseCookies_DictFormat_MissingPSID(t *testing.T) {
	dictFormat := `{
  "__Secure-1PSIDTS": "test_psidts_value"
}`

	_, err := parseCookies([]byte(dictFormat))
	if err == nil {
		t.Error("parseCookies() with missing PSID should return error")
	}
}

func TestParseCookies_ListFormat(t *testing.T) {
	listFormat := `[
  {"name": "__Secure-1PSID", "value": "test_psid_value"},
  {"name": "__Secure-1PSIDTS", "value": "test_psidts_value"}
]`

	cookies, err := parseCookies([]byte(listFormat))
	if err != nil {
		t.Fatalf("parseCookies() with list format returned error: %v", err)
	}

	if cookies.Secure1PSID != "test_psid_value" {
		t.Errorf("Expected Secure1PSID 'test_psid_value', got '%s'", cookies.Secure1PSID)
	}

	if cookies.Secure1PSIDTS != "test_psidts_value" {
		t.Errorf("Expected Secure1PSIDTS 'test_psidts_value', got '%s'", cookies.Secure1PSIDTS)
	}
}

func TestParseCookies_ListFormat_OnlyPSID(t *testing.T) {
	listFormat := `[
  {"name": "__Secure-1PSID", "value": "test_psid_value"}
]`

	cookies, err := parseCookies([]byte(listFormat))
	if err != nil {
		t.Fatalf("parseCookies() with list format (only PSID) returned error: %v", err)
	}

	if cookies.Secure1PSID != "test_psid_value" {
		t.Errorf("Expected Secure1PSID 'test_psid_value', got '%s'", cookies.Secure1PSID)
	}

	if cookies.Secure1PSIDTS != "" {
		t.Errorf("Expected Secure1PSIDTS to be empty, got '%s'", cookies.Secure1PSIDTS)
	}
}

func TestParseCookies_ListFormat_ExtraCookies(t *testing.T) {
	listFormat := `[
  {"name": "__Secure-1PSID", "value": "test_psid_value"},
  {"name": "__Secure-1PSIDTS", "value": "test_psidts_value"},
  {"name": "other_cookie", "value": "should_ignore"},
  {"name": "__Secure-SOMECOOKIE", "value": "ignored"}
]`

	cookies, err := parseCookies([]byte(listFormat))
	if err != nil {
		t.Fatalf("parseCookies() with extra cookies returned error: %v", err)
	}

	if cookies.Secure1PSID != "test_psid_value" {
		t.Errorf("Expected Secure1PSID 'test_psid_value', got '%s'", cookies.Secure1PSID)
	}

	if cookies.Secure1PSIDTS != "test_psidts_value" {
		t.Errorf("Expected Secure1PSIDTS 'test_psidts_value', got '%s'", cookies.Secure1PSIDTS)
	}
}

func TestParseCookies_ListFormat_MissingPSID(t *testing.T) {
	listFormat := `[
  {"name": "__Secure-1PSIDTS", "value": "test_psidts_value"}
]`

	_, err := parseCookies([]byte(listFormat))
	if err == nil {
		t.Error("parseCookies() with missing PSID in list format should return error")
	}
}

func TestParseCookies_InvalidJSON(t *testing.T) {
	invalidJSON := `{"invalid": json`

	_, err := parseCookies([]byte(invalidJSON))
	if err == nil {
		t.Error("parseCookies() with invalid JSON should return error")
	}
}

func TestParseCookies_NeitherFormat(t *testing.T) {
	otherFormat := `{"some_other_field": "value"}`

	_, err := parseCookies([]byte(otherFormat))
	if err == nil {
		t.Error("parseCookies() with unknown format should return error")
	}
}

func TestCookies_Fields(t *testing.T) {
	cookies := &Cookies{
		Secure1PSID:   "psid_value",
		Secure1PSIDTS: "psidts_value",
	}

	if cookies.Secure1PSID != "psid_value" {
		t.Error("Secure1PSID mismatch")
	}
	if cookies.Secure1PSIDTS != "psidts_value" {
		t.Error("Secure1PSIDTS mismatch")
	}
}

func TestValidateCookies(t *testing.T) {
	tests := []struct {
		name    string
		cookies *Cookies
		wantErr bool
	}{
		{
			name:    "nil cookies",
			cookies: nil,
			wantErr: true,
		},
		{
			name:    "empty PSID",
			cookies: &Cookies{Secure1PSID: ""},
			wantErr: true,
		},
		{
			name:    "valid with both cookies",
			cookies: &Cookies{Secure1PSID: "psid", Secure1PSIDTS: "psidts"},
			wantErr: false,
		},
		{
			name:    "valid with only PSID",
			cookies: &Cookies{Secure1PSID: "psid"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCookies(tt.cookies)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCookies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCookieListItem_Fields(t *testing.T) {
	item := CookieListItem{
		Name:  "test_name",
		Value: "test_value",
	}

	if item.Name != "test_name" {
		t.Error("Name mismatch")
	}
	if item.Value != "test_value" {
		t.Error("Value mismatch")
	}
}

// Helper to setup isolated test environment
func setupCookiesTestEnv(t *testing.T) (tmpDir string, cleanup func()) {
	tmpDir = t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)

	// Create the config directory
	configDir := filepath.Join(tmpDir, ".geminiweb")
	_ = os.MkdirAll(configDir, 0o755)

	cleanup = func() {
		_ = os.Setenv("HOME", oldHome)
	}
	return tmpDir, cleanup
}

func TestLoadCookies_FileNotExists(t *testing.T) {
	_, cleanup := setupCookiesTestEnv(t)
	defer cleanup()

	_, err := LoadCookies()
	if err == nil {
		t.Error("LoadCookies() with non-existent file should return error")
	}
}

func TestSaveAndLoadCookies(t *testing.T) {
	_, cleanup := setupCookiesTestEnv(t)
	defer cleanup()

	cookies := &Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	err := SaveCookies(cookies)
	if err != nil {
		t.Fatalf("SaveCookies() returned error: %v", err)
	}

	loaded, err := LoadCookies()
	if err != nil {
		t.Fatalf("LoadCookies() returned error: %v", err)
	}

	if loaded.Secure1PSID != cookies.Secure1PSID {
		t.Errorf("Secure1PSID = %s, want %s", loaded.Secure1PSID, cookies.Secure1PSID)
	}
	if loaded.Secure1PSIDTS != cookies.Secure1PSIDTS {
		t.Errorf("Secure1PSIDTS = %s, want %s", loaded.Secure1PSIDTS, cookies.Secure1PSIDTS)
	}
}

func TestImportCookies_SourceNotExists(t *testing.T) {
	err := ImportCookies("/path/to/nonexistent/file.json")
	if err == nil {
		t.Error("ImportCookies() with non-existent source file should return error")
	}
}

func TestImportCookies_ValidSource(t *testing.T) {
	tmpDir, cleanup := setupCookiesTestEnv(t)
	defer cleanup()

	sourceFile := filepath.Join(tmpDir, "source_cookies.json")
	sourceCookies := `[
  {"name": "__Secure-1PSID", "value": "imported_psid"},
  {"name": "__Secure-1PSIDTS", "value": "imported_psidts"}
]`

	if err := os.WriteFile(sourceFile, []byte(sourceCookies), 0o644); err != nil {
		t.Fatalf("Failed to write source cookies file: %v", err)
	}

	err := ImportCookies(sourceFile)
	if err != nil {
		t.Fatalf("ImportCookies() returned error: %v", err)
	}

	loaded, err := LoadCookies()
	if err != nil {
		t.Fatalf("LoadCookies() returned error: %v", err)
	}

	if loaded.Secure1PSID != "imported_psid" {
		t.Errorf("Secure1PSID = %s, want imported_psid", loaded.Secure1PSID)
	}
}

func TestCookies_ToMap(t *testing.T) {
	cookies := &Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	m := cookies.ToMap()

	if len(m) != 2 {
		t.Errorf("ToMap() returned map with %d entries, want 2", len(m))
	}

	if m["__Secure-1PSID"] != "test_psid" {
		t.Errorf("__Secure-1PSID = %s, want test_psid", m["__Secure-1PSID"])
	}

	if m["__Secure-1PSIDTS"] != "test_psidts" {
		t.Errorf("__Secure-1PSIDTS = %s, want test_psidts", m["__Secure-1PSIDTS"])
	}
}

func TestCookies_ToMap_WithoutPSIDTS(t *testing.T) {
	cookies := &Cookies{
		Secure1PSID: "test_psid",
	}

	m := cookies.ToMap()

	if len(m) != 1 {
		t.Errorf("ToMap() returned map with %d entries, want 1", len(m))
	}

	if m["__Secure-1PSID"] != "test_psid" {
		t.Errorf("__Secure-1PSID = %s, want test_psid", m["__Secure-1PSID"])
	}

	if _, ok := m["__Secure-1PSIDTS"]; ok {
		t.Error("__Secure-1PSIDTS should not be in map when empty")
	}
}

func TestCookies_Update1PSIDTS(t *testing.T) {
	cookies := &Cookies{
		Secure1PSID:   "original_psid",
		Secure1PSIDTS: "original_psidts",
	}

	cookies.Update1PSIDTS("updated_psidts")

	if cookies.Secure1PSIDTS != "updated_psidts" {
		t.Errorf("Secure1PSIDTS = %s, want updated_psidts", cookies.Secure1PSIDTS)
	}

	// PSID should not change
	if cookies.Secure1PSID != "original_psid" {
		t.Errorf("Secure1PSID = %s, want original_psid", cookies.Secure1PSID)
	}
}

func TestCookies_Update1PSIDTS_Empty(t *testing.T) {
	cookies := &Cookies{
		Secure1PSID: "test_psid",
	}

	cookies.Update1PSIDTS("")

	if cookies.Secure1PSIDTS != "" {
		t.Errorf("Secure1PSIDTS = %s, want empty", cookies.Secure1PSIDTS)
	}
}
