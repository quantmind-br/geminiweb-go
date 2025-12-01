package commands

import (
	"bytes"
	"os"
	"testing"

	"github.com/diogo/geminiweb/internal/history"
)

func TestHistoryCommand(t *testing.T) {
	// Test that the command is properly configured
	if historyCmd.Use != "history" {
		t.Errorf("Expected use 'history', got %s", historyCmd.Use)
	}

	if historyCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Test that subcommands are registered
	expectedSubcommands := []string{"list", "show", "delete", "clear"}
	for _, sub := range expectedSubcommands {
		found := false
		for _, cmd := range historyCmd.Commands() {
			if cmd.Name() == sub {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Subcommand %s not found", sub)
		}
	}
}

func TestHistoryListCommand(t *testing.T) {
	// Test command structure
	if historyListCmd.Use != "list" {
		t.Errorf("Expected use 'list', got %s", historyListCmd.Use)
	}

	if historyListCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if historyListCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Note: Argument validation is handled by Cobra's Args field, not tested here
	// since calling RunE directly bypasses Cobra's validation
}

func TestHistoryShowCommand(t *testing.T) {
	// Test command structure
	if historyShowCmd.Use != "show <id>" {
		t.Errorf("Expected use 'show <id>', got %s", historyShowCmd.Use)
	}

	if historyShowCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Verify Args validation is configured
	if historyShowCmd.Args == nil {
		t.Error("Args validation should be configured")
	}

	// Note: Argument validation (cobra.ExactArgs(1)) is handled by Cobra,
	// not tested here since calling RunE directly bypasses validation
}

func TestHistoryDeleteCommand(t *testing.T) {
	// Test command structure
	if historyDeleteCmd.Use != "delete <id>" {
		t.Errorf("Expected use 'delete <id>', got %s", historyDeleteCmd.Use)
	}

	if historyDeleteCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestHistoryClearCommand(t *testing.T) {
	// Test command structure
	if historyClearCmd.Use != "clear" {
		t.Errorf("Expected use 'clear', got %s", historyClearCmd.Use)
	}

	if historyClearCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Note: Argument validation is handled by Cobra's Args field, not tested here
	// since calling RunE directly bypasses Cobra's validation
}

// Test that history commands work with a temporary store
func TestHistoryCommands_WithStore(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a store
	store, err := history.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create a test conversation
	conv, err := store.CreateConversation("test-model")
	if err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// Add a message
	err = store.AddMessage(conv.ID, "user", "test message", "")
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Verify the conversation was created
	conversations, err := store.ListConversations()
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 1 {
		t.Errorf("Expected 1 conversation, got %d", len(conversations))
	}

	// Test GetConversation
	retrieved, err := store.GetConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	if retrieved.ID != conv.ID {
		t.Errorf("Conversation ID mismatch")
	}

	// Test DeleteConversation
	err = store.DeleteConversation(conv.ID)
	if err != nil {
		t.Fatalf("Failed to delete conversation: %v", err)
	}

	// Verify it's deleted
	_, err = store.GetConversation(conv.ID)
	if err == nil {
		t.Error("Expected error for deleted conversation")
	}

	// Test ClearAll
	store.CreateConversation("model-1")
	store.CreateConversation("model-2")
	store.CreateConversation("model-3")

	err = store.ClearAll()
	if err != nil {
		t.Fatalf("Failed to clear all: %v", err)
	}

	conversations, err = store.ListConversations()
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations after clear, got %d", len(conversations))
	}
}

func TestRunHistoryList_Empty(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command with no conversations
	err := runHistoryList(historyListCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryList failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should print "No conversations found"
	if output != "No conversations found.\n" {
		t.Errorf("Expected 'No conversations found.', got: %s", output)
	}
}

func TestRunHistoryList_WithConversations(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create conversations directly with DefaultStore
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create test conversations
	conv1, _ := store.CreateConversation("gemini-2.5-flash")
	store.AddMessage(conv1.ID, "user", "First message", "")

	conv2, _ := store.CreateConversation("gemini-2.5-pro")
	store.AddMessage(conv2.ID, "user", "Second message", "")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	err = runHistoryList(historyListCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryList failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain the headers
	if !bytes.Contains([]byte(output), []byte("ID")) {
		t.Error("Output should contain 'ID' header")
	}
	if !bytes.Contains([]byte(output), []byte("TITLE")) {
		t.Error("Output should contain 'TITLE' header")
	}
}

func TestRunHistoryShow_Success(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a conversation
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv, _ := store.CreateConversation("test-model")
	store.AddMessage(conv.ID, "user", "test message", "")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	err = runHistoryShow(historyShowCmd, []string{conv.ID})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryShow failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain the conversation ID
	if !bytes.Contains([]byte(output), []byte(conv.ID)) {
		t.Errorf("Output should contain conversation ID: %s", conv.ID)
	}
}

func TestRunHistoryDelete_Success(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a conversation
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv, _ := store.CreateConversation("test-model")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	err = runHistoryDelete(historyDeleteCmd, []string{conv.ID})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryDelete failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success message
	if !bytes.Contains([]byte(output), []byte("Deleted conversation:")) {
		t.Error("Output should contain success message")
	}

	// Verify the conversation was actually deleted
	_, err = store.GetConversation(conv.ID)
	if err == nil {
		t.Error("Conversation should be deleted")
	}
}

func TestRunHistoryClear_Success(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create conversations
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	store.CreateConversation("model-1")
	store.CreateConversation("model-2")
	store.CreateConversation("model-3")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	err = runHistoryClear(historyClearCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryClear failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success message
	if !bytes.Contains([]byte(output), []byte("All conversations deleted.")) {
		t.Error("Output should contain success message")
	}

	// Verify all conversations were deleted
	conversations, err := store.ListConversations()
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations after clear, got %d", len(conversations))
	}
}
