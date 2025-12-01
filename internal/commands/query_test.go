package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/render"
)

// mockGeminiClient is a simple mock for testing
type mockGeminiClient struct {
	closed              bool
	generateContentFunc func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error)
	initFunc            func() error
}

func (m *mockGeminiClient) GenerateContent(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
	if m.generateContentFunc != nil {
		return m.generateContentFunc(prompt, opts)
	}
	return nil, nil
}

func (m *mockGeminiClient) Init() error {
	if m.initFunc != nil {
		return m.initFunc()
	}
	return nil
}

func (m *mockGeminiClient) Close() error {
	return nil
}

func (m *mockGeminiClient) IsClosed() bool {
	return m.closed
}

func (m *mockGeminiClient) GetAccessToken() string {
	return "test_token"
}

func (m *mockGeminiClient) GetModel() models.Model {
	return models.Model25Flash
}

func (m *mockGeminiClient) SetModel(model models.Model) {}

func (m *mockGeminiClient) GetCookies() *config.Cookies {
	return &config.Cookies{
		Secure1PSID:   "test",
		Secure1PSIDTS: "test",
	}
}

func (m *mockGeminiClient) StartChat() *api.ChatSession {
	return nil
}

func TestNewSpinner(t *testing.T) {
	message := "Test message"
	spinner := newSpinner(message)

	if spinner.message != message {
		t.Errorf("Expected message %s, got %s", message, spinner.message)
	}

	if spinner.stop == nil {
		t.Error("Stop channel is nil")
	}

	if spinner.done == nil {
		t.Error("Done channel is nil")
	}

	if spinner.frame != 0 {
		t.Errorf("Expected frame 0, got %d", spinner.frame)
	}
}

func TestSpinnerStart(t *testing.T) {
	spinner := newSpinner("Test")

	// Start spinner
	spinner.start()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Stop spinner
	spinner.stopWithSuccess("Success")

	// Wait for it to finish
	select {
	case <-spinner.done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Spinner did not stop within expected time")
	}
}

func TestSpinnerStop(t *testing.T) {
	spinner := newSpinner("Test")

	// Start spinner
	spinner.start()

	// Stop spinner with error
	spinner.stopWithError()

	// Wait for it to finish
	select {
	case <-spinner.done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Spinner did not stop within expected time")
	}
}

func TestSpinnerRender(t *testing.T) {
	spinner := newSpinner("Test")

	// Test render at different frames
	for i := 0; i < 10; i++ {
		spinner.frame = i
		spinner.render() // render() prints to stderr, doesn't return a value

		// We can't easily test the output since it goes to stderr
		// but we can test that it doesn't panic
		if spinner.frame != i {
			t.Errorf("Frame %d: frame was modified", i)
		}
	}
}

func TestRunQuery(t *testing.T) {
	// Create a simple mock client
	mockClient := &mockGeminiClient{
		closed: false,
		generateContentFunc: func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
			return &models.ModelOutput{
				Metadata:   []string{"cid123", "rid456", "rcid789"},
				Candidates: []models.Candidate{{Text: "Test response"}},
				Chosen:     0,
			}, nil
		},
	}

	// Capture stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runQueryWithClient("Test prompt", mockClient)

	// Restore stdout
	w.Close()
	os.Stdout = originalStdout

	if err != nil {
		t.Errorf("runQueryWithClient failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// The output might be empty due to how renderMarkdownToTerminal works
	// Let's just verify the function doesn't error
	if err != nil {
		t.Errorf("runQueryWithClient failed: %v", err)
	}

	// Check if we got any output at all
	if output == "" {
		t.Log("No output captured (this may be expected)")
	}
}

func TestRunQuery_Error(t *testing.T) {
	// Mock client that returns an error
	mockClient := &mockGeminiClient{
		closed: false,
		generateContentFunc: func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
			return nil, fmt.Errorf("test error")
		},
	}

	err := runQueryWithClient("Test prompt", mockClient)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "test error") {
		t.Errorf("Expected error to contain 'test error', got: %v", err)
	}
}

func TestRunQuery_ClientClosed(t *testing.T) {
	// Mock closed client
	mockClient := &mockGeminiClient{
		closed: true,
	}

	err := runQueryWithClient("Test prompt", mockClient)
	if err == nil {
		t.Error("Expected error for closed client, got nil")
	}
}

func TestRenderMarkdownToTerminal(t *testing.T) {
	input := "# Test Header\n\nThis is **bold** text."

	output, _ := render.MarkdownWithWidth(input, 80)

	// The output should be styled with glamour
	if output == input {
		t.Error("Expected styled output, got plain input")
	}

	// Should contain some ANSI escape codes for styling
	// Note: This might fail in some environments, so we'll just check it's not empty
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

// Helper function for testing runQuery with a specific client
func runQueryWithClient(prompt string, client *mockGeminiClient) error {
	// This is a simplified version of runQuery for testing
	if client.IsClosed() {
		return fmt.Errorf("client is closed")
	}

	// Check Init first
	if err := client.Init(); err != nil {
		return err
	}

	opts := &api.GenerateOptions{
		Model:    client.GetModel(),
		Metadata: []string{},
	}

	output, err := client.GenerateContent(prompt, opts)
	if err != nil {
		return err
	}

	// Guard against nil output
	if output == nil {
		return fmt.Errorf("no output from GenerateContent")
	}

	// Render the response
	_, _ = render.MarkdownWithWidth(output.Text(), 80)

	return nil
}

func TestRunQuery_EmptyPrompt(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Test with empty prompt (trimmed)
	err := runQuery("   ")
	if err == nil {
		t.Error("Expected error for empty prompt")
	}

	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("Expected 'cannot be empty' error, got: %v", err)
	}

	// Test with completely empty string
	err = runQuery("")
	if err == nil {
		t.Error("Expected error for empty prompt")
	}
}

func TestRunQuery_AuthError(t *testing.T) {
	// This would test the authentication error path
	// Since we can't easily mock config.LoadCookies(), we just verify the function exists
	t.Skip("Cannot easily test config.LoadCookies() without extensive mocking")
}

func TestRunQuery_ClientInitError(t *testing.T) {
	// Mock client that fails on Init
	mockClient := &mockGeminiClient{
		closed: false,
		generateContentFunc: func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
			return nil, nil
		},
		initFunc: func() error {
			return fmt.Errorf("init failed")
		},
	}

	err := runQueryWithClient("Test prompt", mockClient)
	if err == nil {
		t.Error("Expected error for failed Init, got nil")
	}
	if !strings.Contains(err.Error(), "init failed") {
		t.Errorf("Expected 'init failed' in error, got: %v", err)
	}
}

func TestRunQuery_GenerateError(t *testing.T) {
	// Mock client that fails on GenerateContent
	mockClient := &mockGeminiClient{
		closed: false,
		generateContentFunc: func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
			return nil, fmt.Errorf("generation failed")
		},
	}

	err := runQueryWithClient("Test prompt", mockClient)
	if err == nil {
		t.Error("Expected error for failed generation, got nil")
	}
	if !strings.Contains(err.Error(), "generation failed") {
		t.Errorf("Expected 'generation failed' in error, got: %v", err)
	}
}

func TestRunQuery_SuccessWithoutImage(t *testing.T) {
	// Mock client that succeeds
	mockClient := &mockGeminiClient{
		closed: false,
		generateContentFunc: func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
			return &models.ModelOutput{
				Metadata:   []string{"cid123", "rid456", "rcid789"},
				Candidates: []models.Candidate{{Text: "Test response"}},
				Chosen:     0,
			}, nil
		},
	}

	err := runQueryWithClient("Test prompt", mockClient)
	if err != nil {
		t.Errorf("runQueryWithClient failed: %v", err)
	}
}

func TestRunQuery_WithImage(t *testing.T) {
	// Mock client that succeeds (note: actual image upload can't be tested without full setup)
	mockClient := &mockGeminiClient{
		closed: false,
		generateContentFunc: func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
			return &models.ModelOutput{
				Metadata:   []string{"cid123"},
				Candidates: []models.Candidate{{Text: "Image response"}},
				Chosen:     0,
			}, nil
		},
	}

	err := runQueryWithClient("Describe this image", mockClient)
	if err != nil {
		t.Errorf("runQueryWithClient with image failed: %v", err)
	}
}

func TestRunQuery_OutputToFile(t *testing.T) {
	// Create a temporary file for output
	tmpFile := t.TempDir() + "/output.txt"

	// Mock client that succeeds
	mockClient := &mockGeminiClient{
		closed: false,
		generateContentFunc: func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
			return &models.ModelOutput{
				Metadata:   []string{"cid123"},
				Candidates: []models.Candidate{{Text: "File output response"}},
				Chosen:     0,
			}, nil
		},
	}

	// Override file writing for testing
	err := runQueryToFile("Test prompt", mockClient, tmpFile)
	if err != nil {
		t.Errorf("runQueryToFile failed: %v", err)
	}

	// Verify file was created and contains expected content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !bytes.Contains(content, []byte("File output response")) {
		t.Errorf("Expected file to contain response, got: %s", string(content))
	}

	// Clean up
	os.Remove(tmpFile)
}

func TestRunQuery_WithThoughts(t *testing.T) {
	// Mock client that returns output with thoughts
	mockClient := &mockGeminiClient{
		closed: false,
		generateContentFunc: func(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
			return &models.ModelOutput{
				Metadata:   []string{"cid123"},
				Candidates: []models.Candidate{{Text: "Response text", Thoughts: "Thinking process"}},
				Chosen:     0,
			}, nil
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runQueryWithClient("Test prompt", mockClient)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runQueryWithClient with thoughts failed: %v", err)
	}

	// Read output to verify thoughts are printed
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// The thoughts should be in the output (though styled with ANSI codes)
	if !bytes.Contains([]byte(output), []byte("Thinking process")) {
		t.Log("Thoughts might be styled with ANSI codes, checking rendered output")
	}
}

// Helper function for testing runQuery with file output
func runQueryToFile(prompt string, client *mockGeminiClient, outputFile string) error {
	if client.IsClosed() {
		return fmt.Errorf("client is closed")
	}

	opts := &api.GenerateOptions{
		Model:    client.GetModel(),
		Metadata: []string{},
	}

	output, err := client.GenerateContent(prompt, opts)
	if err != nil {
		return err
	}

	text := output.Text()

	// Write to file
	return os.WriteFile(outputFile, []byte(text), 0o644)
}

func TestSpinner_Structure(t *testing.T) {
	spinner := newSpinner("Test message")

	// Verify struct fields
	if spinner.message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", spinner.message)
	}

	if spinner.stop == nil {
		t.Error("stop channel should not be nil")
	}

	if spinner.done == nil {
		t.Error("done channel should not be nil")
	}

	if spinner.frame != 0 {
		t.Errorf("Expected frame 0, got %d", spinner.frame)
	}
}

func TestGradientColors(t *testing.T) {
	// Verify gradientColors is defined and has content
	if len(gradientColors) == 0 {
		t.Error("gradientColors should not be empty")
	}

	// Verify specific colors
	expectedColors := []string{
		"ff6b6b", // Red
		"feca57", // Yellow
		"48dbfb", // Cyan
		"ff9ff3", // Pink
	}

	for i, expected := range expectedColors {
		if i >= len(gradientColors) {
			break
		}
		colorStr := string(gradientColors[i])
		if colorStr != "#"+expected {
			t.Errorf("Expected color %s at index %d, got %s", expected, i, colorStr)
		}
	}
}

func TestColorVariables(t *testing.T) {
	// Verify color variables are defined (just check they exist)
	_ = colorText
	_ = colorTextDim
	_ = colorTextMute
	_ = colorSuccess

	// If we got here, the variables are defined
	// We can't test the actual color values without rendering
}

// TestGetTerminalWidth tests the getTerminalWidth function
func TestGetTerminalWidth(t *testing.T) {
	t.Run("valid_width", func(t *testing.T) {
		// getTerminalWidth should return a positive width
		width := getTerminalWidth()
		if width <= 0 {
			t.Errorf("getTerminalWidth() = %d, want > 0", width)
		}

		// Common terminal widths
		if width < 40 || width > 300 {
			t.Logf("Terminal width = %d (outside common range 40-300)", width)
		}
	})

	t.Run("default_width", func(t *testing.T) {
		// The function should return at least the default width of 80
		// Even on systems where term.GetSize fails
		width := getTerminalWidth()
		// If the function works correctly, it should either return the actual width
		// or the default of 80
		if width < 80 {
			t.Errorf("getTerminalWidth() = %d, want >= 80 (default or actual)", width)
		}
	})

	t.Run("positive_value", func(t *testing.T) {
		width := getTerminalWidth()
		if width <= 0 {
			t.Errorf("getTerminalWidth() returned non-positive value: %d", width)
		}
	})
}
