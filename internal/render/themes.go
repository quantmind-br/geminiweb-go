package render

import (
	_ "embed"
	"os"
	"path/filepath"
	"sync"
)

//go:embed themes/dark.json
var darkTheme []byte

//go:embed themes/tokyonight.json
var tokyoNightTheme []byte

//go:embed themes/catppuccin.json
var catppuccinTheme []byte

// BuiltinTheme represents a built-in theme name
const (
	ThemeDark       = "dark"
	ThemeLight      = "light"
	ThemeTokyoNight = "tokyonight"
	ThemeCatppuccin = "catppuccin"
)

// themeFileCache stores paths to written theme files
var (
	themeFileMu    sync.RWMutex
	themeFileCache = make(map[string]string)
)

// GetBuiltinTheme returns the content of a built-in theme by name.
// Returns nil and false if the theme is not a built-in theme.
func GetBuiltinTheme(name string) ([]byte, bool) {
	switch name {
	case ThemeDark:
		return darkTheme, true
	case ThemeTokyoNight:
		return tokyoNightTheme, true
	case ThemeCatppuccin:
		return catppuccinTheme, true
	default:
		return nil, false
	}
}

// IsBuiltinStyle returns true if the style is a built-in style
// (either glamour built-in or our custom built-in themes).
func IsBuiltinStyle(style string) bool {
	switch style {
	case ThemeDark, ThemeLight, "dracula", "notty", "ascii":
		return true
	case ThemeTokyoNight, ThemeCatppuccin:
		return true
	default:
		return false
	}
}

// WriteThemeToTempFile writes a built-in theme to a temporary file
// and returns the file path. Thread-safe and caches file paths.
func WriteThemeToTempFile(name string) (string, error) {
	content, ok := GetBuiltinTheme(name)
	if !ok {
		return "", nil // Not a built-in theme, return empty string
	}

	// Check if already written and file still exists
	themeFileMu.RLock()
	if path, ok := themeFileCache[name]; ok {
		if _, err := os.Stat(path); err == nil {
			themeFileMu.RUnlock()
			return path, nil
		}
		// File was deleted, need to rewrite
	}
	themeFileMu.RUnlock()

	// Write the theme file
	themeFileMu.Lock()
	defer themeFileMu.Unlock()

	// Double-check after acquiring write lock
	if path, ok := themeFileCache[name]; ok {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "geminiweb-theme-"+name+".json")

	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		return "", err
	}

	themeFileCache[name] = tmpFile
	return tmpFile, nil
}

// ClearThemeCache clears the theme file cache (useful for testing).
func ClearThemeCache() {
	themeFileMu.Lock()
	defer themeFileMu.Unlock()
	themeFileCache = make(map[string]string)
}

// ThemeInfo contains information about a theme for display purposes.
type ThemeInfo struct {
	Name        string
	Description string
}

// AvailableThemes returns a list of all available themes (built-in and glamour styles).
func AvailableThemes() []ThemeInfo {
	return []ThemeInfo{
		{Name: ThemeDark, Description: "Dark theme (default)"},
		{Name: ThemeTokyoNight, Description: "Tokyo Night color scheme"},
		{Name: ThemeCatppuccin, Description: "Catppuccin Mocha color scheme"},
		{Name: ThemeLight, Description: "Light theme for bright terminals"},
		{Name: "dracula", Description: "Dracula color scheme"},
		{Name: "notty", Description: "Plain text (no styling)"},
		{Name: "ascii", Description: "ASCII-only output"},
	}
}

// ThemeNames returns just the theme names for selection.
func ThemeNames() []string {
	themes := AvailableThemes()
	names := make([]string, len(themes))
	for i, t := range themes {
		names[i] = t.Name
	}
	return names
}
