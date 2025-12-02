package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/history"
)

// HistoryStore defines the interface for history operations needed by the selector
type HistoryStore interface {
	ListConversations() ([]*history.Conversation, error)
	CreateConversation(model string) (*history.Conversation, error)
}

// historyLoadedMsg is sent when conversations are loaded
type historyLoadedMsg struct {
	conversations []*history.Conversation
	err           error
}

// HistorySelectorModel represents the history selector TUI state
type HistorySelectorModel struct {
	store     HistoryStore
	modelName string

	// Data
	conversations []*history.Conversation

	// Navigation
	cursor int

	// State
	loading   bool
	err       error
	confirmed bool

	// Result
	selectedConv *history.Conversation // nil means new conversation
	isNewConv    bool

	// Dimensions
	width  int
	height int
	ready  bool
}

// NewHistorySelectorModel creates a new history selector model
func NewHistorySelectorModel(store HistoryStore, modelName string) HistorySelectorModel {
	return HistorySelectorModel{
		store:     store,
		modelName: modelName,
		loading:   true,
		cursor:    0, // Start at "New Conversation"
	}
}

// Init initializes the model and starts loading conversations
func (m HistorySelectorModel) Init() tea.Cmd {
	return m.loadConversations()
}

// loadConversations returns a command that loads conversations from the store
func (m HistorySelectorModel) loadConversations() tea.Cmd {
	return func() tea.Msg {
		conversations, err := m.store.ListConversations()
		if err != nil {
			return historyLoadedMsg{err: err}
		}
		return historyLoadedMsg{conversations: conversations}
	}
}

// Update handles messages and updates the model
func (m HistorySelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case historyLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.conversations = msg.conversations
		}

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc", "q":
			return m, tea.Quit

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				// Wrap to last item (+1 for "New Conversation" option)
				m.cursor = len(m.conversations)
			}

		case "down", "j":
			m.cursor++
			// +1 for "New Conversation" option
			if m.cursor > len(m.conversations) {
				m.cursor = 0
			}

		case "enter":
			m.confirmed = true
			if m.cursor == 0 {
				// "New Conversation" selected
				m.isNewConv = true
				m.selectedConv = nil
			} else {
				// Existing conversation selected
				m.isNewConv = false
				m.selectedConv = m.conversations[m.cursor-1]
			}
			return m, tea.Quit

		case "home", "g":
			m.cursor = 0

		case "end", "G":
			m.cursor = len(m.conversations)
		}
	}

	return m, nil
}

// View renders the TUI
func (m HistorySelectorModel) View() string {
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

	// Status bar
	statusBar := m.renderStatusBar(contentWidth)
	sections = append(sections, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the header panel
func (m HistorySelectorModel) renderHeader(width int) string {
	title := configTitleStyle.Render("Select Conversation")
	subtitle := hintStyle.Render(fmt.Sprintf("  Model: %s", m.modelName))
	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle)
	return configHeaderStyle.Width(width).Render(headerContent)
}

// renderList renders the conversation list
func (m HistorySelectorModel) renderList(width int) string {
	title := configSectionTitleStyle.Render("Conversations")

	var items []string

	// "New Conversation" option (always first)
	newConvItem := m.renderItem(0, "+ New Conversation", "", time.Time{}, true, width-6)
	items = append(items, newConvItem)

	// Existing conversations
	if len(m.conversations) == 0 {
		items = append(items, hintStyle.Render("  No saved conversations"))
	} else {
		// Calculate visible items based on available height
		availableHeight := m.height - 12
		maxItems := max(5, availableHeight/2)

		// Calculate scroll offset
		scrollOffset := 0
		if m.cursor >= maxItems {
			scrollOffset = m.cursor - maxItems + 1
		}

		endIdx := min(scrollOffset+maxItems, len(m.conversations)+1)

		// Render visible items
		for i := scrollOffset; i < endIdx; i++ {
			if i == 0 {
				// Already rendered "New Conversation"
				continue
			}
			conv := m.conversations[i-1]
			item := m.renderItem(i, conv.Title, conv.Model, conv.UpdatedAt, false, width-6)
			items = append(items, item)
		}

		// Scroll indicators
		if scrollOffset > 0 {
			items = append([]string{hintStyle.Render("  ...")}, items...)
		}
		if endIdx < len(m.conversations)+1 {
			items = append(items, hintStyle.Render("  ..."))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, ""}, items...)...)
	return configPanelStyle.Width(width).Render(content)
}

// renderItem renders a single conversation item
func (m HistorySelectorModel) renderItem(index int, title, model string, updatedAt time.Time, isNew bool, width int) string {
	cursor := "  "
	titleStyle := configMenuItemStyle
	if index == m.cursor {
		cursor = configCursorStyle.Render("> ")
		titleStyle = configMenuSelectedStyle
	}

	titleText := titleStyle.Render(title)

	if isNew {
		return fmt.Sprintf("%s%s", cursor, titleText)
	}

	// Format updated time
	timeStr := ""
	if !updatedAt.IsZero() {
		now := time.Now()
		diff := now.Sub(updatedAt)

		switch {
		case diff < time.Hour:
			timeStr = fmt.Sprintf("%dm ago", int(diff.Minutes()))
		case diff < 24*time.Hour:
			timeStr = fmt.Sprintf("%dh ago", int(diff.Hours()))
		case diff < 7*24*time.Hour:
			timeStr = fmt.Sprintf("%dd ago", int(diff.Hours()/24))
		default:
			timeStr = updatedAt.Format("Jan 2")
		}
	}

	// Build the line
	modelInfo := ""
	if model != "" {
		modelInfo = hintStyle.Render(fmt.Sprintf(" [%s]", model))
	}

	timeInfo := ""
	if timeStr != "" {
		timeInfo = configDisabledStyle.Render(fmt.Sprintf(" - %s", timeStr))
	}

	return fmt.Sprintf("%s%s%s%s", cursor, titleText, modelInfo, timeInfo)
}

// renderStatusBar renders the bottom status bar
func (m HistorySelectorModel) renderStatusBar(width int) string {
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"", "Navigate"},
		{"Enter", "Select"},
		{"Esc", "Quit"},
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

	bar := lipgloss.JoinHorizontal(lipgloss.Center, strings.Join(items, "  |  "))
	return configStatusBarStyle.Width(width).Align(lipgloss.Center).Render(bar)
}

// Result returns the selected conversation (nil for new) and whether confirmed
func (m HistorySelectorModel) Result() (*history.Conversation, bool, bool) {
	return m.selectedConv, m.isNewConv, m.confirmed
}

// HistorySelectorResult contains the result of running the history selector
type HistorySelectorResult struct {
	Conversation *history.Conversation // nil for new conversation
	IsNew        bool                  // true if user selected "New Conversation"
	Confirmed    bool                  // true if user confirmed selection
}

// RunHistorySelector starts the history selector TUI and returns the result
func RunHistorySelector(store HistoryStore, modelName string) (HistorySelectorResult, error) {
	m := NewHistorySelectorModel(store, modelName)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return HistorySelectorResult{}, err
	}

	if hm, ok := finalModel.(HistorySelectorModel); ok {
		conv, isNew, confirmed := hm.Result()
		return HistorySelectorResult{
			Conversation: conv,
			IsNew:        isNew,
			Confirmed:    confirmed,
		}, nil
	}

	return HistorySelectorResult{}, nil
}
