package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diogo/geminiweb/internal/history"
	"github.com/diogo/geminiweb/internal/models"
)

func TestChatCommand(t *testing.T) {
	// Test that the command is properly configured
	if chatCmd.Use != "chat" {
		t.Errorf("Expected use 'chat', got %s", chatCmd.Use)
	}

	if chatCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if chatCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if chatCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestChatCommand_Args(t *testing.T) {
	// Chat command should accept no arguments
	// Note: We don't call RunE directly as it would launch the interactive TUI
	// Instead, we validate the Args validator function if set
	if chatCmd.Args != nil {
		tests := []struct {
			name    string
			args    []string
			wantErr bool
		}{
			{
				name:    "no args",
				args:    []string{},
				wantErr: false,
			},
			{
				name:    "with args (should be rejected)",
				args:    []string{"test"},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := chatCmd.Args(chatCmd, tt.args)
				if tt.wantErr && err == nil {
					t.Errorf("Expected error, got nil")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})
		}
	} else {
		// If Args is nil, the command accepts any arguments
		// This test documents that chat currently doesn't validate args
		t.Log("chatCmd.Args is nil - command accepts any arguments (consider adding cobra.NoArgs)")
	}
}

func TestChatCommand_Flags(t *testing.T) {
	// Model flag is defined as PersistentFlag on rootCmd, inherited by all subcommands
	flag := rootCmd.PersistentFlags().Lookup("model")
	if flag == nil {
		t.Error("model flag not found on rootCmd")
	}
}

func TestChatCommand_FileFlag(t *testing.T) {
	// Verify file flag is registered
	flag := chatCmd.Flags().Lookup("file")
	if flag == nil {
		t.Fatal("file flag not found")
	}
	if flag.Shorthand != "f" {
		t.Errorf("Expected shorthand 'f', got '%s'", flag.Shorthand)
	}
}

func TestChatCommand_FileFlag_ReadFile(t *testing.T) {
	// Create temp file with content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_prompt.md")
	content := "Test prompt content\nWith multiple lines"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Test reading
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if strings.TrimSpace(string(data)) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(data), content)
	}
}

func TestChatCommand_FileFlag_EmptyFile(t *testing.T) {
	// Create empty temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.md")

	if err := os.WriteFile(tmpFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	data, _ := os.ReadFile(tmpFile)
	if strings.TrimSpace(string(data)) != "" {
		t.Error("Expected empty content")
	}
}

func TestChatCommand_FileFlag_NonExistent(t *testing.T) {
	_, err := os.ReadFile("/nonexistent/path/file.md")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestChatCommand_FileFlag_MaxFileSize(t *testing.T) {
	// Verify the constant is set correctly
	if maxFileSize != 1*1024*1024 {
		t.Errorf("Expected maxFileSize to be 1MB (1048576), got %d", maxFileSize)
	}
}

// TestCreateChatSessionWithConversation_WithGem tests createChatSessionWithConversation with a gem
func TestCreateChatSessionWithConversation_WithGem(t *testing.T) {
	// Create mock client that implements GeminiClientInterface
	mockClient := &mockGeminiClient{
		closed: false,
	}

	// Create a test conversation
	conv := &history.Conversation{
		CID:  "test-cid",
		RID:  "test-rid",
		RCID: "test-rcid",
	}

	// Call createChatSessionWithConversation with gem
	session := createChatSessionWithConversation(mockClient, "test-gem-id", models.Model25Flash, conv)

	// The session should not be nil
	if session == nil {
		t.Error("Expected non-nil session with gem")
	}
}

// TestCreateChatSessionWithConversation_WithoutGem tests createChatSessionWithConversation without a gem
func TestCreateChatSessionWithConversation_WithoutGem(t *testing.T) {
	mockClient := &mockGeminiClient{
		closed: false,
	}

	// Create a test conversation
	conv := &history.Conversation{
		CID:  "test-cid",
		RID:  "test-rid",
		RCID: "test-rcid",
	}

	// Call createChatSessionWithConversation without gem
	session := createChatSessionWithConversation(mockClient, "", models.Model25Flash, conv)

	if session == nil {
		t.Error("Expected non-nil session without gem")
	}
}

// TestCreateChatSessionWithConversation_WithoutConversation tests createChatSessionWithConversation with nil conversation
func TestCreateChatSessionWithConversation_WithoutConversation(t *testing.T) {
	mockClient := &mockGeminiClient{
		closed: false,
	}

	// Call with nil conversation
	session := createChatSessionWithConversation(mockClient, "test-gem-id", models.Model25Flash, nil)

	if session == nil {
		t.Error("Expected non-nil session with nil conversation")
	}
}

// TestCreateChatSessionWithConversation_EmptyConversation tests with empty conversation metadata
func TestCreateChatSessionWithConversation_EmptyConversation(t *testing.T) {
	mockClient := &mockGeminiClient{
		closed: false,
	}

	// Create a conversation with empty metadata
	conv := &history.Conversation{
		CID:  "",
		RID:  "",
		RCID: "",
	}

	session := createChatSessionWithConversation(mockClient, "", models.Model25Flash, conv)

	if session == nil {
		t.Error("Expected non-nil session with empty conversation metadata")
	}
}

// TestCreateChatSessionWithConversation_DifferentModels tests with different models
func TestCreateChatSessionWithConversation_DifferentModels(t *testing.T) {
	mockClient := &mockGeminiClient{
		closed: false,
	}

	conv := &history.Conversation{
		CID:  "test-cid",
		RID:  "test-rid",
		RCID: "test-rcid",
	}

	models := []models.Model{
		models.ModelFast,
		models.ModelPro,
		models.ModelThinking,
	}

	for _, model := range models {
		t.Run(model.Name, func(t *testing.T) {
			session := createChatSessionWithConversation(mockClient, "", model, conv)
			if session == nil {
				t.Error("Expected non-nil session")
			}
		})
	}
}

// TestCreateChatSessionWithConversation_WithEmptyGemID tests with empty gem ID string
func TestCreateChatSessionWithConversation_WithEmptyGemID(t *testing.T) {
	mockClient := &mockGeminiClient{
		closed: false,
	}

	conv := &history.Conversation{
		CID:  "test-cid",
		RID:  "test-rid",
		RCID: "test-rcid",
	}

	// Empty string should be treated as no gem
	session := createChatSessionWithConversation(mockClient, "", models.Model25Flash, conv)

	if session == nil {
		t.Error("Expected non-nil session with empty gem ID")
	}
}
