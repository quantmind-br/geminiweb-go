// Package render provides TUI theme definitions for the terminal interface.
package render

import (
	"github.com/charmbracelet/lipgloss"
)

// TUITheme defines the color scheme for the TUI interface
type TUITheme struct {
	Name        string
	Description string

	// Base colors
	Background lipgloss.Color
	Surface    lipgloss.Color
	Border     lipgloss.Color

	// Accent colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color

	// Text colors
	Text     lipgloss.Color
	TextDim  lipgloss.Color
	TextMute lipgloss.Color
}

// Built-in TUI themes
var (
	// TokyoNightTheme is the default dark theme based on Tokyo Night color scheme
	TokyoNightTheme = TUITheme{
		Name:        "tokyonight",
		Description: "Tokyo Night - Dark theme with blue accents",

		Background: lipgloss.Color("#1a1b26"),
		Surface:    lipgloss.Color("#24283b"),
		Border:     lipgloss.Color("#414868"),

		Primary:   lipgloss.Color("#7aa2f7"),
		Secondary: lipgloss.Color("#9ece6a"),
		Accent:    lipgloss.Color("#bb9af7"),
		Warning:   lipgloss.Color("#e0af68"),
		Error:     lipgloss.Color("#f7768e"),

		Text:     lipgloss.Color("#c0caf5"),
		TextDim:  lipgloss.Color("#565f89"),
		TextMute: lipgloss.Color("#3b4261"),
	}

	// CatppuccinMochaTheme is based on Catppuccin Mocha palette
	CatppuccinMochaTheme = TUITheme{
		Name:        "catppuccin",
		Description: "Catppuccin Mocha - Warm dark theme with pastel colors",

		Background: lipgloss.Color("#1e1e2e"),
		Surface:    lipgloss.Color("#313244"),
		Border:     lipgloss.Color("#45475a"),

		Primary:   lipgloss.Color("#89b4fa"), // Blue
		Secondary: lipgloss.Color("#a6e3a1"), // Green
		Accent:    lipgloss.Color("#cba6f7"), // Mauve
		Warning:   lipgloss.Color("#f9e2af"), // Yellow
		Error:     lipgloss.Color("#f38ba8"), // Red

		Text:     lipgloss.Color("#cdd6f4"),
		TextDim:  lipgloss.Color("#6c7086"),
		TextMute: lipgloss.Color("#45475a"),
	}

	// NordTheme is based on the Nord color palette
	NordTheme = TUITheme{
		Name:        "nord",
		Description: "Nord - Arctic-inspired theme with cool tones",

		Background: lipgloss.Color("#2e3440"),
		Surface:    lipgloss.Color("#3b4252"),
		Border:     lipgloss.Color("#4c566a"),

		Primary:   lipgloss.Color("#88c0d0"), // Frost
		Secondary: lipgloss.Color("#a3be8c"), // Aurora green
		Accent:    lipgloss.Color("#b48ead"), // Aurora purple
		Warning:   lipgloss.Color("#ebcb8b"), // Aurora yellow
		Error:     lipgloss.Color("#bf616a"), // Aurora red

		Text:     lipgloss.Color("#eceff4"),
		TextDim:  lipgloss.Color("#7b88a1"),
		TextMute: lipgloss.Color("#4c566a"),
	}

	// DraculaTheme is based on the Dracula color palette
	DraculaTheme = TUITheme{
		Name:        "dracula",
		Description: "Dracula - Dark theme with vibrant colors",

		Background: lipgloss.Color("#282a36"),
		Surface:    lipgloss.Color("#44475a"),
		Border:     lipgloss.Color("#6272a4"),

		Primary:   lipgloss.Color("#8be9fd"), // Cyan
		Secondary: lipgloss.Color("#50fa7b"), // Green
		Accent:    lipgloss.Color("#ff79c6"), // Pink
		Warning:   lipgloss.Color("#f1fa8c"), // Yellow
		Error:     lipgloss.Color("#ff5555"), // Red

		Text:     lipgloss.Color("#f8f8f2"),
		TextDim:  lipgloss.Color("#6272a4"),
		TextMute: lipgloss.Color("#44475a"),
	}
)

// currentTUITheme holds the currently active TUI theme
var currentTUITheme = TokyoNightTheme

// GetTUITheme returns the currently active TUI theme
func GetTUITheme() TUITheme {
	return currentTUITheme
}

// SetTUITheme sets the active TUI theme by name
func SetTUITheme(name string) bool {
	theme, ok := GetTUIThemeByName(name)
	if ok {
		currentTUITheme = theme
		return true
	}
	return false
}

// GetTUIThemeByName returns a TUI theme by its name
func GetTUIThemeByName(name string) (TUITheme, bool) {
	switch name {
	case "tokyonight":
		return TokyoNightTheme, true
	case "catppuccin":
		return CatppuccinMochaTheme, true
	case "nord":
		return NordTheme, true
	case "dracula":
		return DraculaTheme, true
	default:
		return TUITheme{}, false
	}
}

// AvailableTUIThemes returns a list of all available TUI themes
func AvailableTUIThemes() []TUITheme {
	return []TUITheme{
		TokyoNightTheme,
		CatppuccinMochaTheme,
		NordTheme,
		DraculaTheme,
	}
}

// TUIThemeNames returns just the theme names for selection
func TUIThemeNames() []string {
	themes := AvailableTUIThemes()
	names := make([]string, len(themes))
	for i, t := range themes {
		names[i] = t.Name
	}
	return names
}
