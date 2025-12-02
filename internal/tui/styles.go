// Package tui provides the terminal user interface for geminiweb.
package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/render"
)

// Color variables (updated from theme)
var (
	// Base colors
	colorBackground lipgloss.Color
	colorSurface    lipgloss.Color
	colorBorder     lipgloss.Color

	// Accent colors
	colorPrimary   lipgloss.Color
	colorSecondary lipgloss.Color
	colorAccent    lipgloss.Color
	colorWarning   lipgloss.Color
	colorError     lipgloss.Color

	// Text colors
	colorText     lipgloss.Color
	colorTextDim  lipgloss.Color
	colorTextMute lipgloss.Color
)

// Style variables (rebuilt when theme changes)
var (
	// Header panel style
	headerStyle lipgloss.Style

	// Title style for header
	titleStyle lipgloss.Style

	// Subtitle/model name style
	subtitleStyle lipgloss.Style

	// Hint text style
	hintStyle lipgloss.Style

	// Messages area panel
	messagesAreaStyle lipgloss.Style

	// User message bubble
	userBubbleStyle lipgloss.Style

	// User label style
	userLabelStyle lipgloss.Style

	// Assistant message bubble
	assistantBubbleStyle lipgloss.Style

	// Assistant label style
	assistantLabelStyle lipgloss.Style

	// Thoughts panel style
	thoughtsStyle lipgloss.Style

	// Image section styles
	imageSectionStyle       lipgloss.Style
	imageSectionHeaderStyle lipgloss.Style
	imageLinkStyle          lipgloss.Style
	imageTitleStyle         lipgloss.Style

	// Input area panel
	inputPanelStyle lipgloss.Style

	// Input label style
	inputLabelStyle lipgloss.Style

	// Loading/spinner style
	loadingStyle lipgloss.Style

	// Status bar styles
	statusBarStyle  lipgloss.Style
	statusKeyStyle  lipgloss.Style
	statusDescStyle lipgloss.Style

	// Error style
	errorStyle lipgloss.Style

	// Welcome styles
	welcomeStyle      lipgloss.Style
	welcomeTitleStyle lipgloss.Style
	welcomeIconStyle  lipgloss.Style

	// Config menu styles
	configHeaderStyle       lipgloss.Style
	configTitleStyle        lipgloss.Style
	configPanelStyle        lipgloss.Style
	configSectionTitleStyle lipgloss.Style
	configMenuItemStyle     lipgloss.Style
	configMenuSelectedStyle lipgloss.Style
	configCursorStyle       lipgloss.Style
	configValueStyle        lipgloss.Style
	configEnabledStyle      lipgloss.Style
	configDisabledStyle     lipgloss.Style
	configPathStyle         lipgloss.Style
	configStatusOkStyle     lipgloss.Style
	configStatusErrorStyle  lipgloss.Style
	configFeedbackStyle     lipgloss.Style
	configStatusBarStyle    lipgloss.Style
)

// Gradient colors for animated spinner (fixed colors)
var gradientColors = []lipgloss.Color{
	lipgloss.Color("#ff6b6b"), // Red
	lipgloss.Color("#feca57"), // Yellow
	lipgloss.Color("#48dbfb"), // Cyan
	lipgloss.Color("#ff9ff3"), // Pink
	lipgloss.Color("#54a0ff"), // Blue
	lipgloss.Color("#5f27cd"), // Purple
	lipgloss.Color("#00d2d3"), // Teal
	lipgloss.Color("#1dd1a1"), // Green
}

// init loads the default theme on package initialization
func init() {
	UpdateTheme()
}

// UpdateTheme refreshes all styles based on the current TUI theme
func UpdateTheme() {
	theme := render.GetTUITheme()

	// Update color variables
	colorBackground = theme.Background
	colorSurface = theme.Surface
	colorBorder = theme.Border
	colorPrimary = theme.Primary
	colorSecondary = theme.Secondary
	colorAccent = theme.Accent
	colorWarning = theme.Warning
	colorError = theme.Error
	colorText = theme.Text
	colorTextDim = theme.TextDim
	colorTextMute = theme.TextMute

	// Rebuild all styles with new colors
	rebuildStyles()
}

// rebuildStyles creates all lipgloss styles with current color values
func rebuildStyles() {
	// Header panel style
	headerStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 2).
		MarginBottom(1)

	// Title style for header
	titleStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	// Subtitle/model name style
	subtitleStyle = lipgloss.NewStyle().
		Foreground(colorAccent)

	// Hint text style
	hintStyle = lipgloss.NewStyle().
		Foreground(colorTextDim).
		Italic(true)

	// Messages area panel
	messagesAreaStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(1, 1)

	// User message bubble
	userBubbleStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Foreground(colorText).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	// User label style
	userLabelStyle = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		MarginBottom(0)

	// Assistant message bubble
	assistantBubbleStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Foreground(colorText).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	// Assistant label style
	assistantLabelStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		MarginBottom(0)

	// Thoughts panel style
	thoughtsStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorTextDim).
		BorderLeft(true).
		Foreground(colorTextDim).
		PaddingLeft(1).
		MarginLeft(1).
		Italic(true)

	// Image section styles
	imageSectionStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorWarning).
		BorderLeft(true).
		PaddingLeft(1).
		MarginLeft(1).
		MarginTop(1)

	imageSectionHeaderStyle = lipgloss.NewStyle().
		Foreground(colorWarning).
		Bold(true)

	imageLinkStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Underline(true)

	imageTitleStyle = lipgloss.NewStyle().
		Foreground(colorText)

	// Input area panel
	inputPanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Padding(0, 1).
		MarginTop(1)

	// Input label style
	inputLabelStyle = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true)

	// Loading/spinner style
	loadingStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
		Foreground(colorTextDim).
		MarginTop(1)

	// Status bar key style
	statusKeyStyle = lipgloss.NewStyle().
		Foreground(colorText).
		Background(colorSurface).
		Padding(0, 1)

	// Status bar description style
	statusDescStyle = lipgloss.NewStyle().
		Foreground(colorTextDim)

	// Error style
	errorStyle = lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true)

	// Welcome message style
	welcomeStyle = lipgloss.NewStyle().
		Foreground(colorTextDim).
		Align(lipgloss.Center)

	// Welcome title style
	welcomeTitleStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Align(lipgloss.Center)

	// Welcome icon style
	welcomeIconStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Align(lipgloss.Center)

	// ═══════════════════════════════════════════════════════════════════════════════
	// CONFIG MENU STYLES
	// ═══════════════════════════════════════════════════════════════════════════════

	// Config header style
	configHeaderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 2).
		MarginBottom(1)

	// Config title style
	configTitleStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	// Config panel style (for paths and settings)
	configPanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(1, 2).
		MarginBottom(1)

	// Config section title style
	configSectionTitleStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		MarginBottom(1)

	// Config menu item style (not selected)
	configMenuItemStyle = lipgloss.NewStyle().
		Foreground(colorText).
		PaddingLeft(2)

	// Config menu item style (selected/highlighted)
	configMenuSelectedStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	// Config cursor style
	configCursorStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	// Config value style (for settings values)
	configValueStyle = lipgloss.NewStyle().
		Foreground(colorAccent)

	// Config enabled value style
	configEnabledStyle = lipgloss.NewStyle().
		Foreground(colorSecondary)

	// Config disabled value style
	configDisabledStyle = lipgloss.NewStyle().
		Foreground(colorTextDim)

	// Config path style
	configPathStyle = lipgloss.NewStyle().
		Foreground(colorTextDim)

	// Config status ok style
	configStatusOkStyle = lipgloss.NewStyle().
		Foreground(colorSecondary)

	// Config status error style
	configStatusErrorStyle = lipgloss.NewStyle().
		Foreground(colorError)

	// Config feedback message style
	configFeedbackStyle = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		MarginTop(1)

	// Config status bar style
	configStatusBarStyle = lipgloss.NewStyle().
		Foreground(colorTextDim).
		MarginTop(1)
}

// GetCurrentThemeName returns the name of the current TUI theme
func GetCurrentThemeName() string {
	return render.GetTUITheme().Name
}
