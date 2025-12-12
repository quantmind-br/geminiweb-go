package history

import (
	"os"
	"testing"
)

func TestToggleFavorite(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Create a conversation
	conv, err := store.CreateConversation("test-model")
	if err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}

	// Toggle favorite on
	isFav, err := store.ToggleFavorite(conv.ID)
	if err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	if !isFav {
		t.Error("expected favorite to be true after first toggle")
	}

	// Verify via IsFavorite
	isFav, err = store.IsFavorite(conv.ID)
	if err != nil {
		t.Fatalf("IsFavorite failed: %v", err)
	}
	if !isFav {
		t.Error("IsFavorite should return true")
	}

	// Toggle favorite off
	isFav, err = store.ToggleFavorite(conv.ID)
	if err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	if isFav {
		t.Error("expected favorite to be false after second toggle")
	}
}

func TestSetFavorite(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	// Set favorite to true
	err := store.SetFavorite(conv.ID, true)
	if err != nil {
		t.Fatalf("SetFavorite failed: %v", err)
	}

	isFav, _ := store.IsFavorite(conv.ID)
	if !isFav {
		t.Error("expected favorite to be true")
	}

	// Set favorite to false
	err = store.SetFavorite(conv.ID, false)
	if err != nil {
		t.Fatalf("SetFavorite failed: %v", err)
	}

	isFav, _ = store.IsFavorite(conv.ID)
	if isFav {
		t.Error("expected favorite to be false")
	}
}

func TestMoveConversation(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create 3 conversations
	conv1, _ := store.CreateConversation("model-1")
	conv2, _ := store.CreateConversation("model-2")
	conv3, _ := store.CreateConversation("model-3")

	// Verify initial order (newest first: 3, 2, 1)
	conversations, _ := store.ListConversations()
	if conversations[0].ID != conv3.ID {
		t.Error("expected conv3 at position 0")
	}
	if conversations[1].ID != conv2.ID {
		t.Error("expected conv2 at position 1")
	}
	if conversations[2].ID != conv1.ID {
		t.Error("expected conv1 at position 2")
	}

	// Move conv1 to position 0
	err := store.MoveConversation(conv1.ID, 0)
	if err != nil {
		t.Fatalf("MoveConversation failed: %v", err)
	}

	// Verify new order: 1, 3, 2
	conversations, _ = store.ListConversations()
	if conversations[0].ID != conv1.ID {
		t.Errorf("expected conv1 at position 0, got %s", conversations[0].ID)
	}
}

func TestSwapConversations(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create 2 conversations
	conv1, _ := store.CreateConversation("model-1")
	conv2, _ := store.CreateConversation("model-2")

	// Verify initial order (newest first: 2, 1)
	conversations, _ := store.ListConversations()
	if conversations[0].ID != conv2.ID {
		t.Error("expected conv2 at position 0 initially")
	}

	// Swap positions
	err := store.SwapConversations(conv1.ID, conv2.ID)
	if err != nil {
		t.Fatalf("SwapConversations failed: %v", err)
	}

	// Verify new order: 1, 2
	conversations, _ = store.ListConversations()
	if conversations[0].ID != conv1.ID {
		t.Errorf("expected conv1 at position 0 after swap, got %s", conversations[0].ID)
	}
}

func TestGetOrderIndex(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create conversations
	conv1, _ := store.CreateConversation("model-1")
	conv2, _ := store.CreateConversation("model-2")

	// conv2 should be at index 0 (newest first)
	idx, err := store.GetOrderIndex(conv2.ID)
	if err != nil {
		t.Fatalf("GetOrderIndex failed: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}

	// conv1 should be at index 1
	idx, _ = store.GetOrderIndex(conv1.ID)
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
}

func TestListConversations_PopulatesComputedFields(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create a conversation
	conv, _ := store.CreateConversation("test-model")

	// Mark as favorite
	_ = store.SetFavorite(conv.ID, true)

	// List conversations
	conversations, err := store.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}

	if len(conversations) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(conversations))
	}

	// Check computed fields
	if !conversations[0].IsFavorite {
		t.Error("IsFavorite should be true")
	}
	if conversations[0].OrderIndex != 0 {
		t.Errorf("OrderIndex should be 0, got %d", conversations[0].OrderIndex)
	}
}

func TestDeleteConversation_RemovesFromMeta(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create and favorite a conversation
	conv, _ := store.CreateConversation("test-model")
	_ = store.SetFavorite(conv.ID, true)

	// Delete the conversation
	err := store.DeleteConversation(conv.ID)
	if err != nil {
		t.Fatalf("DeleteConversation failed: %v", err)
	}

	// Verify it's not in favorites
	isFav, _ := store.IsFavorite(conv.ID)
	if isFav {
		t.Error("deleted conversation should not be a favorite")
	}
}

func TestUpdateTitle_UpdatesMeta(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create a conversation
	conv, _ := store.CreateConversation("test-model")

	// Update title
	err := store.UpdateTitle(conv.ID, "New Title")
	if err != nil {
		t.Fatalf("UpdateTitle failed: %v", err)
	}

	// Load meta and verify title is cached
	meta, err := store.loadMeta()
	if err != nil {
		t.Fatalf("loadMeta failed: %v", err)
	}

	if m, ok := meta.Meta[conv.ID]; ok {
		if m.Title != "New Title" {
			t.Errorf("meta title = %s, want New Title", m.Title)
		}
	} else {
		t.Error("conversation not found in meta")
	}
}

func TestMetaPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store and add conversations
	store1, _ := NewStore(tmpDir)
	conv, _ := store1.CreateConversation("test-model")
	_ = store1.SetFavorite(conv.ID, true)

	// Create new store instance (simulating restart)
	store2, _ := NewStore(tmpDir)

	// Verify favorite status persists
	isFav, err := store2.IsFavorite(conv.ID)
	if err != nil {
		t.Fatalf("IsFavorite failed: %v", err)
	}
	if !isFav {
		t.Error("favorite status should persist across store instances")
	}
}

func TestOrphanedMetaCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create a conversation
	conv, _ := store.CreateConversation("test-model")
	_ = store.SetFavorite(conv.ID, true)

	// Manually delete the conversation file (simulating corruption)
	convPath := store.conversationPath(conv.ID)
	if err := deleteFile(convPath); err != nil {
		t.Fatalf("failed to delete conversation file: %v", err)
	}

	// List conversations (should trigger cleanup)
	conversations, err := store.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}

	if len(conversations) != 0 {
		t.Errorf("expected 0 conversations after orphan cleanup, got %d", len(conversations))
	}
}

// Helper to delete a file
func deleteFile(path string) error {
	return os.Remove(path)
}

func TestClearAll_ClearsMeta(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create conversations and mark as favorites
	conv1, _ := store.CreateConversation("model-1")
	conv2, _ := store.CreateConversation("model-2")
	_ = store.SetFavorite(conv1.ID, true)
	_ = store.SetFavorite(conv2.ID, true)

	// Clear all
	err := store.ClearAll()
	if err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	// Verify meta is also cleared
	meta, _ := store.loadMeta()
	if len(meta.Order) != 0 {
		t.Errorf("expected 0 items in order, got %d", len(meta.Order))
	}
	if len(meta.Meta) != 0 {
		t.Errorf("expected 0 items in meta, got %d", len(meta.Meta))
	}
}
