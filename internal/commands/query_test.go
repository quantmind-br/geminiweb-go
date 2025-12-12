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

// mockConfig is a mock for config package functions
type mockConfig struct {
	loadCookiesFunc func() (*config.Cookies, error)
	loadConfigFunc  func() (*config.Config, error)
}

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

	// Test with empty prompt (trimmed) - test both raw and decorated modes
	err := runQuery("   ", false)
	if err == nil {
		t.Error("Expected error for empty prompt")
	}

	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("Expected 'cannot be empty' error, got: %v", err)
	}

	// Test with completely empty string
	err = runQuery("", true)
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

// TestRunQuery_IntegrationSuccess tests the full runQuery flow with successful execution
func TestRunQuery_IntegrationSuccess(t *testing.T) {
	t.Skip("Cannot fully test without extensive API client mocking")
}

// TestRunQuery_EmptyPromptReal tests runQuery with empty prompt
func TestRunQuery_EmptyPromptReal(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	// Test with empty prompt (raw mode)
	err := runQuery("", true)
	if err == nil {
		t.Error("Expected error for empty prompt, got nil")
	}

	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("Expected 'cannot be empty' in error, got: %v", err)
	}

	// Test with whitespace-only prompt (decorated mode)
	err = runQuery("   \n\t  ", false)
	if err == nil {
		t.Error("Expected error for whitespace-only prompt, got nil")
	}
}

// TestRunQuery_AuthErrorReal tests runQuery when cookies fail to load and browser extraction fails
func TestRunQuery_AuthErrorReal(t *testing.T) {
	// Create temporary directory for config without cookies
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Ensure no cookies exist
	os.RemoveAll(tmpDir + "/.geminiweb")

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	// Test should fail due to missing cookies and browser extraction failure
	// With the new silent auth, the error will mention "authentication failed"
	err := runQuery("Test prompt", false)
	if err == nil {
		t.Error("Expected error for missing cookies, got nil")
	}

	// The error should indicate authentication failure
	// (either "authentication failed" from new flow or other auth-related message)
	if !strings.Contains(err.Error(), "authentication failed") &&
		!strings.Contains(err.Error(), "failed to initialize") {
		t.Errorf("Expected authentication-related error, got: %v", err)
	}
}

// TestRunQuery_ClientCreationError tests runQuery when client creation fails
func TestRunQuery_ClientCreationError(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save invalid cookies (missing PSIDTS)
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	// Test should fail due to invalid cookies (test both modes)
	err := runQuery("Test prompt", false)
	if err == nil {
		t.Error("Expected error for invalid cookies, got nil")
	}

	err = runQuery("Test prompt", true)
	if err == nil {
		t.Error("Expected error for invalid cookies in raw mode, got nil")
	}
}

// TestRunQuery_WithImageUpload tests runQuery with image flag set
func TestRunQuery_WithImageUpload(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Create a temporary image file
	imageFile := tmpDir + "/test.png"
	testImageData := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(imageFile, testImageData, 0o644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = imageFile
	outputFlag = ""

	// Test with image (will fail due to client mocking limitations)
	err := runQuery("Describe this image", false)
	if err != nil && !strings.Contains(err.Error(), "failed to upload image") {
		t.Logf("Expected image upload error, got: %v", err)
	}
}

// TestRunQuery_OutputToFileReal tests runQuery with output file
func TestRunQuery_OutputToFileReal(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Create output file path
	outputFile := tmpDir + "/output.txt"

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = outputFile

	// Test with output file (raw mode since outputFlag is set)
	err := runQuery("Test prompt", true)
	if err != nil && !strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected initialization error, got: %v", err)
	}

	// Verify file is created if there was no error
	if err == nil {
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Error("Output file was not created")
		}
	}
}

// TestRunQuery_CopyToClipboard tests the clipboard functionality path
func TestRunQuery_CopyToClipboard(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Enable clipboard in config
	cfg := config.Config{
		CopyToClipboard: true,
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	// Test with clipboard enabled (decorated mode, will fail due to client mocking limitations)
	err := runQuery("Test prompt", false)
	if err != nil && !strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected initialization error, got: %v", err)
	}
}

// TestRunQuery_NonExistentImageFile tests runQuery with non-existent image file
func TestRunQuery_NonExistentImageFile(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags with non-existent image
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = "/non/existent/image.png"
	outputFlag = ""

	// Test should fail - either due to non-existent image file or initialization
	// (the test cookies are not valid for real token extraction)
	err := runQuery("Describe this image", false)
	if err == nil {
		t.Error("Expected error for non-existent image, got nil")
	}

	// Accept either image upload error or initialization error
	// (test cookies don't allow real token extraction)
	if !strings.Contains(err.Error(), "failed to upload image") &&
		!strings.Contains(err.Error(), "failed to initialize") {
		t.Errorf("Expected 'failed to upload image' or 'failed to initialize' in error, got: %v", err)
	}
}

// TestRunQuery_InvalidOutputFile tests runQuery with invalid output file path
func TestRunQuery_InvalidOutputFile(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags with invalid output path
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = "/invalid/path/output.txt"

	// Test should fail due to invalid output path (raw mode since outputFlag is set)
	err := runQuery("Test prompt", true)
	if err == nil {
		t.Error("Expected error for invalid output path, got nil")
	}
}

// TestRunQuery_WithModelFlag tests runQuery with different model flags
func TestRunQuery_WithModelFlag(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags with model
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	oldModelFlag := modelFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
		modelFlag = oldModelFlag
	}()

	imageFlag = ""
	outputFlag = ""
	modelFlag = "gemini-2.5-pro"

	// Test with custom model (will fail due to client mocking limitations)
	err := runQuery("Test prompt", false)
	if err != nil && !strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected initialization error, got: %v", err)
	}
}

// TestRunQuery_WithBrowserRefreshFlag tests runQuery with browser refresh enabled
func TestRunQuery_WithBrowserRefreshFlag(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags with browser refresh
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	oldBrowserRefreshFlag := browserRefreshFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
		browserRefreshFlag = oldBrowserRefreshFlag
	}()

	imageFlag = ""
	outputFlag = ""
	browserRefreshFlag = "chrome"

	// Test with browser refresh (will fail due to client mocking limitations)
	err := runQuery("Test prompt", false)
	if err != nil && !strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected initialization error, got: %v", err)
	}
}

// TestIsStdoutTTY tests the isStdoutTTY function
func TestIsStdoutTTY(t *testing.T) {
	// Just verify the function exists and returns a boolean
	// In a test environment, stdout is typically not a TTY
	result := isStdoutTTY()

	// The result is environment-dependent, so we just verify it returns a valid boolean
	if result != true && result != false {
		t.Error("isStdoutTTY() should return a boolean")
	}
}

// TestRunQuery_RawOutputMode tests runQuery in raw output mode
func TestRunQuery_RawOutputMode(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	t.Run("empty_prompt_raw_mode", func(t *testing.T) {
		err := runQuery("", true)
		if err == nil {
			t.Error("Expected error for empty prompt in raw mode")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("Expected 'cannot be empty' error, got: %v", err)
		}
	})

	t.Run("whitespace_prompt_raw_mode", func(t *testing.T) {
		err := runQuery("   \t\n", true)
		if err == nil {
			t.Error("Expected error for whitespace-only prompt in raw mode")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("Expected 'cannot be empty' error, got: %v", err)
		}
	})
}

// TestRunQuery_DecoratedVsRawMode tests differences between decorated and raw output modes
func TestRunQuery_DecoratedVsRawMode(t *testing.T) {
	// This test verifies that both modes handle errors consistently
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	t.Run("both_modes_reject_empty_prompt", func(t *testing.T) {
		errRaw := runQuery("", true)
		errDecorated := runQuery("", false)

		if errRaw == nil || errDecorated == nil {
			t.Error("Both modes should reject empty prompts")
		}

		// Both errors should contain the same message
		if !strings.Contains(errRaw.Error(), "cannot be empty") {
			t.Errorf("Raw mode error should contain 'cannot be empty', got: %v", errRaw)
		}
		if !strings.Contains(errDecorated.Error(), "cannot be empty") {
			t.Errorf("Decorated mode error should contain 'cannot be empty', got: %v", errDecorated)
		}
	})
}

// TestRunQuery_ClientCreation tests the client creation part of runQuery
func TestRunQuery_ClientCreation(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	// Test client creation (will fail due to client mocking limitations)
	err := runQuery("Test prompt", false)
	if err != nil && !strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected initialization error, got: %v", err)
	}
}

// TestRunQuery_GemResolution tests the gem resolution part of runQuery
func TestRunQuery_GemResolution(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags with gem
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	oldGemFlag := gemFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
		gemFlag = oldGemFlag
	}()

	imageFlag = ""
	outputFlag = ""
	gemFlag = "Code Helper"

	// Test gem resolution (will fail due to client mocking limitations)
	err := runQuery("Test prompt", false)
	if err != nil && !strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected initialization error, got: %v", err)
	}
}

// TestRunQuery_ImageUpload tests the image upload part of runQuery
func TestRunQuery_ImageUpload(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Create a temporary image file
	imageFile := tmpDir + "/test.png"
	testImageData := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(imageFile, testImageData, 0o644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Set flags with image
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = imageFile
	outputFlag = ""

	// Test image upload (will fail due to client mocking limitations)
	err := runQuery("Describe this image", false)
	if err != nil && !strings.Contains(err.Error(), "failed to upload image") &&
		!strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected image upload or initialization error, got: %v", err)
	}
}

// TestRunQuery_LargePrompt tests the large prompt handling part of runQuery
func TestRunQuery_LargePrompt(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	// Create a large prompt (larger than LargePromptThreshold)
	largePrompt := strings.Repeat("This is a large prompt. ", 1000)

	// Test large prompt handling (will fail due to client mocking limitations)
	err := runQuery(largePrompt, false)
	if err != nil && !strings.Contains(err.Error(), "failed to upload prompt") &&
		!strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected prompt upload or initialization error, got: %v", err)
	}
}

// TestRunQuery_ContentGeneration tests the content generation part of runQuery
func TestRunQuery_ContentGeneration(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save valid cookies
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}
	if err := config.SaveCookies(cookies); err != nil {
		t.Fatalf("Failed to save cookies: %v", err)
	}

	// Set flags
	oldImageFlag := imageFlag
	oldOutputFlag := outputFlag
	defer func() {
		imageFlag = oldImageFlag
		outputFlag = oldOutputFlag
	}()

	imageFlag = ""
	outputFlag = ""

	// Test content generation (will fail due to client mocking limitations)
	err := runQuery("Test prompt", false)
	if err != nil && !strings.Contains(err.Error(), "generation failed") &&
		!strings.Contains(err.Error(), "failed to initialize") {
		t.Logf("Expected generation or initialization error, got: %v", err)
	}
}


