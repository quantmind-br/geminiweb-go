package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if store == nil {
		t.Fatal("NewStore returned nil")
	}

	// Check that history directory was created
	historyDir := filepath.Join(tmpDir, "history")
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		t.Error("history directory was not created")
	}
}

func TestStore_CreateConversation(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, err := store.CreateConversation("gemini-2.5-flash")
	if err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}

	if conv.ID == "" {
		t.Error("conversation ID is empty")
	}

	if conv.Model != "gemini-2.5-flash" {
		t.Errorf("Model = %s, want gemini-2.5-flash", conv.Model)
	}

	if conv.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	if len(conv.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(conv.Messages))
	}
}

func TestStore_GetConversation(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	created, _ := store.CreateConversation("test-model")

	retrieved, err := store.GetConversation(created.ID)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, created.ID)
	}

	if retrieved.Model != created.Model {
		t.Errorf("Model = %s, want %s", retrieved.Model, created.Model)
	}
}

func TestStore_GetConversation_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	_, err := store.GetConversation("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent conversation")
	}
}

func TestStore_AddMessage(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	err := store.AddMessage(conv.ID, "user", "Hello!", "")
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	updated, _ := store.GetConversation(conv.ID)
	if len(updated.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(updated.Messages))
	}

	msg := updated.Messages[0]
	if msg.Role != "user" {
		t.Errorf("Role = %s, want user", msg.Role)
	}
	if msg.Content != "Hello!" {
		t.Errorf("Content = %s, want Hello!", msg.Content)
	}
}

func TestStore_AddMessage_UpdatesTitle(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	originalTitle := conv.Title

	store.AddMessage(conv.ID, "user", "What is Go programming?", "")

	updated, _ := store.GetConversation(conv.ID)
	if updated.Title == originalTitle {
		t.Error("title should be updated from first user message")
	}

	if updated.Title != "What is Go programming?" {
		t.Errorf("Title = %s, want What is Go programming?", updated.Title)
	}
}

func TestStore_AddMessage_TruncatesLongTitle(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	longMessage := "This is a very long message that should be truncated when used as a title because it exceeds the maximum length"
	store.AddMessage(conv.ID, "user", longMessage, "")

	updated, _ := store.GetConversation(conv.ID)
	if len(updated.Title) > 60 { // 50 chars + "..."
		t.Errorf("Title too long: %d chars", len(updated.Title))
	}
}

func TestStore_AddMessage_WithThoughts(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	store.AddMessage(conv.ID, "assistant", "Response", "Thinking about this...")

	updated, _ := store.GetConversation(conv.ID)
	if updated.Messages[0].Thoughts != "Thinking about this..." {
		t.Error("thoughts not saved")
	}
}

func TestStore_UpdateMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	err := store.UpdateMetadata(conv.ID, "cid123", "rid456", "rcid789")
	if err != nil {
		t.Fatalf("UpdateMetadata failed: %v", err)
	}

	updated, _ := store.GetConversation(conv.ID)
	if updated.CID != "cid123" {
		t.Errorf("CID = %s, want cid123", updated.CID)
	}
	if updated.RID != "rid456" {
		t.Errorf("RID = %s, want rid456", updated.RID)
	}
	if updated.RCID != "rcid789" {
		t.Errorf("RCID = %s, want rcid789", updated.RCID)
	}
}

func TestStore_DeleteConversation(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	err := store.DeleteConversation(conv.ID)
	if err != nil {
		t.Fatalf("DeleteConversation failed: %v", err)
	}

	_, err = store.GetConversation(conv.ID)
	if err == nil {
		t.Error("conversation should be deleted")
	}
}

func TestStore_DeleteConversation_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	err := store.DeleteConversation("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent conversation")
	}
}

func TestStore_ListConversations(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create multiple conversations
	store.CreateConversation("model-1")
	time.Sleep(10 * time.Millisecond)
	store.CreateConversation("model-2")
	time.Sleep(10 * time.Millisecond)
	store.CreateConversation("model-3")

	conversations, err := store.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}

	if len(conversations) != 3 {
		t.Errorf("expected 3 conversations, got %d", len(conversations))
	}

	// Should be sorted by UpdatedAt descending (newest first)
	if conversations[0].Model != "model-3" {
		t.Error("conversations not sorted correctly")
	}
}

func TestStore_ListConversations_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conversations, err := store.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}

	if len(conversations) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(conversations))
	}
}

func TestStore_UpdateTitle(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	err := store.UpdateTitle(conv.ID, "New Title")
	if err != nil {
		t.Fatalf("UpdateTitle failed: %v", err)
	}

	updated, _ := store.GetConversation(conv.ID)
	if updated.Title != "New Title" {
		t.Errorf("Title = %s, want New Title", updated.Title)
	}
}

func TestStore_ClearAll(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	store.CreateConversation("model-1")
	store.CreateConversation("model-2")
	store.CreateConversation("model-3")

	err := store.ClearAll()
	if err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	conversations, _ := store.ListConversations()
	if len(conversations) != 0 {
		t.Errorf("expected 0 conversations after clear, got %d", len(conversations))
	}
}

func TestMessage_Fields(t *testing.T) {
	msg := Message{
		Role:      "user",
		Content:   "Hello",
		Thoughts:  "thinking",
		Timestamp: time.Now(),
	}

	if msg.Role != "user" {
		t.Error("Role mismatch")
	}
	if msg.Content != "Hello" {
		t.Error("Content mismatch")
	}
	if msg.Thoughts != "thinking" {
		t.Error("Thoughts mismatch")
	}
	if msg.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

func TestConversation_Fields(t *testing.T) {
	conv := Conversation{
		ID:        "conv-123",
		Title:     "Test Chat",
		Model:     "gemini-2.5-flash",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []Message{},
		CID:       "cid",
		RID:       "rid",
		RCID:      "rcid",
	}

	if conv.ID != "conv-123" {
		t.Error("ID mismatch")
	}
	if conv.CID != "cid" {
		t.Error("CID mismatch")
	}
}

func TestGenerateConvID(t *testing.T) {
	id1 := generateConvID()
	id2 := generateConvID()

	if id1 == "" {
		t.Error("generated ID is empty")
	}

	if id1 == id2 {
		t.Log("Warning: consecutive IDs are same (possible but rare)")
	}
}

func TestGetHistoryDir(t *testing.T) {
	dir, err := GetHistoryDir()
	if err != nil {
		t.Fatalf("GetHistoryDir failed: %v", err)
	}

	if dir == "" {
		t.Error("history dir is empty")
	}
}

func TestDefaultStore(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	store, err := DefaultStore()
	if err != nil {
		t.Fatalf("DefaultStore() returned error: %v", err)
	}

	if store == nil {
		t.Error("DefaultStore() returned nil")
	}

	// Verify the store uses the correct directory
	expectedDir := filepath.Join(tmpDir, ".geminiweb", "history")
	if store.baseDir != expectedDir {
		t.Errorf("baseDir = %s, want %s", store.baseDir, expectedDir)
	}

	// Verify directory was created
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("history directory was not created")
	}
}

func TestClearAll_WithEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Clear an empty directory should not error
	err := store.ClearAll()
	if err != nil {
		t.Fatalf("ClearAll() on empty directory returned error: %v", err)
	}
}

func TestClearAll_RemovesOnlyJSONFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create a conversation
	store.CreateConversation("test-model")

	// Create a non-JSON file that should not be touched
	otherFile := filepath.Join(tmpDir, "history", "other.txt")
	if err := os.WriteFile(otherFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create other file: %v", err)
	}

	err := store.ClearAll()
	if err != nil {
		t.Fatalf("ClearAll() returned error: %v", err)
	}

	// Verify JSON files are gone
	conversations, _ := store.ListConversations()
	if len(conversations) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(conversations))
	}

	// Verify non-JSON file still exists
	if _, err := os.Stat(otherFile); os.IsNotExist(err) {
		t.Error("non-JSON file should not be removed")
	}
}
