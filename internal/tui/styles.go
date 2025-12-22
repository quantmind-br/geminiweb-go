// Package tui provides the terminal user interface for geminiweb.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/render"
)

// Color variables (updated from theme)
var (
	// Base colors
	colorSurface lipgloss.Color
	colorBorder  lipgloss.Color

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

	// Tool message bubble
	toolBubbleStyle lipgloss.Style

	// Tool label style
	toolLabelStyle lipgloss.Style

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
		Foreground(colorTextDim)

	// Hint text style
	hintStyle = lipgloss.NewStyle().
		Foreground(colorTextMute).
		Italic(true)

	// Messages area panel
	messagesAreaStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(1)

	// User message bubble
	userBubbleStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Padding(0, 1).
		MarginLeft(4)

	// User label style
	userLabelStyle = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		MarginBottom(0).
		MarginLeft(4)

	// Assistant message bubble
	assistantBubbleStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Foreground(colorText).
		Padding(0, 1).
		MarginRight(4)

	// Assistant label style
	assistantLabelStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		MarginBottom(0)

	// Tool message bubble
	toolBubbleStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorTextDim).
		Foreground(colorTextDim).
		Padding(0, 1).
		MarginLeft(2).
		MarginRight(2)

	// Tool label style
	toolLabelStyle = lipgloss.NewStyle().
		Foreground(colorTextDim).
		Italic(true).
		MarginBottom(0).
		MarginLeft(2)

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
		MarginTop(1).
		MarginBottom(1)

	imageSectionHeaderStyle = lipgloss.NewStyle().
		Foreground(colorTextDim).
		Bold(true).
		MarginBottom(0)

	imageLinkStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Underline(true)

	imageTitleStyle = lipgloss.NewStyle().
		Foreground(colorText).
		Italic(true)

	// Input area panel
	inputPanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		MarginTop(1)

	// Input label style
	inputLabelStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		MarginRight(1)

	// Loading/spinner style
	loadingStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	// Status bar styles
	statusBarStyle = lipgloss.NewStyle().
		Foreground(colorTextMute).
		MarginTop(1)

	statusKeyStyle = lipgloss.NewStyle().
		Foreground(colorTextDim).
		Bold(true)

	statusDescStyle = lipgloss.NewStyle().
		Foreground(colorTextMute)

	// Error style
	errorStyle = lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true)

	// Welcome styles
	welcomeStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		MarginBottom(1).
		Align(lipgloss.Center)

	welcomeTitleStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Copy().
		MarginBottom(1)

	welcomeIconStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		MarginBottom(1)

	// Config menu styles
	configHeaderStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		MarginBottom(1).
		Align(lipgloss.Center)

	configTitleStyle = lipgloss.NewStyle().
		Foreground(colorText).
		Bold(true).
		MarginBottom(1).
		PaddingLeft(1)

	configPanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(1, 2)

	configSectionTitleStyle = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		MarginTop(1).
		MarginBottom(0)

	configMenuItemStyle = lipgloss.NewStyle().
		Foreground(colorText).
		PaddingLeft(2)

	configMenuSelectedStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		PaddingLeft(0).
		SetString("> ")

	configCursorStyle = lipgloss.NewStyle().
		Foreground(colorAccent)

	configValueStyle = lipgloss.NewStyle().
		Foreground(colorTextDim)

	configEnabledStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ece6a")) // Green

	configDisabledStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f7768e")) // Red

	configPathStyle = lipgloss.NewStyle().
		Foreground(colorTextMute).
		Italic(true)

	configStatusOkStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ece6a"))

	configStatusErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f7768e"))

	configFeedbackStyle = lipgloss.NewStyle().
		Foreground(colorTextDim).
		Italic(true).
		MarginTop(1)

	configStatusBarStyle = lipgloss.NewStyle().
		Foreground(colorTextMute).
		MarginTop(1).
		Align(lipgloss.Center)
}

// FormatError returns a styled error message with additional context.
// It extracts details from structured GeminiError types if available.
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	// Use colors from theme
	errStyle := lipgloss.NewStyle().Foreground(colorError)
	dimStyle := lipgloss.NewStyle().Foreground(colorTextDim)

	var sb strings.Builder
	sb.WriteString(errStyle.Render(fmt.Sprintf("✗ %v", err)))

	// Extract additional context from structured errors
	if status := errors.GetHTTPStatus(err); status > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  HTTP Status: %d", status)))
	}

	if code := errors.GetErrorCode(err); code != errors.ErrCodeUnknown {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  Error Code: %d (%s)", code, code.String())))
	}

	if endpoint := errors.GetEndpoint(err); endpoint != "" {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  Endpoint: %s", endpoint)))
	}

	// Show response body if available (contains detailed error info like blocking URLs)
	if body := errors.GetResponseBody(err); body != "" {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n\n  %s", strings.ReplaceAll(body, "\n", "\n  "))))
	} else {
		// Provide helpful hints based on error type only if no body
		switch {
		case errors.IsAuthError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: Try running 'geminiweb auto-login' to refresh your session"))
		case errors.IsRateLimitError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: You've hit the usage limit. Try again later or use a different model"))
		case errors.IsNetworkError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: Check your internet connection and try again"))
		case errors.IsTimeoutError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: Request timed out. Try again or check your connection"))
		case errors.IsUploadError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: File upload failed. Check the file exists and is accessible"))
		}
	}

	return sb.String()
}

// PrintError prints a styled error message to stderr.
func PrintError(err error) {
	if err == nil {
		return
	}
	fmt.Println(FormatError(err))
}