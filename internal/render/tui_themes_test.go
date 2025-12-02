package render

import (
	"testing"
)

func TestTUITheme_Structure(t *testing.T) {
	t.Run("TokyoNight theme has all colors defined", func(t *testing.T) {
		theme := TokyoNightTheme

		if theme.Name == "" {
			t.Error("theme name should not be empty")
		}
		if theme.Description == "" {
			t.Error("theme description should not be empty")
		}
		if string(theme.Background) == "" {
			t.Error("background color should not be empty")
		}
		if string(theme.Primary) == "" {
			t.Error("primary color should not be empty")
		}
		if string(theme.Text) == "" {
			t.Error("text color should not be empty")
		}
	})

	t.Run("all themes have required fields", func(t *testing.T) {
		themes := AvailableTUIThemes()

		for _, theme := range themes {
			if theme.Name == "" {
				t.Errorf("theme has empty name")
			}
			if theme.Description == "" {
				t.Errorf("theme %s has empty description", theme.Name)
			}
			if string(theme.Background) == "" {
				t.Errorf("theme %s has empty background color", theme.Name)
			}
			if string(theme.Surface) == "" {
				t.Errorf("theme %s has empty surface color", theme.Name)
			}
			if string(theme.Border) == "" {
				t.Errorf("theme %s has empty border color", theme.Name)
			}
			if string(theme.Primary) == "" {
				t.Errorf("theme %s has empty primary color", theme.Name)
			}
			if string(theme.Secondary) == "" {
				t.Errorf("theme %s has empty secondary color", theme.Name)
			}
			if string(theme.Accent) == "" {
				t.Errorf("theme %s has empty accent color", theme.Name)
			}
			if string(theme.Warning) == "" {
				t.Errorf("theme %s has empty warning color", theme.Name)
			}
			if string(theme.Error) == "" {
				t.Errorf("theme %s has empty error color", theme.Name)
			}
			if string(theme.Text) == "" {
				t.Errorf("theme %s has empty text color", theme.Name)
			}
			if string(theme.TextDim) == "" {
				t.Errorf("theme %s has empty textDim color", theme.Name)
			}
			if string(theme.TextMute) == "" {
				t.Errorf("theme %s has empty textMute color", theme.Name)
			}
		}
	})
}

func TestGetTUITheme(t *testing.T) {
	// Reset to default theme after test
	defer SetTUITheme("tokyonight")

	t.Run("returns current theme", func(t *testing.T) {
		theme := GetTUITheme()

		if theme.Name == "" {
			t.Error("current theme name should not be empty")
		}
	})

	t.Run("default theme is TokyoNight", func(t *testing.T) {
		// Reset to ensure default
		SetTUITheme("tokyonight")
		theme := GetTUITheme()

		if theme.Name != "tokyonight" {
			t.Errorf("expected default theme 'tokyonight', got '%s'", theme.Name)
		}
	})
}

func TestSetTUITheme(t *testing.T) {
	// Reset to default theme after test
	defer SetTUITheme("tokyonight")

	t.Run("sets valid theme", func(t *testing.T) {
		ok := SetTUITheme("catppuccin")

		if !ok {
			t.Error("should return true for valid theme")
		}

		theme := GetTUITheme()
		if theme.Name != "catppuccin" {
			t.Errorf("expected theme 'catppuccin', got '%s'", theme.Name)
		}
	})

	t.Run("returns false for invalid theme", func(t *testing.T) {
		// First set a known theme
		SetTUITheme("tokyonight")

		ok := SetTUITheme("nonexistent")

		if ok {
			t.Error("should return false for invalid theme")
		}

		// Theme should remain unchanged
		theme := GetTUITheme()
		if theme.Name != "tokyonight" {
			t.Errorf("theme should remain 'tokyonight', got '%s'", theme.Name)
		}
	})
}

func TestGetTUIThemeByName(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{"tokyonight", true},
		{"catppuccin", true},
		{"nord", true},
		{"dracula", true},
		{"nonexistent", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			theme, ok := GetTUIThemeByName(tc.name)

			if ok != tc.expected {
				t.Errorf("GetTUIThemeByName(%q) ok = %v, want %v", tc.name, ok, tc.expected)
			}

			if ok && theme.Name != tc.name {
				t.Errorf("GetTUIThemeByName(%q) returned theme with name %q", tc.name, theme.Name)
			}
		})
	}
}

func TestAvailableTUIThemes(t *testing.T) {
	t.Run("returns at least 4 themes", func(t *testing.T) {
		themes := AvailableTUIThemes()

		if len(themes) < 4 {
			t.Errorf("expected at least 4 themes, got %d", len(themes))
		}
	})

	t.Run("includes all known themes", func(t *testing.T) {
		themes := AvailableTUIThemes()
		expectedNames := []string{"tokyonight", "catppuccin", "nord", "dracula"}

		nameMap := make(map[string]bool)
		for _, theme := range themes {
			nameMap[theme.Name] = true
		}

		for _, name := range expectedNames {
			if !nameMap[name] {
				t.Errorf("expected theme %q not found in available themes", name)
			}
		}
	})
}

func TestTUIThemeNames(t *testing.T) {
	t.Run("returns theme names", func(t *testing.T) {
		names := TUIThemeNames()

		if len(names) == 0 {
			t.Error("should return at least one theme name")
		}

		// All names should be non-empty
		for i, name := range names {
			if name == "" {
				t.Errorf("theme name at index %d is empty", i)
			}
		}
	})

	t.Run("matches available themes", func(t *testing.T) {
		names := TUIThemeNames()
		themes := AvailableTUIThemes()

		if len(names) != len(themes) {
			t.Errorf("names count (%d) != themes count (%d)", len(names), len(themes))
		}

		for i, name := range names {
			if name != themes[i].Name {
				t.Errorf("name[%d] = %q, themes[%d].Name = %q", i, name, i, themes[i].Name)
			}
		}
	})
}

func TestThemeColors_AreValidHex(t *testing.T) {
	themes := AvailableTUIThemes()

	for _, theme := range themes {
		t.Run(theme.Name, func(t *testing.T) {
			colors := []struct {
				name  string
				color string
			}{
				{"Background", string(theme.Background)},
				{"Surface", string(theme.Surface)},
				{"Border", string(theme.Border)},
				{"Primary", string(theme.Primary)},
				{"Secondary", string(theme.Secondary)},
				{"Accent", string(theme.Accent)},
				{"Warning", string(theme.Warning)},
				{"Error", string(theme.Error)},
				{"Text", string(theme.Text)},
				{"TextDim", string(theme.TextDim)},
				{"TextMute", string(theme.TextMute)},
			}

			for _, c := range colors {
				// Check that colors start with # and have proper length
				if len(c.color) == 0 {
					t.Errorf("%s color is empty", c.name)
					continue
				}
				if c.color[0] != '#' {
					t.Errorf("%s color %q should start with #", c.name, c.color)
				}
				// Hex colors should be #RRGGBB (7 chars)
				if len(c.color) != 7 {
					t.Errorf("%s color %q has invalid length (expected 7, got %d)", c.name, c.color, len(c.color))
				}
			}
		})
	}
}
