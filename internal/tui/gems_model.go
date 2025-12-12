package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	gemsViewCreate
	gemsViewEdit
	gemsViewDelete
)

// Form field indices
const (
	formFieldName = iota
	formFieldDescription
	formFieldPrompt
	formFieldCount
)

// gemsLoadedMsg is sent when gems are loaded
type gemsLoadedMsg struct {
	gems *models.GemJar
	err  error
}

// gemCreatedMsg is sent when a gem is created
type gemCreatedMsg struct {
	gem *models.Gem
	err error
}

// gemUpdatedMsg is sent when a gem is updated
type gemUpdatedMsg struct {
	gem *models.Gem
	err error
}

// gemDeletedMsg is sent when a gem is deleted
type gemDeletedMsg struct {
	gemID   string
	gemName string
	err     error
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

	// Form inputs for create/edit
	formInputs     []textinput.Model
	promptTextarea textarea.Model
	formFocus      int  // Current focused field in form
	useTextarea    bool // Whether prompt is using textarea

	// Navigation
	view   gemsView
	cursor int

	// Feedback
	feedback        string
	feedbackTimeout time.Duration

	// State
	loading    bool
	submitting bool // True when creating/updating/deleting
	err        error

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

	// Create form inputs
	formInputs := make([]textinput.Model, formFieldCount)

	// Name input
	formInputs[formFieldName] = textinput.New()
	formInputs[formFieldName].Placeholder = "Gem name..."
	formInputs[formFieldName].CharLimit = 100
	formInputs[formFieldName].Width = 50

	// Description input
	formInputs[formFieldDescription] = textinput.New()
	formInputs[formFieldDescription].Placeholder = "Description (optional)..."
	formInputs[formFieldDescription].CharLimit = 500
	formInputs[formFieldDescription].Width = 50

	// Prompt input (single line, can switch to textarea)
	formInputs[formFieldPrompt] = textinput.New()
	formInputs[formFieldPrompt].Placeholder = "System prompt..."
	formInputs[formFieldPrompt].CharLimit = 5000
	formInputs[formFieldPrompt].Width = 50

	// Textarea for longer prompts
	ta := textarea.New()
	ta.Placeholder = "System prompt (multi-line)..."
	ta.CharLimit = 5000
	ta.SetWidth(50)
	ta.SetHeight(6)
	ta.ShowLineNumbers = false

	return GemsModel{
		client:          client,
		searchInput:     ti,
		formInputs:      formInputs,
		promptTextarea:  ta,
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

// createGem returns a command that creates a new gem
func (m GemsModel) createGem(name, prompt, description string) tea.Cmd {
	return func() tea.Msg {
		gem, err := m.client.CreateGem(name, prompt, description)
		if err != nil {
			return gemCreatedMsg{err: err}
		}
		return gemCreatedMsg{gem: gem}
	}
}

// updateGem returns a command that updates an existing gem
func (m GemsModel) updateGem(id, name, prompt, description string) tea.Cmd {
	return func() tea.Msg {
		gem, err := m.client.UpdateGem(id, name, prompt, description)
		if err != nil {
			return gemUpdatedMsg{err: err}
		}
		return gemUpdatedMsg{gem: gem}
	}
}

// deleteGem returns a command that deletes a gem
func (m GemsModel) deleteGem(id, name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteGem(id)
		return gemDeletedMsg{gemID: id, gemName: name, err: err}
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
		// Update form input widths
		formWidth := m.width - 20
		if formWidth < 30 {
			formWidth = 30
		}
		for i := range m.formInputs {
			m.formInputs[i].Width = formWidth
		}
		m.promptTextarea.SetWidth(formWidth)

	case gemsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else if msg.gems != nil {
			m.allGems = sortGems(msg.gems.Values())
			m.filteredGems = m.allGems
		}

	case gemCreatedMsg:
		m.submitting = false
		if msg.err != nil {
			m.feedback = fmt.Sprintf("Error creating gem: %v", msg.err)
		} else {
			m.feedback = fmt.Sprintf("Created gem '%s'", msg.gem.Name)
			m.view = gemsViewList
			m.resetForm()
			// Reload gems
			return m, tea.Batch(m.loadGems(), clearGemsFeedback(m.feedbackTimeout))
		}
		return m, clearGemsFeedback(m.feedbackTimeout)

	case gemUpdatedMsg:
		m.submitting = false
		if msg.err != nil {
			m.feedback = fmt.Sprintf("Error updating gem: %v", msg.err)
		} else {
			m.feedback = fmt.Sprintf("Updated gem '%s'", msg.gem.Name)
			m.view = gemsViewList
			m.resetForm()
			// Reload gems
			return m, tea.Batch(m.loadGems(), clearGemsFeedback(m.feedbackTimeout))
		}
		return m, clearGemsFeedback(m.feedbackTimeout)

	case gemDeletedMsg:
		m.submitting = false
		if msg.err != nil {
			m.feedback = fmt.Sprintf("Error deleting gem: %v", msg.err)
		} else {
			m.feedback = fmt.Sprintf("Deleted gem '%s'", msg.gemName)
			m.view = gemsViewList
			m.selectedGem = nil
			// Reload gems
			return m, tea.Batch(m.loadGems(), clearGemsFeedback(m.feedbackTimeout))
		}
		return m, clearGemsFeedback(m.feedbackTimeout)

	case gemsFeedbackClearMsg:
		m.feedback = ""

	case tea.KeyMsg:
		// Don't process keys while submitting
		if m.submitting {
			return m, nil
		}
		// Handle key messages based on current view
		switch m.view {
		case gemsViewCreate, gemsViewEdit:
			return m.handleFormInput(msg)
		case gemsViewDelete:
			return m.handleDeleteConfirm(msg)
		default:
			if m.searching {
				return m.handleSearchInput(msg)
			}
			return m.handleKeyMsg(msg)
		}
	}

	// Update focused input based on view
	cmds = append(cmds, m.updateFocusedInput(msg)...)

	return m, tea.Batch(cmds...)
}

// updateFocusedInput updates the currently focused input component
func (m *GemsModel) updateFocusedInput(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	switch m.view {
	case gemsViewCreate, gemsViewEdit:
		if m.formFocus == formFieldPrompt && m.useTextarea {
			var cmd tea.Cmd
			m.promptTextarea, cmd = m.promptTextarea.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.formFocus < len(m.formInputs) {
			var cmd tea.Cmd
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
			cmds = append(cmds, cmd)
		}
	default:
		if m.searching {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return cmds
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
		m.view = gemsViewList
		return m, nil

	case "esc":
		if m.view == gemsViewDetails {
			m.view = gemsViewList
			return m, nil
		}
		return m, tea.Quit

	case "/":
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

	case "n":
		// Create new gem
		if m.view == gemsViewList {
			m.resetForm()
			m.view = gemsViewCreate
			m.formInputs[formFieldName].Focus()
			return m, textinput.Blink
		}

	case "e":
		// Edit selected gem
		gem := m.getSelectedGem()
		if gem != nil && !gem.Predefined {
			m.populateForm(gem)
			m.view = gemsViewEdit
			m.formInputs[formFieldName].Focus()
			return m, textinput.Blink
		} else if gem != nil && gem.Predefined {
			m.feedback = "Cannot edit system gems"
			return m, clearGemsFeedback(m.feedbackTimeout)
		}

	case "d":
		// Delete selected gem
		gem := m.getSelectedGem()
		if gem != nil && !gem.Predefined {
			m.view = gemsViewDelete
			return m, nil
		} else if gem != nil && gem.Predefined {
			m.feedback = "Cannot delete system gems"
			return m, clearGemsFeedback(m.feedbackTimeout)
		}

	case "y":
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
		m.searching = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.filteredGems = m.allGems
		m.cursor = 0
		return m, nil

	case "enter":
		m.searching = false
		m.searchInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.filterGems()
		return m, cmd
	}
}

// handleFormInput handles key messages in create/edit form
func (m GemsModel) handleFormInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.view = gemsViewList
		m.resetForm()
		return m, nil

	case "tab", "down":
		// Move to next field
		m.blurCurrentField()
		m.formFocus++
		if m.formFocus >= formFieldCount {
			m.formFocus = 0
		}
		m.focusCurrentField()
		return m, textinput.Blink

	case "shift+tab", "up":
		// Move to previous field
		m.blurCurrentField()
		m.formFocus--
		if m.formFocus < 0 {
			m.formFocus = formFieldCount - 1
		}
		m.focusCurrentField()
		return m, textinput.Blink

	case "ctrl+t":
		// Toggle textarea for prompt field
		if m.formFocus == formFieldPrompt {
			m.useTextarea = !m.useTextarea
			if m.useTextarea {
				// Copy content from textinput to textarea
				m.promptTextarea.SetValue(m.formInputs[formFieldPrompt].Value())
				m.formInputs[formFieldPrompt].Blur()
				m.promptTextarea.Focus()
			} else {
				// Copy content from textarea to textinput
				m.formInputs[formFieldPrompt].SetValue(m.promptTextarea.Value())
				m.promptTextarea.Blur()
				m.formInputs[formFieldPrompt].Focus()
			}
			return m, textinput.Blink
		}

	case "ctrl+s", "ctrl+enter":
		// Submit form
		return m.submitForm()

	default:
		// Update focused input
		var cmd tea.Cmd
		if m.formFocus == formFieldPrompt && m.useTextarea {
			m.promptTextarea, cmd = m.promptTextarea.Update(msg)
		} else if m.formFocus < len(m.formInputs) {
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

// handleDeleteConfirm handles key messages in delete confirmation
func (m GemsModel) handleDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "n", "N":
		m.view = gemsViewList
		return m, nil

	case "y", "Y", "enter":
		gem := m.getSelectedGem()
		if gem != nil && !gem.Predefined {
			m.submitting = true
			return m, m.deleteGem(gem.ID, gem.Name)
		}
		m.view = gemsViewList
		return m, nil
	}

	return m, nil
}

// getSelectedGem returns the currently selected gem
func (m GemsModel) getSelectedGem() *models.Gem {
	if m.view == gemsViewDetails && m.selectedGem != nil {
		return m.selectedGem
	}
	if m.view == gemsViewList && len(m.filteredGems) > 0 && m.cursor < len(m.filteredGems) {
		return m.filteredGems[m.cursor]
	}
	return m.selectedGem
}

// resetForm resets all form inputs
func (m *GemsModel) resetForm() {
	for i := range m.formInputs {
		m.formInputs[i].SetValue("")
		m.formInputs[i].Blur()
	}
	m.promptTextarea.SetValue("")
	m.promptTextarea.Blur()
	m.formFocus = 0
	m.useTextarea = false
}

// populateForm fills the form with gem data for editing
func (m *GemsModel) populateForm(gem *models.Gem) {
	m.formInputs[formFieldName].SetValue(gem.Name)
	m.formInputs[formFieldDescription].SetValue(gem.Description)
	m.formInputs[formFieldPrompt].SetValue(gem.Prompt)
	m.promptTextarea.SetValue(gem.Prompt)
	m.formFocus = 0
	// Use textarea if prompt has multiple lines
	m.useTextarea = strings.Contains(gem.Prompt, "\n")
}

// blurCurrentField removes focus from the current field
func (m *GemsModel) blurCurrentField() {
	if m.formFocus == formFieldPrompt && m.useTextarea {
		m.promptTextarea.Blur()
	} else if m.formFocus < len(m.formInputs) {
		m.formInputs[m.formFocus].Blur()
	}
}

// focusCurrentField sets focus on the current field
func (m *GemsModel) focusCurrentField() {
	if m.formFocus == formFieldPrompt && m.useTextarea {
		m.promptTextarea.Focus()
	} else if m.formFocus < len(m.formInputs) {
		m.formInputs[m.formFocus].Focus()
	}
}

// submitForm validates and submits the form
func (m GemsModel) submitForm() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.formInputs[formFieldName].Value())
	description := strings.TrimSpace(m.formInputs[formFieldDescription].Value())

	var prompt string
	if m.useTextarea {
		prompt = strings.TrimSpace(m.promptTextarea.Value())
	} else {
		prompt = strings.TrimSpace(m.formInputs[formFieldPrompt].Value())
	}

	// Validation
	if name == "" {
		m.feedback = "Name is required"
		return m, clearGemsFeedback(m.feedbackTimeout)
	}
	if prompt == "" {
		m.feedback = "Prompt is required"
		return m, clearGemsFeedback(m.feedbackTimeout)
	}

	m.submitting = true

	if m.view == gemsViewCreate {
		return m, m.createGem(name, prompt, description)
	}

	// Edit mode - update existing gem
	if m.selectedGem != nil {
		return m, m.updateGem(m.selectedGem.ID, name, prompt, description)
	}

	m.submitting = false
	m.feedback = "No gem selected for update"
	return m, clearGemsFeedback(m.feedbackTimeout)
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

	// Search bar (when in search mode or has search value, only in list view)
	if m.view == gemsViewList && (m.searching || m.searchInput.Value() != "") {
		searchBar := m.renderSearchBar(contentWidth)
		sections = append(sections, searchBar)
	}

	// Main content based on view
	switch m.view {
	case gemsViewList:
		sections = append(sections, m.renderListView(contentWidth))
	case gemsViewDetails:
		sections = append(sections, m.renderDetailsView(contentWidth))
	case gemsViewCreate:
		sections = append(sections, m.renderCreateView(contentWidth))
	case gemsViewEdit:
		sections = append(sections, m.renderEditView(contentWidth))
	case gemsViewDelete:
		sections = append(sections, m.renderDeleteView(contentWidth))
	}

	// Feedback
	if m.feedback != "" {
		feedbackMsg := configFeedbackStyle.Render("✓ " + m.feedback)
		sections = append(sections, feedbackMsg)
	}

	// Submitting indicator
	if m.submitting {
		sections = append(sections, loadingStyle.Render("  Processing..."))
	}

	// Status bar
	statusBar := m.renderStatusBar(contentWidth)
	sections = append(sections, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the header panel
func (m GemsModel) renderHeader(width int) string {
	var title string
	switch m.view {
	case gemsViewCreate:
		title = configTitleStyle.Render("✦ Create New Gem")
	case gemsViewEdit:
		title = configTitleStyle.Render("✦ Edit Gem")
	case gemsViewDelete:
		title = configTitleStyle.Render("✦ Delete Gem")
	default:
		title = configTitleStyle.Render("✦ Gemini Gems")
	}

	subtitle := ""
	if m.view == gemsViewList {
		if len(m.filteredGems) != len(m.allGems) {
			subtitle = hintStyle.Render(fmt.Sprintf("  %d/%d gems", len(m.filteredGems), len(m.allGems)))
		} else {
			subtitle = hintStyle.Render(fmt.Sprintf("  %d gems", len(m.filteredGems)))
		}
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
		noGems := hintStyle.Render("No gems found. Press 'n' to create one.")
		content := lipgloss.JoinVertical(lipgloss.Left, title, "", noGems)
		return configPanelStyle.Width(width).Render(content)
	}

	// Calculate visible items based on available height
	availableHeight := m.height - 12
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

		promptWidth := width - 8
		if promptWidth < 40 {
			promptWidth = 40
		}

		opts := render.DefaultOptions().WithWidth(promptWidth)
		rendered, err := render.Markdown(gem.Prompt, opts)
		if err != nil {
			rendered = gem.Prompt
		}

		promptBox := thoughtsStyle.Width(promptWidth).Render(strings.TrimSpace(rendered))
		details = append(details, promptBox)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, details...)
	return configPanelStyle.Width(width).Render(content)
}

// renderCreateView renders the create gem form
func (m GemsModel) renderCreateView(width int) string {
	return m.renderFormView(width, "✨ New Gem", "Fill in the details for your new gem:")
}

// renderEditView renders the edit gem form
func (m GemsModel) renderEditView(width int) string {
	gemName := ""
	if m.selectedGem != nil {
		gemName = m.selectedGem.Name
	}
	return m.renderFormView(width, "✏️  Edit Gem", fmt.Sprintf("Editing gem: %s", gemName))
}

// renderFormView renders the create/edit form
func (m GemsModel) renderFormView(width int, title, subtitle string) string {
	titleLine := configSectionTitleStyle.Render(title)
	subtitleLine := hintStyle.Render(subtitle)

	var fields []string
	fields = append(fields, titleLine, subtitleLine, "")

	// Name field
	nameLabel := m.renderFieldLabel("Name", formFieldName, true)
	fields = append(fields, nameLabel)
	fields = append(fields, "   "+m.formInputs[formFieldName].View())
	fields = append(fields, "")

	// Description field
	descLabel := m.renderFieldLabel("Description", formFieldDescription, false)
	fields = append(fields, descLabel)
	fields = append(fields, "   "+m.formInputs[formFieldDescription].View())
	fields = append(fields, "")

	// Prompt field
	promptLabel := m.renderFieldLabel("Prompt", formFieldPrompt, true)
	toggleHint := hintStyle.Render(" (Ctrl+T to toggle multi-line)")
	fields = append(fields, promptLabel+toggleHint)

	if m.useTextarea {
		fields = append(fields, "   "+m.promptTextarea.View())
	} else {
		fields = append(fields, "   "+m.formInputs[formFieldPrompt].View())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, fields...)
	return configPanelStyle.Width(width).Render(content)
}

// renderFieldLabel renders a form field label with focus indication
func (m GemsModel) renderFieldLabel(label string, fieldIndex int, required bool) string {
	style := configMenuItemStyle
	if m.formFocus == fieldIndex {
		style = configMenuSelectedStyle
	}

	reqMark := ""
	if required {
		reqMark = errorStyle.Render("*")
	}

	return fmt.Sprintf("   %s%s:", style.Render(label), reqMark)
}

// renderDeleteView renders the delete confirmation dialog
func (m GemsModel) renderDeleteView(width int) string {
	gem := m.getSelectedGem()
	if gem == nil {
		return configPanelStyle.Width(width).Render("No gem selected")
	}

	title := configSectionTitleStyle.Render("⚠️  Confirm Deletion")
	warning := errorStyle.Render("This action cannot be undone!")

	var lines []string
	lines = append(lines, title, "")
	lines = append(lines, fmt.Sprintf("   Are you sure you want to delete the gem '%s'?", gem.Name))
	lines = append(lines, "")
	lines = append(lines, "   "+warning)
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("   %s  %s", configMenuSelectedStyle.Render("ID:"), configPathStyle.Render(gem.ID)))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return configPanelStyle.Width(width).Render(content)
}

// renderStatusBar renders the bottom status bar
func (m GemsModel) renderStatusBar(width int) string {
	var shortcuts []struct {
		key  string
		desc string
	}

	switch m.view {
	case gemsViewCreate, gemsViewEdit:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"Tab", "Next"},
			{"Shift+Tab", "Prev"},
			{"Ctrl+S", "Save"},
			{"Esc", "Cancel"},
		}
	case gemsViewDelete:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"y", "Yes, delete"},
			{"n", "No, cancel"},
			{"Esc", "Cancel"},
		}
	case gemsViewDetails:
		gem := m.selectedGem
		if gem != nil && !gem.Predefined {
			shortcuts = []struct {
				key  string
				desc string
			}{
				{"c", "Chat"},
				{"e", "Edit"},
				{"d", "Delete"},
				{"y", "Copy ID"},
				{"Esc", "Back"},
			}
		} else {
			shortcuts = []struct {
				key  string
				desc string
			}{
				{"c", "Chat"},
				{"y", "Copy ID"},
				{"Esc", "Back"},
			}
		}
	case gemsViewList:
		if m.searching {
			shortcuts = []struct {
				key  string
				desc string
			}{
				{"Enter", "Confirm"},
				{"Esc", "Cancel"},
			}
		} else {
			shortcuts = []struct {
				key  string
				desc string
			}{
				{"↑↓", "Navigate"},
				{"/", "Search"},
				{"n", "New"},
				{"e", "Edit"},
				{"d", "Delete"},
				{"c", "Chat"},
				{"q", "Quit"},
			}
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
