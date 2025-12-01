// Package tui provides the terminal user interface for geminiweb.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Modern color palette
var (
	// Base colors
	colorBackground = lipgloss.Color("#1a1b26") // Dark background
	colorSurface    = lipgloss.Color("#24283b") // Slightly lighter surface
	colorBorder     = lipgloss.Color("#414868") // Subtle border

	// Accent colors
	colorPrimary   = lipgloss.Color("#7aa2f7") // Soft blue
	colorSecondary = lipgloss.Color("#9ece6a") // Soft green
	colorAccent    = lipgloss.Color("#bb9af7") // Purple accent
	colorWarning   = lipgloss.Color("#e0af68") // Warm yellow
	colorError     = lipgloss.Color("#f7768e") // Soft red

	// Text colors
	colorText     = lipgloss.Color("#c0caf5") // Main text
	colorTextDim  = lipgloss.Color("#565f89") // Dimmed text
	colorTextMute = lipgloss.Color("#3b4261") // Very dim text
)

// Header panel style
var headerStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(0, 2).
	MarginBottom(1)

// Title style for header
var titleStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Subtitle/model name style
var subtitleStyle = lipgloss.NewStyle().
	Foreground(colorAccent)

// Hint text style
var hintStyle = lipgloss.NewStyle().
	Foreground(colorTextDim).
	Italic(true)

// Messages area panel
var messagesAreaStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(1, 1)

// User message bubble
var userBubbleStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorSecondary).
	Foreground(colorText).
	Padding(0, 1).
	MarginTop(1).
	MarginBottom(1)

// User label style
var userLabelStyle = lipgloss.NewStyle().
	Foreground(colorSecondary).
	Bold(true).
	MarginBottom(0)

// Assistant message bubble
var assistantBubbleStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorPrimary).
	Foreground(colorText).
	Padding(0, 1).
	MarginTop(1).
	MarginBottom(1)

// Assistant label style
var assistantLabelStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true).
	MarginBottom(0)

// Thoughts panel style
var thoughtsStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(colorTextDim).
	BorderLeft(true).
	Foreground(colorTextDim).
	PaddingLeft(1).
	MarginLeft(1).
	Italic(true)

// Input area panel
var inputPanelStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorSecondary).
	Padding(0, 1).
	MarginTop(1)

// Input label style
var inputLabelStyle = lipgloss.NewStyle().
	Foreground(colorSecondary).
	Bold(true)

// Loading/spinner style
var loadingStyle = lipgloss.NewStyle().
	Foreground(colorAccent).
	Bold(true)

// Gradient colors for animated spinner
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

// Status bar style
var statusBarStyle = lipgloss.NewStyle().
	Foreground(colorTextDim).
	MarginTop(1)

// Status bar key style
var statusKeyStyle = lipgloss.NewStyle().
	Foreground(colorText).
	Background(colorSurface).
	Padding(0, 1)

// Status bar description style
var statusDescStyle = lipgloss.NewStyle().
	Foreground(colorTextDim)

// Error style
var errorStyle = lipgloss.NewStyle().
	Foreground(colorError).
	Bold(true)

// Welcome message style
var welcomeStyle = lipgloss.NewStyle().
	Foreground(colorTextDim).
	Align(lipgloss.Center)

// Welcome title style
var welcomeTitleStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true).
	Align(lipgloss.Center)

// Welcome icon style
var welcomeIconStyle = lipgloss.NewStyle().
	Foreground(colorAccent).
	Align(lipgloss.Center)

// ═══════════════════════════════════════════════════════════════════════════════
// CONFIG MENU STYLES
// ═══════════════════════════════════════════════════════════════════════════════

// Config header style
var configHeaderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(0, 2).
	MarginBottom(1)

// Config title style
var configTitleStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Config panel style (for paths and settings)
var configPanelStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(1, 2).
	MarginBottom(1)

// Config section title style
var configSectionTitleStyle = lipgloss.NewStyle().
	Foreground(colorAccent).
	Bold(true).
	MarginBottom(1)

// Config menu item style (not selected)
var configMenuItemStyle = lipgloss.NewStyle().
	Foreground(colorText).
	PaddingLeft(2)

// Config menu item style (selected/highlighted)
var configMenuSelectedStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Config cursor style
var configCursorStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Config value style (for settings values)
var configValueStyle = lipgloss.NewStyle().
	Foreground(colorAccent)

// Config enabled value style
var configEnabledStyle = lipgloss.NewStyle().
	Foreground(colorSecondary)

// Config disabled value style
var configDisabledStyle = lipgloss.NewStyle().
	Foreground(colorTextDim)

// Config path style
var configPathStyle = lipgloss.NewStyle().
	Foreground(colorTextDim)

// Config status ok style
var configStatusOkStyle = lipgloss.NewStyle().
	Foreground(colorSecondary)

// Config status error style
var configStatusErrorStyle = lipgloss.NewStyle().
	Foreground(colorError)

// Config feedback message style
var configFeedbackStyle = lipgloss.NewStyle().
	Foreground(colorSecondary).
	Bold(true).
	MarginTop(1)

// Config status bar style
var configStatusBarStyle = lipgloss.NewStyle().
	Foreground(colorTextDim).
	MarginTop(1)
