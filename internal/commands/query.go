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
	apierrors "github.com/diogo/geminiweb/internal/errors"
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
	stopped bool // Flag to prevent double-close
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

// stopOnce safely closes the stop channel only once
func (s *spinner) stopOnce() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped {
		close(s.stop)
		s.stopped = true
	}
}

// stopWithSuccess stops the spinner and shows success message
func (s *spinner) stopWithSuccess(message string) {
	s.stopOnce()
	<-s.done

	checkmark := lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("✓")
	msg := lipgloss.NewStyle().Foreground(colorSuccess).Render(message)
	fmt.Fprintf(os.Stderr, "%s %s\n", checkmark, msg)
}

// stopWithError stops the spinner and shows error
func (s *spinner) stopWithError() {
	s.stopOnce()
	<-s.done
}

// runQuery executes a single query and outputs the response
// If rawOutput is true, only the raw response text is printed without decoration
func runQuery(prompt string, rawOutput bool) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	// Load config for verbose logging
	cfg, _ := config.LoadConfig()

	modelName := getModel()
	model := models.ModelFromName(modelName)

	// Apply persona system prompt if specified
	var persona *config.Persona
	if personaFlag != "" {
		var err error
		persona, err = config.GetPersona(personaFlag)
		if err != nil {
			return fmt.Errorf("failed to load persona '%s': %w", personaFlag, err)
		}
		if cfg.Verbose && !rawOutput {
			fmt.Fprintf(os.Stderr, "[verbose] Using persona: %s\n", persona.Name)
		}
	} else {
		// Check for default persona (if not "default")
		defaultPersona, err := config.GetDefaultPersona()
		if err == nil && defaultPersona != nil && defaultPersona.Name != "default" && defaultPersona.SystemPrompt != "" {
			persona = defaultPersona
			if cfg.Verbose && !rawOutput {
				fmt.Fprintf(os.Stderr, "[verbose] Using default persona: %s\n", persona.Name)
			}
		}
	}

	// Verbose: show model being used
	if cfg.Verbose && !rawOutput {
		fmt.Fprintf(os.Stderr, "[verbose] Model: %s\n", modelName)
	}

	// Build client options
	clientOpts := []api.ClientOption{
		api.WithModel(model),
		api.WithAutoRefresh(false),
	}

	// Add browser refresh if enabled (also enables silent auto-login fallback)
	if browserType, enabled := getBrowserRefresh(); enabled {
		clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
	}

	// Add auto-close options from config (less relevant for single queries, but consistent)
	if cfg.AutoClose {
		clientOpts = append(clientOpts,
			api.WithAutoClose(true),
			api.WithCloseDelay(time.Duration(cfg.CloseDelay)*time.Second),
			api.WithAutoReInit(cfg.AutoReInit),
		)
	}

	// Create client with nil cookies - Init() will load from disk or browser
	client, err := api.NewClient(nil, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Initialize client
	// Init() handles cookie loading from disk and browser fallback
	var spin *spinner
	if !rawOutput {
		spin = newSpinner("Connecting to Gemini")
		spin.start()
	}

	if err := client.Init(); err != nil {
		if !rawOutput {
			spin.stopWithError()
			fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Failed to initialize"))
		}
		return fmt.Errorf("failed to initialize: %w", err)
	}
	if !rawOutput {
		spin.stopWithSuccess("Connected")
	}

	// Resolve gem if provided
	var gemID string
	if gemFlag != "" {
		if !rawOutput {
			spin = newSpinner("Loading gems")
			spin.start()
		}

		gem, err := resolveGem(client, gemFlag)
		if err != nil {
			if !rawOutput {
				spin.stopWithError()
				fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Gem resolution failed"))
			}
			return fmt.Errorf("gem resolution failed: %w", err)
		}
		gemID = gem.ID
		if !rawOutput {
			spin.stopWithSuccess(fmt.Sprintf("Using gem: %s", gem.Name))
		}
	}

	// Upload image if provided
	var images []*api.UploadedImage
	if imageFlag != "" {
		if !rawOutput {
			spin = newSpinner("Uploading image")
			spin.start()
		}

		img, err := client.UploadImage(imageFlag)
		if err != nil {
			if !rawOutput {
				spin.stopWithError()
				fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Failed to upload image"))
			}
			return fmt.Errorf("failed to upload image: %w", err)
		}
		images = append(images, img)
		if !rawOutput {
			spin.stopWithSuccess("Image uploaded")
		}
	}

	// Apply persona system prompt to the user's message
	if persona != nil && persona.SystemPrompt != "" {
		prompt = config.FormatSystemPrompt(persona, prompt)
	}

	// For large prompts, upload as a file and use a reference prompt
	var actualPrompt string
	if len(prompt) > api.LargePromptThreshold {
		if !rawOutput {
			spin = newSpinner("Uploading large prompt as file")
			spin.start()
		}

		// Upload the prompt as a text file
		uploadedFile, err := client.UploadText(prompt, "prompt.md")
		if err != nil {
			if !rawOutput {
				spin.stopWithError()
				fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Failed to upload prompt"))
			}
			return fmt.Errorf("failed to upload prompt: %w", err)
		}

		// Add as an "image" (the API treats uploaded files similarly)
		images = append(images, uploadedFile)

		// Use minimal prompt - the uploaded file already contains the user's instructions
		actualPrompt = "."
		if !rawOutput {
			spin.stopWithSuccess(fmt.Sprintf("Prompt uploaded (%d KB)", len(prompt)/1024))
		}
	} else {
		actualPrompt = prompt
	}

	// Generate content
	if !rawOutput {
		spin = newSpinner("Generating response")
		spin.start()
	}

	opts := &api.GenerateOptions{
		Files: images,
		GemID: gemID,
	}

	// Track request timing for verbose output
	startTime := time.Now()
	output, err := client.GenerateContent(actualPrompt, opts)
	requestDuration := time.Since(startTime)

	if err != nil {
		if !rawOutput {
			spin.stopWithError()
			fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Generation failed"))
		}
		return fmt.Errorf("generation failed: %w", err)
	}
	if !rawOutput {
		spin.stopWithSuccess("Done")
	}

	// Verbose: show request timing
	if cfg.Verbose && !rawOutput {
		fmt.Fprintf(os.Stderr, "[verbose] Request took %s\n", requestDuration.Round(time.Millisecond))
		if candidate := output.ChosenCandidate(); candidate != nil {
			if candidate.Thoughts != "" {
				fmt.Fprintf(os.Stderr, "[verbose] Response includes thoughts\n")
			}
			if len(candidate.WebImages) > 0 {
				fmt.Fprintf(os.Stderr, "[verbose] Response includes %d web images\n", len(candidate.WebImages))
			}
			if len(candidate.GeneratedImages) > 0 {
				fmt.Fprintf(os.Stderr, "[verbose] Response includes %d generated images\n", len(candidate.GeneratedImages))
			}
		}
	}

	// Download images if --save-images flag is set
	if saveImagesFlag != "" && output != nil {
		allImages := output.Images()
		if len(allImages) > 0 {
			if !rawOutput {
				spin = newSpinner("Downloading images")
				spin.start()
			}

			opts := api.ImageDownloadOptions{
				Directory: saveImagesFlag,
				FullSize:  true,
			}
			paths, err := client.DownloadAllImages(output, opts)
			if err != nil {
				if !rawOutput {
					spin.stopWithError()
					fmt.Fprintf(os.Stderr, "Warning: failed to save some images: %v\n", err)
				}
			} else if len(paths) > 0 {
				if !rawOutput {
					spin.stopWithSuccess(fmt.Sprintf("Saved %d images to %s", len(paths), saveImagesFlag))
				}
			} else if !rawOutput {
				spin.stopWithSuccess("No images to download")
			}
		}
	}

	text := output.Text()

	// Raw output mode: output only the raw text
	if rawOutput {
		// Output to file if specified
		if outputFlag != "" {
			if err := os.WriteFile(outputFlag, []byte(text), 0o644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			return nil
		}
		// Output raw text to stdout
		fmt.Print(text)
		return nil
	}

	// Decorated output mode (TTY)
	// Add spacing
	fmt.Fprintln(os.Stderr)

	// Copy to clipboard if enabled in config (cfg was loaded at function start)
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

	// Render markdown for terminal output using user config
	renderOpts := render.LoadOptionsFromConfigWithWidth(contentWidth)
	rendered, err := render.Markdown(text, renderOpts)
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

// isStdoutTTY returns true if stdout is connected to a terminal
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// formatErrorMessage formats an error with additional context from structured errors
func formatErrorMessage(err error, context string) string {
	if err == nil {
		return ""
	}

	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	dimStyle := lipgloss.NewStyle().Foreground(colorTextDim)

	var sb strings.Builder
	sb.WriteString(errorStyle.Render(fmt.Sprintf("✗ %s: %v", context, err)))

	// Extract additional context from structured errors
	if status := apierrors.GetHTTPStatus(err); status > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  HTTP Status: %d", status)))
	}

	if code := apierrors.GetErrorCode(err); code != apierrors.ErrCodeUnknown {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  Error Code: %d (%s)", code, code.String())))
	}

	if endpoint := apierrors.GetEndpoint(err); endpoint != "" {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  Endpoint: %s", endpoint)))
	}

	// Show response body if available (contains detailed error info like blocking URLs)
	if body := apierrors.GetResponseBody(err); body != "" {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n\n  %s", strings.ReplaceAll(body, "\n", "\n  "))))
	} else {
		// Provide helpful hints based on error type only if no body
		switch {
		case apierrors.IsAuthError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: Try running 'geminiweb auto-login' to refresh your session"))
		case apierrors.IsRateLimitError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: You've hit the usage limit. Try again later or use a different model"))
		case apierrors.IsNetworkError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: Check your internet connection and try again"))
		case apierrors.IsTimeoutError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: Request timed out. Try again or check your connection"))
		case apierrors.IsUploadError(err):
			sb.WriteString(dimStyle.Render("\n  Hint: File upload failed. Check the file exists and is accessible"))
		}
	}

	return sb.String()
}
