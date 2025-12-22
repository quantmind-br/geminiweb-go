package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/history"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/render"
	"github.com/diogo/geminiweb/pkg/toolexec"
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
	toolExecutionMsg struct {
		call   toolexec.ToolCall
		result *toolexec.Result
	}
	// gemsLoadedForChatMsg is sent when gems are loaded for the chat selector
	gemsLoadedForChatMsg struct {
		gems []*models.Gem
		err  error
	}
	// historyLoadedForChatMsg is sent when history is loaded for the /history command
	historyLoadedForChatMsg struct {
		conversations []*history.Conversation
		err           error
	}
	// exportResultMsg is sent when a conversation export completes
	exportResultMsg struct {
		path      string // Absolute path of exported file
		format    string // "markdown" or "json"
		size      int64  // File size in bytes
		overwrite bool   // If file was overwritten
		err       error  // Error, if any
	}
	// downloadImagesResultMsg is sent when image download completes
	downloadImagesResultMsg struct {
		paths []string // Paths to downloaded images
		count int      // Number of images downloaded
		err   error    // Error, if any
	}
	// initialPromptMsg is sent when an initial prompt from file needs to be processed
	initialPromptMsg struct {
		prompt string
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

// FullHistoryStore extends HistoryStoreInterface with read operations for /history command
// Also implements HistoryManagerStore for /manage command
type FullHistoryStore interface {
	HistoryStoreInterface
	ListConversations() ([]*history.Conversation, error)
	GetConversation(id string) (*history.Conversation, error)
	CreateConversation(model string) (*history.Conversation, error)
	DeleteConversation(id string) error
	ToggleFavorite(id string) (bool, error)
	MoveConversation(id string, newIndex int) error
	SwapConversations(id1, id2 string) error
	ExportToMarkdown(id string) (string, error)
	ExportToJSON(id string) ([]byte, error)
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

	// Tool execution state
	toolRegistry     toolexec.Registry
	toolExecutor     toolexec.Executor
	pendingToolCalls []toolexec.ToolCall
	toolResultBlocks []string
	confirmingTool   bool
	toolConfirmCall  *toolexec.ToolCall
	autoApproveTools bool

	// Gem selection state
	selectingGem  bool
	gemsList      []*models.Gem
	gemsCursor    int
	gemsLoading   bool
	gemsFilter    string
	activeGemName string // Name of currently active gem

	// History/conversation state
	conversation *history.Conversation // Current conversation (nil for unsaved)
	historyStore HistoryStoreInterface // Store for persisting messages

	// History selection state (for /history command)
	selectingHistory bool
	historyList      []*history.Conversation
	historyCursor    int
	historyLoading   bool
	historyFilter    string
	fullHistoryStore FullHistoryStore // Full store interface for /history command

	// File attachments (for /file and /image commands)
	attachments []*api.UploadedFile

	// Image download state (for /save command)
	selectingImages bool
	imageSelector   ImageSelectorModel
	lastOutput      *models.ModelOutput // Store last response for image access
	downloadDir     string              // Directory for saving images

	// Extension state
	detectedExtension models.Extension // Extension detected in prompt (e.g., @Gmail)

	// Local persona (system prompt)
	persona *config.Persona

	// Initial prompt to send automatically on start
	initialPrompt string

	// Dimensions
	width  int
	height int
}

// chatMessage represents a message in the chat
type chatMessage struct {
	role     string // "user", "assistant", or "tool"
	content  string
	thoughts string
	images   []models.WebImage // Images from ModelOutput (for assistant messages)
}

// createTextarea creates and configures a textarea for multi-line input
// Enter sends the message, \ + Enter inserts a newline (line continuation)
func createTextarea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (\\ + Enter for newline)"
	ta.CharLimit = 4000
	ta.ShowLineNumbers = false
	ta.SetHeight(3) // Multi-line input support
	ta.Focus()

	// Disable default InsertNewline binding - we handle newlines via \ + Enter in Update()
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys(""), // Disabled - handled manually
	)

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(colorText)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(colorTextDim)
	ta.BlurredStyle = ta.FocusedStyle

	return ta
}

// NewChatModel creates a new chat TUI model
func NewChatModel(client api.GeminiClientInterface, modelName string) Model {
	ta := createTextarea()

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = loadingStyle

	toolRegistry := defaultToolRegistry()
	toolExecutor := defaultToolExecutor(toolRegistry)
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	return Model{
		client:           client,
		session:          client.StartChat(),
		modelName:        modelName,
		textarea:         ta,
		spinner:          s,
		messages:         []chatMessage{},
		toolRegistry:     toolRegistry,
		toolExecutor:     toolExecutor,
		autoApproveTools: cfg.AutoApproveTools,
	}
}

func defaultToolRegistry() toolexec.Registry {
	return toolexec.NewRegistryWithOptions(
		toolexec.WithTools(
			toolexec.NewBashTool(),
			toolexec.NewFileReadTool(),
			toolexec.NewFileWriteTool(),
			toolexec.NewSearchTool(),
		),
	)
}

func defaultToolExecutor(registry toolexec.Registry) toolexec.Executor {
	return toolexec.NewExecutor(
		registry,
		toolexec.WithDefaultSecurityPolicy(),
		toolexec.WithDefaultMiddleware(),
	)
}

func (m *Model) ensureTooling() {
	if m.toolRegistry == nil {
		m.toolRegistry = defaultToolRegistry()
	}
	if m.toolExecutor == nil {
		m.toolExecutor = defaultToolExecutor(m.toolRegistry)
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textarea.Blink,
		m.spinner.Tick,
	}

	// If there's an initial prompt, send it automatically
	if m.initialPrompt != "" {
		cmds = append(cmds, m.sendInitialPrompt())
	}

	return tea.Batch(cmds...)
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

	// Handle tool confirmation mode
	if m.confirmingTool {
		return m.updateToolConfirmation(msg)
	}

	// Handle gem selection mode
	if m.selectingGem {
		return m.updateGemSelection(msg)
	}

	// Handle history selection mode
	if m.selectingHistory {
		return m.updateHistorySelection(msg)
	}

	// Handle image selection mode (for /save command)
	if m.selectingImages {
		return m.updateImageSelection(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate component heights
		headerHeight := 4 // Header panel with border
		inputHeight := 7  // Input panel with border (includes multi-line textarea)
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

		case "ctrl+g":
			// Shortcut to open gems selector (same as /gems)
			m.textarea.Reset()
			m.selectingGem = true
			m.gemsLoading = true
			m.gemsCursor = 0
			m.gemsFilter = ""
			return m, m.loadGemsForChat()

		case "ctrl+e":
			// Shortcut to export conversation (same as /export without args)
			return m.handleExportCommand("")

		case "enter":
			if !m.loading {
				rawInput := m.textarea.Value()

				// Check for line continuation: if line ends with \, insert newline instead of sending
				if strings.HasSuffix(rawInput, "\\") {
					// Remove the trailing backslash and insert a newline
					m.textarea.SetValue(strings.TrimSuffix(rawInput, "\\") + "\n")
					// Move cursor to end
					m.textarea.CursorEnd()
					return m, nil
				}

				// Empty input - do nothing
				if strings.TrimSpace(rawInput) == "" {
					return m, nil
				}

				input := strings.TrimSpace(rawInput)
				parsed := parseCommand(input)

				// Handle commands
				if parsed.IsCommand {
					switch parsed.Command {
					case "exit", "quit":
						return m, tea.Quit

					case "gems", "gem":
						m.textarea.Reset()
						m.selectingGem = true
						m.gemsLoading = true
						m.gemsCursor = 0
						m.gemsFilter = ""
						return m, m.loadGemsForChat()

					case "history", "hist":
						if m.fullHistoryStore == nil {
							m.err = fmt.Errorf("history not available")
							return m, nil
						}
						m.textarea.Reset()
						m.selectingHistory = true
						m.historyLoading = true
						m.historyCursor = 0
						m.historyFilter = ""
						return m, m.loadHistoryForChat()

					case "manage":
						// Open full history manager
						if m.fullHistoryStore == nil {
							m.err = fmt.Errorf("history not available")
							return m, nil
						}
						m.textarea.Reset()
						// Run the history manager synchronously
						result, err := RunHistoryManager(m.fullHistoryStore)
						if err != nil {
							m.err = fmt.Errorf("history manager error: %w", err)
							return m, nil
						}
						// If a conversation was selected, switch to it
						if result.Conversation != nil {
							return m.switchConversation(result.Conversation)
						}
						return m, nil

					case "favorite", "fav":
						// Toggle favorite status of current conversation
						if m.fullHistoryStore == nil {
							m.err = fmt.Errorf("history not available")
							return m, nil
						}
						if m.conversation == nil {
							m.err = fmt.Errorf("no active conversation to favorite")
							return m, nil
						}
						m.textarea.Reset()
						isFav, err := m.fullHistoryStore.ToggleFavorite(m.conversation.ID)
						if err != nil {
							m.err = fmt.Errorf("failed to toggle favorite: %w", err)
							return m, nil
						}
						m.conversation.IsFavorite = isFav
						if isFav {
							m.err = fmt.Errorf("★ Added to favorites")
						} else {
							m.err = fmt.Errorf("☆ Removed from favorites")
						}
						return m, nil

					case "file":
						return m.handleFileCommand(parsed.Args)

					case "image":
						return m.handleImageCommand(parsed.Args)

					case "clear":
						// Clear all attachments
						m.attachments = nil
						m.textarea.Reset()
						m.err = nil
						return m, nil

					case "export":
						return m.handleExportCommand(parsed.Args)

					case "save", "download":
						return m.handleSaveCommand(parsed.Args)

					default:
						// Unknown command - show error but don't send as message
						m.err = fmt.Errorf("unknown command: /%s", parsed.Command)
						return m, nil
					}
				}

				// Handle exit commands without slash
				if input == "exit" || input == "quit" {
					return m, tea.Quit
				}

				// Add user message
				m.messages = append(m.messages, chatMessage{
					role:    "user",
					content: input,
				})
				m.updateViewport()
				m.viewport.GotoBottom()

				// Auto-save user message to history
				m.saveMessageToHistory("user", input, "")

				// Start loading
				m.loading = true
				m.err = nil
				m.animationFrame = 0
				userMsg := m.textarea.Value()
				m.textarea.Reset()

				// Detect extensions in the prompt
				if ext, found := models.DetectExtension(userMsg); found {
					m.detectedExtension = ext
				} else {
					m.detectedExtension = ""
				}

				// Send message with attachments
				cmd = m.sendMessageWithAttachments(userMsg)

				// Clear attachments after sending
				m.attachments = nil

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

	case historyLoadedForChatMsg:
		m.historyLoading = false
		if msg.err != nil {
			m.selectingHistory = false
			m.err = msg.err
		} else {
			m.historyList = msg.conversations
		}

	case fileUploadedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("file upload failed: %w", msg.err)
		} else {
			// Add file to attachments
			m.attachments = append(m.attachments, msg.file)
			// Show success feedback (could use a toast/notification style)
			m.err = nil
		}

	case exportResultMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Show success feedback (using m.err for feedback - existing pattern)
			feedback := fmt.Sprintf("✓ Exported to %s", msg.path)
			if msg.overwrite {
				feedback += " (overwritten)"
			}
			m.err = fmt.Errorf("%s", feedback)
		}

	case downloadImagesResultMsg:
		if msg.err != nil {
			m.err = msg.err
		} else if msg.count > 0 {
			m.err = fmt.Errorf("✓ Downloaded %d image(s) to %s", msg.count, m.imageSelector.TargetDir())
		} else {
			m.err = fmt.Errorf("no images were downloaded")
		}

	case toolExecutionMsg:
		cmd = m.handleToolResult(msg.call, msg.result)
		if cmd != nil {
			cmds = append(cmds, cmd)
			if m.loading {
				cmds = append(cmds, animationTick())
			}
		}

	case responseMsg:
		m.loading = false
		m.lastOutput = msg.output // Store for /save command
		responseText := msg.output.Text()
		thoughts := msg.output.Thoughts()
		images := msg.output.Images()
		toolCalls, cleanText := toolexec.ExtractToolCallsLenient(responseText)
		displayText := responseText
		if len(toolCalls) > 0 {
			displayText = cleanText
		}

		if strings.TrimSpace(displayText) != "" || thoughts != "" || len(images) > 0 {
			m.messages = append(m.messages, chatMessage{
				role:     "assistant",
				content:  displayText,
				thoughts: thoughts,
				images:   images,
			})
			m.updateViewport()
			m.viewport.GotoBottom()

			// Auto-save assistant message to history
			m.saveMessageToHistory("assistant", displayText, thoughts)
		}

		// Update conversation metadata for session resumption
		m.saveMetadataToHistory()

		if len(toolCalls) > 0 {
			m.ensureTooling()
			m.pendingToolCalls = toolCalls
			m.toolResultBlocks = nil
			cmd = m.startNextToolCall()
			if cmd != nil {
				cmds = append(cmds, cmd)
				if m.loading {
					cmds = append(cmds, animationTick())
				}
			}
		}

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

	case initialPromptMsg:
		// Process initial prompt from file as if user typed it
		prompt := msg.prompt

		// Apply persona system prompt if set
		finalPrompt := prompt
		if m.persona != nil && m.persona.SystemPrompt != "" {
			finalPrompt = config.FormatSystemPrompt(m.persona, prompt)
		}

		// Add user message to chat
		m.messages = append(m.messages, chatMessage{
			role:    "user",
			content: prompt, // Show original prompt, not with system prompt
		})

		// Save to history if available
		if m.historyStore != nil && m.conversation != nil {
			_ = m.historyStore.AddMessage(m.conversation.ID, "user", prompt, "")
		}

		// Set loading state and send message
		m.loading = true
		m.updateViewport()
		return m, tea.Batch(
			m.sendMessage(finalPrompt),
			animationTick(),
		)
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

	if m.confirmingTool {
		return m.renderToolConfirmation()
	}

	// If selecting gem, show the gem selector overlay
	if m.selectingGem {
		return m.renderGemSelector()
	}

	// If selecting history, show the history selector overlay
	if m.selectingHistory {
		return m.renderHistorySelector()
	}

	// If selecting images, show the image selector overlay
	if m.selectingImages {
		return m.imageSelector.View()
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
		// Build label with attachment indicator
		label := "You"
		if len(m.attachments) > 0 {
			attachmentInfo := fmt.Sprintf(" 📎 %d file", len(m.attachments))
			if len(m.attachments) > 1 {
				attachmentInfo += "s"
			}
			label += attachmentInfo
		}

		inputContent = lipgloss.JoinVertical(
			lipgloss.Left,
			inputLabelStyle.Render(label),
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

func (m Model) renderToolConfirmation() string {
	width := m.width - 8
	if width < 40 {
		width = 40
	}

	var content strings.Builder
	content.WriteString("Tool execution requested\n\n")

	if m.toolConfirmCall == nil {
		content.WriteString("No tool call pending.")
	} else {
		call := m.toolConfirmCall
		content.WriteString("Tool: ")
		content.WriteString(call.Name)

		if call.Reason != "" {
			content.WriteString("\nReason: ")
			content.WriteString(call.Reason)
		}

		if len(call.Args) > 0 {
			if data, err := json.MarshalIndent(call.Args, "", "  "); err == nil {
				content.WriteString("\nArgs:\n")
				content.Write(data)
			}
		}
	}

	content.WriteString("\n\nConfirm execution? (y/n)")

	panel := messagesAreaStyle.Width(width).Render(content.String())
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
	}
	return panel
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
		{"\\+Enter", "Newline"},
		{"^E", "Export"},
		{"^G", "Gems"},
		{"Esc", "Quit"},
		{"↑↓", "Scroll"},
	}

	var items []string

	// Show extension indicator if one was detected
	if m.detectedExtension != "" {
		extIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7dcfff")). // Cyan color for extension
			Bold(true).
			Render(string(m.detectedExtension))
		items = append(items, extIndicator)
	}

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

// sendMessageWithAttachments creates a command to send a message with file attachments
func (m Model) sendMessageWithAttachments(prompt string) tea.Cmd {
	// Capture attachments in closure (they will be cleared after this returns)
	attachments := m.attachments

	// Apply persona system prompt if set
	finalPrompt := prompt
	if m.persona != nil && m.persona.SystemPrompt != "" {
		finalPrompt = config.FormatSystemPrompt(m.persona, prompt)
	}

	return func() tea.Msg {
		output, err := m.session.SendMessage(finalPrompt, attachments)
		if err != nil {
			return errMsg{err: err}
		}
		return responseMsg{output: output}
	}
}

func (m *Model) startNextToolCall() tea.Cmd {
	if len(m.pendingToolCalls) == 0 {
		return nil
	}

	m.ensureTooling()

	call := m.pendingToolCalls[0]
	m.pendingToolCalls = m.pendingToolCalls[1:]

	tool, err := m.toolRegistry.Get(call.Name)
	if err != nil {
		result := toolexec.NewErrorResult(call.Name, err).WithTiming(time.Now(), time.Now())
		return func() tea.Msg {
			return toolExecutionMsg{call: call, result: result}
		}
	}

	if tool.RequiresConfirmation(call.Args) && !m.autoApproveTools {
		m.confirmingTool = true
		m.toolConfirmCall = &call
		m.loading = false
		return nil
	}

	m.loading = true
	m.animationFrame = 0
	return m.executeToolCall(call)
}

func (m Model) executeToolCall(call toolexec.ToolCall) tea.Cmd {
	registry := m.toolRegistry
	executor := m.toolExecutor

	return func() tea.Msg {
		if registry == nil || executor == nil {
			err := toolexec.NewExecutionError(call.Name, "tool executor not configured")
			result := toolexec.NewErrorResult(call.Name, err).WithTiming(time.Now(), time.Now())
			return toolExecutionMsg{call: call, result: result}
		}

		input := call.ToInput()
		start := time.Now()
		output, err := executor.Execute(context.Background(), call.Name, input)
		end := time.Now()

		result := toolexec.NewResult(call.Name, output, err).WithTiming(start, end)
		return toolExecutionMsg{call: call, result: result}
	}
}

func (m *Model) handleToolResult(call toolexec.ToolCall, result *toolexec.Result) tea.Cmd {
	if result == nil {
		result = toolexec.NewErrorResult(call.Name, toolexec.NewExecutionError(call.Name, "missing tool result"))
	}
	if result.ToolName == "" {
		result.ToolName = call.Name
	}

	toolMessage := formatToolMessage(call, result)
	if strings.TrimSpace(toolMessage) != "" {
		m.messages = append(m.messages, chatMessage{
			role:    "tool",
			content: toolMessage,
		})
		m.updateViewport()
		m.viewport.GotoBottom()
		m.saveMessageToHistory("tool", toolMessage, "")
	}

	resultBlock := toolexec.NewToolCallResult(result).FormatAsBlock()
	m.toolResultBlocks = append(m.toolResultBlocks, resultBlock)

	if len(m.pendingToolCalls) > 0 {
		return m.startNextToolCall()
	}

	if len(m.toolResultBlocks) == 0 {
		return nil
	}

	payload := strings.Join(m.toolResultBlocks, "\n")
	m.toolResultBlocks = nil
	m.loading = true
	m.animationFrame = 0

	return m.sendMessage(payload)
}

func formatToolMessage(call toolexec.ToolCall, result *toolexec.Result) string {
	var sb strings.Builder

	sb.WriteString("Tool: ")
	sb.WriteString(call.Name)

	if call.Reason != "" {
		sb.WriteString("\nReason: ")
		sb.WriteString(call.Reason)
	}

	if len(call.Args) > 0 {
		if data, err := json.Marshal(call.Args); err == nil {
			sb.WriteString("\nArgs: ")
			sb.Write(data)
		}
	}

	var outputText string
	if result != nil && result.Output != nil {
		if len(result.Output.Data) > 0 {
			outputText = string(result.Output.Data)
		} else if result.Output.Message != "" {
			outputText = result.Output.Message
		}
		if result.Output.Truncated {
			outputText = strings.TrimRight(outputText, "\n") + "\n[output truncated]"
		}
	}

	if result != nil && result.Error != nil {
		if outputText != "" {
			outputText += "\n"
		}
		outputText += "Error: " + result.Error.Error()
	}

	if strings.TrimSpace(outputText) != "" {
		sb.WriteString("\nOutput:\n")
		sb.WriteString(strings.TrimRight(outputText, "\n"))
	}

	return strings.TrimSpace(sb.String())
}

// sendInitialPrompt creates a command to send the initial prompt from file
// This is called automatically on Init() when initialPrompt is set
func (m *Model) sendInitialPrompt() tea.Cmd {
	prompt := m.initialPrompt
	m.initialPrompt = "" // Clear to prevent re-sending

	return func() tea.Msg {
		// Return a message that triggers the send flow
		return initialPromptMsg{prompt: prompt}
	}
}

// fileUploadedMsg is sent when a file upload completes
type fileUploadedMsg struct {
	file *api.UploadedFile
	err  error
}

// handleFileCommand handles the /file <path> command
func (m Model) handleFileCommand(path string) (tea.Model, tea.Cmd) {
	if path == "" {
		m.err = fmt.Errorf("usage: /file <path>")
		return m, nil
	}

	// Expand home directory if needed
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = strings.Replace(path, "~", home, 1)
		}
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		m.err = fmt.Errorf("file not found: %s", path)
		return m, nil
	}

	// Check if client supports file upload
	if m.client == nil {
		m.err = fmt.Errorf("client not available for file upload")
		return m, nil
	}

	m.textarea.Reset()
	m.err = nil

	// Upload file asynchronously
	return m, m.uploadFile(path)
}

// handleImageCommand handles the /image <path> command (alias for /file)
func (m Model) handleImageCommand(path string) (tea.Model, tea.Cmd) {
	// Image is just a specialized file upload
	return m.handleFileCommand(path)
}

// handleExportCommand handles the /export <path> [-f format] command
func (m Model) handleExportCommand(args string) (tea.Model, tea.Cmd) {
	// If no args given and we have a conversation with title, use that as default filename
	if strings.TrimSpace(args) == "" {
		// Generate default filename from conversation title or timestamp
		var filename string
		if m.conversation != nil && m.conversation.Title != "" {
			filename = sanitizeFilename(m.conversation.Title) + ".md"
		} else {
			filename = fmt.Sprintf("conversation_%s.md", time.Now().Format("20060102_150405"))
		}
		args = filename
	}

	// Parse arguments
	path, format, err := parseExportArgs(args)
	if err != nil {
		m.err = err
		return m, nil
	}

	// Validate and expand path
	absPath, err := validateExportPath(path)
	if err != nil {
		m.err = err
		return m, nil
	}

	// Check for conversation to export
	if m.conversation != nil && m.conversation.ID != "" && m.fullHistoryStore != nil {
		// Export from store (persisted conversation)
		return m, exportCommand(m.fullHistoryStore, m.conversation.ID, format, absPath)
	}

	// Check for in-memory messages
	if len(m.messages) > 0 {
		// Export from memory (unsaved conversation)
		var title string
		if m.conversation != nil && m.conversation.Title != "" {
			title = m.conversation.Title
		} else {
			title = "Conversation"
		}
		return m, exportFromMemory(m.messages, title, format, absPath)
	}

	m.err = fmt.Errorf("no conversation to export")
	return m, nil
}

// handleSaveCommand handles the /save command to download images
func (m Model) handleSaveCommand(args string) (tea.Model, tea.Cmd) {
	m.textarea.Reset()

	// Check if we have a last response with images
	if m.lastOutput == nil {
		m.err = fmt.Errorf("no images to save - send a message first")
		return m, nil
	}

	images := m.lastOutput.Images()
	if len(images) == 0 {
		m.err = fmt.Errorf("no images in the last response")
		return m, nil
	}

	// Determine target directory
	targetDir := m.downloadDir
	if args != "" {
		targetDir = strings.TrimSpace(args)
	}
	if targetDir == "" {
		// Use default from config
		homeDir, _ := os.UserHomeDir()
		targetDir = filepath.Join(homeDir, ".geminiweb", "images")
	}

	// Open image selector
	m.selectingImages = true
	m.imageSelector = NewImageSelectorModel(images, targetDir)
	m.imageSelector.width = m.width
	m.imageSelector.height = m.height
	m.imageSelector.ready = true

	return m, nil
}

// downloadSelectedImages creates a command to download selected images
func (m Model) downloadSelectedImages(indices []int, targetDir string) tea.Cmd {
	return func() tea.Msg {
		if m.lastOutput == nil {
			return downloadImagesResultMsg{err: fmt.Errorf("no output available")}
		}

		opts := api.ImageDownloadOptions{
			Directory: targetDir,
			FullSize:  true,
		}

		paths, err := m.client.DownloadSelectedImages(m.lastOutput, indices, opts)
		if err != nil {
			return downloadImagesResultMsg{err: err}
		}

		return downloadImagesResultMsg{
			paths: paths,
			count: len(paths),
		}
	}
}

// uploadFile creates a command to upload a file
func (m Model) uploadFile(path string) tea.Cmd {
	return func() tea.Msg {
		file, err := m.client.UploadFile(path)
		if err != nil {
			return fileUploadedMsg{err: err}
		}
		return fileUploadedMsg{file: file}
	}
}

// parseExportArgs parses /export command arguments
// Returns path, format, and error
// Examples:
//   - "/export chat.md" -> path="chat.md", format="markdown"
//   - "/export chat.json" -> path="chat.json", format="json"
//   - "/export chat" -> path="chat.md", format="markdown" (default)
//   - "/export chat -f json" -> path="chat.json", format="json"
func parseExportArgs(args string) (path, format string, err error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", fmt.Errorf("usage: /export <path> [-f json|md]")
	}

	parts := strings.Fields(args)
	format = "markdown" // default

	// Parse flags
	var pathParts []string
	for i := 0; i < len(parts); i++ {
		if parts[i] == "-f" && i+1 < len(parts) {
			f := strings.ToLower(parts[i+1])
			switch f {
			case "json":
				format = "json"
			case "md", "markdown":
				format = "markdown"
			default:
				return "", "", fmt.Errorf("unknown format: %s (use json or md)", f)
			}
			i++ // skip format value
		} else {
			pathParts = append(pathParts, parts[i])
		}
	}

	if len(pathParts) == 0 {
		return "", "", fmt.Errorf("missing filename")
	}

	path = strings.Join(pathParts, " ")

	// Infer format from extension if not explicitly set via flag
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		format = "json"
	} else if !strings.HasSuffix(strings.ToLower(path), ".md") {
		// Add default extension
		if format == "json" {
			if !strings.HasSuffix(path, ".json") {
				path += ".json"
			}
		} else {
			if !strings.HasSuffix(path, ".md") {
				path += ".md"
			}
		}
	}

	return path, format, nil
}

// validateExportPath validates and expands an export path
// Returns absolute path or error
func validateExportPath(path string) (string, error) {
	// Expand home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot expand ~: %w", err)
		}
		path = strings.Replace(path, "~", home, 1)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if parent directory exists
	dir := filepath.Dir(absPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", dir)
	}

	return absPath, nil
}

// sanitizeFilename removes or replaces characters invalid in filenames
func sanitizeFilename(title string) string {
	// Characters invalid on Windows and/or Unix
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := title

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Trim whitespace and dots from ends
	result = strings.Trim(result, " .")

	// Truncate to 200 characters
	if len(result) > 200 {
		result = result[:200]
	}

	// If empty or all underscores after sanitization, use fallback
	if result == "" || strings.Trim(result, "_") == "" {
		result = "conversation"
	}

	return result
}

// exportCommand creates a tea.Cmd that exports a conversation from store
func exportCommand(store FullHistoryStore, convID, format, path string) tea.Cmd {
	return func() tea.Msg {
		// Check if file exists (for overwrite flag)
		overwrite := false
		if _, err := os.Stat(path); err == nil {
			overwrite = true
		}

		var data []byte
		var err error

		if format == "json" {
			data, err = store.ExportToJSON(convID)
		} else {
			var md string
			md, err = store.ExportToMarkdown(convID)
			data = []byte(md)
		}

		if err != nil {
			return exportResultMsg{err: fmt.Errorf("export failed: %w", err)}
		}

		// Write to file
		if err := os.WriteFile(path, data, 0644); err != nil {
			return exportResultMsg{err: fmt.Errorf("write failed: %w", err)}
		}

		return exportResultMsg{
			path:      path,
			format:    format,
			size:      int64(len(data)),
			overwrite: overwrite,
		}
	}
}

// exportFromMemory creates a tea.Cmd that exports in-memory messages
func exportFromMemory(messages []chatMessage, title, format, path string) tea.Cmd {
	return func() tea.Msg {
		// Check if file exists (for overwrite flag)
		overwrite := false
		if _, err := os.Stat(path); err == nil {
			overwrite = true
		}

		var data []byte

		if format == "json" {
			// Build JSON structure for in-memory export
			type exportMessage struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				Thoughts  string `json:"thoughts,omitempty"`
				Timestamp string `json:"timestamp,omitempty"`
			}
			type exportData struct {
				Title    string          `json:"title"`
				Messages []exportMessage `json:"messages"`
			}

			export := exportData{Title: title}
			for _, msg := range messages {
				export.Messages = append(export.Messages, exportMessage{
					Role:    msg.role,
					Content: msg.content,
				})
			}

			var err error
			data, err = jsonMarshalIndent(export, "", "  ")
			if err != nil {
				return exportResultMsg{err: fmt.Errorf("json marshal failed: %w", err)}
			}
		} else {
			// Build markdown for in-memory export
			var md strings.Builder
			if title != "" {
				md.WriteString("# ")
				md.WriteString(title)
				md.WriteString("\n\n")
			}

			for i, msg := range messages {
				if i > 0 {
					md.WriteString("\n---\n\n")
				}
				switch msg.role {
				case "user":
					md.WriteString("**User:**\n\n")
				case "tool":
					md.WriteString("**Tool:**\n\n")
				default:
					md.WriteString("**Gemini:**\n\n")
				}
				md.WriteString(msg.content)
				md.WriteString("\n")
			}

			data = []byte(md.String())
		}

		// Write to file
		if err := os.WriteFile(path, data, 0644); err != nil {
			return exportResultMsg{err: fmt.Errorf("write failed: %w", err)}
		}

		return exportResultMsg{
			path:      path,
			format:    format,
			size:      int64(len(data)),
			overwrite: overwrite,
		}
	}
}

// jsonMarshalIndent is a helper to marshal JSON with indentation
// Note: We use gjson for reading JSON but encoding/json for writing
func jsonMarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// saveMessageToHistory saves a message to the history store if available
func (m *Model) saveMessageToHistory(role, content, thoughts string) {
	if m.historyStore == nil || m.conversation == nil {
		return
	}
	// Errors are logged but not exposed to user (best-effort persistence)
	_ = m.historyStore.AddMessage(m.conversation.ID, role, content, thoughts)
}

// saveMetadataToHistory saves session metadata for conversation resumption
func (m *Model) saveMetadataToHistory() {
	if m.historyStore == nil || m.conversation == nil || m.session == nil {
		return
	}
	cid := m.session.CID()
	rid := m.session.RID()
	rcid := m.session.RCID()
	if cid != "" || rid != "" || rcid != "" {
		_ = m.historyStore.UpdateMetadata(m.conversation.ID, cid, rid, rcid)
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

		switch msg.role {
		case "user":
			// User message
			label := userLabelStyle.Render("⬤ You")
			bubble := userBubbleStyle.Width(bubbleWidth).Render(msg.content)
			content.WriteString(label + "\n" + bubble)

		case "tool":
			// Tool message
			label := toolLabelStyle.Render("Tool")
			bubble := toolBubbleStyle.Width(bubbleWidth).Render(msg.content)
			content.WriteString(label + "\n" + bubble)

		default:
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

			// Render images if present
			if len(msg.images) > 0 {
				imagesContent := renderImageLinks(msg.images, bubbleWidth-4)
				content.WriteString("\n" + imagesContent)
			}
		}
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
}

// renderImageLinks renders image URLs in a styled format
func renderImageLinks(images []models.WebImage, width int) string {
	var sb strings.Builder

	// Header
	header := imageSectionHeaderStyle.Render(fmt.Sprintf("🖼 Images (%d)", len(images)))
	sb.WriteString(header)
	sb.WriteString("\n")

	// Render each image link
	for i, img := range images {
		// Use title if available, otherwise use "Image N"
		title := img.Title
		if title == "" {
			if img.Alt != "" {
				title = img.Alt
			} else {
				title = fmt.Sprintf("Image %d", i+1)
			}
		}

		// Truncate title if too long
		maxTitleLen := width - 10
		if maxTitleLen < 20 {
			maxTitleLen = 20
		}
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}

		// Format: [Title] URL
		titlePart := imageTitleStyle.Render("[" + title + "]")
		urlPart := imageLinkStyle.Render(img.URL)
		sb.WriteString(fmt.Sprintf("  %s %s\n", titlePart, urlPart))
	}

	// Wrap in section style
	return imageSectionStyle.Width(width).Render(sb.String())
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
	ta := createTextarea()

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = loadingStyle

	toolRegistry := defaultToolRegistry()
	toolExecutor := defaultToolExecutor(toolRegistry)
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	return Model{
		client:           client,
		session:          session,
		modelName:        modelName,
		textarea:         ta,
		spinner:          s,
		messages:         []chatMessage{},
		toolRegistry:     toolRegistry,
		toolExecutor:     toolExecutor,
		autoApproveTools: cfg.AutoApproveTools,
	}
}

// RunChatWithConversation starts the chat TUI with a pre-configured session and conversation
func RunChatWithConversation(client api.GeminiClientInterface, session ChatSessionInterface, modelName string, conv *history.Conversation, store HistoryStoreInterface) error {
	return RunChatWithConversationAndGem(client, session, modelName, conv, store, "")
}

// RunChatWithConversationAndGem starts the chat TUI with a pre-configured session, conversation, and initial gem name
func RunChatWithConversationAndGem(client api.GeminiClientInterface, session ChatSessionInterface, modelName string, conv *history.Conversation, store HistoryStoreInterface, gemName string) error {
	return RunChatWithPersona(client, session, modelName, conv, store, gemName, nil)
}

// RunChatWithPersona starts the chat TUI with a pre-configured session, conversation, gem name, and local persona
func RunChatWithPersona(client api.GeminiClientInterface, session ChatSessionInterface, modelName string, conv *history.Conversation, store HistoryStoreInterface, gemName string, persona *config.Persona) error {
	return RunChatWithInitialPrompt(client, session, modelName, conv, store, gemName, persona, "")
}

// RunChatWithInitialPrompt starts the chat TUI with all options including an initial prompt
// If initialPrompt is non-empty, it will be sent automatically when the TUI starts
func RunChatWithInitialPrompt(
	client api.GeminiClientInterface,
	session ChatSessionInterface,
	modelName string,
	conv *history.Conversation,
	store HistoryStoreInterface,
	gemName string,
	persona *config.Persona,
	initialPrompt string,
) error {
	m := NewChatModelWithConversation(client, session, modelName, conv, store)
	m.activeGemName = gemName
	m.persona = persona
	m.initialPrompt = initialPrompt

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}

// NewChatModelWithConversation creates a new chat TUI model with a conversation for persistence
func NewChatModelWithConversation(client api.GeminiClientInterface, session ChatSessionInterface, modelName string, conv *history.Conversation, store HistoryStoreInterface) Model {
	ta := createTextarea()

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = loadingStyle

	toolRegistry := defaultToolRegistry()
	toolExecutor := defaultToolExecutor(toolRegistry)
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

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

	m := Model{
		client:           client,
		session:          session,
		modelName:        modelName,
		textarea:         ta,
		spinner:          s,
		messages:         messages,
		conversation:     conv,
		historyStore:     store,
		toolRegistry:     toolRegistry,
		toolExecutor:     toolExecutor,
		autoApproveTools: cfg.AutoApproveTools,
	}

	// Check if store implements FullHistoryStore for /history command
	if fullStore, ok := store.(FullHistoryStore); ok {
		m.fullHistoryStore = fullStore
	}

	return m
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

// updateToolConfirmation handles input when confirming tool execution
func (m Model) updateToolConfirmation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "y", "Y":
			if m.toolConfirmCall == nil {
				m.confirmingTool = false
				return m, nil
			}
			call := *m.toolConfirmCall
			m.toolConfirmCall = nil
			m.confirmingTool = false
			m.loading = true
			m.animationFrame = 0
			return m, tea.Batch(
				m.executeToolCall(call),
				animationTick(),
			)

		case "n", "N", "esc":
			if m.toolConfirmCall == nil {
				m.confirmingTool = false
				return m, nil
			}
			call := *m.toolConfirmCall
			m.toolConfirmCall = nil
			m.confirmingTool = false
			result := toolexec.NewErrorResult(call.Name, toolexec.NewUserDeniedError(call.Name)).
				WithTiming(time.Now(), time.Now())
			return m, func() tea.Msg {
				return toolExecutionMsg{call: call, result: result}
			}
		}
	}

	return m, nil
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

// loadHistoryForChat returns a command that loads conversations from the history store
func (m Model) loadHistoryForChat() tea.Cmd {
	return func() tea.Msg {
		if m.fullHistoryStore == nil {
			return historyLoadedForChatMsg{err: fmt.Errorf("history store not available")}
		}

		conversations, err := m.fullHistoryStore.ListConversations()
		if err != nil {
			return historyLoadedForChatMsg{err: err}
		}

		return historyLoadedForChatMsg{conversations: conversations}
	}
}

// updateHistorySelection handles updates when in history selection mode
func (m Model) updateHistorySelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case historyLoadedForChatMsg:
		m.historyLoading = false
		if msg.err != nil {
			m.selectingHistory = false
			m.err = msg.err
		} else {
			m.historyList = msg.conversations
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			// Cancel history selection
			m.selectingHistory = false
			m.historyList = nil
			m.historyCursor = 0
			m.historyFilter = ""

		case "up", "k":
			totalItems := len(m.filteredHistory()) + 1 // +1 for "New Conversation"
			if totalItems > 0 {
				m.historyCursor--
				if m.historyCursor < 0 {
					m.historyCursor = totalItems - 1
				}
			}

		case "down", "j":
			totalItems := len(m.filteredHistory()) + 1 // +1 for "New Conversation"
			if totalItems > 0 {
				m.historyCursor++
				if m.historyCursor >= totalItems {
					m.historyCursor = 0
				}
			}

		case "enter":
			if m.historyCursor == 0 {
				// "New Conversation" selected
				return m.startNewConversation()
			}

			// Existing conversation selected
			filtered := m.filteredHistory()
			convIdx := m.historyCursor - 1
			if convIdx >= 0 && convIdx < len(filtered) {
				return m.switchConversation(filtered[convIdx])
			}

		case "backspace":
			if len(m.historyFilter) > 0 {
				m.historyFilter = m.historyFilter[:len(m.historyFilter)-1]
				m.historyCursor = 0
			}

		default:
			// Handle typing for filter (only printable characters)
			if len(msg.String()) == 1 {
				r := []rune(msg.String())[0]
				if r >= ' ' && r <= '~' {
					m.historyFilter += msg.String()
					m.historyCursor = 0
				}
			}
		}
	}

	return m, nil
}

// updateImageSelection handles input when selecting images for download
func (m Model) updateImageSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.imageSelector.width = msg.Width
		m.imageSelector.height = msg.Height

	case tea.KeyMsg:
		// Update the image selector
		var cmd tea.Cmd
		m.imageSelector, cmd = m.imageSelector.Update(msg)

		// Check if selection is complete (confirmed or cancelled)
		if m.imageSelector.IsConfirmed() || m.imageSelector.IsCancelled() {
			m.selectingImages = false

			if m.imageSelector.IsCancelled() {
				// User cancelled
				return m, cmd
			}

			// User confirmed - start download
			indices := m.imageSelector.SelectedIndices()
			if len(indices) == 0 {
				m.err = fmt.Errorf("no images selected")
				return m, cmd
			}

			return m, m.downloadSelectedImages(indices, m.imageSelector.TargetDir())
		}

		return m, cmd
	}

	return m, nil
}

// filteredHistory returns the history list filtered by historyFilter
func (m Model) filteredHistory() []*history.Conversation {
	if m.historyFilter == "" {
		return m.historyList
	}

	filter := strings.ToLower(m.historyFilter)
	var filtered []*history.Conversation
	for _, conv := range m.historyList {
		if strings.Contains(strings.ToLower(conv.Title), filter) ||
			strings.Contains(strings.ToLower(conv.Model), filter) {
			filtered = append(filtered, conv)
		}
	}
	return filtered
}

// renderHistorySelector renders the history selection overlay
func (m Model) renderHistorySelector() string {
	width := m.width - 8
	if width < 40 {
		width = 40
	}

	var content strings.Builder

	// Header
	title := configTitleStyle.Render("📚 Select Conversation")
	if m.conversation != nil {
		title += hintStyle.Render(fmt.Sprintf("  (current: %s)", m.conversation.Title))
	}
	content.WriteString(title)
	content.WriteString("\n\n")

	// Filter input
	if m.historyFilter != "" {
		filterLine := inputLabelStyle.Render("🔍 ") + m.historyFilter + "_"
		content.WriteString(filterLine)
		content.WriteString("\n\n")
	}

	if m.historyLoading {
		content.WriteString(loadingStyle.Render("  Loading conversations..."))
	} else {
		filtered := m.filteredHistory()

		// Show "New Conversation" option first (index 0)
		newConvCursor := "  "
		newConvStyle := configMenuItemStyle
		if m.historyCursor == 0 {
			newConvCursor = configCursorStyle.Render("▸ ")
			newConvStyle = configMenuSelectedStyle
		}
		content.WriteString(fmt.Sprintf("%s%s\n", newConvCursor, newConvStyle.Render("+ New Conversation")))
		content.WriteString("\n")

		if len(filtered) == 0 && len(m.historyList) == 0 {
			content.WriteString(hintStyle.Render("  No saved conversations"))
		} else if len(filtered) == 0 {
			content.WriteString(hintStyle.Render("  No conversations match filter"))
		} else {
			// Show up to 7 conversations (8 - 1 for "New Conversation")
			maxItems := 7
			startIdx := 0
			// Adjust for cursor position relative to filtered list (cursor 0 is "New Conversation")
			effectiveCursor := m.historyCursor - 1
			if effectiveCursor >= maxItems {
				startIdx = effectiveCursor - maxItems + 1
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
				conv := filtered[i]
				cursor := "  "
				titleStyle := configMenuItemStyle
				// Cursor index in the full list (accounting for "New Conversation" at 0)
				if i+1 == m.historyCursor {
					cursor = configCursorStyle.Render("▸ ")
					titleStyle = configMenuSelectedStyle
				}

				// Format time
				timeStr := formatTimeAgo(conv.UpdatedAt)

				// Show model
				modelInfo := configDisabledStyle.Render(fmt.Sprintf("[%s]", conv.Model))

				line := fmt.Sprintf("%s%s %s %s",
					cursor,
					titleStyle.Render(conv.Title),
					modelInfo,
					hintStyle.Render(" - "+timeStr),
				)

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

// formatTimeAgo formats a time as a relative string
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}

// ParsedCommand represents a parsed command from user input
type ParsedCommand struct {
	Command   string // The command name (e.g., "file", "image", "history", "gems")
	Args      string // The arguments after the command
	IsCommand bool   // Whether the input was a command
}

// parseCommand parses user input to detect commands
// Commands start with / and may have arguments separated by space
// Examples:
//   - "/file path/to/file.txt" -> {Command: "file", Args: "path/to/file.txt", IsCommand: true}
//   - "/history" -> {Command: "history", Args: "", IsCommand: true}
//   - "hello world" -> {Command: "", Args: "", IsCommand: false}
func parseCommand(input string) ParsedCommand {
	input = strings.TrimSpace(input)

	// Check if input starts with /
	if !strings.HasPrefix(input, "/") {
		return ParsedCommand{IsCommand: false}
	}

	// Remove the leading /
	cmdLine := input[1:]

	// Split into command and args
	parts := strings.SplitN(cmdLine, " ", 2)
	command := strings.ToLower(parts[0])

	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	return ParsedCommand{
		Command:   command,
		Args:      args,
		IsCommand: true,
	}
}

// switchConversation switches to a different conversation
func (m Model) switchConversation(conv *history.Conversation) (tea.Model, tea.Cmd) {
	// Clear current state
	m.selectingHistory = false
	m.historyList = nil
	m.historyCursor = 0
	m.historyFilter = ""

	// Set the new conversation
	m.conversation = conv

	// Load messages from the conversation
	m.messages = make([]chatMessage, 0, len(conv.Messages))
	for _, msg := range conv.Messages {
		m.messages = append(m.messages, chatMessage{
			role:     msg.Role,
			content:  msg.Content,
			thoughts: msg.Thoughts,
		})
	}

	// Update session metadata for resumption
	if m.session != nil && (conv.CID != "" || conv.RID != "" || conv.RCID != "") {
		m.session.SetMetadata(conv.CID, conv.RID, conv.RCID)
	}

	// Update viewport with new messages
	m.updateViewport()
	m.viewport.GotoBottom()

	return m, nil
}

// startNewConversation starts a fresh conversation
func (m Model) startNewConversation() (tea.Model, tea.Cmd) {
	// Clear current state
	m.selectingHistory = false
	m.historyList = nil
	m.historyCursor = 0
	m.historyFilter = ""

	// Create new conversation if store is available
	if m.fullHistoryStore != nil {
		newConv, err := m.fullHistoryStore.CreateConversation(m.modelName)
		if err == nil {
			m.conversation = newConv
		} else {
			m.err = fmt.Errorf("failed to create conversation: %w", err)
		}
	}

	// Clear messages
	m.messages = []chatMessage{}

	// Reset session metadata
	if m.session != nil {
		m.session.SetMetadata("", "", "")
	}

	// Update viewport
	m.updateViewport()

	return m, nil
}
