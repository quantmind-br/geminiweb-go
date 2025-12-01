package commands

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/render"
)

// Gradient colors for animation
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

var (
	colorText     = lipgloss.Color("#c0caf5")
	colorTextDim  = lipgloss.Color("#565f89")
	colorTextMute = lipgloss.Color("#3b4261")
	colorSuccess  = lipgloss.Color("#9ece6a")
	colorPrimary  = lipgloss.Color("#7aa2f7")
)

// Styles matching the chat TUI
var (
	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				MarginBottom(0)

	assistantBubbleStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Foreground(colorText).
				Padding(0, 1).
				MarginTop(1).
				MarginBottom(1)

	thoughtsStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorTextDim).
			BorderLeft(true).
			Foreground(colorTextDim).
			PaddingLeft(1).
			MarginLeft(1).
			Italic(true)
)

// spinner handles the animated loading indicator
type spinner struct {
	message string
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
	frame   int
}

// newSpinner creates a new animated spinner
func newSpinner(message string) *spinner {
	return &spinner{
		message: message,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// start begins the animation
func (s *spinner) start() {
	go func() {
		defer close(s.done)

		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		// Hide cursor
		fmt.Fprint(os.Stderr, "\033[?25l")

		for {
			select {
			case <-s.stop:
				// Clear line and show cursor
				fmt.Fprint(os.Stderr, "\r\033[K\033[?25h")
				return
			case <-ticker.C:
				s.mu.Lock()
				s.render()
				s.frame++
				s.mu.Unlock()
			}
		}
	}()
}

// render draws the current animation frame
func (s *spinner) render() {
	// Spinner characters
	chars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	barChars := []string{"█", "█", "█", "█", "█", "█", "▓", "▒", "░"}

	// Build spinner character with color
	spinIdx := s.frame % len(chars)
	spinColor := gradientColors[s.frame%len(gradientColors)]
	spinnerChar := lipgloss.NewStyle().Foreground(spinColor).Bold(true).Render(chars[spinIdx])

	// Build animated bar
	barWidth := 16
	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		colorIdx := (i + s.frame) % len(gradientColors)
		charIdx := (i + s.frame/2) % len(barChars)
		style := lipgloss.NewStyle().Foreground(gradientColors[colorIdx])
		bar.WriteString(style.Render(barChars[charIdx]))
	}

	// Build animated dots
	var dots strings.Builder
	numDots := (s.frame / 3) % 4
	for i := 0; i < 3; i++ {
		if i < numDots {
			dotColor := gradientColors[(s.frame+i)%len(gradientColors)]
			dots.WriteString(lipgloss.NewStyle().Foreground(dotColor).Render("●"))
		} else {
			dots.WriteString(lipgloss.NewStyle().Foreground(colorTextMute).Render("○"))
		}
	}

	// Message with color
	msg := lipgloss.NewStyle().Foreground(colorText).Render(s.message)

	// Print animation (clear line first)
	fmt.Fprintf(os.Stderr, "\r\033[K%s %s %s %s", spinnerChar, bar.String(), msg, dots.String())
}

// stopWithSuccess stops the spinner and shows success message
func (s *spinner) stopWithSuccess(message string) {
	close(s.stop)
	<-s.done

	checkmark := lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("✓")
	msg := lipgloss.NewStyle().Foreground(colorSuccess).Render(message)
	fmt.Fprintf(os.Stderr, "%s %s\n", checkmark, msg)
}

// stopWithError stops the spinner and shows error
func (s *spinner) stopWithError() {
	close(s.stop)
	<-s.done
}

// runQuery executes a single query and outputs the response
func runQuery(prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	// Load cookies
	cookies, err := config.LoadCookies()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	modelName := getModel()
	model := models.ModelFromName(modelName)

	// Build client options
	clientOpts := []api.ClientOption{
		api.WithModel(model),
		api.WithAutoRefresh(false),
	}

	// Add browser refresh if enabled
	if browserType, enabled := getBrowserRefresh(); enabled {
		clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
	}

	// Create client
	client, err := api.NewClient(cookies, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Initialize client with animation
	spin := newSpinner("Connecting to Gemini")
	spin.start()

	if err := client.Init(); err != nil {
		spin.stopWithError()
		return fmt.Errorf("failed to initialize: %w", err)
	}
	spin.stopWithSuccess("Connected")

	// Upload image if provided
	var images []*api.UploadedImage
	if imageFlag != "" {
		spin = newSpinner("Uploading image")
		spin.start()

		img, err := client.UploadImage(imageFlag)
		if err != nil {
			spin.stopWithError()
			return fmt.Errorf("failed to upload image: %w", err)
		}
		images = append(images, img)
		spin.stopWithSuccess("Image uploaded")
	}

	// Generate content with animation
	spin = newSpinner("Generating response")
	spin.start()

	opts := &api.GenerateOptions{
		Images: images,
	}

	output, err := client.GenerateContent(prompt, opts)
	if err != nil {
		spin.stopWithError()
		return fmt.Errorf("generation failed: %w", err)
	}
	spin.stopWithSuccess("Done")

	// Add spacing
	fmt.Fprintln(os.Stderr)

	text := output.Text()

	// Copy to clipboard if enabled in config
	cfg, _ := config.LoadConfig()
	if cfg.CopyToClipboard {
		if err := clipboard.WriteAll(text); err != nil {
			// Log warning but don't fail
			warnMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")).Render(
				fmt.Sprintf("⚠ Failed to copy to clipboard: %v", err),
			)
			fmt.Fprintln(os.Stderr, warnMsg)
		} else {
			clipMsg := lipgloss.NewStyle().Foreground(colorSuccess).Render("✓ Copied to clipboard")
			fmt.Fprintln(os.Stderr, clipMsg)
		}
	}

	// Output to file if specified
	if outputFlag != "" {
		if err := os.WriteFile(outputFlag, []byte(text), 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		successMsg := lipgloss.NewStyle().Foreground(colorSuccess).Render(
			fmt.Sprintf("✓ Response saved to %s", outputFlag),
		)
		fmt.Fprintln(os.Stderr, successMsg)
		return nil
	}

	// Get terminal width for proper formatting
	termWidth := getTerminalWidth()
	bubbleWidth := termWidth - 4
	if bubbleWidth < 40 {
		bubbleWidth = 40
	}
	if bubbleWidth > 120 {
		bubbleWidth = 120
	}
	contentWidth := bubbleWidth - 4

	// Print assistant label (similar to chat TUI)
	label := assistantLabelStyle.Render("✦ Gemini")
	fmt.Println(label)

	// Print thoughts if present (with styled border)
	if thoughts := output.Thoughts(); thoughts != "" {
		thoughtsContent := thoughtsStyle.Width(contentWidth).Render("💭 " + thoughts)
		fmt.Println(thoughtsContent)
	}

	// Render markdown for terminal output
	rendered, err := render.MarkdownWithWidth(text, contentWidth)
	if err != nil {
		rendered = text
	}
	// Trim trailing newlines from glamour
	rendered = strings.TrimRight(rendered, "\n")

	// Wrap content in assistant bubble style
	bubble := assistantBubbleStyle.Width(bubbleWidth).Render(rendered)
	fmt.Println(bubble)

	return nil
}


// getTerminalWidth returns the terminal width or a default value
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80 // default width
	}
	return width
}
