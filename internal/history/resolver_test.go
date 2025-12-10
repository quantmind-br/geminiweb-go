package history

import (
	"strings"
	"testing"
	"time"
)

func TestResolver_ResolveAtLast(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	// Create conversations with time gaps
	// Most recent conversations are added at the beginning of the order
	conv1, _ := store.CreateConversation("model-1")
	time.Sleep(10 * time.Millisecond)
	_, _ = store.CreateConversation("model-2") // conv2 - not used directly
	time.Sleep(10 * time.Millisecond)
	conv3, _ := store.CreateConversation("model-3")

	resolver := NewResolver(store)

	// @last should return the first in the order (newest created)
	// Since conversations are prepended to meta.Order, conv3 should be first
	id, err := resolver.Resolve("@last")
	if err != nil {
		t.Fatalf("Resolve @last failed: %v", err)
	}
	if id != conv3.ID {
		t.Errorf("@last = %s, want %s", id, conv3.ID)
	}

	// Move conv1 to the top of the order
	store.MoveConversation(conv1.ID, 0)
	id, _ = resolver.Resolve("@last")
	if id != conv1.ID {
		t.Errorf("@last after move = %s, want %s", id, conv1.ID)
	}
}

func TestResolver_ResolveAtFirst(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv1, _ := store.CreateConversation("model-1")
	time.Sleep(10 * time.Millisecond)
	store.CreateConversation("model-2")

	resolver := NewResolver(store)

	// @first should return the first (oldest) conversation
	id, err := resolver.Resolve("@first")
	if err != nil {
		t.Fatalf("Resolve @first failed: %v", err)
	}
	if id != conv1.ID {
		t.Errorf("@first = %s, want %s", id, conv1.ID)
	}
}

func TestResolver_ResolveNumericIndex(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv1, _ := store.CreateConversation("model-1")
	time.Sleep(10 * time.Millisecond)
	conv2, _ := store.CreateConversation("model-2")
	time.Sleep(10 * time.Millisecond)
	conv3, _ := store.CreateConversation("model-3")

	resolver := NewResolver(store)

	// Index 1 should be the first (most recent)
	id, err := resolver.Resolve("1")
	if err != nil {
		t.Fatalf("Resolve 1 failed: %v", err)
	}
	if id != conv3.ID {
		t.Errorf("index 1 = %s, want %s (most recent)", id, conv3.ID)
	}

	// Index 2
	id, _ = resolver.Resolve("2")
	if id != conv2.ID {
		t.Errorf("index 2 = %s, want %s", id, conv2.ID)
	}

	// Index 3 (oldest)
	id, _ = resolver.Resolve("3")
	if id != conv1.ID {
		t.Errorf("index 3 = %s, want %s (oldest)", id, conv1.ID)
	}
}

func TestResolver_ResolveNumericIndex_OutOfRange(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	store.CreateConversation("model-1")

	resolver := NewResolver(store)

	// Index 0 should fail (1-based)
	_, err := resolver.Resolve("0")
	if err == nil {
		t.Error("expected error for index 0")
	}

	// Index 99 should fail
	_, err = resolver.Resolve("99")
	if err == nil {
		t.Error("expected error for index 99")
	}
}

func TestResolver_ResolveDirectID(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	resolver := NewResolver(store)

	// Direct ID should work
	id, err := resolver.Resolve(conv.ID)
	if err != nil {
		t.Fatalf("Resolve direct ID failed: %v", err)
	}
	if id != conv.ID {
		t.Errorf("direct ID = %s, want %s", id, conv.ID)
	}
}

func TestResolver_ResolveDirectID_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	resolver := NewResolver(store)

	_, err := resolver.Resolve("conv-nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}
}

func TestResolver_ResolveSubstring(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	store.UpdateTitle(conv.ID, "API Development Discussion")

	resolver := NewResolver(store)

	// Substring match (case insensitive)
	id, err := resolver.Resolve("api")
	if err != nil {
		t.Fatalf("Resolve substring failed: %v", err)
	}
	if id != conv.ID {
		t.Errorf("substring match = %s, want %s", id, conv.ID)
	}

	// Another substring
	id, _ = resolver.Resolve("Discussion")
	if id != conv.ID {
		t.Errorf("substring match = %s, want %s", id, conv.ID)
	}
}

func TestResolver_ResolveSubstring_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	store.UpdateTitle(conv.ID, "API Development")

	resolver := NewResolver(store)

	_, err := resolver.Resolve("xyz123")
	if err == nil {
		t.Error("expected error for no match")
	}
}

func TestResolver_ResolveSubstring_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv1, _ := store.CreateConversation("model-1")
	conv2, _ := store.CreateConversation("model-2")
	store.UpdateTitle(conv1.ID, "API v1 Discussion")
	store.UpdateTitle(conv2.ID, "API v2 Discussion")

	resolver := NewResolver(store)

	// "API" matches both - should error
	_, err := resolver.Resolve("API")
	if err == nil {
		t.Error("expected error for multiple matches")
	}
	if !strings.Contains(err.Error(), "multiple") {
		t.Errorf("error should mention 'multiple', got: %v", err)
	}
}

func TestResolver_ResolveEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	resolver := NewResolver(store)

	_, err := resolver.Resolve("")
	if err == nil {
		t.Error("expected error for empty reference")
	}
}

func TestResolver_ResolveNoConversations(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	resolver := NewResolver(store)

	_, err := resolver.Resolve("@last")
	if err == nil {
		t.Error("expected error for no conversations")
	}
}

func TestResolver_ResolveWithInfo(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")
	store.UpdateTitle(conv.ID, "Test Title")

	resolver := NewResolver(store)

	// ResolveWithInfo should return full conversation
	result, err := resolver.ResolveWithInfo("1")
	if err != nil {
		t.Fatalf("ResolveWithInfo failed: %v", err)
	}
	if result.ID != conv.ID {
		t.Errorf("ID = %s, want %s", result.ID, conv.ID)
	}
	if result.Title != "Test Title" {
		t.Errorf("Title = %s, want Test Title", result.Title)
	}
}

func TestResolver_MustResolve(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	resolver := NewResolver(store)

	// Should not panic for valid reference
	id := resolver.MustResolve("1")
	if id != conv.ID {
		t.Errorf("MustResolve = %s, want %s", id, conv.ID)
	}
}

func TestResolver_MustResolve_Panics(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	resolver := NewResolver(store)

	// Should panic for invalid reference
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustResolve should panic for invalid reference")
		}
	}()

	resolver.MustResolve("@last") // Should panic - no conversations
}

func TestResolver_CaseInsensitiveAliases(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	conv, _ := store.CreateConversation("test-model")

	resolver := NewResolver(store)

	// @LAST, @Last, @last should all work
	tests := []string{"@last", "@LAST", "@Last", "@LaSt"}
	for _, alias := range tests {
		id, err := resolver.Resolve(alias)
		if err != nil {
			t.Errorf("Resolve %s failed: %v", alias, err)
			continue
		}
		if id != conv.ID {
			t.Errorf("Resolve %s = %s, want %s", alias, id, conv.ID)
		}
	}
}

func TestListAliases(t *testing.T) {
	help := ListAliases()
	if help == "" {
		t.Error("ListAliases should return non-empty help text")
	}
	if !strings.Contains(help, "@last") {
		t.Error("help should mention @last")
	}
	if !strings.Contains(help, "@first") {
		t.Error("help should mention @first")
	}
}
