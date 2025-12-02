package models

import (
	"testing"
)

func TestGemJarGet(t *testing.T) {
	jar := make(GemJar)
	jar["abc123"] = &Gem{ID: "abc123", Name: "Test Gem", Predefined: false}
	jar["def456"] = &Gem{ID: "def456", Name: "System Gem", Predefined: true}

	tests := []struct {
		name    string
		id      string
		gemName string
		wantID  string
		wantNil bool
	}{
		{"by ID", "abc123", "", "abc123", false},
		{"by name", "", "Test Gem", "abc123", false},
		{"by name case sensitive", "", "test gem", "", true},
		{"not found", "xyz", "Unknown", "", true},
		{"empty search", "", "", "", true},
		{"by ID takes priority", "abc123", "System Gem", "abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gem := jar.Get(tt.id, tt.gemName)
			if tt.wantNil && gem != nil {
				t.Error("Expected nil, got gem")
			}
			if !tt.wantNil && (gem == nil || gem.ID != tt.wantID) {
				t.Errorf("Expected ID %s, got %v", tt.wantID, gem)
			}
		})
	}
}

func TestGemJarFilter(t *testing.T) {
	jar := make(GemJar)
	jar["1"] = &Gem{ID: "1", Name: "Code Helper", Predefined: true}
	jar["2"] = &Gem{ID: "2", Name: "My Coder", Predefined: false}
	jar["3"] = &Gem{ID: "3", Name: "Writer", Predefined: false}

	t.Run("filter by predefined true", func(t *testing.T) {
		predefined := true
		result := jar.Filter(&predefined, "")
		if len(result) != 1 {
			t.Errorf("Expected 1, got %d", len(result))
		}
	})

	t.Run("filter by predefined false", func(t *testing.T) {
		predefined := false
		result := jar.Filter(&predefined, "")
		if len(result) != 2 {
			t.Errorf("Expected 2, got %d", len(result))
		}
	})

	t.Run("filter by name contains", func(t *testing.T) {
		result := jar.Filter(nil, "code")
		if len(result) != 2 {
			t.Errorf("Expected 2, got %d", len(result))
		}
	})

	t.Run("filter by name contains case insensitive", func(t *testing.T) {
		result := jar.Filter(nil, "CODE")
		if len(result) != 2 {
			t.Errorf("Expected 2, got %d", len(result))
		}
	})

	t.Run("filter combined", func(t *testing.T) {
		predefined := false
		result := jar.Filter(&predefined, "coder")
		if len(result) != 1 {
			t.Errorf("Expected 1, got %d", len(result))
		}
		if _, ok := result["2"]; !ok {
			t.Error("Expected gem with ID '2'")
		}
	})

	t.Run("filter no match", func(t *testing.T) {
		result := jar.Filter(nil, "xyz")
		if len(result) != 0 {
			t.Errorf("Expected 0, got %d", len(result))
		}
	})
}

func TestGemJarCustom(t *testing.T) {
	jar := make(GemJar)
	jar["1"] = &Gem{ID: "1", Name: "Code Helper", Predefined: true}
	jar["2"] = &Gem{ID: "2", Name: "My Coder", Predefined: false}
	jar["3"] = &Gem{ID: "3", Name: "Writer", Predefined: false}

	custom := jar.Custom()
	if len(custom) != 2 {
		t.Errorf("Expected 2 custom gems, got %d", len(custom))
	}

	for _, gem := range custom {
		if gem.Predefined {
			t.Errorf("Expected custom gem, got predefined: %s", gem.Name)
		}
	}
}

func TestGemJarSystem(t *testing.T) {
	jar := make(GemJar)
	jar["1"] = &Gem{ID: "1", Name: "Code Helper", Predefined: true}
	jar["2"] = &Gem{ID: "2", Name: "Another System", Predefined: true}
	jar["3"] = &Gem{ID: "3", Name: "Writer", Predefined: false}

	system := jar.System()
	if len(system) != 2 {
		t.Errorf("Expected 2 system gems, got %d", len(system))
	}

	for _, gem := range system {
		if !gem.Predefined {
			t.Errorf("Expected system gem, got custom: %s", gem.Name)
		}
	}
}

func TestGemJarValues(t *testing.T) {
	jar := make(GemJar)
	jar["1"] = &Gem{ID: "1", Name: "Gem1"}
	jar["2"] = &Gem{ID: "2", Name: "Gem2"}
	jar["3"] = &Gem{ID: "3", Name: "Gem3"}

	values := jar.Values()
	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}

	// Verificar que todos os gems estão presentes
	ids := make(map[string]bool)
	for _, gem := range values {
		ids[gem.ID] = true
	}
	for expectedID := range jar {
		if !ids[expectedID] {
			t.Errorf("Missing gem with ID %s", expectedID)
		}
	}
}

func TestGemJarLen(t *testing.T) {
	jar := make(GemJar)
	if jar.Len() != 0 {
		t.Errorf("Expected 0, got %d", jar.Len())
	}

	jar["1"] = &Gem{ID: "1"}
	if jar.Len() != 1 {
		t.Errorf("Expected 1, got %d", jar.Len())
	}

	jar["2"] = &Gem{ID: "2"}
	jar["3"] = &Gem{ID: "3"}
	if jar.Len() != 3 {
		t.Errorf("Expected 3, got %d", jar.Len())
	}
}

func TestEmptyGemJar(t *testing.T) {
	jar := make(GemJar)

	t.Run("Get on empty", func(t *testing.T) {
		gem := jar.Get("any", "any")
		if gem != nil {
			t.Error("Expected nil on empty jar")
		}
	})

	t.Run("Filter on empty", func(t *testing.T) {
		result := jar.Filter(nil, "test")
		if len(result) != 0 {
			t.Error("Expected empty result")
		}
	})

	t.Run("Custom on empty", func(t *testing.T) {
		result := jar.Custom()
		if len(result) != 0 {
			t.Error("Expected empty result")
		}
	})

	t.Run("System on empty", func(t *testing.T) {
		result := jar.System()
		if len(result) != 0 {
			t.Error("Expected empty result")
		}
	})

	t.Run("Values on empty", func(t *testing.T) {
		result := jar.Values()
		if len(result) != 0 {
			t.Error("Expected empty result")
		}
	})
}

func TestGemStruct(t *testing.T) {
	gem := Gem{
		ID:          "test-id",
		Name:        "Test Gem",
		Description: "A test description",
		Prompt:      "You are a test assistant",
		Predefined:  false,
	}

	if gem.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", gem.ID)
	}
	if gem.Name != "Test Gem" {
		t.Errorf("Expected Name 'Test Gem', got %s", gem.Name)
	}
	if gem.Description != "A test description" {
		t.Errorf("Expected Description 'A test description', got %s", gem.Description)
	}
	if gem.Prompt != "You are a test assistant" {
		t.Errorf("Expected Prompt 'You are a test assistant', got %s", gem.Prompt)
	}
	if gem.Predefined {
		t.Error("Expected Predefined false")
	}
}
