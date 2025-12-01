package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPersonas(t *testing.T) {
	personas := DefaultPersonas()

	if len(personas) < 5 {
		t.Errorf("expected at least 5 default personas, got %d", len(personas))
	}

	// Check that 'default' persona exists
	foundDefault := false
	for _, p := range personas {
		if p.Name == "default" {
			foundDefault = true
			if p.SystemPrompt != "" {
				t.Error("default persona should have empty system prompt")
			}
		}
	}

	if !foundDefault {
		t.Error("default persona not found")
	}
}

func TestDefaultPersonas_AllHaveNames(t *testing.T) {
	personas := DefaultPersonas()

	for i, p := range personas {
		if p.Name == "" {
			t.Errorf("persona %d has empty name", i)
		}
		if p.Description == "" {
			t.Errorf("persona %s has empty description", p.Name)
		}
	}
}

func TestPersona_Fields(t *testing.T) {
	p := Persona{
		Name:         "test",
		Description:  "Test persona",
		SystemPrompt: "Be helpful",
		Model:        "gemini-2.5-pro",
		Temperature:  0.7,
	}

	if p.Name != "test" {
		t.Error("Name mismatch")
	}
	if p.Description != "Test persona" {
		t.Error("Description mismatch")
	}
	if p.SystemPrompt != "Be helpful" {
		t.Error("SystemPrompt mismatch")
	}
	if p.Model != "gemini-2.5-pro" {
		t.Error("Model mismatch")
	}
	if p.Temperature != 0.7 {
		t.Error("Temperature mismatch")
	}
}

func TestPersonaConfig_Fields(t *testing.T) {
	config := PersonaConfig{
		Personas: []Persona{
			{Name: "test"},
		},
		DefaultPersona: "test",
	}

	if len(config.Personas) != 1 {
		t.Error("Personas length mismatch")
	}
	if config.DefaultPersona != "test" {
		t.Error("DefaultPersona mismatch")
	}
}

func TestMergePersonas(t *testing.T) {
	defaults := []Persona{
		{Name: "default", Description: "Default"},
		{Name: "coder", Description: "Coder"},
	}

	custom := []Persona{
		{Name: "coder", Description: "Custom Coder"}, // Override
		{Name: "mybot", Description: "My Bot"},       // New
	}

	result := mergePersonas(defaults, custom)

	if len(result) != 3 {
		t.Errorf("expected 3 personas, got %d", len(result))
	}

	// Check override
	for _, p := range result {
		if p.Name == "coder" && p.Description != "Custom Coder" {
			t.Error("coder persona should be overridden")
		}
	}

	// Check new persona added
	foundMyBot := false
	for _, p := range result {
		if p.Name == "mybot" {
			foundMyBot = true
		}
	}
	if !foundMyBot {
		t.Error("mybot persona not found")
	}
}

func TestMergePersonas_EmptyCustom(t *testing.T) {
	defaults := DefaultPersonas()
	result := mergePersonas(defaults, nil)

	if len(result) != len(defaults) {
		t.Error("empty custom should return defaults")
	}
}

func TestFormatSystemPrompt_WithPersona(t *testing.T) {
	persona := &Persona{
		Name:         "test",
		SystemPrompt: "Be helpful and concise",
	}

	result := FormatSystemPrompt(persona, "Hello")

	if result == "Hello" {
		t.Error("should format with system prompt")
	}

	if result != `[System Instructions]
Be helpful and concise

[User Message]
Hello` {
		t.Errorf("unexpected format: %s", result)
	}
}

func TestFormatSystemPrompt_NilPersona(t *testing.T) {
	result := FormatSystemPrompt(nil, "Hello")

	if result != "Hello" {
		t.Error("nil persona should return message unchanged")
	}
}

func TestFormatSystemPrompt_EmptySystemPrompt(t *testing.T) {
	persona := &Persona{
		Name:         "default",
		SystemPrompt: "",
	}

	result := FormatSystemPrompt(persona, "Hello")

	if result != "Hello" {
		t.Error("empty system prompt should return message unchanged")
	}
}

// Tests that require file system
func setupTestConfig(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, ".geminiweb")
	os.MkdirAll(configDir, 0o755)

	cleanup := func() {
		os.Setenv("HOME", oldHome)
	}

	return tmpDir, cleanup
}

func TestLoadPersonas_NoFile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	config, err := LoadPersonas()
	if err != nil {
		t.Fatalf("LoadPersonas failed: %v", err)
	}

	if config == nil {
		t.Fatal("config is nil")
	}

	if len(config.Personas) == 0 {
		t.Error("should return default personas")
	}

	if config.DefaultPersona != "default" {
		t.Errorf("DefaultPersona = %s, want default", config.DefaultPersona)
	}
}

func TestSaveAndLoadPersonas(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	config := &PersonaConfig{
		Personas: []Persona{
			{Name: "test", Description: "Test Persona", SystemPrompt: "Be test"},
		},
		DefaultPersona: "test",
	}

	err := SavePersonas(config)
	if err != nil {
		t.Fatalf("SavePersonas failed: %v", err)
	}

	loaded, err := LoadPersonas()
	if err != nil {
		t.Fatalf("LoadPersonas failed: %v", err)
	}

	// Should have merged with defaults
	if len(loaded.Personas) < 5 {
		t.Error("should merge with defaults")
	}

	if loaded.DefaultPersona != "test" {
		t.Errorf("DefaultPersona = %s, want test", loaded.DefaultPersona)
	}
}

func TestGetPersona(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	persona, err := GetPersona("coder")
	if err != nil {
		t.Fatalf("GetPersona failed: %v", err)
	}

	if persona.Name != "coder" {
		t.Errorf("Name = %s, want coder", persona.Name)
	}
}

func TestGetPersona_NotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	_, err := GetPersona("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent persona")
	}
}

func TestListPersonaNames(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	names, err := ListPersonaNames()
	if err != nil {
		t.Fatalf("ListPersonaNames failed: %v", err)
	}

	if len(names) == 0 {
		t.Error("expected at least one persona name")
	}

	// Check default is in list
	foundDefault := false
	for _, name := range names {
		if name == "default" {
			foundDefault = true
		}
	}
	if !foundDefault {
		t.Error("default not in list")
	}
}

func TestAddPersona(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	newPersona := Persona{
		Name:         "mybot",
		Description:  "My custom bot",
		SystemPrompt: "Be awesome",
	}

	err := AddPersona(newPersona)
	if err != nil {
		t.Fatalf("AddPersona failed: %v", err)
	}

	retrieved, err := GetPersona("mybot")
	if err != nil {
		t.Fatalf("GetPersona failed: %v", err)
	}

	if retrieved.Description != "My custom bot" {
		t.Error("persona not saved correctly")
	}
}

func TestAddPersona_Duplicate(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	err := AddPersona(Persona{Name: "coder"})
	if err == nil {
		t.Error("expected error for duplicate persona")
	}
}

func TestUpdatePersona(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	updated := Persona{
		Name:         "coder",
		Description:  "Updated Coder",
		SystemPrompt: "New prompt",
	}

	err := UpdatePersona(updated)
	if err != nil {
		t.Fatalf("UpdatePersona failed: %v", err)
	}

	retrieved, _ := GetPersona("coder")
	if retrieved.Description != "Updated Coder" {
		t.Error("persona not updated")
	}
}

func TestUpdatePersona_NotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	err := UpdatePersona(Persona{Name: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent persona")
	}
}

func TestDeletePersona(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a persona first
	AddPersona(Persona{Name: "todelete"})

	err := DeletePersona("todelete")
	if err != nil {
		t.Fatalf("DeletePersona failed: %v", err)
	}

	_, err = GetPersona("todelete")
	if err == nil {
		t.Error("persona should be deleted")
	}
}

func TestDeletePersona_CannotDeleteDefault(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	err := DeletePersona("default")
	if err == nil {
		t.Error("should not allow deleting default persona")
	}
}

func TestSetDefaultPersona(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	err := SetDefaultPersona("coder")
	if err != nil {
		t.Fatalf("SetDefaultPersona failed: %v", err)
	}

	config, _ := LoadPersonas()
	if config.DefaultPersona != "coder" {
		t.Errorf("DefaultPersona = %s, want coder", config.DefaultPersona)
	}
}

func TestSetDefaultPersona_NotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	err := SetDefaultPersona("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent persona")
	}
}

func TestGetDefaultPersona(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	persona, err := GetDefaultPersona()
	if err != nil {
		t.Fatalf("GetDefaultPersona failed: %v", err)
	}

	if persona.Name != "default" {
		t.Errorf("default persona Name = %s, want default", persona.Name)
	}
}
