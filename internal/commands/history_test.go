package commands

import (
	"bytes"
	"os"
	"strings"
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
	expectedSubcommands := []string{"list", "show", "delete", "clear", "rename", "favorite", "export", "search"}
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
	// Test command structure - now uses <ref> instead of <id>
	if historyShowCmd.Use != "show <ref>" {
		t.Errorf("Expected use 'show <ref>', got %s", historyShowCmd.Use)
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
	// Test command structure - now uses <ref> instead of <id>
	if historyDeleteCmd.Use != "delete <ref>" {
		t.Errorf("Expected use 'delete <ref>', got %s", historyDeleteCmd.Use)
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

func TestHistoryRenameCommand(t *testing.T) {
	// Test command structure
	if historyRenameCmd.Use != "rename <ref> <title>" {
		t.Errorf("Expected use 'rename <ref> <title>', got %s", historyRenameCmd.Use)
	}

	if historyRenameCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	if historyRenameCmd.Args == nil {
		t.Error("Args validation should be configured")
	}
}

func TestHistoryFavoriteCommand(t *testing.T) {
	// Test command structure
	if historyFavoriteCmd.Use != "favorite <ref>" {
		t.Errorf("Expected use 'favorite <ref>', got %s", historyFavoriteCmd.Use)
	}

	if historyFavoriteCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	if historyFavoriteCmd.Args == nil {
		t.Error("Args validation should be configured")
	}
}

func TestHistoryExportCommand(t *testing.T) {
	// Test command structure
	if historyExportCmd.Use != "export <ref>" {
		t.Errorf("Expected use 'export <ref>', got %s", historyExportCmd.Use)
	}

	if historyExportCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	if historyExportCmd.Args == nil {
		t.Error("Args validation should be configured")
	}
}

func TestHistorySearchCommand(t *testing.T) {
	// Test command structure
	if historySearchCmd.Use != "search <query>" {
		t.Errorf("Expected use 'search <query>', got %s", historySearchCmd.Use)
	}

	if historySearchCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	if historySearchCmd.Args == nil {
		t.Error("Args validation should be configured")
	}
}

// Test that history commands work with a temporary store
func TestHistoryCommands_WithStore(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

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
	_, _ = store.CreateConversation("model-1")
	_, _ = store.CreateConversation("model-2")
	_, _ = store.CreateConversation("model-3")

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
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command with no conversations
	err := runHistoryList(historyListCmd, []string{})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryList failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should print "No conversation history found"
	if !strings.Contains(output, "No conversation history found.") {
		t.Errorf("Expected 'No conversation history found.', got: %s", output)
	}
	if !strings.Contains(output, "geminiweb chat") {
		t.Errorf("Expected hint about starting new chat, got: %s", output)
	}
}

func TestRunHistoryList_WithConversations(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create conversations directly with DefaultStore
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create test conversations
	conv1, _ := store.CreateConversation("gemini-2.5-flash")
	_ = store.AddMessage(conv1.ID, "user", "First message", "")

	conv2, _ := store.CreateConversation("gemini-2.5-pro")
	_ = store.AddMessage(conv2.ID, "user", "Second message", "")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	err = runHistoryList(historyListCmd, []string{})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryList failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// New format shows indices like [1], [2]
	if !strings.Contains(output, "[1]") {
		t.Error("Output should contain index [1]")
	}
	if !strings.Contains(output, "[2]") {
		t.Error("Output should contain index [2]")
	}
	// Should contain message count
	if !strings.Contains(output, "msg") {
		t.Error("Output should contain message count indicator")
	}
}

func TestRunHistoryShow_Success(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create a conversation
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv, _ := store.CreateConversation("test-model")
	_ = store.AddMessage(conv.ID, "user", "test message", "")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command - use index "1" or direct ID
	err = runHistoryShow(historyShowCmd, []string{conv.ID})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryShow failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain the conversation ID
	if !strings.Contains(output, conv.ID) {
		t.Errorf("Output should contain conversation ID: %s", conv.ID)
	}
}

func TestRunHistoryShow_WithAlias(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create a conversation
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv, _ := store.CreateConversation("test-model")
	_ = store.AddMessage(conv.ID, "user", "test message", "")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command with @last alias
	err = runHistoryShow(historyShowCmd, []string{"@last"})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryShow with @last failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain the conversation ID
	if !strings.Contains(output, conv.ID) {
		t.Errorf("Output should contain conversation ID: %s", conv.ID)
	}
}

func TestRunHistoryShow_WithNumericIndex(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create a conversation
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv, _ := store.CreateConversation("test-model")
	_ = store.AddMessage(conv.ID, "user", "test message", "")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command with numeric index
	err = runHistoryShow(historyShowCmd, []string{"1"})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryShow with numeric index failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain the conversation ID
	if !strings.Contains(output, conv.ID) {
		t.Errorf("Output should contain conversation ID: %s", conv.ID)
	}
}

func TestRunHistoryDelete_Success(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

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

	// Set force flag to skip confirmation
	historyForceFlag = true
	defer func() { historyForceFlag = false }()

	// Run the command
	err = runHistoryDelete(historyDeleteCmd, []string{conv.ID})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryDelete failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success message (now uses ✓)
	if !strings.Contains(output, "Deleted") {
		t.Errorf("Output should contain success message, got: %s", output)
	}

	// Verify the conversation was actually deleted
	_, err = store.GetConversation(conv.ID)
	if err == nil {
		t.Error("Conversation should be deleted")
	}
}

func TestRunHistoryDelete_WithAlias(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

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

	// Set force flag to skip confirmation
	historyForceFlag = true
	defer func() { historyForceFlag = false }()

	// Run the command with @last alias
	err = runHistoryDelete(historyDeleteCmd, []string{"@last"})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryDelete with @last failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success message
	if !strings.Contains(output, "Deleted") {
		t.Errorf("Output should contain success message, got: %s", output)
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
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create conversations
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	_, _ = store.CreateConversation("model-1")
	_, _ = store.CreateConversation("model-2")
	_, _ = store.CreateConversation("model-3")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set force flag to skip confirmation
	historyForceFlag = true
	defer func() { historyForceFlag = false }()

	// Run the command
	err = runHistoryClear(historyClearCmd, []string{})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryClear failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success message (now uses ✓ and count)
	if !strings.Contains(output, "Deleted") || !strings.Contains(output, "3") {
		t.Errorf("Output should contain success message with count, got: %s", output)
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

func TestRunHistoryRename_Success(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

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
	err = runHistoryRename(historyRenameCmd, []string{conv.ID, "New Title"})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryRename failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success message
	if !strings.Contains(output, "Renamed") || !strings.Contains(output, "New Title") {
		t.Errorf("Output should contain success message, got: %s", output)
	}

	// Verify the title was changed
	updated, _ := store.GetConversation(conv.ID)
	if updated.Title != "New Title" {
		t.Errorf("Title should be 'New Title', got: %s", updated.Title)
	}
}

func TestRunHistoryFavorite_Toggle(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

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

	// Run the command - first toggle (add to favorites)
	err = runHistoryFavorite(historyFavoriteCmd, []string{conv.ID})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryFavorite failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success message about adding
	if !strings.Contains(output, "★") || !strings.Contains(output, "favorites") {
		t.Errorf("Output should indicate added to favorites, got: %s", output)
	}

	// Verify favorite status
	isFav, _ := store.IsFavorite(conv.ID)
	if !isFav {
		t.Error("Conversation should be favorited")
	}

	// Toggle again (remove from favorites)
	r, w, _ = os.Pipe()
	os.Stdout = w

	err = runHistoryFavorite(historyFavoriteCmd, []string{conv.ID})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryFavorite second toggle failed: %v", err)
	}

	buf.Reset()
	_, _ = buf.ReadFrom(r)
	output = buf.String()

	// Should contain message about removing
	if !strings.Contains(output, "☆") || !strings.Contains(strings.ToLower(output), "removed") {
		t.Errorf("Output should indicate removed from favorites, got: %s", output)
	}

	// Verify favorite status
	isFav, _ = store.IsFavorite(conv.ID)
	if isFav {
		t.Error("Conversation should not be favorited")
	}
}

func TestRunHistoryExport_Markdown(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create a conversation with messages
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv, _ := store.CreateConversation("test-model")
	_ = store.AddMessage(conv.ID, "user", "Hello", "")
	_ = store.AddMessage(conv.ID, "assistant", "Hi there!", "")
	_ = store.UpdateTitle(conv.ID, "Test Export")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags
	historyOutputFlag = ""
	historyFormatFlag = ""

	// Run the command
	err = runHistoryExport(historyExportCmd, []string{conv.ID})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryExport failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain markdown content
	if !strings.Contains(output, "# Test Export") {
		t.Errorf("Output should contain markdown title, got: %s", output)
	}
	if !strings.Contains(output, "Hello") {
		t.Errorf("Output should contain message content, got: %s", output)
	}
}

func TestRunHistorySearch_TitleMatch(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create conversations
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv1, _ := store.CreateConversation("test-model")
	_ = store.UpdateTitle(conv1.ID, "API Development Chat")

	conv2, _ := store.CreateConversation("test-model")
	_ = store.UpdateTitle(conv2.ID, "Database Design")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flag
	historyContentFlag = false

	// Run the command
	err = runHistorySearch(historySearchCmd, []string{"API"})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistorySearch failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should find the API conversation
	if !strings.Contains(output, "API Development") {
		t.Errorf("Output should contain matched conversation, got: %s", output)
	}
	if strings.Contains(output, "Database Design") {
		t.Errorf("Output should not contain non-matching conversation, got: %s", output)
	}
}

func TestRunHistorySearch_NoResults(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create a conversation
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv, _ := store.CreateConversation("test-model")
	_ = store.UpdateTitle(conv.ID, "General Chat")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flag
	historyContentFlag = false

	// Run the command
	err = runHistorySearch(historySearchCmd, []string{"xyz123nonexistent"})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistorySearch failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should indicate no results
	if !strings.Contains(output, "No conversations matching") {
		t.Errorf("Output should indicate no results, got: %s", output)
	}
}

func TestRunHistoryList_Favorites(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create conversations
	store, err := history.DefaultStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	conv1, _ := store.CreateConversation("model-1")
	_ = store.UpdateTitle(conv1.ID, "Favorite Chat")
	_ = store.SetFavorite(conv1.ID, true)

	conv2, _ := store.CreateConversation("model-2")
	_ = store.UpdateTitle(conv2.ID, "Regular Chat")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set favorites filter
	historyFavoritesFlag = true
	defer func() { historyFavoritesFlag = false }()

	// Run the command
	err = runHistoryList(historyListCmd, []string{})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runHistoryList with favorites failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain only the favorite conversation
	if !strings.Contains(output, "Favorite Chat") {
		t.Errorf("Output should contain favorited conversation, got: %s", output)
	}
	if strings.Contains(output, "Regular Chat") {
		t.Errorf("Output should not contain non-favorited conversation, got: %s", output)
	}
}
