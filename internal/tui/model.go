package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/api"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/history"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/render"
)

// Animation tick message
type animationTickMsg time.Time

// Message types for the TUI
type (
	responseMsg struct {
		output *models.ModelOutput
	}
	errMsg struct {
		err error
	}
	// gemsLoadedForChatMsg is sent when gems are loaded for the chat selector
	gemsLoadedForChatMsg struct {
		gems []*models.Gem
		err  error
	}
)

// ChatSessionInterface defines the interface for chat session operations needed by the TUI
type ChatSessionInterface interface {
	SendMessage(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error)
	SetMetadata(cid, rid, rcid string)
	GetMetadata() []string
	CID() string
	RID() string
	RCID() string
	GetModel() models.Model
	SetModel(model models.Model)
	LastOutput() *models.ModelOutput
	ChooseCandidate(index int) error
	SetGem(gemID string)
	GetGemID() string
}

// HistoryStoreInterface defines the interface for history operations needed by the TUI
type HistoryStoreInterface interface {
	AddMessage(id, role, content, thoughts string) error
	UpdateMetadata(id, cid, rid, rcid string) error
	UpdateTitle(id, title string) error
}

// Model represents the TUI state
type Model struct {
	client    api.GeminiClientInterface
	session   ChatSessionInterface
	modelName string

	// UI components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// State
	messages       []chatMessage
	loading        bool
	ready          bool
	err            error
	animationFrame int // Frame counter for loading animation

	// Gem selection state
	selectingGem  bool
	gemsList      []*models.Gem
	gemsCursor    int
	gemsLoading   bool
	gemsFilter    string
	activeGemName string // Name of currently active gem

	// History/conversation state
	conversation *history.Conversation   // Current conversation (nil for unsaved)
	historyStore HistoryStoreInterface   // Store for persisting messages

	// Dimensions
	width  int
	height int
}

// chatMessage represents a message in the chat
type chatMessage struct {
	role     string // "user" or "assistant"
	content  string
	thoughts string
}

// NewChatModel creates a new chat TUI model
func NewChatModel(client api.GeminiClientInterface, modelName string) Model {
	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.CharLimit = 4000
	ta.ShowLineNumbers = false
	ta.SetHeight(2)
	ta.Focus()

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(colorText)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(colorTextDim)
	ta.BlurredStyle = ta.FocusedStyle

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = loadingStyle

	return Model{
		client:    client,
		session:   client.StartChat(),
		modelName: modelName,
		textarea:  ta,
		spinner:   s,
		messages:  []chatMessage{},
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// animationTick returns a command that sends animation tick messages
func animationTick() tea.Cmd {
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return animationTickMsg(t)
	})
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// Handle gem selection mode
	if m.selectingGem {
		return m.updateGemSelection(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate component heights
		headerHeight := 4 // Header panel with border
		inputHeight := 6  // Input panel with border
		statusHeight := 1 // Status bar
		padding := 2      // Extra spacing

		vpHeight := m.height - headerHeight - inputHeight - statusHeight - padding
		if vpHeight < 5 {
			vpHeight = 5
		}

		contentWidth := m.width - 4

		// Initialize viewport on first size message
		if !m.ready {
			m.viewport = viewport.New(contentWidth, vpHeight)
			m.textarea.SetWidth(contentWidth - 4)
			m.ready = true
		} else {
			m.viewport.Width = contentWidth
			m.viewport.Height = vpHeight
			m.textarea.SetWidth(contentWidth - 4)
		}
		m.updateViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.loading {
				m.loading = false
			} else {
				return m, tea.Quit
			}

		case "enter":
			if !m.loading && strings.TrimSpace(m.textarea.Value()) != "" {
				// Check for exit commands
				input := strings.TrimSpace(m.textarea.Value())
				if input == "exit" || input == "quit" || input == "/exit" || input == "/quit" {
					return m, tea.Quit
				}

				// Check for /gems command
				if input == "/gems" || input == "/gem" {
					m.textarea.Reset()
					m.selectingGem = true
					m.gemsLoading = true
					m.gemsCursor = 0
					m.gemsFilter = ""
					return m, m.loadGemsForChat()
				}

				// Add user message
				m.messages = append(m.messages, chatMessage{
					role:    "user",
					content: input,
				})
				m.updateViewport()
				m.viewport.GotoBottom()

				// Start loading
				m.loading = true
				m.err = nil
				m.animationFrame = 0
				userMsg := m.textarea.Value()
				m.textarea.Reset()

				cmd = m.sendMessage(userMsg)

				return m, tea.Batch(
					cmd,
					m.spinner.Tick,
					animationTick(),
				)
			}
		}

	case gemsLoadedForChatMsg:
		m.gemsLoading = false
		if msg.err != nil {
			m.selectingGem = false
			m.err = msg.err
		} else {
			m.gemsList = msg.gems
		}

	case responseMsg:
		m.loading = false
		m.messages = append(m.messages, chatMessage{
			role:     "assistant",
			content:  msg.output.Text(),
			thoughts: msg.output.Thoughts(),
		})
		m.updateViewport()
		m.viewport.GotoBottom()

	case errMsg:
		m.loading = false
		m.err = msg.err

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case animationTickMsg:
		if m.loading {
			m.animationFrame++
			cmds = append(cmds, animationTick())
		}
	}

	// Update child components - only pass KeyMsg to textarea to prevent escape sequence leaks
	if !m.loading {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return loadingStyle.Render("  Initializing...")
	}

	// If selecting gem, show the gem selector overlay
	if m.selectingGem {
		return m.renderGemSelector()
	}

	var sections []string
	contentWidth := m.width - 4

	// ═══════════════════════════════════════════════════════════════
	// HEADER
	// ═══════════════════════════════════════════════════════════════
	headerParts := []string{
		titleStyle.Render("✦ Gemini Chat"),
		hintStyle.Render("  •  "),
		subtitleStyle.Render(m.modelName),
	}
	// Show active gem if set
	if m.activeGemName != "" {
		headerParts = append(headerParts,
			hintStyle.Render("  •  "),
			configValueStyle.Render("📦 "+m.activeGemName),
		)
	}
	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, headerParts...)
	header := headerStyle.Width(contentWidth).Render(headerContent)
	sections = append(sections, header)

	// ═══════════════════════════════════════════════════════════════
	// MESSAGES AREA
	// ═══════════════════════════════════════════════════════════════
	var messagesContent string
	if len(m.messages) == 0 {
		// Welcome message when empty
		messagesContent = m.renderWelcome()
	} else {
		messagesContent = m.viewport.View()
	}

	messagesPanel := messagesAreaStyle.
		Width(contentWidth).
		Height(m.viewport.Height).
		Render(messagesContent)
	sections = append(sections, messagesPanel)

	// ═══════════════════════════════════════════════════════════════
	// INPUT AREA
	// ═══════════════════════════════════════════════════════════════
	var inputContent string
	if m.loading {
		// Use colorful animated loading indicator
		inputContent = m.renderLoadingAnimation()
	} else {
		inputContent = lipgloss.JoinVertical(
			lipgloss.Left,
			inputLabelStyle.Render("You"),
			m.textarea.View(),
		)
	}

	inputPanel := inputPanelStyle.Width(contentWidth).Render(inputContent)
	sections = append(sections, inputPanel)

	// ═══════════════════════════════════════════════════════════════
	// STATUS BAR
	// ═══════════════════════════════════════════════════════════════
	statusBar := m.renderStatusBar(contentWidth)
	sections = append(sections, statusBar)

	// ═══════════════════════════════════════════════════════════════
	// ERROR DISPLAY
	// ═══════════════════════════════════════════════════════════════
	if m.err != nil {
		errorDisplay := m.formatError(m.err)
		sections = append(sections, errorDisplay)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderWelcome renders the welcome screen when no messages exist
func (m Model) renderWelcome() string {
	width := m.viewport.Width - 4
	height := m.viewport.Height

	icon := welcomeIconStyle.Width(width).Render("✦")
	title := welcomeTitleStyle.Width(width).Render("Welcome to Gemini Chat")
	subtitle := welcomeStyle.Width(width).Render("Start a conversation by typing a message below")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		icon,
		"",
		title,
		"",
		subtitle,
		"",
	)

	// Center vertically
	contentHeight := lipgloss.Height(content)
	topPadding := (height - contentHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	return strings.Repeat("\n", topPadding) + content
}

// renderLoadingAnimation renders a colorful animated loading indicator
func (m Model) renderLoadingAnimation() string {
	// Animation characters
	chars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	barChars := []string{"█", "█", "█", "█", "█", "█", "█", "█", "▓", "▒", "░"}

	// Get current animation frame
	frame := m.animationFrame

	// Render spinning character with color
	spinIdx := frame % len(chars)
	spinColor := gradientColors[frame%len(gradientColors)]
	spinner := lipgloss.NewStyle().Foreground(spinColor).Bold(true).Render(chars[spinIdx])

	// Render animated bar with gradient
	barWidth := 20
	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		// Calculate which color to use based on position and frame
		colorIdx := (i + frame) % len(gradientColors)
		charIdx := (i + frame/2) % len(barChars)

		style := lipgloss.NewStyle().Foreground(gradientColors[colorIdx])
		bar.WriteString(style.Render(barChars[charIdx]))
	}

	// Animated dots
	dots := ""
	numDots := (frame / 3) % 4
	for i := 0; i < numDots; i++ {
		dotColor := gradientColors[(frame+i)%len(gradientColors)]
		dots += lipgloss.NewStyle().Foreground(dotColor).Render("●")
	}
	for i := numDots; i < 3; i++ {
		dots += lipgloss.NewStyle().Foreground(colorTextMute).Render("○")
	}

	// Combine elements
	text := lipgloss.NewStyle().Foreground(colorText).Render(" Gemini is thinking ")

	return fmt.Sprintf("%s %s %s %s", spinner, bar.String(), text, dots)
}

// renderStatusBar renders the bottom status bar with shortcuts
func (m Model) renderStatusBar(width int) string {
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"Enter", "Send"},
		{"Esc", "Quit"},
		{"↑↓", "Scroll"},
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
	return statusBarStyle.Width(width).Align(lipgloss.Center).Render(bar)
}

// sendMessage creates a command to send a message to the API
func (m Model) sendMessage(prompt string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.session.SendMessage(prompt, nil)
		if err != nil {
			return errMsg{err: err}
		}
		return responseMsg{output: output}
	}
}

// updateViewport refreshes the viewport content with styled messages
func (m *Model) updateViewport() {
	var content strings.Builder
	bubbleWidth := m.viewport.Width - 6

	for i, msg := range m.messages {
		if i > 0 {
			content.WriteString("\n")
		}

		if msg.role == "user" {
			// User message
			label := userLabelStyle.Render("⬤ You")
			bubble := userBubbleStyle.Width(bubbleWidth).Render(msg.content)
			content.WriteString(label + "\n" + bubble)
		} else {
			// Assistant message
			label := assistantLabelStyle.Render("✦ Gemini")

			// Render thoughts if present
			if msg.thoughts != "" {
				thoughtsContent := thoughtsStyle.Width(bubbleWidth - 4).Render(
					"💭 " + msg.thoughts,
				)
				content.WriteString(label + "\n" + thoughtsContent + "\n")
			} else {
				content.WriteString(label + "\n")
			}

			// Render markdown content
			rendered, err := render.MarkdownWithWidth(msg.content, bubbleWidth-4)
			if err != nil {
				rendered = msg.content
			}
			// Trim trailing newlines from glamour
			rendered = strings.TrimRight(rendered, "\n")

			bubble := assistantBubbleStyle.Width(bubbleWidth).Render(rendered)
			content.WriteString(bubble)
		}
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
}


// formatError formats an error with structured error details for display
func (m Model) formatError(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	// Main error message
	sb.WriteString(errorStyle.Render(fmt.Sprintf("⚠ Error: %v", err)))

	// Add structured error details
	detailStyle := lipgloss.NewStyle().Foreground(colorTextDim).PaddingLeft(2)

	if status := apierrors.GetHTTPStatus(err); status > 0 {
		sb.WriteString("\n")
		sb.WriteString(detailStyle.Render(fmt.Sprintf("HTTP Status: %d", status)))
	}

	if code := apierrors.GetErrorCode(err); code != apierrors.ErrCodeUnknown {
		sb.WriteString("\n")
		sb.WriteString(detailStyle.Render(fmt.Sprintf("Error Code: %d (%s)", code, code.String())))
	}

	// Add helpful hints
	hintStyle := lipgloss.NewStyle().Foreground(colorPrimary).PaddingLeft(2)
	switch {
	case apierrors.IsAuthError(err):
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("💡 Try 'geminiweb auto-login' to refresh your session"))
	case apierrors.IsRateLimitError(err):
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("💡 Usage limit reached. Try again later or use a different model"))
	case apierrors.IsNetworkError(err):
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("💡 Check your internet connection"))
	case apierrors.IsTimeoutError(err):
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("💡 Request timed out. Try again"))
	}

	return sb.String()
}

// RunChat starts the chat TUI
func RunChat(client api.GeminiClientInterface, modelName string) error {
	m := NewChatModel(client, modelName)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}

// RunChatWithSession starts the chat TUI with a pre-configured session
func RunChatWithSession(client api.GeminiClientInterface, session ChatSessionInterface, modelName string) error {
	m := NewChatModelWithSession(client, session, modelName)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}

// NewChatModelWithSession creates a new chat TUI model with a pre-configured session
func NewChatModelWithSession(client api.GeminiClientInterface, session ChatSessionInterface, modelName string) Model {
	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.CharLimit = 4000
	ta.ShowLineNumbers = false
	ta.SetHeight(2)
	ta.Focus()

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(colorText)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(colorTextDim)
	ta.BlurredStyle = ta.FocusedStyle

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = loadingStyle

	return Model{
		client:    client,
		session:   session,
		modelName: modelName,
		textarea:  ta,
		spinner:   s,
		messages:  []chatMessage{},
	}
}

// RunChatWithConversation starts the chat TUI with a pre-configured session and conversation
func RunChatWithConversation(client api.GeminiClientInterface, session ChatSessionInterface, modelName string, conv *history.Conversation, store HistoryStoreInterface) error {
	m := NewChatModelWithConversation(client, session, modelName, conv, store)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}

// NewChatModelWithConversation creates a new chat TUI model with a conversation for persistence
func NewChatModelWithConversation(client api.GeminiClientInterface, session ChatSessionInterface, modelName string, conv *history.Conversation, store HistoryStoreInterface) Model {
	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.CharLimit = 4000
	ta.ShowLineNumbers = false
	ta.SetHeight(2)
	ta.Focus()

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(colorText)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(colorTextDim)
	ta.BlurredStyle = ta.FocusedStyle

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = loadingStyle

	// Load existing messages from conversation
	var messages []chatMessage
	if conv != nil {
		for _, msg := range conv.Messages {
			messages = append(messages, chatMessage{
				role:     msg.Role,
				content:  msg.Content,
				thoughts: msg.Thoughts,
			})
		}
	}

	return Model{
		client:       client,
		session:      session,
		modelName:    modelName,
		textarea:     ta,
		spinner:      s,
		messages:     messages,
		conversation: conv,
		historyStore: store,
	}
}

// loadGemsForChat returns a command that loads gems from the API
func (m Model) loadGemsForChat() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return gemsLoadedForChatMsg{err: fmt.Errorf("client not available")}
		}

		jar, err := m.client.FetchGems(false) // Don't include hidden system gems
		if err != nil {
			return gemsLoadedForChatMsg{err: err}
		}

		// Sort gems: custom first, then by name
		gems := jar.Values()
		sortedGems := make([]*models.Gem, len(gems))
		copy(sortedGems, gems)

		// Sort: custom gems first, then alphabetically by name
		for i := 0; i < len(sortedGems)-1; i++ {
			for j := i + 1; j < len(sortedGems); j++ {
				// Custom gems before system gems
				if sortedGems[i].Predefined && !sortedGems[j].Predefined {
					sortedGems[i], sortedGems[j] = sortedGems[j], sortedGems[i]
				} else if sortedGems[i].Predefined == sortedGems[j].Predefined {
					// Alphabetically by name
					if strings.ToLower(sortedGems[i].Name) > strings.ToLower(sortedGems[j].Name) {
						sortedGems[i], sortedGems[j] = sortedGems[j], sortedGems[i]
					}
				}
			}
		}

		return gemsLoadedForChatMsg{gems: sortedGems}
	}
}

// updateGemSelection handles updates when in gem selection mode
func (m Model) updateGemSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case gemsLoadedForChatMsg:
		m.gemsLoading = false
		if msg.err != nil {
			m.selectingGem = false
			m.err = msg.err
		} else {
			m.gemsList = msg.gems
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			// Cancel gem selection
			m.selectingGem = false
			m.gemsList = nil
			m.gemsCursor = 0
			m.gemsFilter = ""

		case "up", "k":
			if len(m.filteredGems()) > 0 {
				m.gemsCursor--
				if m.gemsCursor < 0 {
					m.gemsCursor = len(m.filteredGems()) - 1
				}
			}

		case "down", "j":
			if len(m.filteredGems()) > 0 {
				m.gemsCursor++
				if m.gemsCursor >= len(m.filteredGems()) {
					m.gemsCursor = 0
				}
			}

		case "enter":
			filtered := m.filteredGems()
			if len(filtered) > 0 && m.gemsCursor < len(filtered) {
				selectedGem := filtered[m.gemsCursor]
				m.session.SetGem(selectedGem.ID)
				m.activeGemName = selectedGem.Name
				m.selectingGem = false
				m.gemsList = nil
				m.gemsCursor = 0
				m.gemsFilter = ""
			}

		case "backspace":
			if len(m.gemsFilter) > 0 {
				m.gemsFilter = m.gemsFilter[:len(m.gemsFilter)-1]
				m.gemsCursor = 0
			}

		default:
			// Handle typing for filter (only printable characters)
			if len(msg.String()) == 1 {
				r := []rune(msg.String())[0]
				if r >= ' ' && r <= '~' {
					m.gemsFilter += msg.String()
					m.gemsCursor = 0
				}
			}
		}
	}

	return m, nil
}

// filteredGems returns the gems list filtered by gemsFilter
func (m Model) filteredGems() []*models.Gem {
	if m.gemsFilter == "" {
		return m.gemsList
	}

	filter := strings.ToLower(m.gemsFilter)
	var filtered []*models.Gem
	for _, gem := range m.gemsList {
		if strings.Contains(strings.ToLower(gem.Name), filter) ||
			strings.Contains(strings.ToLower(gem.Description), filter) {
			filtered = append(filtered, gem)
		}
	}
	return filtered
}

// renderGemSelector renders the gem selection overlay
func (m Model) renderGemSelector() string {
	width := m.width - 8
	if width < 40 {
		width = 40
	}

	var content strings.Builder

	// Header
	title := configTitleStyle.Render("📦 Select a Gem")
	if m.activeGemName != "" {
		title += hintStyle.Render(fmt.Sprintf("  (current: %s)", m.activeGemName))
	}
	content.WriteString(title)
	content.WriteString("\n\n")

	// Filter input
	if m.gemsFilter != "" {
		filterLine := inputLabelStyle.Render("🔍 ") + m.gemsFilter + "_"
		content.WriteString(filterLine)
		content.WriteString("\n\n")
	}

	if m.gemsLoading {
		content.WriteString(loadingStyle.Render("  Loading gems..."))
	} else if len(m.gemsList) == 0 {
		content.WriteString(hintStyle.Render("  No gems found"))
	} else {
		filtered := m.filteredGems()
		if len(filtered) == 0 {
			content.WriteString(hintStyle.Render("  No gems match filter"))
		} else {
			// Show up to 8 gems
			maxItems := 8
			startIdx := 0
			if m.gemsCursor >= maxItems {
				startIdx = m.gemsCursor - maxItems + 1
			}
			endIdx := startIdx + maxItems
			if endIdx > len(filtered) {
				endIdx = len(filtered)
			}

			// Scroll indicator
			if startIdx > 0 {
				content.WriteString(hintStyle.Render("  ↑ more above"))
				content.WriteString("\n")
			}

			for i := startIdx; i < endIdx; i++ {
				gem := filtered[i]
				cursor := "  "
				nameStyle := configMenuItemStyle
				if i == m.gemsCursor {
					cursor = configCursorStyle.Render("▸ ")
					nameStyle = configMenuSelectedStyle
				}

				gemType := configValueStyle.Render("[custom]")
				if gem.Predefined {
					gemType = configDisabledStyle.Render("[system]")
				}

				name := nameStyle.Render(gem.Name)
				line := fmt.Sprintf("%s%s %s", cursor, name, gemType)

				// Add truncated description
				if gem.Description != "" {
					maxDesc := width - len(gem.Name) - 15
					if maxDesc > 10 {
						desc := gem.Description
						if len(desc) > maxDesc {
							desc = desc[:maxDesc-3] + "..."
						}
						line += hintStyle.Render(" - " + desc)
					}
				}

				content.WriteString(line)
				content.WriteString("\n")
			}

			// Scroll indicator
			if endIdx < len(filtered) {
				content.WriteString(hintStyle.Render("  ↓ more below"))
				content.WriteString("\n")
			}
		}
	}

	content.WriteString("\n")

	// Status bar
	shortcuts := []string{
		statusKeyStyle.Render("↑↓") + statusDescStyle.Render(" Navigate"),
		statusKeyStyle.Render("Enter") + statusDescStyle.Render(" Select"),
		statusKeyStyle.Render("Esc") + statusDescStyle.Render(" Cancel"),
	}
	statusBar := strings.Join(shortcuts, "  │  ")
	content.WriteString(statusBar)

	// Wrap in a box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Width(width)

	return boxStyle.Render(content.String())
}
