package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/diogo/geminiweb/internal/history"
)

// mockHistoryStore is a mock implementation of HistoryStore for testing
type mockHistoryStore struct {
	conversations []*history.Conversation
	err           error
	createErr     error
	createdModel  string
}

func (m *mockHistoryStore) ListConversations() ([]*history.Conversation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.conversations, nil
}

func (m *mockHistoryStore) CreateConversation(model string) (*history.Conversation, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.createdModel = model
	return &history.Conversation{
		ID:    "new-conv-id",
		Model: model,
		Title: "New Conversation",
	}, nil
}

func TestNewHistorySelectorModel(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")

	if m.store != store {
		t.Error("Store not set correctly")
	}
	if m.modelName != "gemini-2.5-flash" {
		t.Errorf("modelName = %s, want gemini-2.5-flash", m.modelName)
	}
	if !m.loading {
		t.Error("Model should be loading initially")
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestHistorySelectorModel_Init(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestHistorySelectorModel_Update_WindowSize(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")

	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(msg)

	if model, ok := updatedModel.(HistorySelectorModel); ok {
		if model.width != 100 {
			t.Errorf("width = %d, want 100", model.width)
		}
		if model.height != 40 {
			t.Errorf("height = %d, want 40", model.height)
		}
		if !model.ready {
			t.Error("Model should be ready after WindowSizeMsg")
		}
	} else {
		t.Error("Update should return HistorySelectorModel")
	}
}

func TestHistorySelectorModel_Update_HistoryLoaded(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.ready = true

	conversations := []*history.Conversation{
		{ID: "conv-1", Title: "First Chat", Model: "gemini-2.5-flash"},
		{ID: "conv-2", Title: "Second Chat", Model: "gemini-3.0-pro"},
	}

	msg := historyLoadedMsg{conversations: conversations}
	updatedModel, _ := m.Update(msg)

	if model, ok := updatedModel.(HistorySelectorModel); ok {
		if model.loading {
			t.Error("Model should not be loading after historyLoadedMsg")
		}
		if len(model.conversations) != 2 {
			t.Errorf("conversations length = %d, want 2", len(model.conversations))
		}
		if model.err != nil {
			t.Errorf("err = %v, want nil", model.err)
		}
	} else {
		t.Error("Update should return HistorySelectorModel")
	}
}

func TestHistorySelectorModel_Update_HistoryLoadedError(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.ready = true

	testErr := errors.New("failed to load")
	msg := historyLoadedMsg{err: testErr}
	updatedModel, _ := m.Update(msg)

	if model, ok := updatedModel.(HistorySelectorModel); ok {
		if model.loading {
			t.Error("Model should not be loading after historyLoadedMsg")
		}
		if model.err == nil {
			t.Error("err should be set")
		}
	} else {
		t.Error("Update should return HistorySelectorModel")
	}
}

func TestHistorySelectorModel_Update_Navigation(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.loading = false
	m.ready = true
	m.conversations = []*history.Conversation{
		{ID: "conv-1", Title: "First Chat"},
		{ID: "conv-2", Title: "Second Chat"},
	}
	m.cursor = 0

	t.Run("down key", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			if model.cursor != 1 {
				t.Errorf("cursor = %d, want 1", model.cursor)
			}
		}
	})

	t.Run("up key", func(t *testing.T) {
		m.cursor = 1
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			if model.cursor != 0 {
				t.Errorf("cursor = %d, want 0", model.cursor)
			}
		}
	})

	t.Run("j key (vim down)", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			if model.cursor != 1 {
				t.Errorf("cursor = %d, want 1", model.cursor)
			}
		}
	})

	t.Run("k key (vim up)", func(t *testing.T) {
		m.cursor = 1
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			if model.cursor != 0 {
				t.Errorf("cursor = %d, want 0", model.cursor)
			}
		}
	})

	t.Run("wrap around down", func(t *testing.T) {
		m.cursor = len(m.conversations) // Last item
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			if model.cursor != 0 {
				t.Errorf("cursor = %d, want 0 (wrapped)", model.cursor)
			}
		}
	})

	t.Run("wrap around up", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			expectedCursor := len(m.conversations) // +1 for "New Conversation"
			if model.cursor != expectedCursor {
				t.Errorf("cursor = %d, want %d (wrapped)", model.cursor, expectedCursor)
			}
		}
	})
}

func TestHistorySelectorModel_Update_Enter_NewConversation(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.loading = false
	m.ready = true
	m.conversations = []*history.Conversation{
		{ID: "conv-1", Title: "First Chat"},
	}
	m.cursor = 0 // "New Conversation" is at index 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Enter should return a quit command")
	}

	if model, ok := updatedModel.(HistorySelectorModel); ok {
		if !model.confirmed {
			t.Error("confirmed should be true")
		}
		if !model.isNewConv {
			t.Error("isNewConv should be true")
		}
		if model.selectedConv != nil {
			t.Error("selectedConv should be nil for new conversation")
		}
	} else {
		t.Error("Update should return HistorySelectorModel")
	}
}

func TestHistorySelectorModel_Update_Enter_ExistingConversation(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.loading = false
	m.ready = true
	m.conversations = []*history.Conversation{
		{ID: "conv-1", Title: "First Chat"},
		{ID: "conv-2", Title: "Second Chat"},
	}
	m.cursor = 1 // First existing conversation (index 0 is "New Conversation")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Enter should return a quit command")
	}

	if model, ok := updatedModel.(HistorySelectorModel); ok {
		if !model.confirmed {
			t.Error("confirmed should be true")
		}
		if model.isNewConv {
			t.Error("isNewConv should be false")
		}
		if model.selectedConv == nil {
			t.Error("selectedConv should not be nil")
		}
		if model.selectedConv.ID != "conv-1" {
			t.Errorf("selectedConv.ID = %s, want conv-1", model.selectedConv.ID)
		}
	} else {
		t.Error("Update should return HistorySelectorModel")
	}
}

func TestHistorySelectorModel_Update_Quit(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.loading = false
	m.ready = true

	tests := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}},
		{"esc", tea.KeyMsg{Type: tea.KeyEscape}},
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := m.Update(tt.msg)
			if cmd == nil {
				t.Errorf("%s should return a quit command", tt.name)
			}
		})
	}
}

func TestHistorySelectorModel_View_NotReady(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.ready = false

	view := m.View()
	if !strings.Contains(view, "Initializing") {
		t.Error("View should show initializing message when not ready")
	}
}

func TestHistorySelectorModel_View_Loading(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.ready = true
	m.loading = true

	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("View should show loading message")
	}
}

func TestHistorySelectorModel_View_Error(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.ready = true
	m.loading = false
	m.err = errors.New("test error")

	view := m.View()
	if !strings.Contains(view, "Error") || !strings.Contains(view, "test error") {
		t.Error("View should show error message")
	}
}

func TestHistorySelectorModel_View_WithConversations(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.ready = true
	m.loading = false
	m.width = 80
	m.height = 24
	m.conversations = []*history.Conversation{
		{ID: "conv-1", Title: "First Chat", Model: "gemini-2.5-flash", UpdatedAt: time.Now().Add(-1 * time.Hour)},
		{ID: "conv-2", Title: "Second Chat", Model: "gemini-3.0-pro", UpdatedAt: time.Now().Add(-24 * time.Hour)},
	}

	view := m.View()

	// Should contain header
	if !strings.Contains(view, "Select Conversation") {
		t.Error("View should contain header")
	}

	// Should contain "New Conversation" option
	if !strings.Contains(view, "New Conversation") {
		t.Error("View should contain 'New Conversation' option")
	}

	// Should contain conversation titles
	if !strings.Contains(view, "First Chat") {
		t.Error("View should contain conversation title")
	}
}

func TestHistorySelectorModel_View_EmptyConversations(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.ready = true
	m.loading = false
	m.width = 80
	m.height = 24
	m.conversations = []*history.Conversation{}

	view := m.View()

	// Should contain "New Conversation" option
	if !strings.Contains(view, "New Conversation") {
		t.Error("View should contain 'New Conversation' option")
	}

	// Should indicate no saved conversations
	if !strings.Contains(view, "No saved conversations") {
		t.Error("View should indicate no saved conversations")
	}
}

func TestHistorySelectorModel_Result(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")

	t.Run("initial state", func(t *testing.T) {
		conv, isNew, confirmed := m.Result()
		if conv != nil {
			t.Error("Initial conv should be nil")
		}
		if isNew {
			t.Error("Initial isNew should be false")
		}
		if confirmed {
			t.Error("Initial confirmed should be false")
		}
	})

	t.Run("after new conversation selection", func(t *testing.T) {
		m.confirmed = true
		m.isNewConv = true
		m.selectedConv = nil

		conv, isNew, confirmed := m.Result()
		if conv != nil {
			t.Error("conv should be nil for new conversation")
		}
		if !isNew {
			t.Error("isNew should be true")
		}
		if !confirmed {
			t.Error("confirmed should be true")
		}
	})

	t.Run("after existing conversation selection", func(t *testing.T) {
		existingConv := &history.Conversation{ID: "test-id", Title: "Test"}
		m.confirmed = true
		m.isNewConv = false
		m.selectedConv = existingConv

		conv, isNew, confirmed := m.Result()
		if conv != existingConv {
			t.Error("conv should be the selected conversation")
		}
		if isNew {
			t.Error("isNew should be false")
		}
		if !confirmed {
			t.Error("confirmed should be true")
		}
	})
}

func TestHistorySelectorResult_Struct(t *testing.T) {
	conv := &history.Conversation{ID: "test"}
	result := HistorySelectorResult{
		Conversation: conv,
		IsNew:        false,
		Confirmed:    true,
	}

	if result.Conversation != conv {
		t.Error("Conversation not set correctly")
	}
	if result.IsNew {
		t.Error("IsNew should be false")
	}
	if !result.Confirmed {
		t.Error("Confirmed should be true")
	}
}

func TestHistorySelectorModel_Update_HomeEnd(t *testing.T) {
	store := &mockHistoryStore{}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")
	m.loading = false
	m.ready = true
	m.conversations = []*history.Conversation{
		{ID: "conv-1"},
		{ID: "conv-2"},
		{ID: "conv-3"},
	}

	t.Run("home key", func(t *testing.T) {
		m.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyHome}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			if model.cursor != 0 {
				t.Errorf("cursor = %d, want 0", model.cursor)
			}
		}
	})

	t.Run("g key (vim home)", func(t *testing.T) {
		m.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			if model.cursor != 0 {
				t.Errorf("cursor = %d, want 0", model.cursor)
			}
		}
	})

	t.Run("end key", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyEnd}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			expected := len(m.conversations)
			if model.cursor != expected {
				t.Errorf("cursor = %d, want %d", model.cursor, expected)
			}
		}
	})

	t.Run("G key (vim end)", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		updatedModel, _ := m.Update(msg)
		if model, ok := updatedModel.(HistorySelectorModel); ok {
			expected := len(m.conversations)
			if model.cursor != expected {
				t.Errorf("cursor = %d, want %d", model.cursor, expected)
			}
		}
	})
}

func TestHistorySelectorModel_LoadConversations(t *testing.T) {
	conversations := []*history.Conversation{
		{ID: "conv-1", Title: "Test"},
	}
	store := &mockHistoryStore{conversations: conversations}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")

	cmd := m.loadConversations()
	if cmd == nil {
		t.Error("loadConversations should return a command")
	}

	// Execute the command
	msg := cmd()
	if loaded, ok := msg.(historyLoadedMsg); ok {
		if len(loaded.conversations) != 1 {
			t.Errorf("Expected 1 conversation, got %d", len(loaded.conversations))
		}
	} else {
		t.Errorf("Expected historyLoadedMsg, got %T", msg)
	}
}

func TestHistorySelectorModel_LoadConversations_Error(t *testing.T) {
	testErr := errors.New("load error")
	store := &mockHistoryStore{err: testErr}
	m := NewHistorySelectorModel(store, "gemini-2.5-flash")

	cmd := m.loadConversations()
	msg := cmd()

	if loaded, ok := msg.(historyLoadedMsg); ok {
		if loaded.err == nil {
			t.Error("Expected error in historyLoadedMsg")
		}
	} else {
		t.Errorf("Expected historyLoadedMsg, got %T", msg)
	}
}
