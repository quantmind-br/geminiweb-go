package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/render"
)

// personaView represents the current view in the persona TUI
type personaView int

const (
	personaViewList personaView = iota
	personaViewDetails
	personaViewCreate
	personaViewEdit
	personaViewDelete
	personaViewHelp
)

// Form field indices
const (
	personaFieldName = iota
	personaFieldDescription
	personaFieldPrompt
	personaFieldCount
)

// personaLoadedMsg is sent when personas are loaded
type personaLoadedMsg struct {
	personas       []config.Persona
	defaultPersona *config.Persona
	err            error
}

// personaSavedMsg is sent when a persona is saved
type personaSavedMsg struct {
	persona config.Persona
	err     error
}

// personaDeletedMsg is sent when a persona is deleted
type personaDeletedMsg struct {
	name string
	err  error
}

// personaDefaultSetMsg is sent when default persona is changed
type personaDefaultSetMsg struct {
	name string
	err  error
}

// personaFeedbackClearMsg is sent to clear feedback messages
type personaFeedbackClearMsg struct{}

// PersonaManagerModel represents the persona manager TUI state
type PersonaManagerModel struct {
	store PersonaStore

	// Data
	allPersonas      []config.Persona // All personas (sorted)
	filteredPersonas []config.Persona // Filtered personas based on search
	selectedPersona  *config.Persona  // Currently selected persona for details view
	defaultPersona   *config.Persona  // Current default persona

	// Search
	searchInput textinput.Model
	searching   bool

	// Form inputs for create/edit
	formInputs     []textinput.Model
	promptTextarea textarea.Model
	formFocus      int  // Current focused field in form
	useTextarea    bool // Whether prompt is using textarea

	// Navigation
	view   personaView
	cursor int

	// Feedback
	feedback        string
	feedbackTimeout time.Duration

	// State
	loading    bool
	submitting bool
	err        error

	// Dimensions
	width  int
	height int
	ready  bool
}

// NewPersonaManagerModel creates a new persona manager TUI model
func NewPersonaManagerModel(store PersonaStore) PersonaManagerModel {
	// Create form inputs
	formInputs := make([]textinput.Model, personaFieldCount)

	// Name input
	formInputs[personaFieldName] = textinput.New()
	formInputs[personaFieldName].Placeholder = "Persona name..."
	formInputs[personaFieldName].CharLimit = config.MaxNameLength
	formInputs[personaFieldName].Width = 50

	// Description input
	formInputs[personaFieldDescription] = textinput.New()
	formInputs[personaFieldDescription].Placeholder = "Description (optional)..."
	formInputs[personaFieldDescription].CharLimit = config.MaxDescriptionLength
	formInputs[personaFieldDescription].Width = 50

	// Prompt input (single line, can switch to textarea)
	formInputs[personaFieldPrompt] = textinput.New()
	formInputs[personaFieldPrompt].Placeholder = "System prompt..."
	formInputs[personaFieldPrompt].CharLimit = 500
	formInputs[personaFieldPrompt].Width = 50

	// Textarea for longer prompts
	ta := textarea.New()
	ta.Placeholder = "System prompt (multi-line)..."
	ta.CharLimit = config.MaxPromptLength
	ta.SetWidth(50)
	ta.SetHeight(6)
	ta.ShowLineNumbers = false

	// Search input
	si := textinput.New()
	si.Placeholder = "Type to filter personas..."
	si.CharLimit = 100
	si.Width = 40

	return PersonaManagerModel{
		store:           store,
		formInputs:      formInputs,
		promptTextarea:  ta,
		searchInput:     si,
		view:            personaViewList,
		feedbackTimeout: 2 * time.Second,
		loading:         true,
	}
}

// Init initializes the model and starts loading personas
func (m PersonaManagerModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadPersonas(),
		textinput.Blink,
	)
}

// loadPersonas returns a command that loads personas from the store
func (m PersonaManagerModel) loadPersonas() tea.Cmd {
	return func() tea.Msg {
		personas, err := m.store.List()
		if err != nil {
			return personaLoadedMsg{err: err}
		}

		defaultPersona, err := m.store.GetDefault()
		if err != nil {
			return personaLoadedMsg{err: err}
		}

		return personaLoadedMsg{
			personas:       personas,
			defaultPersona: defaultPersona,
		}
	}
}

// savePersona returns a command that saves a persona
func (m PersonaManagerModel) savePersona(persona config.Persona) tea.Cmd {
	return func() tea.Msg {
		err := m.store.Save(persona)
		if err != nil {
			return personaSavedMsg{err: err}
		}
		return personaSavedMsg{persona: persona}
	}
}

// deletePersona returns a command that deletes a persona
func (m PersonaManagerModel) deletePersona(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.store.Delete(name)
		return personaDeletedMsg{name: name, err: err}
	}
}

// setDefaultPersona returns a command that sets the default persona
func (m PersonaManagerModel) setDefaultPersona(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.store.SetDefault(name)
		return personaDefaultSetMsg{name: name, err: err}
	}
}

// Update handles messages and updates the model
func (m PersonaManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		// Update form input widths
		formWidth := m.width - 20
		if formWidth < 30 {
			formWidth = 30
		}
		for i := range m.formInputs {
			m.formInputs[i].Width = formWidth
		}
		m.promptTextarea.SetWidth(formWidth)
		// Update search input width
		m.searchInput.Width = m.width - 10
		if m.searchInput.Width < 20 {
			m.searchInput.Width = 20
		}

	case personaLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.allPersonas = sortPersonas(msg.personas)
			m.filteredPersonas = m.allPersonas
			m.defaultPersona = msg.defaultPersona
			// Reset cursor to avoid out of bounds after reload
			if m.cursor >= len(m.filteredPersonas) {
				m.cursor = max(0, len(m.filteredPersonas)-1)
			}
		}

	case personaSavedMsg:
		m.submitting = false
		if msg.err != nil {
			m.feedback = fmt.Sprintf("Error saving persona: %v", msg.err)
		} else {
			m.feedback = fmt.Sprintf("Saved persona '%s'", msg.persona.Name)
			m.view = personaViewList
			m.resetForm()
			// Reload personas
			return m, tea.Batch(m.loadPersonas(), clearPersonaFeedback(m.feedbackTimeout))
		}
		return m, clearPersonaFeedback(m.feedbackTimeout)

	case personaDeletedMsg:
		m.submitting = false
		if msg.err != nil {
			m.feedback = fmt.Sprintf("Error deleting persona: %v", msg.err)
		} else {
			m.feedback = fmt.Sprintf("Deleted persona '%s'", msg.name)
			m.view = personaViewList
			m.selectedPersona = nil
			return m, tea.Batch(m.loadPersonas(), clearPersonaFeedback(m.feedbackTimeout))
		}
		return m, clearPersonaFeedback(m.feedbackTimeout)

	case personaDefaultSetMsg:
		if msg.err != nil {
			m.feedback = fmt.Sprintf("Error setting default: %v", msg.err)
		} else {
			m.feedback = fmt.Sprintf("Set '%s' as default", msg.name)
			return m, tea.Batch(m.loadPersonas(), clearPersonaFeedback(m.feedbackTimeout))
		}
		return m, clearPersonaFeedback(m.feedbackTimeout)

	case personaFeedbackClearMsg:
		m.feedback = ""

	case tea.KeyMsg:
		if m.submitting {
			return m, nil
		}
		// Handle key messages based on current view
		switch m.view {
		case personaViewCreate, personaViewEdit:
			return m.handleFormInput(msg)
		case personaViewDelete:
			return m.handleDeleteConfirm(msg)
		default:
			return m.handleKeyMsg(msg)
		}
	}

	// Update focused input based on view
	cmds = append(cmds, m.updateFocusedInput(msg)...)

	return m, tea.Batch(cmds...)
}

// updateFocusedInput updates the currently focused input component
func (m *PersonaManagerModel) updateFocusedInput(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	switch m.view {
	case personaViewCreate, personaViewEdit:
		if m.formFocus == personaFieldPrompt && m.useTextarea {
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

// handleKeyMsg handles key messages when not in form mode
func (m PersonaManagerModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search mode
	if m.searching && m.view == personaViewList {
		return m.handleSearchInput(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.view == personaViewList {
			return m, tea.Quit
		}
		m.view = personaViewList
		return m, nil

	case "?":
		m.view = personaViewHelp
		return m, nil

	case "/":
		if m.view == personaViewList {
			m.searching = true
			m.searchInput.Focus()
			return m, textinput.Blink
		}

	case "esc":
		if m.searching {
			m.searching = false
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.filteredPersonas = m.allPersonas
			m.cursor = 0
			return m, nil
		}
		if m.view == personaViewDetails || m.view == personaViewHelp {
			m.view = personaViewList
			return m, nil
		}
		return m, tea.Quit

	case "up", "k":
		if m.view == personaViewList {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = max(0, len(m.filteredPersonas)-1)
			}
		}

	case "down", "j":
		if m.view == personaViewList {
			m.cursor++
			if m.cursor >= len(m.filteredPersonas) {
				m.cursor = 0
			}
		}

	case "enter":
		if m.view == personaViewList && len(m.filteredPersonas) > 0 {
			m.selectedPersona = &m.filteredPersonas[m.cursor]
			m.view = personaViewDetails
		}

	case "n":
		// Create new persona
		if m.view == personaViewList {
			m.resetForm()
			m.view = personaViewCreate
			m.formInputs[personaFieldName].Focus()
			return m, textinput.Blink
		}

	case "e":
		// Edit selected persona
		persona := m.getSelectedPersona()
		if persona != nil && persona.Name != "default" {
			m.populateForm(persona)
			m.view = personaViewEdit
			m.formInputs[personaFieldName].Focus()
			// Don't allow editing the name for existing personas
			m.formInputs[personaFieldName].Blur()
			return m, textinput.Blink
		} else if persona != nil && persona.Name == "default" {
			m.feedback = "Cannot edit the default persona"
			return m, clearPersonaFeedback(m.feedbackTimeout)
		}

	case "d":
		// Delete selected persona
		persona := m.getSelectedPersona()
		if persona != nil && persona.Name != "default" {
			m.view = personaViewDelete
			return m, nil
		} else if persona != nil && persona.Name == "default" {
			m.feedback = "Cannot delete the default persona"
			return m, clearPersonaFeedback(m.feedbackTimeout)
		}

	case "s":
		// Set selected persona as default
		persona := m.getSelectedPersona()
		if persona != nil {
			return m, m.setDefaultPersona(persona.Name)
		}

	case "x":
		// Export selected persona to Markdown
		persona := m.getSelectedPersona()
		if persona != nil {
			model, cmd := m.exportPersonaToMarkdown(persona)
			return model, cmd
		}

	case "home", "g":
		if m.view == personaViewList {
			m.cursor = 0
		}

	case "end", "G":
		if m.view == personaViewList {
			m.cursor = max(0, len(m.filteredPersonas)-1)
		}
	}

	return m, nil
}

// handleSearchInput handles key messages in search mode
func (m PersonaManagerModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.filteredPersonas = m.allPersonas
		m.cursor = 0
		return m, nil

	case "enter":
		m.searching = false
		m.searchInput.Blur()
		return m, nil

	case "up", "k":
		m.cursor--
		if m.cursor < 0 {
			m.cursor = max(0, len(m.filteredPersonas)-1)
		}
		return m, nil

	case "down", "j":
		m.cursor++
		if m.cursor >= len(m.filteredPersonas) {
			m.cursor = 0
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.filterPersonas()
		// Reset cursor when filtering
		if m.cursor >= len(m.filteredPersonas) {
			m.cursor = max(0, len(m.filteredPersonas)-1)
		}
		return m, cmd
	}
}

// filterPersonas filters personas based on search input
func (m *PersonaManagerModel) filterPersonas() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	if query == "" {
		m.filteredPersonas = m.allPersonas
		m.cursor = 0
		return
	}

	var filtered []config.Persona
	for _, p := range m.allPersonas {
		name := strings.ToLower(p.Name)
		desc := strings.ToLower(p.Description)

		if strings.Contains(name, query) || strings.Contains(desc, query) {
			filtered = append(filtered, p)
		}
	}

	m.filteredPersonas = filtered
	if m.cursor >= len(m.filteredPersonas) {
		m.cursor = max(0, len(m.filteredPersonas)-1)
	}
}

// handleFormInput handles key messages in create/edit form
func (m PersonaManagerModel) handleFormInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.view = personaViewList
		m.resetForm()
		return m, nil

	case "tab", "down":
		m.blurCurrentField()
		m.formFocus++
		if m.formFocus >= personaFieldCount {
			m.formFocus = 0
		}
		m.focusCurrentField()
		return m, textinput.Blink

	case "shift+tab", "up":
		m.blurCurrentField()
		m.formFocus--
		if m.formFocus < 0 {
			m.formFocus = personaFieldCount - 1
		}
		m.focusCurrentField()
		return m, textinput.Blink

	case "ctrl+t":
		// Toggle textarea for prompt field
		if m.formFocus == personaFieldPrompt {
			m.useTextarea = !m.useTextarea
			if m.useTextarea {
				m.promptTextarea.SetValue(m.formInputs[personaFieldPrompt].Value())
				m.formInputs[personaFieldPrompt].Blur()
				m.promptTextarea.Focus()
			} else {
				m.formInputs[personaFieldPrompt].SetValue(m.promptTextarea.Value())
				m.promptTextarea.Blur()
				m.formInputs[personaFieldPrompt].Focus()
			}
			return m, textinput.Blink
		}

	case "ctrl+s", "ctrl+enter":
		return m.submitForm()

	default:
		var cmd tea.Cmd
		if m.formFocus == personaFieldPrompt && m.useTextarea {
			m.promptTextarea, cmd = m.promptTextarea.Update(msg)
		} else if m.formFocus < len(m.formInputs) {
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

// handleDeleteConfirm handles key messages in delete confirmation
func (m PersonaManagerModel) handleDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "n", "N":
		m.view = personaViewList
		return m, nil

	case "y", "Y", "enter":
		persona := m.getSelectedPersona()
		if persona != nil && persona.Name != "default" {
			m.submitting = true
			return m, m.deletePersona(persona.Name)
		}
		m.view = personaViewList
		return m, nil
	}

	return m, nil
}

// getSelectedPersona returns the currently selected persona
func (m PersonaManagerModel) getSelectedPersona() *config.Persona {
	if m.view == personaViewDetails && m.selectedPersona != nil {
		return m.selectedPersona
	}
	if m.view == personaViewList && len(m.filteredPersonas) > 0 && m.cursor < len(m.filteredPersonas) {
		return &m.filteredPersonas[m.cursor]
	}
	return m.selectedPersona
}

// resetForm resets all form inputs
func (m *PersonaManagerModel) resetForm() {
	for i := range m.formInputs {
		m.formInputs[i].SetValue("")
		m.formInputs[i].Blur()
	}
	m.promptTextarea.SetValue("")
	m.promptTextarea.Blur()
	m.formFocus = 0
	m.useTextarea = false
}

// populateForm fills the form with persona data for editing
func (m *PersonaManagerModel) populateForm(persona *config.Persona) {
	m.formInputs[personaFieldName].SetValue(persona.Name)
	m.formInputs[personaFieldDescription].SetValue(persona.Description)
	m.formInputs[personaFieldPrompt].SetValue(persona.SystemPrompt)
	m.promptTextarea.SetValue(persona.SystemPrompt)
	m.formFocus = 0
	// Use textarea if prompt has multiple lines
	m.useTextarea = strings.Contains(persona.SystemPrompt, "\n")
}

// blurCurrentField removes focus from the current field
func (m *PersonaManagerModel) blurCurrentField() {
	if m.formFocus == personaFieldPrompt && m.useTextarea {
		m.promptTextarea.Blur()
	} else if m.formFocus < len(m.formInputs) {
		m.formInputs[m.formFocus].Blur()
	}
}

// focusCurrentField sets focus on the current field
func (m *PersonaManagerModel) focusCurrentField() {
	if m.formFocus == personaFieldPrompt && m.useTextarea {
		m.promptTextarea.Focus()
	} else if m.formFocus < len(m.formInputs) {
		m.formInputs[m.formFocus].Focus()
	}
}

// submitForm validates and submits the form
func (m PersonaManagerModel) submitForm() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.formInputs[personaFieldName].Value())
	description := strings.TrimSpace(m.formInputs[personaFieldDescription].Value())

	var prompt string
	if m.useTextarea {
		prompt = strings.TrimSpace(m.promptTextarea.Value())
	} else {
		prompt = strings.TrimSpace(m.formInputs[personaFieldPrompt].Value())
	}

	// Validation
	if name == "" {
		m.feedback = "Name is required"
		return m, clearPersonaFeedback(m.feedbackTimeout)
	}
	if prompt == "" {
		m.feedback = "System prompt is required"
		return m, clearPersonaFeedback(m.feedbackTimeout)
	}

	// Validate using config validation
	persona := config.Persona{
		Name:         name,
		Description:  description,
		SystemPrompt: prompt,
	}
	if err := config.ValidatePersona(persona); err != nil {
		m.feedback = fmt.Sprintf("Validation failed: %v", err)
		return m, clearPersonaFeedback(m.feedbackTimeout)
	}

	m.submitting = true

	if m.view == personaViewCreate {
		return m, m.savePersona(persona)
	}

	// Edit mode - update existing persona
	if m.selectedPersona != nil {
		persona.Name = m.selectedPersona.Name // Keep original name
		return m, m.savePersona(persona)
	}

	m.submitting = false
	m.feedback = "No persona selected for update"
	return m, clearPersonaFeedback(m.feedbackTimeout)
}

// exportPersonaToMarkdown exports a persona to a Markdown file
func (m PersonaManagerModel) exportPersonaToMarkdown(persona *config.Persona) (tea.Model, tea.Cmd) {
	// Get download directory
	cfg, err := config.LoadConfig()
	if err != nil {
		m.feedback = fmt.Sprintf("Error loading config: %v", err)
		return m, clearPersonaFeedback(m.feedbackTimeout)
	}
	downloadDir, err := config.GetDownloadDir(cfg)
	if err != nil {
		m.feedback = fmt.Sprintf("Error getting download directory: %v", err)
		return m, clearPersonaFeedback(m.feedbackTimeout)
	}

	// Create personas subdirectory
	personasDir := filepath.Join(downloadDir, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		m.feedback = fmt.Sprintf("Error creating personas directory: %v", err)
		return m, clearPersonaFeedback(m.feedbackTimeout)
	}

	// Generate Markdown content
	filename := fmt.Sprintf("%s.md", strings.ToLower(strings.ReplaceAll(persona.Name, " ", "_")))
	filePath := filepath.Join(personasDir, filename)

	var md strings.Builder
	md.WriteString("---\n")
	md.WriteString(fmt.Sprintf("name: %s\n", persona.Name))
	if persona.Description != "" {
		md.WriteString(fmt.Sprintf("description: %s\n", persona.Description))
	}
	if persona.Model != "" {
		md.WriteString(fmt.Sprintf("model: %s\n", persona.Model))
	}
	if persona.Temperature != 0 {
		md.WriteString(fmt.Sprintf("temperature: %.1f\n", persona.Temperature))
	}
	md.WriteString("---\n\n")
	md.WriteString(fmt.Sprintf("# %s\n\n", persona.Name))
	if persona.Description != "" {
		md.WriteString(fmt.Sprintf("**%s**\n\n", persona.Description))
	}
	md.WriteString("## System Prompt\n\n")
	md.WriteString("```\n")
	md.WriteString(persona.SystemPrompt)
	md.WriteString("\n```\n")

	// Write file
	if err := os.WriteFile(filePath, []byte(md.String()), 0o644); err != nil {
		m.feedback = fmt.Sprintf("Error exporting persona: %v", err)
		return m, clearPersonaFeedback(m.feedbackTimeout)
	}

	m.feedback = fmt.Sprintf("Exported '%s' to %s", persona.Name, filePath)
	return m, clearPersonaFeedback(m.feedbackTimeout)
}

// clearPersonaFeedback returns a command that clears the feedback message after a delay
func clearPersonaFeedback(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return personaFeedbackClearMsg{}
	})
}

// View renders the TUI
func (m PersonaManagerModel) View() string {
	if !m.ready {
		return loadingStyle.Render("  Initializing...")
	}

	if m.loading {
		return loadingStyle.Render("  Loading personas...")
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

	// Main content based on view
	switch m.view {
	case personaViewList:
		sections = append(sections, m.renderListView(contentWidth))
	case personaViewDetails:
		sections = append(sections, m.renderDetailsView(contentWidth))
	case personaViewCreate:
		sections = append(sections, m.renderCreateView(contentWidth))
	case personaViewEdit:
		sections = append(sections, m.renderEditView(contentWidth))
	case personaViewDelete:
		sections = append(sections, m.renderDeleteView(contentWidth))
	case personaViewHelp:
		sections = append(sections, m.renderHelpView(contentWidth))
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
func (m PersonaManagerModel) renderHeader(width int) string {
	var title string
	switch m.view {
	case personaViewCreate:
		title = configTitleStyle.Render("✦ Create New Persona")
	case personaViewEdit:
		title = configTitleStyle.Render("✦ Edit Persona")
	case personaViewDelete:
		title = configTitleStyle.Render("✦ Delete Persona")
	default:
		title = configTitleStyle.Render("✦ Persona Manager")
	}

	subtitle := ""
	if m.view == personaViewList {
		subtitle = hintStyle.Render(fmt.Sprintf("  %d personas", len(m.filteredPersonas)))
	}

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle)
	return configHeaderStyle.Width(width).Render(headerContent)
}

// renderListView renders the personas list view
func (m PersonaManagerModel) renderListView(width int) string {
	title := configSectionTitleStyle.Render("📦 Personas")

	// Add search bar if searching or has search value
	var contentSections []string
	if m.searching || m.searchInput.Value() != "" {
		searchLabel := inputLabelStyle.Render("🔍 ")
		searchContent := lipgloss.JoinHorizontal(lipgloss.Center, searchLabel, m.searchInput.View())
		contentSections = append(contentSections, inputPanelStyle.Width(width).Render(searchContent))
	}

	if len(m.filteredPersonas) == 0 {
		noPersonas := hintStyle.Render("No personas found. Press 'n' to create one.")
		contentSections = append(contentSections, title, "", noPersonas)
		content := lipgloss.JoinVertical(lipgloss.Left, contentSections...)
		return configPanelStyle.Width(width).Render(content)
	}

	// Calculate visible items based on available height
	availableHeight := m.height - 10
	maxItems := max(5, availableHeight/2)

	// Calculate scroll offset to keep cursor visible
	scrollOffset := 0
	if m.cursor >= maxItems {
		scrollOffset = m.cursor - maxItems + 1
	}

	var items []string
	endIdx := min(scrollOffset+maxItems, len(m.filteredPersonas))

	for i := scrollOffset; i < endIdx; i++ {
		persona := m.filteredPersonas[i]
		item := m.renderPersonaItem(&persona, i == m.cursor, width-6)
		items = append(items, item)
	}

	// Add scroll indicators if needed
	if scrollOffset > 0 {
		items = append([]string{hintStyle.Render("  ↑ more above")}, items...)
	}
	if endIdx < len(m.filteredPersonas) {
		items = append(items, hintStyle.Render("  ↓ more below"))
	}

	contentSections = append(contentSections, title)
	contentSections = append(contentSections, "")
	contentSections = append(contentSections, items...)
	content := lipgloss.JoinVertical(lipgloss.Left, contentSections...)
	return configPanelStyle.Width(width).Render(content)
}

// renderPersonaItem renders a single persona item in the list
func (m PersonaManagerModel) renderPersonaItem(persona *config.Persona, selected bool, width int) string {
	cursor := "  "
	nameStyle := configMenuItemStyle
	if selected {
		cursor = configCursorStyle.Render("▸ ")
		nameStyle = configMenuSelectedStyle
	}

	// Default indicator
	defaultIndicator := ""
	if m.defaultPersona != nil && persona.Name == m.defaultPersona.Name {
		defaultIndicator = configValueStyle.Render("[default]")
	}

	// Name
	name := nameStyle.Render(persona.Name)

	// Description (truncated)
	desc := ""
	if persona.Description != "" {
		maxDesc := width - len(persona.Name) - 20
		if maxDesc > 10 {
			desc = hintStyle.Render(" - " + truncate(persona.Description, maxDesc))
		}
	}

	return fmt.Sprintf("%s%s %s%s", cursor, name, defaultIndicator, desc)
}

// renderDetailsView renders the persona details view
func (m PersonaManagerModel) renderDetailsView(width int) string {
	if m.selectedPersona == nil {
		return configPanelStyle.Width(width).Render("No persona selected")
	}

	persona := m.selectedPersona
	title := configSectionTitleStyle.Render("📋 Persona Details")

	// Default indicator
	defaultIndicator := ""
	if m.defaultPersona != nil && persona.Name == m.defaultPersona.Name {
		defaultIndicator = configEnabledStyle.Render("✓ Default")
	} else {
		defaultIndicator = hintStyle.Render("(not default)")
	}

	var details []string
	details = append(details, title, "")
	details = append(details, fmt.Sprintf("   %s  %s", configMenuSelectedStyle.Render("Name:"), persona.Name))
	details = append(details, fmt.Sprintf("   %s  %s", configMenuSelectedStyle.Render("Status:"), defaultIndicator))

	if persona.Description != "" {
		details = append(details, fmt.Sprintf("   %s  %s", configMenuSelectedStyle.Render("Desc:"), persona.Description))
	}

	// Prompt (rendered as markdown if possible)
	if persona.SystemPrompt != "" {
		details = append(details, "")
		details = append(details, configSectionTitleStyle.Render("📝 System Prompt"))

		promptWidth := width - 8
		if promptWidth < 40 {
			promptWidth = 40
		}

		opts := render.DefaultOptions().WithWidth(promptWidth)
		rendered, err := render.Markdown(persona.SystemPrompt, opts)
		if err != nil {
			rendered = persona.SystemPrompt
		}

		promptBox := thoughtsStyle.Width(promptWidth).Render(strings.TrimSpace(rendered))
		details = append(details, promptBox)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, details...)
	return configPanelStyle.Width(width).Render(content)
}

// renderCreateView renders the create persona form
func (m PersonaManagerModel) renderCreateView(width int) string {
	return m.renderFormView(width, "✨ New Persona", "Fill in the details for your new persona:")
}

// renderEditView renders the edit persona form
func (m PersonaManagerModel) renderEditView(width int) string {
	personaName := ""
	if m.selectedPersona != nil {
		personaName = m.selectedPersona.Name
	}
	return m.renderFormView(width, "✏️  Edit Persona", fmt.Sprintf("Editing persona: %s", personaName))
}

// renderFormView renders the create/edit form
func (m PersonaManagerModel) renderFormView(width int, title, subtitle string) string {
	titleLine := configSectionTitleStyle.Render(title)
	subtitleLine := hintStyle.Render(subtitle)

	var fields []string
	fields = append(fields, titleLine, subtitleLine, "")

	// Name field (disabled in edit mode)
	nameLabel := m.renderFieldLabel("Name", personaFieldName, true)
	fields = append(fields, nameLabel)
	if m.view == personaViewEdit {
		fields = append(fields, "   "+configDisabledStyle.Render(m.formInputs[personaFieldName].Value()))
	} else {
		fields = append(fields, "   "+m.formInputs[personaFieldName].View())
	}
	fields = append(fields, "")

	// Description field
	descLabel := m.renderFieldLabel("Description", personaFieldDescription, false)
	fields = append(fields, descLabel)
	fields = append(fields, "   "+m.formInputs[personaFieldDescription].View())
	fields = append(fields, "")

	// Prompt field
	promptLabel := m.renderFieldLabel("System Prompt", personaFieldPrompt, true)
	toggleHint := hintStyle.Render(" (Ctrl+T to toggle multi-line)")
	fields = append(fields, promptLabel+toggleHint)

	if m.useTextarea {
		fields = append(fields, "   "+m.promptTextarea.View())
	} else {
		fields = append(fields, "   "+m.formInputs[personaFieldPrompt].View())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, fields...)
	return configPanelStyle.Width(width).Render(content)
}

// renderFieldLabel renders a form field label with focus indication
func (m PersonaManagerModel) renderFieldLabel(label string, fieldIndex int, required bool) string {
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
func (m PersonaManagerModel) renderDeleteView(width int) string {
	persona := m.getSelectedPersona()
	if persona == nil {
		return configPanelStyle.Width(width).Render("No persona selected")
	}

	title := configSectionTitleStyle.Render("⚠️  Confirm Deletion")
	warning := errorStyle.Render("This action cannot be undone!")

	var lines []string
	lines = append(lines, title, "")
	lines = append(lines, fmt.Sprintf("   Are you sure you want to delete the persona '%s'?", persona.Name))
	lines = append(lines, "")
	lines = append(lines, "   "+warning)
	lines = append(lines, "")

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return configPanelStyle.Width(width).Render(content)
}

// renderHelpView renders the help overlay
func (m PersonaManagerModel) renderHelpView(width int) string {
	title := configSectionTitleStyle.Render("❓ Keyboard Shortcuts")

	var sections []string
	sections = append(sections, title, "")

	// List View shortcuts
	sections = append(sections, configMenuSelectedStyle.Render("List View:"))
	sections = append(sections, "   ↑/↓ or j/k  - Navigate personas")
	sections = append(sections, "   Enter       - View persona details")
	sections = append(sections, "   /           - Search/filter personas")
	sections = append(sections, "   n           - Create new persona")
	sections = append(sections, "   e           - Edit selected persona")
	sections = append(sections, "   d           - Delete selected persona")
	sections = append(sections, "   s           - Set selected as default")
	sections = append(sections, "   x           - Export to Markdown")
	sections = append(sections, "   g/G         - Jump to top/bottom")
	sections = append(sections, "   q           - Quit")
	sections = append(sections, "")

	// Details View shortcuts
	sections = append(sections, configMenuSelectedStyle.Render("Details View:"))
	sections = append(sections, "   e           - Edit persona")
	sections = append(sections, "   d           - Delete persona")
	sections = append(sections, "   s           - Set as default")
	sections = append(sections, "   Esc         - Back to list")
	sections = append(sections, "")

	// Form View shortcuts
	sections = append(sections, configMenuSelectedStyle.Render("Form View (Create/Edit):"))
	sections = append(sections, "   Tab/↑/↓    - Navigate fields")
	sections = append(sections, "   Shift+Tab  - Previous field")
	sections = append(sections, "   Ctrl+T      - Toggle multi-line for prompt")
	sections = append(sections, "   Ctrl+S      - Save persona")
	sections = append(sections, "   Esc         - Cancel")
	sections = append(sections, "")

	// Delete View shortcuts
	sections = append(sections, configMenuSelectedStyle.Render("Delete Confirmation:"))
	sections = append(sections, "   y           - Confirm deletion")
	sections = append(sections, "   n/Esc       - Cancel")
	sections = append(sections, "")

	// Global shortcuts
	sections = append(sections, configMenuSelectedStyle.Render("Global:"))
	sections = append(sections, "   ?           - Show this help")
	sections = append(sections, "   Ctrl+C      - Quit")

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return configPanelStyle.Width(width).Render(content)
}

// renderStatusBar renders the bottom status bar
func (m PersonaManagerModel) renderStatusBar(width int) string {
	var shortcuts []struct {
		key  string
		desc string
	}

	switch m.view {
	case personaViewCreate, personaViewEdit:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"Tab", "Next"},
			{"Shift+Tab", "Prev"},
			{"Ctrl+S", "Save"},
			{"Esc", "Cancel"},
		}
	case personaViewDelete:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"y", "Yes, delete"},
			{"n", "No, cancel"},
			{"Esc", "Cancel"},
		}
	case personaViewDetails:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"e", "Edit"},
			{"d", "Delete"},
			{"s", "Set Default"},
			{"x", "Export"},
			{"Esc", "Back"},
		}
	case personaViewList:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"↑↓", "Navigate"},
			{"/", "Search"},
			{"Enter", "Details"},
			{"n", "New"},
			{"e", "Edit"},
			{"d", "Delete"},
			{"s", "Set Default"},
			{"x", "Export"},
			{"?", "Help"},
			{"q", "Quit"},
		}
	case personaViewHelp:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"Esc", "Close"},
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

// sortPersonas sorts personas (default first, then alphabetically)
func sortPersonas(personas []config.Persona) []config.Persona {
	sorted := make([]config.Persona, len(personas))
	copy(sorted, personas)

	sort.Slice(sorted, func(i, j int) bool {
		// Default persona first
		if sorted[i].Name == "default" {
			return true
		}
		if sorted[j].Name == "default" {
			return false
		}
		// Then alphabetically
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	return sorted
}

// RunPersonaManagerTUI starts the persona manager TUI
func RunPersonaManagerTUI(store PersonaStore) error {
	m := NewPersonaManagerModel(store)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
