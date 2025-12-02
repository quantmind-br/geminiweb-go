package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/render"
)

// gemsView represents the current view in the gems TUI
type gemsView int

const (
	gemsViewList gemsView = iota
	gemsViewDetails
)

// gemsLoadedMsg is sent when gems are loaded
type gemsLoadedMsg struct {
	gems *models.GemJar
	err  error
}

// gemsFeedbackClearMsg is sent to clear feedback messages
type gemsFeedbackClearMsg struct{}

// StartChatMsg signals that a chat should be started with the selected gem
type StartChatMsg struct {
	GemID   string
	GemName string
}

// GemsModel represents the gems TUI state
type GemsModel struct {
	client api.GeminiClientInterface

	// Data
	allGems      []*models.Gem // All gems (sorted)
	filteredGems []*models.Gem // Filtered gems based on search
	selectedGem  *models.Gem   // Currently selected gem for details view

	// Search
	searchInput textinput.Model
	searching   bool

	// Navigation
	view   gemsView
	cursor int

	// Feedback
	feedback        string
	feedbackTimeout time.Duration

	// State
	loading bool
	err     error

	// Dimensions
	width  int
	height int
	ready  bool

	// Include hidden gems
	includeHidden bool

	// Chat transition
	startChatGemID   string // Set when user presses 'c' to start chat
	startChatGemName string // Name of the gem to start chat with
}

// NewGemsModel creates a new gems TUI model
func NewGemsModel(client api.GeminiClientInterface, includeHidden bool) GemsModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter gems..."
	ti.CharLimit = 100
	ti.Width = 40

	return GemsModel{
		client:          client,
		searchInput:     ti,
		view:            gemsViewList,
		feedbackTimeout: 2 * time.Second,
		loading:         true,
		includeHidden:   includeHidden,
	}
}

// Init initializes the model and starts loading gems
func (m GemsModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadGems(),
		textinput.Blink,
	)
}

// loadGems returns a command that loads gems from the API
func (m GemsModel) loadGems() tea.Cmd {
	return func() tea.Msg {
		gems, err := m.client.FetchGems(m.includeHidden)
		if err != nil {
			return gemsLoadedMsg{err: err}
		}
		return gemsLoadedMsg{gems: gems}
	}
}

// Update handles messages and updates the model
func (m GemsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.searchInput.Width = m.width - 10
		if m.searchInput.Width < 20 {
			m.searchInput.Width = 20
		}

	case gemsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else if msg.gems != nil {
			m.allGems = sortGems(msg.gems.Values())
			m.filteredGems = m.allGems
		}

	case gemsFeedbackClearMsg:
		m.feedback = ""

	case tea.KeyMsg:
		// Handle key messages based on current state
		if m.searching {
			return m.handleSearchInput(msg)
		}
		return m.handleKeyMsg(msg)
	}

	// Update search input if searching
	if m.searching {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles key messages when not in search mode
func (m GemsModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "q":
		if m.view == gemsViewList {
			return m, tea.Quit
		}
		// In details view, go back to list
		m.view = gemsViewList
		return m, nil

	case "esc":
		if m.view == gemsViewDetails {
			m.view = gemsViewList
			return m, nil
		}
		return m, tea.Quit

	case "/":
		// Enter search mode
		m.searching = true
		m.searchInput.Focus()
		return m, textinput.Blink

	case "up", "k":
		if m.view == gemsViewList {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = max(0, len(m.filteredGems)-1)
			}
		}

	case "down", "j":
		if m.view == gemsViewList {
			m.cursor++
			if m.cursor >= len(m.filteredGems) {
				m.cursor = 0
			}
		}

	case "enter":
		if m.view == gemsViewList && len(m.filteredGems) > 0 {
			m.selectedGem = m.filteredGems[m.cursor]
			m.view = gemsViewDetails
		}

	case "c":
		// Start chat with selected gem
		if m.view == gemsViewList && len(m.filteredGems) > 0 {
			gem := m.filteredGems[m.cursor]
			m.startChatGemID = gem.ID
			m.startChatGemName = gem.Name
			return m, tea.Quit
		} else if m.view == gemsViewDetails && m.selectedGem != nil {
			m.startChatGemID = m.selectedGem.ID
			m.startChatGemName = m.selectedGem.Name
			return m, tea.Quit
		}

	case "y":
		// Copy ID to clipboard
		if m.view == gemsViewDetails && m.selectedGem != nil {
			return m.copyIDToClipboard()
		} else if m.view == gemsViewList && len(m.filteredGems) > 0 {
			m.selectedGem = m.filteredGems[m.cursor]
			return m.copyIDToClipboard()
		}

	case "home", "g":
		if m.view == gemsViewList {
			m.cursor = 0
		}

	case "end", "G":
		if m.view == gemsViewList {
			m.cursor = max(0, len(m.filteredGems)-1)
		}
	}

	return m, nil
}

// handleSearchInput handles key messages in search mode
func (m GemsModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Exit search mode
		m.searching = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.filteredGems = m.allGems
		m.cursor = 0
		return m, nil

	case "enter":
		// Confirm search and exit search mode
		m.searching = false
		m.searchInput.Blur()
		return m, nil

	default:
		// Update search input and filter
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.filterGems()
		return m, cmd
	}
}

// filterGems filters gems based on search input
func (m *GemsModel) filterGems() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	if query == "" {
		m.filteredGems = m.allGems
		m.cursor = 0
		return
	}

	var filtered []*models.Gem
	for _, gem := range m.allGems {
		name := strings.ToLower(gem.Name)
		desc := strings.ToLower(gem.Description)

		// Simple fuzzy matching: check if query is contained in name or description
		if strings.Contains(name, query) || strings.Contains(desc, query) {
			filtered = append(filtered, gem)
		}
	}

	m.filteredGems = filtered
	if m.cursor >= len(m.filteredGems) {
		m.cursor = max(0, len(m.filteredGems)-1)
	}
}

// copyIDToClipboard copies the selected gem ID to clipboard
func (m GemsModel) copyIDToClipboard() (tea.Model, tea.Cmd) {
	if m.selectedGem == nil {
		return m, nil
	}

	err := clipboard.WriteAll(m.selectedGem.ID)
	if err != nil {
		m.feedback = fmt.Sprintf("Failed to copy: %v", err)
	} else {
		m.feedback = fmt.Sprintf("Copied ID: %s", truncate(m.selectedGem.ID, 30))
	}

	return m, clearGemsFeedback(m.feedbackTimeout)
}

// clearGemsFeedback returns a command that clears the feedback message after a delay
func clearGemsFeedback(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return gemsFeedbackClearMsg{}
	})
}

// View renders the TUI
func (m GemsModel) View() string {
	if !m.ready {
		return loadingStyle.Render("  Initializing...")
	}

	if m.loading {
		return loadingStyle.Render("  Loading gems...")
	}

	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("  Error: %v", m.err))
	}

	var sections []string
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Header
	header := m.renderHeader(contentWidth)
	sections = append(sections, header)

	// Search bar (when in search mode or has search value)
	if m.searching || m.searchInput.Value() != "" {
		searchBar := m.renderSearchBar(contentWidth)
		sections = append(sections, searchBar)
	}

	// Main content
	switch m.view {
	case gemsViewList:
		listPanel := m.renderListView(contentWidth)
		sections = append(sections, listPanel)
	case gemsViewDetails:
		detailsPanel := m.renderDetailsView(contentWidth)
		sections = append(sections, detailsPanel)
	}

	// Feedback
	if m.feedback != "" {
		feedbackMsg := configFeedbackStyle.Render("✓ " + m.feedback)
		sections = append(sections, feedbackMsg)
	}

	// Status bar
	statusBar := m.renderStatusBar(contentWidth)
	sections = append(sections, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the header panel
func (m GemsModel) renderHeader(width int) string {
	title := configTitleStyle.Render("✦ Gemini Gems")
	subtitle := hintStyle.Render(fmt.Sprintf("  %d gems", len(m.filteredGems)))
	if len(m.filteredGems) != len(m.allGems) {
		subtitle = hintStyle.Render(fmt.Sprintf("  %d/%d gems", len(m.filteredGems), len(m.allGems)))
	}
	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle)
	return configHeaderStyle.Width(width).Render(headerContent)
}

// renderSearchBar renders the search input bar
func (m GemsModel) renderSearchBar(width int) string {
	searchLabel := inputLabelStyle.Render("🔍 ")
	searchContent := lipgloss.JoinHorizontal(lipgloss.Center, searchLabel, m.searchInput.View())
	return inputPanelStyle.Width(width).Render(searchContent)
}

// renderListView renders the gems list view
func (m GemsModel) renderListView(width int) string {
	title := configSectionTitleStyle.Render("📦 Gems")

	if len(m.filteredGems) == 0 {
		noGems := hintStyle.Render("No gems found")
		content := lipgloss.JoinVertical(lipgloss.Left, title, "", noGems)
		return configPanelStyle.Width(width).Render(content)
	}

	// Calculate visible items based on available height
	availableHeight := m.height - 12 // Reserve space for header, search, status bar
	if m.searching || m.searchInput.Value() != "" {
		availableHeight -= 3
	}
	maxItems := max(5, availableHeight/2)

	// Calculate scroll offset to keep cursor visible
	scrollOffset := 0
	if m.cursor >= maxItems {
		scrollOffset = m.cursor - maxItems + 1
	}

	var items []string
	endIdx := min(scrollOffset+maxItems, len(m.filteredGems))

	for i := scrollOffset; i < endIdx; i++ {
		gem := m.filteredGems[i]
		item := m.renderGemItem(gem, i == m.cursor, width-6)
		items = append(items, item)
	}

	// Add scroll indicators if needed
	if scrollOffset > 0 {
		items = append([]string{hintStyle.Render("  ↑ more above")}, items...)
	}
	if endIdx < len(m.filteredGems) {
		items = append(items, hintStyle.Render("  ↓ more below"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, ""}, items...)...)
	return configPanelStyle.Width(width).Render(content)
}

// renderGemItem renders a single gem item in the list
func (m GemsModel) renderGemItem(gem *models.Gem, selected bool, width int) string {
	cursor := "  "
	nameStyle := configMenuItemStyle
	if selected {
		cursor = configCursorStyle.Render("▸ ")
		nameStyle = configMenuSelectedStyle
	}

	// Type indicator
	gemType := configValueStyle.Render("[custom]")
	if gem.Predefined {
		gemType = configDisabledStyle.Render("[system]")
	}

	// Name
	name := nameStyle.Render(gem.Name)

	// Description (truncated)
	desc := ""
	if gem.Description != "" {
		maxDesc := width - len(gem.Name) - 15
		if maxDesc > 10 {
			desc = hintStyle.Render(" - " + truncate(gem.Description, maxDesc))
		}
	}

	return fmt.Sprintf("%s%s %s%s", cursor, name, gemType, desc)
}

// renderDetailsView renders the gem details view
func (m GemsModel) renderDetailsView(width int) string {
	if m.selectedGem == nil {
		return configPanelStyle.Width(width).Render("No gem selected")
	}

	gem := m.selectedGem
	title := configSectionTitleStyle.Render("📋 Gem Details")

	// Type indicator
	gemType := configEnabledStyle.Render("custom")
	if gem.Predefined {
		gemType = configDisabledStyle.Render("system")
	}

	// Build details
	var details []string
	details = append(details, title, "")
	details = append(details, fmt.Sprintf("   %s  %s", configMenuSelectedStyle.Render("Name:"), gem.Name))
	details = append(details, fmt.Sprintf("   %s  %s", configMenuSelectedStyle.Render("Type:"), gemType))
	details = append(details, fmt.Sprintf("   %s  %s", configMenuSelectedStyle.Render("ID:"), configPathStyle.Render(gem.ID)))

	if gem.Description != "" {
		details = append(details, fmt.Sprintf("   %s  %s", configMenuSelectedStyle.Render("Desc:"), gem.Description))
	}

	// Prompt (rendered as markdown if possible)
	if gem.Prompt != "" {
		details = append(details, "")
		details = append(details, configSectionTitleStyle.Render("📝 Prompt"))

		// Try to render prompt as markdown
		promptWidth := width - 8
		if promptWidth < 40 {
			promptWidth = 40
		}

		opts := render.DefaultOptions().WithWidth(promptWidth)
		rendered, err := render.Markdown(gem.Prompt, opts)
		if err != nil {
			// Fallback to plain text
			rendered = gem.Prompt
		}

		// Wrap in a style
		promptBox := thoughtsStyle.Width(promptWidth).Render(strings.TrimSpace(rendered))
		details = append(details, promptBox)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, details...)
	return configPanelStyle.Width(width).Render(content)
}

// renderStatusBar renders the bottom status bar
func (m GemsModel) renderStatusBar(width int) string {
	var shortcuts []struct {
		key  string
		desc string
	}

	if m.searching {
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"Enter", "Confirm"},
			{"Esc", "Cancel"},
		}
	} else if m.view == gemsViewList {
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"↑↓", "Navigate"},
			{"/", "Search"},
			{"Enter", "Details"},
			{"c", "Chat"},
			{"y", "Copy ID"},
			{"q", "Quit"},
		}
	} else {
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"c", "Chat"},
			{"y", "Copy ID"},
			{"Esc", "Back"},
			{"q", "Quit"},
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

// Helper functions

// sortGems sorts gems by name (custom first, then system)
func sortGems(gems []*models.Gem) []*models.Gem {
	sorted := make([]*models.Gem, len(gems))
	copy(sorted, gems)

	sort.Slice(sorted, func(i, j int) bool {
		// Custom gems before system gems
		if sorted[i].Predefined != sorted[j].Predefined {
			return !sorted[i].Predefined
		}
		// Then by name
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	return sorted
}

// truncate truncates a string to maxLen and adds "..." if needed
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// GemsTUIResult contains the result of running the gems TUI
type GemsTUIResult struct {
	GemID   string // Set if user pressed 'c' to start chat
	GemName string // Name of the gem if chat was initiated
}

// RunGemsTUI starts the gems TUI and returns the result
func RunGemsTUI(client api.GeminiClientInterface, includeHidden bool) (GemsTUIResult, error) {
	m := NewGemsModel(client, includeHidden)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return GemsTUIResult{}, err
	}

	// Check if chat was initiated
	if gm, ok := finalModel.(GemsModel); ok {
		if gm.startChatGemID != "" {
			return GemsTUIResult{
				GemID:   gm.startChatGemID,
				GemName: gm.startChatGemName,
			}, nil
		}
	}

	return GemsTUIResult{}, nil
}
