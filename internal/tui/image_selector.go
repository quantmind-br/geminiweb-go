package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/models"
)

// ImageSelectorModel represents the image selector TUI state
type ImageSelectorModel struct {
	// Data
	images    []models.WebImage
	targetDir string

	// Selection state
	selected map[int]bool
	cursor   int

	// State
	confirmed bool
	cancelled bool

	// Dimensions
	width  int
	height int
	ready  bool
}

// NewImageSelectorModel creates a new image selector model
func NewImageSelectorModel(images []models.WebImage, targetDir string) ImageSelectorModel {
	return ImageSelectorModel{
		images:    images,
		targetDir: targetDir,
		selected:  make(map[int]bool),
		cursor:    0,
	}
}

// Init initializes the model
func (m ImageSelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m ImageSelectorModel) Update(msg tea.Msg) (ImageSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.cancelled = true
			m.confirmed = true
			return m, nil

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.images) - 1
			}

		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.images) {
				m.cursor = 0
			}

		case " ": // Space - toggle selection
			if m.cursor >= 0 && m.cursor < len(m.images) {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}

		case "a": // Select all
			for i := range m.images {
				m.selected[i] = true
			}

		case "n": // Select none
			m.selected = make(map[int]bool)

		case "enter":
			m.confirmed = true
			return m, nil

		case "home", "g":
			m.cursor = 0

		case "end", "G":
			m.cursor = len(m.images) - 1
		}
	}

	return m, nil
}

// View renders the TUI
func (m ImageSelectorModel) View() string {
	if !m.ready {
		return "  Initializing..."
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	b.WriteString(headerStyle.Render("Select images to download"))
	b.WriteString("\n\n")

	// Calculate visible area
	maxVisible := m.height - 8 // Reserve space for header and footer
	if maxVisible < 3 {
		maxVisible = 3
	}

	// Calculate scroll offset
	startIdx := 0
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(m.images) {
		endIdx = len(m.images)
	}

	// Render items
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true)

	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242"))

	for i := startIdx; i < endIdx; i++ {
		img := m.images[i]

		// Cursor indicator
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		// Selection checkbox
		checkbox := "[ ] "
		if m.selected[i] {
			checkbox = selectedStyle.Render("[x] ")
		}

		// Title (truncate if needed)
		title := img.Title
		if title == "" {
			title = img.Alt
		}
		if title == "" {
			title = fmt.Sprintf("Image %d", i+1)
		}

		maxTitleLen := m.width - 10
		if maxTitleLen < 20 {
			maxTitleLen = 20
		}
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}

		// URL (truncated)
		url := img.URL
		maxURLLen := m.width - 8
		if maxURLLen < 30 {
			maxURLLen = 30
		}
		if len(url) > maxURLLen {
			url = url[:maxURLLen-3] + "..."
		}

		if i == m.cursor {
			b.WriteString(cursor + checkbox + cursorStyle.Render(title) + "\n")
			b.WriteString("      " + dimStyle.Render(url) + "\n")
		} else {
			b.WriteString(cursor + checkbox + title + "\n")
			b.WriteString("      " + dimStyle.Render(url) + "\n")
		}
	}

	// Footer with keybindings
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242"))

	selectedCount := m.SelectedCount()
	countInfo := fmt.Sprintf("  %d of %d selected", selectedCount, len(m.images))
	b.WriteString(footerStyle.Render(countInfo))
	b.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	help := "  Space: toggle  a: all  n: none  Enter: download  Esc: cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

// SelectedCount returns the number of selected images
func (m ImageSelectorModel) SelectedCount() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

// SelectedIndices returns the indices of selected images
func (m ImageSelectorModel) SelectedIndices() []int {
	var indices []int
	for i := 0; i < len(m.images); i++ {
		if m.selected[i] {
			indices = append(indices, i)
		}
	}
	return indices
}

// IsConfirmed returns whether the user confirmed the selection
func (m ImageSelectorModel) IsConfirmed() bool {
	return m.confirmed && !m.cancelled
}

// IsCancelled returns whether the user cancelled
func (m ImageSelectorModel) IsCancelled() bool {
	return m.cancelled
}

// TargetDir returns the target directory for downloads
func (m ImageSelectorModel) TargetDir() string {
	return m.targetDir
}
