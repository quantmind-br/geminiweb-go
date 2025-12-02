package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/render"
)

// configView represents the current view in the config menu
type configView int

const (
	viewMain configView = iota
	viewModelSelect
	viewThemeSelect    // Markdown theme
	viewTUIThemeSelect // TUI color theme
)

// Menu item indices for main view
const (
	menuDefaultModel = iota
	menuVerbose
	menuAutoClose
	menuCopyToClipboard
	menuTheme    // Markdown theme
	menuTUITheme // TUI color theme
	menuExit
	menuItemCount
)

// feedbackClearMsg is sent to clear feedback messages
type feedbackClearMsg struct{}

// ConfigModel represents the config TUI state
type ConfigModel struct {
	config       config.Config
	configDir    string
	cookiesPath  string
	cookiesExist bool

	// Navigation
	view           configView
	cursor         int
	modelCursor    int
	themeCursor    int // Markdown theme cursor
	tuiThemeCursor int // TUI theme cursor

	// Feedback
	feedback        string
	feedbackTimeout time.Duration

	// Dimensions
	width  int
	height int
	ready  bool
}

// NewConfigModel creates a new config TUI model
func NewConfigModel() ConfigModel {
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	configDir, _ := config.GetConfigDir()
	cookiesPath, _ := config.GetCookiesPath()

	cookiesExist := false
	if _, err := os.Stat(cookiesPath); err == nil {
		cookiesExist = true
	}

	// Find current model index
	modelCursor := 0
	models := config.AvailableModels()
	for i, m := range models {
		if m == cfg.DefaultModel {
			modelCursor = i
			break
		}
	}

	// Find current markdown theme index
	themeCursor := 0
	themes := render.ThemeNames()
	currentTheme := cfg.Markdown.Style
	if currentTheme == "" {
		currentTheme = render.ThemeDark
	}
	for i, t := range themes {
		if t == currentTheme {
			themeCursor = i
			break
		}
	}

	// Find current TUI theme index
	tuiThemeCursor := 0
	tuiThemes := render.TUIThemeNames()
	currentTUITheme := cfg.TUITheme
	if currentTUITheme == "" {
		currentTUITheme = "tokyonight"
	}
	for i, t := range tuiThemes {
		if t == currentTUITheme {
			tuiThemeCursor = i
			break
		}
	}

	// Apply the configured TUI theme at startup
	if currentTUITheme != "" {
		render.SetTUITheme(currentTUITheme)
		UpdateTheme()
	}

	return ConfigModel{
		config:          cfg,
		configDir:       configDir,
		cookiesPath:     cookiesPath,
		cookiesExist:    cookiesExist,
		view:            viewMain,
		cursor:          0,
		modelCursor:     modelCursor,
		themeCursor:     themeCursor,
		tuiThemeCursor:  tuiThemeCursor,
		feedbackTimeout: 2 * time.Second,
	}
}

// Init initializes the model
func (m ConfigModel) Init() tea.Cmd {
	return nil
}

// clearFeedback returns a command that clears the feedback message after a delay
func clearFeedback(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return feedbackClearMsg{}
	})
}

// Update handles messages and updates the model
func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case feedbackClearMsg:
		m.feedback = ""

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.view == viewModelSelect || m.view == viewThemeSelect || m.view == viewTUIThemeSelect {
				m.view = viewMain
			} else {
				return m, tea.Quit
			}

		case "up", "k":
			if m.view == viewMain {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = menuItemCount - 1
				}
			} else if m.view == viewModelSelect {
				m.modelCursor--
				if m.modelCursor < 0 {
					m.modelCursor = len(config.AvailableModels()) - 1
				}
			} else if m.view == viewThemeSelect {
				m.themeCursor--
				if m.themeCursor < 0 {
					m.themeCursor = len(render.ThemeNames()) - 1
				}
			} else if m.view == viewTUIThemeSelect {
				m.tuiThemeCursor--
				if m.tuiThemeCursor < 0 {
					m.tuiThemeCursor = len(render.TUIThemeNames()) - 1
				}
			}

		case "down", "j":
			if m.view == viewMain {
				m.cursor++
				if m.cursor >= menuItemCount {
					m.cursor = 0
				}
			} else if m.view == viewModelSelect {
				m.modelCursor++
				if m.modelCursor >= len(config.AvailableModels()) {
					m.modelCursor = 0
				}
			} else if m.view == viewThemeSelect {
				m.themeCursor++
				if m.themeCursor >= len(render.ThemeNames()) {
					m.themeCursor = 0
				}
			} else if m.view == viewTUIThemeSelect {
				m.tuiThemeCursor++
				if m.tuiThemeCursor >= len(render.TUIThemeNames()) {
					m.tuiThemeCursor = 0
				}
			}

		case "enter", " ":
			return m.handleSelect()
		}
	}

	return m, nil
}

// handleSelect handles menu item selection
func (m ConfigModel) handleSelect() (tea.Model, tea.Cmd) {
	if m.view == viewMain {
		switch m.cursor {
		case menuDefaultModel:
			m.view = viewModelSelect
			return m, nil

		case menuVerbose:
			m.config.Verbose = !m.config.Verbose
			if err := config.SaveConfig(m.config); err != nil {
				m.feedback = fmt.Sprintf("Error: %v", err)
			} else {
				state := "disabled"
				if m.config.Verbose {
					state = "enabled"
				}
				m.feedback = fmt.Sprintf("Verbose logging %s", state)
			}
			return m, clearFeedback(m.feedbackTimeout)

		case menuAutoClose:
			m.config.AutoClose = !m.config.AutoClose
			if err := config.SaveConfig(m.config); err != nil {
				m.feedback = fmt.Sprintf("Error: %v", err)
			} else {
				state := "disabled"
				if m.config.AutoClose {
					state = "enabled"
				}
				m.feedback = fmt.Sprintf("Auto-close %s", state)
			}
			return m, clearFeedback(m.feedbackTimeout)

		case menuCopyToClipboard:
			m.config.CopyToClipboard = !m.config.CopyToClipboard
			if err := config.SaveConfig(m.config); err != nil {
				m.feedback = fmt.Sprintf("Error: %v", err)
			} else {
				state := "disabled"
				if m.config.CopyToClipboard {
					state = "enabled"
				}
				m.feedback = fmt.Sprintf("Copy to clipboard %s", state)
			}
			return m, clearFeedback(m.feedbackTimeout)

		case menuTheme:
			m.view = viewThemeSelect
			return m, nil

		case menuTUITheme:
			m.view = viewTUIThemeSelect
			return m, nil

		case menuExit:
			return m, tea.Quit
		}
	} else if m.view == viewModelSelect {
		models := config.AvailableModels()
		m.config.DefaultModel = models[m.modelCursor]
		if err := config.SaveConfig(m.config); err != nil {
			m.feedback = fmt.Sprintf("Error: %v", err)
		} else {
			m.feedback = fmt.Sprintf("Model set to %s", m.config.DefaultModel)
		}
		m.view = viewMain
		return m, clearFeedback(m.feedbackTimeout)
	} else if m.view == viewThemeSelect {
		themes := render.ThemeNames()
		m.config.Markdown.Style = themes[m.themeCursor]
		if err := config.SaveConfig(m.config); err != nil {
			m.feedback = fmt.Sprintf("Error: %v", err)
		} else {
			m.feedback = fmt.Sprintf("Markdown theme set to %s", m.config.Markdown.Style)
		}
		m.view = viewMain
		return m, clearFeedback(m.feedbackTimeout)
	} else if m.view == viewTUIThemeSelect {
		tuiThemes := render.TUIThemeNames()
		selectedTheme := tuiThemes[m.tuiThemeCursor]
		m.config.TUITheme = selectedTheme

		// Apply the new TUI theme immediately
		render.SetTUITheme(selectedTheme)
		UpdateTheme()

		if err := config.SaveConfig(m.config); err != nil {
			m.feedback = fmt.Sprintf("Error: %v", err)
		} else {
			m.feedback = fmt.Sprintf("TUI theme set to %s", selectedTheme)
		}
		m.view = viewMain
		return m, clearFeedback(m.feedbackTimeout)
	}

	return m, nil
}

// View renders the TUI
func (m ConfigModel) View() string {
	if !m.ready {
		return loadingStyle.Render("  Initializing...")
	}

	var sections []string
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	// ═══════════════════════════════════════════════════════════════
	// HEADER
	// ═══════════════════════════════════════════════════════════════
	headerContent := configTitleStyle.Render("✦ Configuration")
	header := configHeaderStyle.Width(contentWidth).Render(headerContent)
	sections = append(sections, header)

	// ═══════════════════════════════════════════════════════════════
	// PATHS PANEL
	// ═══════════════════════════════════════════════════════════════
	pathsTitle := configSectionTitleStyle.Render("📁 Paths")

	configPath := configPathStyle.Render(m.configDir + "/config.json")
	cookiesPath := configPathStyle.Render(m.cookiesPath)

	var cookiesStatus string
	if m.cookiesExist {
		cookiesStatus = configStatusOkStyle.Render("✓ exists")
	} else {
		cookiesStatus = configStatusErrorStyle.Render("✗ not found")
	}

	pathsContent := lipgloss.JoinVertical(lipgloss.Left,
		pathsTitle,
		fmt.Sprintf("   Config:  %s", configPath),
		fmt.Sprintf("   Cookies: %s  %s", cookiesPath, cookiesStatus),
	)
	pathsPanel := configPanelStyle.Width(contentWidth).Render(pathsContent)
	sections = append(sections, pathsPanel)

	// ═══════════════════════════════════════════════════════════════
	// SETTINGS/MENU PANEL
	// ═══════════════════════════════════════════════════════════════
	var settingsContent string
	switch m.view {
	case viewMain:
		settingsContent = m.renderMainMenu(contentWidth)
	case viewModelSelect:
		settingsContent = m.renderModelSelect(contentWidth)
	case viewThemeSelect:
		settingsContent = m.renderThemeSelect(contentWidth)
	case viewTUIThemeSelect:
		settingsContent = m.renderTUIThemeSelect(contentWidth)
	}

	settingsPanel := configPanelStyle.Width(contentWidth).Render(settingsContent)
	sections = append(sections, settingsPanel)

	// ═══════════════════════════════════════════════════════════════
	// FEEDBACK MESSAGE
	// ═══════════════════════════════════════════════════════════════
	if m.feedback != "" {
		feedbackMsg := configFeedbackStyle.Render("✓ " + m.feedback)
		sections = append(sections, feedbackMsg)
	}

	// ═══════════════════════════════════════════════════════════════
	// STATUS BAR
	// ═══════════════════════════════════════════════════════════════
	statusBar := m.renderStatusBar(contentWidth)
	sections = append(sections, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderMainMenu renders the main settings menu
func (m ConfigModel) renderMainMenu(width int) string {
	title := configSectionTitleStyle.Render("⚙ Settings")

	var items []string

	// Default Model
	cursor := "  "
	style := configMenuItemStyle
	if m.cursor == menuDefaultModel {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	modelValue := configValueStyle.Render(m.config.DefaultModel)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Default Model"),
		strings.Repeat(" ", 8),
		modelValue,
	))

	// Verbose
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuVerbose {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	verboseValue := m.renderBoolValue(m.config.Verbose)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Verbose Logging"),
		strings.Repeat(" ", 5),
		verboseValue,
	))

	// Auto Close
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuAutoClose {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	autoCloseValue := m.renderBoolValue(m.config.AutoClose)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Auto Close"),
		strings.Repeat(" ", 10),
		autoCloseValue,
	))

	// Copy to Clipboard
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuCopyToClipboard {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	clipboardValue := m.renderBoolValue(m.config.CopyToClipboard)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Copy to Clipboard"),
		strings.Repeat(" ", 3),
		clipboardValue,
	))

	// Markdown Theme
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuTheme {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	currentTheme := m.config.Markdown.Style
	if currentTheme == "" {
		currentTheme = render.ThemeDark
	}
	themeValue := configValueStyle.Render(currentTheme)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Markdown Theme"),
		strings.Repeat(" ", 6),
		themeValue,
	))

	// TUI Theme
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuTUITheme {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	currentTUITheme := m.config.TUITheme
	if currentTUITheme == "" {
		currentTUITheme = "tokyonight"
	}
	tuiThemeValue := configValueStyle.Render(currentTUITheme)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("TUI Theme"),
		strings.Repeat(" ", 11),
		tuiThemeValue,
	))

	// Separator
	items = append(items, "")

	// Exit
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuExit {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	items = append(items, cursor+style.Render("Exit"))

	return lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, items...)...,
	)
}

// renderModelSelect renders the model selection sub-menu
func (m ConfigModel) renderModelSelect(width int) string {
	title := configSectionTitleStyle.Render("🤖 Select Model")

	models := config.AvailableModels()
	var items []string

	for i, model := range models {
		cursor := "  "
		style := configMenuItemStyle
		if m.modelCursor == i {
			cursor = configCursorStyle.Render("▸ ")
			style = configMenuSelectedStyle
		}

		current := ""
		if model == m.config.DefaultModel {
			current = configStatusOkStyle.Render(" (current)")
		}

		items = append(items, cursor+style.Render(model)+current)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, items...)...,
	)
}


// renderThemeSelect renders the markdown theme selection sub-menu
func (m ConfigModel) renderThemeSelect(width int) string {
	title := configSectionTitleStyle.Render("🎨 Select Markdown Theme")

	themes := render.AvailableThemes()
	var items []string

	currentTheme := m.config.Markdown.Style
	if currentTheme == "" {
		currentTheme = render.ThemeDark
	}

	for i, theme := range themes {
		cursor := "  "
		style := configMenuItemStyle
		if m.themeCursor == i {
			cursor = configCursorStyle.Render("▸ ")
			style = configMenuSelectedStyle
		}

		current := ""
		if theme.Name == currentTheme {
			current = configStatusOkStyle.Render(" (current)")
		}

		// Format: "theme-name - description"
		themeText := fmt.Sprintf("%s - %s", theme.Name, theme.Description)
		items = append(items, cursor+style.Render(themeText)+current)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, items...)...,
	)
}

// renderTUIThemeSelect renders the TUI color theme selection sub-menu
func (m ConfigModel) renderTUIThemeSelect(width int) string {
	title := configSectionTitleStyle.Render("🎨 Select TUI Theme")

	themes := render.AvailableTUIThemes()
	var items []string

	currentTUITheme := m.config.TUITheme
	if currentTUITheme == "" {
		currentTUITheme = "tokyonight"
	}

	for i, theme := range themes {
		cursor := "  "
		style := configMenuItemStyle
		if m.tuiThemeCursor == i {
			cursor = configCursorStyle.Render("▸ ")
			style = configMenuSelectedStyle
		}

		current := ""
		if theme.Name == currentTUITheme {
			current = configStatusOkStyle.Render(" (current)")
		}

		// Format: "theme-name - description"
		themeText := fmt.Sprintf("%s - %s", theme.Name, theme.Description)
		items = append(items, cursor+style.Render(themeText)+current)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, items...)...,
	)
}

// renderBoolValue renders a boolean value with appropriate styling
func (m ConfigModel) renderBoolValue(value bool) string {
	if value {
		return configEnabledStyle.Render("enabled")
	}
	return configDisabledStyle.Render("disabled")
}

// renderStatusBar renders the bottom status bar
func (m ConfigModel) renderStatusBar(width int) string {
	var shortcuts []struct {
		key  string
		desc string
	}

	if m.view == viewMain {
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"↑↓", "Navigate"},
			{"Enter", "Select"},
			{"Esc", "Exit"},
		}
	} else {
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"↑↓", "Navigate"},
			{"Enter", "Select"},
			{"Esc", "Back"},
		}
	}

	var items []string
	for _, s := range shortcuts {
		item := lipgloss.JoinHorizontal(
			lipgloss.Center,
			statusKeyStyle.Render(s.key),
			statusDescStyle.Render(" "+s.desc),
		)
		items = append(items, item)
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Center, strings.Join(items, "  │  "))
	return configStatusBarStyle.Width(width).Align(lipgloss.Center).Render(bar)
}

// RunConfig starts the config TUI
func RunConfig() error {
	m := NewConfigModel()

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
