package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/history"
)

// HistoryManagerStore defines the interface for history operations needed by the manager
type HistoryManagerStore interface {
	ListConversations() ([]*history.Conversation, error)
	GetConversation(id string) (*history.Conversation, error)
	DeleteConversation(id string) error
	UpdateTitle(id, title string) error
	ToggleFavorite(id string) (bool, error)
	MoveConversation(id string, newIndex int) error
	SwapConversations(id1, id2 string) error
	ExportToMarkdown(id string) (string, error)
}

// HistoryManagerMode represents the current mode of the manager
type HistoryManagerMode int

const (
	ModeNormal HistoryManagerMode = iota
	ModeRename
	ModeSearch
	ModeConfirmDelete
)

// HistoryManagerFilter represents the current filter
type HistoryManagerFilter int

const (
	FilterAll HistoryManagerFilter = iota
	FilterFavorites
)

// historyManagerLoadedMsg is sent when conversations are loaded
type historyManagerLoadedMsg struct {
	conversations []*history.Conversation
	err           error
}

// HistoryManagerModel represents the history manager TUI state
type HistoryManagerModel struct {
	store HistoryManagerStore

	// Data
	conversations         []*history.Conversation
	filteredConversations []*history.Conversation

	// Navigation
	cursor int

	// State
	loading bool
	err     error
	mode    HistoryManagerMode
	filter  HistoryManagerFilter

	// Rename mode
	renameInput textinput.Model
	renameID    string

	// Search mode
	searchInput  textinput.Model
	searchQuery  string
	searchActive bool

	// Delete confirmation
	deleteID    string
	deleteTitle string

	// Result
	selectedConv *history.Conversation
	shouldQuit   bool
	feedback     string // Temporary feedback message

	// Dimensions
	width  int
	height int
	ready  bool
}

// NewHistoryManagerModel creates a new history manager model
func NewHistoryManagerModel(store HistoryManagerStore) HistoryManagerModel {
	renameInput := textinput.New()
	renameInput.Placeholder = "New title..."
	renameInput.CharLimit = 100

	searchInput := textinput.New()
	searchInput.Placeholder = "Search..."
	searchInput.CharLimit = 50

	return HistoryManagerModel{
		store:       store,
		loading:     true,
		cursor:      0,
		mode:        ModeNormal,
		filter:      FilterAll,
		renameInput: renameInput,
		searchInput: searchInput,
	}
}

// Init initializes the model and starts loading conversations
func (m HistoryManagerModel) Init() tea.Cmd {
	return m.loadConversations()
}

// loadConversations returns a command that loads conversations from the store
func (m HistoryManagerModel) loadConversations() tea.Cmd {
	return func() tea.Msg {
		conversations, err := m.store.ListConversations()
		if err != nil {
			return historyManagerLoadedMsg{err: err}
		}
		return historyManagerLoadedMsg{conversations: conversations}
	}
}

// Update handles messages and updates the model
func (m HistoryManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case historyManagerLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.conversations = msg.conversations
			m.applyFilter()
		}

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch m.mode {
		case ModeRename:
			return m.updateRenameMode(msg)
		case ModeSearch:
			return m.updateSearchMode(msg)
		case ModeConfirmDelete:
			return m.updateConfirmDeleteMode(msg)
		default:
			return m.updateNormalMode(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateNormalMode handles input in normal mode
func (m HistoryManagerModel) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.shouldQuit = true
		return m, tea.Quit

	case "esc", "q":
		m.shouldQuit = true
		return m, tea.Quit

	case "up", "k":
		if len(m.filteredConversations) > 0 {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.filteredConversations) - 1
			}
		}

	case "down", "j":
		if len(m.filteredConversations) > 0 {
			m.cursor++
			if m.cursor >= len(m.filteredConversations) {
				m.cursor = 0
			}
		}

	case "ctrl+up", "ctrl+k":
		// Move conversation up
		if len(m.filteredConversations) > 0 && m.cursor > 0 {
			conv := m.filteredConversations[m.cursor]
			prevConv := m.filteredConversations[m.cursor-1]
			if err := m.store.SwapConversations(conv.ID, prevConv.ID); err == nil {
				m.cursor--
				return m, m.loadConversations()
			}
		}

	case "ctrl+down", "ctrl+j":
		// Move conversation down
		if len(m.filteredConversations) > 0 && m.cursor < len(m.filteredConversations)-1 {
			conv := m.filteredConversations[m.cursor]
			nextConv := m.filteredConversations[m.cursor+1]
			if err := m.store.SwapConversations(conv.ID, nextConv.ID); err == nil {
				m.cursor++
				return m, m.loadConversations()
			}
		}

	case "enter":
		// Open conversation
		if len(m.filteredConversations) > 0 {
			m.selectedConv = m.filteredConversations[m.cursor]
			return m, tea.Quit
		}

	case "f":
		// Toggle favorite
		if len(m.filteredConversations) > 0 {
			conv := m.filteredConversations[m.cursor]
			isFav, err := m.store.ToggleFavorite(conv.ID)
			if err == nil {
				if isFav {
					m.feedback = fmt.Sprintf("★ '%s' added to favorites", truncateTitle(conv.Title, 30))
				} else {
					m.feedback = fmt.Sprintf("☆ '%s' removed from favorites", truncateTitle(conv.Title, 30))
				}
				return m, m.loadConversations()
			}
		}

	case "r":
		// Enter rename mode
		if len(m.filteredConversations) > 0 {
			conv := m.filteredConversations[m.cursor]
			m.mode = ModeRename
			m.renameID = conv.ID
			m.renameInput.SetValue(conv.Title)
			m.renameInput.Focus()
			return m, textinput.Blink
		}

	case "d":
		// Enter delete confirmation
		if len(m.filteredConversations) > 0 {
			conv := m.filteredConversations[m.cursor]
			m.mode = ModeConfirmDelete
			m.deleteID = conv.ID
			m.deleteTitle = conv.Title
		}

	case "e":
		// Export (show in feedback for now)
		if len(m.filteredConversations) > 0 {
			conv := m.filteredConversations[m.cursor]
			m.feedback = fmt.Sprintf("Use CLI: geminiweb history export %d", m.cursor+1)
			_ = conv // Silence unused warning
		}

	case "/":
		// Enter search mode
		m.mode = ModeSearch
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink

	case "tab":
		// Toggle filter
		if m.filter == FilterAll {
			m.filter = FilterFavorites
		} else {
			m.filter = FilterAll
		}
		m.applyFilter()
		m.cursor = 0

	case "home", "g":
		m.cursor = 0

	case "end", "G":
		if len(m.filteredConversations) > 0 {
			m.cursor = len(m.filteredConversations) - 1
		}

	case "?":
		m.feedback = "↑↓:Nav  Ctrl+↑↓:Move  f:Fav  r:Rename  d:Del  e:Export  /:Search  Tab:Filter"
	}

	return m, nil
}

// updateRenameMode handles input in rename mode
func (m HistoryManagerModel) updateRenameMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeNormal
		m.renameInput.Blur()
		return m, nil

	case "enter":
		newTitle := strings.TrimSpace(m.renameInput.Value())
		if newTitle != "" {
			if err := m.store.UpdateTitle(m.renameID, newTitle); err == nil {
				m.feedback = fmt.Sprintf("✓ Renamed to '%s'", truncateTitle(newTitle, 30))
			}
		}
		m.mode = ModeNormal
		m.renameInput.Blur()
		return m, m.loadConversations()

	default:
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
}

// updateSearchMode handles input in search mode
func (m HistoryManagerModel) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeNormal
		m.searchInput.Blur()
		m.searchQuery = ""
		m.searchActive = false
		m.applyFilter()
		return m, nil

	case "enter":
		m.searchQuery = m.searchInput.Value()
		m.searchActive = m.searchQuery != ""
		m.mode = ModeNormal
		m.searchInput.Blur()
		m.applyFilter()
		m.cursor = 0
		return m, nil

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}
}

// updateConfirmDeleteMode handles input in delete confirmation mode
func (m HistoryManagerModel) updateConfirmDeleteMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if err := m.store.DeleteConversation(m.deleteID); err == nil {
			m.feedback = fmt.Sprintf("✓ Deleted '%s'", truncateTitle(m.deleteTitle, 30))
			if m.cursor >= len(m.filteredConversations)-1 && m.cursor > 0 {
				m.cursor--
			}
		}
		m.mode = ModeNormal
		return m, m.loadConversations()

	case "n", "N", "esc":
		m.mode = ModeNormal
		return m, nil
	}

	return m, nil
}

// applyFilter filters conversations based on current filter and search
func (m *HistoryManagerModel) applyFilter() {
	m.filteredConversations = nil

	for _, conv := range m.conversations {
		// Apply favorites filter
		if m.filter == FilterFavorites && !conv.IsFavorite {
			continue
		}

		// Apply search filter
		if m.searchActive && m.searchQuery != "" {
			if !strings.Contains(strings.ToLower(conv.Title), strings.ToLower(m.searchQuery)) {
				continue
			}
		}

		m.filteredConversations = append(m.filteredConversations, conv)
	}

	// Adjust cursor if out of bounds
	if m.cursor >= len(m.filteredConversations) {
		m.cursor = max(0, len(m.filteredConversations)-1)
	}
}

// View renders the TUI
func (m HistoryManagerModel) View() string {
	if !m.ready {
		return loadingStyle.Render("  Initializing...")
	}

	if m.loading {
		return loadingStyle.Render("  Loading conversations...")
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

	// Conversation list
	listPanel := m.renderList(contentWidth)
	sections = append(sections, listPanel)

	// Feedback message (if any)
	if m.feedback != "" {
		feedbackPanel := configFeedbackStyle.Render("  " + m.feedback)
		sections = append(sections, feedbackPanel)
	}

	// Mode-specific UI
	switch m.mode {
	case ModeRename:
		sections = append(sections, m.renderRenameInput(contentWidth))
	case ModeSearch:
		sections = append(sections, m.renderSearchInput(contentWidth))
	case ModeConfirmDelete:
		sections = append(sections, m.renderDeleteConfirm(contentWidth))
	}

	// Status bar
	statusBar := m.renderStatusBar(contentWidth)
	sections = append(sections, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the header panel
func (m HistoryManagerModel) renderHeader(width int) string {
	title := configTitleStyle.Render("Conversation Manager")

	// Filter tabs
	var allTab, favTab string
	if m.filter == FilterAll {
		allTab = configMenuSelectedStyle.Render("[All]")
		favTab = hintStyle.Render("[★ Favorites]")
	} else {
		allTab = hintStyle.Render("[All]")
		favTab = configMenuSelectedStyle.Render("[★ Favorites]")
	}

	tabs := fmt.Sprintf("  %s  %s", allTab, favTab)

	// Search indicator
	searchInfo := ""
	if m.searchActive && m.searchQuery != "" {
		searchInfo = hintStyle.Render(fmt.Sprintf("  Search: \"%s\"", m.searchQuery))
	}

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, title, tabs, searchInfo)
	return configHeaderStyle.Width(width).Render(headerContent)
}

// renderList renders the conversation list
func (m HistoryManagerModel) renderList(width int) string {
	var items []string

	if len(m.filteredConversations) == 0 {
		if m.filter == FilterFavorites {
			items = append(items, hintStyle.Render("  No favorite conversations"))
			items = append(items, hintStyle.Render("  Press 'f' on a conversation to add to favorites"))
		} else if m.searchActive {
			items = append(items, hintStyle.Render(fmt.Sprintf("  No conversations matching '%s'", m.searchQuery)))
		} else {
			items = append(items, hintStyle.Render("  No conversations found"))
		}
	} else {
		// Calculate visible items based on available height
		availableHeight := m.height - 14
		maxItems := max(5, availableHeight/2)

		// Calculate scroll offset
		scrollOffset := 0
		if m.cursor >= maxItems {
			scrollOffset = m.cursor - maxItems + 1
		}

		endIdx := min(scrollOffset+maxItems, len(m.filteredConversations))

		// Scroll indicator (top)
		if scrollOffset > 0 {
			items = append(items, hintStyle.Render("  ↑ more..."))
		}

		// Render visible items
		for i := scrollOffset; i < endIdx; i++ {
			conv := m.filteredConversations[i]
			item := m.renderItem(i, conv, width-6)
			items = append(items, item)
		}

		// Scroll indicator (bottom)
		if endIdx < len(m.filteredConversations) {
			items = append(items, hintStyle.Render("  ↓ more..."))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, items...)
	return configPanelStyle.Width(width).Render(content)
}

// renderItem renders a single conversation item
func (m HistoryManagerModel) renderItem(index int, conv *history.Conversation, width int) string {
	// Cursor
	cursor := "  "
	titleStyle := configMenuItemStyle
	if index == m.cursor {
		cursor = configCursorStyle.Render("▸ ")
		titleStyle = configMenuSelectedStyle
	}

	// Index (1-based)
	indexStr := configDisabledStyle.Render(fmt.Sprintf("%2d.", index+1))

	// Favorite star
	star := "  "
	if conv.IsFavorite {
		star = lipgloss.NewStyle().Foreground(colorWarning).Render("★ ")
	}

	// Title
	title := conv.Title
	if len(title) > 40 {
		title = title[:40] + "..."
	}
	titleText := titleStyle.Render(title)

	// Message count and time
	relTime := history.FormatRelativeTime(conv.UpdatedAt)
	msgCount := len(conv.Messages)
	msgInfo := configDisabledStyle.Render(fmt.Sprintf(" (%d msgs, %s)", msgCount, relTime))

	// Move indicator
	moveIndicator := ""
	if index == m.cursor {
		moveIndicator = hintStyle.Render(" ↕")
	}

	return fmt.Sprintf("%s%s %s%s%s%s", cursor, indexStr, star, titleText, msgInfo, moveIndicator)
}

// renderRenameInput renders the rename input field
func (m HistoryManagerModel) renderRenameInput(width int) string {
	label := configSectionTitleStyle.Render("Rename:")
	input := m.renameInput.View()
	hint := hintStyle.Render("  Enter: Confirm  Esc: Cancel")
	content := lipgloss.JoinVertical(lipgloss.Left, label, input, hint)
	return configPanelStyle.Width(width).Render(content)
}

// renderSearchInput renders the search input field
func (m HistoryManagerModel) renderSearchInput(width int) string {
	label := configSectionTitleStyle.Render("Search:")
	input := m.searchInput.View()
	hint := hintStyle.Render("  Enter: Search  Esc: Cancel")
	content := lipgloss.JoinVertical(lipgloss.Left, label, input, hint)
	return configPanelStyle.Width(width).Render(content)
}

// renderDeleteConfirm renders the delete confirmation
func (m HistoryManagerModel) renderDeleteConfirm(width int) string {
	title := m.deleteTitle
	if len(title) > 30 {
		title = title[:30] + "..."
	}
	question := errorStyle.Render(fmt.Sprintf("Delete '%s'?", title))
	hint := hintStyle.Render("  Y: Confirm  N/Esc: Cancel")
	content := lipgloss.JoinVertical(lipgloss.Left, question, hint)
	return configPanelStyle.Width(width).Render(content)
}

// renderStatusBar renders the bottom status bar
func (m HistoryManagerModel) renderStatusBar(width int) string {
	var shortcuts []struct {
		key  string
		desc string
	}

	switch m.mode {
	case ModeRename:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"Enter", "Save"},
			{"Esc", "Cancel"},
		}
	case ModeSearch:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"Enter", "Search"},
			{"Esc", "Cancel"},
		}
	case ModeConfirmDelete:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"Y", "Delete"},
			{"N", "Cancel"},
		}
	default:
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"↑↓", "Nav"},
			{"Ctrl+↑↓", "Move"},
			{"f", "Fav"},
			{"r", "Rename"},
			{"d", "Del"},
			{"/", "Search"},
			{"Tab", "Filter"},
			{"Enter", "Open"},
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

	bar := lipgloss.JoinHorizontal(lipgloss.Center, strings.Join(items, "  "))
	return configStatusBarStyle.Width(width).Align(lipgloss.Center).Render(bar)
}

// Result returns the selected conversation (nil if none selected)
func (m HistoryManagerModel) Result() (*history.Conversation, bool) {
	return m.selectedConv, m.shouldQuit
}

// HistoryManagerResult contains the result of running the history manager
type HistoryManagerResult struct {
	Conversation *history.Conversation // nil if no conversation selected
	ShouldQuit   bool                  // true if user quit without selecting
}

// RunHistoryManager starts the history manager TUI and returns the result
func RunHistoryManager(store HistoryManagerStore) (HistoryManagerResult, error) {
	m := NewHistoryManagerModel(store)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return HistoryManagerResult{}, err
	}

	if hm, ok := finalModel.(HistoryManagerModel); ok {
		conv, quit := hm.Result()
		return HistoryManagerResult{
			Conversation: conv,
			ShouldQuit:   quit,
		}, nil
	}

	return HistoryManagerResult{ShouldQuit: true}, nil
}

// truncateTitle truncates a title to maxLen characters
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen] + "..."
}
