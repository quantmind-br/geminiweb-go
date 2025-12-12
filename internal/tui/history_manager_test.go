package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/diogo/geminiweb/internal/history"
)

// mockHistoryManagerStore is a mock implementation of HistoryManagerStore for testing
type mockHistoryManagerStore struct {
	conversations      []*history.Conversation
	listErr            error
	deleteErr          error
	updateTitleErr     error
	toggleFavoriteErr  error
	swapErr            error
	exportErr          error
	deletedID          string
	updatedTitleID     string
	updatedTitle       string
	toggledFavoriteID  string
	swappedIDs         []string
	exportedID         string
	favoriteState      bool
}

func (m *mockHistoryManagerStore) ListConversations() ([]*history.Conversation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.conversations, nil
}

func (m *mockHistoryManagerStore) GetConversation(id string) (*history.Conversation, error) {
	for _, c := range m.conversations {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockHistoryManagerStore) DeleteConversation(id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedID = id
	return nil
}

func (m *mockHistoryManagerStore) UpdateTitle(id, title string) error {
	if m.updateTitleErr != nil {
		return m.updateTitleErr
	}
	m.updatedTitleID = id
	m.updatedTitle = title
	return nil
}

func (m *mockHistoryManagerStore) ToggleFavorite(id string) (bool, error) {
	if m.toggleFavoriteErr != nil {
		return false, m.toggleFavoriteErr
	}
	m.toggledFavoriteID = id
	m.favoriteState = !m.favoriteState
	return m.favoriteState, nil
}

func (m *mockHistoryManagerStore) MoveConversation(id string, newIndex int) error {
	return nil
}

func (m *mockHistoryManagerStore) SwapConversations(id1, id2 string) error {
	if m.swapErr != nil {
		return m.swapErr
	}
	m.swappedIDs = []string{id1, id2}
	return nil
}

func (m *mockHistoryManagerStore) ExportToMarkdown(id string) (string, error) {
	if m.exportErr != nil {
		return "", m.exportErr
	}
	m.exportedID = id
	return "# Exported conversation", nil
}

// Helper function to create test conversations
func createTestConversations() []*history.Conversation {
	return []*history.Conversation{
		{ID: "conv-1", Title: "First Chat", Model: "gemini-2.5-flash", IsFavorite: false, UpdatedAt: time.Now()},
		{ID: "conv-2", Title: "Second Chat", Model: "gemini-3.0-pro", IsFavorite: true, UpdatedAt: time.Now().Add(-1 * time.Hour)},
		{ID: "conv-3", Title: "Third Chat", Model: "gemini-2.5-flash", IsFavorite: false, UpdatedAt: time.Now().Add(-2 * time.Hour)},
	}
}

func TestNewHistoryManagerModel(t *testing.T) {
	store := &mockHistoryManagerStore{}
	m := NewHistoryManagerModel(store)

	if m.store != store {
		t.Error("Store not set correctly")
	}
	if !m.loading {
		t.Error("Model should be loading initially")
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.mode != ModeNormal {
		t.Errorf("mode = %d, want ModeNormal", m.mode)
	}
	if m.filter != FilterAll {
		t.Errorf("filter = %d, want FilterAll", m.filter)
	}
}

func TestHistoryManagerModel_Init(t *testing.T) {
	store := &mockHistoryManagerStore{}
	m := NewHistoryManagerModel(store)

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestHistoryManagerModel_Update_WindowSize(t *testing.T) {
	store := &mockHistoryManagerStore{}
	m := NewHistoryManagerModel(store)

	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(msg)

	if model, ok := updatedModel.(HistoryManagerModel); ok {
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
		t.Error("Update should return HistoryManagerModel")
	}
}

func TestHistoryManagerModel_Update_HistoryLoaded(t *testing.T) {
	store := &mockHistoryManagerStore{}
	m := NewHistoryManagerModel(store)
	m.ready = true

	conversations := createTestConversations()
	msg := historyManagerLoadedMsg{conversations: conversations}
	updatedModel, _ := m.Update(msg)

	if model, ok := updatedModel.(HistoryManagerModel); ok {
		if model.loading {
			t.Error("Model should not be loading after historyManagerLoadedMsg")
		}
		if len(model.conversations) != 3 {
			t.Errorf("conversations = %d, want 3", len(model.conversations))
		}
		if len(model.filteredConversations) != 3 {
			t.Errorf("filteredConversations = %d, want 3", len(model.filteredConversations))
		}
	} else {
		t.Error("Update should return HistoryManagerModel")
	}
}

func TestHistoryManagerModel_Update_HistoryLoadedError(t *testing.T) {
	store := &mockHistoryManagerStore{}
	m := NewHistoryManagerModel(store)
	m.ready = true

	msg := historyManagerLoadedMsg{err: errors.New("load error")}
	updatedModel, _ := m.Update(msg)

	if model, ok := updatedModel.(HistoryManagerModel); ok {
		if model.loading {
			t.Error("Model should not be loading after error")
		}
		if model.err == nil {
			t.Error("Error should be set")
		}
	} else {
		t.Error("Update should return HistoryManagerModel")
	}
}

func TestHistoryManagerModel_Navigation(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()

	t.Run("down key moves cursor", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 1 {
			t.Errorf("cursor = %d, want 1", model.cursor)
		}
	})

	t.Run("up key moves cursor", func(t *testing.T) {
		m.cursor = 1
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 0 {
			t.Errorf("cursor = %d, want 0", model.cursor)
		}
	})

	t.Run("up key wraps around", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 2 {
			t.Errorf("cursor = %d, want 2 (wrap around)", model.cursor)
		}
	})

	t.Run("down key wraps around", func(t *testing.T) {
		m.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 0 {
			t.Errorf("cursor = %d, want 0 (wrap around)", model.cursor)
		}
	})

	t.Run("j key moves down", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 1 {
			t.Errorf("cursor = %d, want 1", model.cursor)
		}
	})

	t.Run("k key moves up", func(t *testing.T) {
		m.cursor = 1
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 0 {
			t.Errorf("cursor = %d, want 0", model.cursor)
		}
	})

	t.Run("g key goes to beginning", func(t *testing.T) {
		m.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 0 {
			t.Errorf("cursor = %d, want 0", model.cursor)
		}
	})

	t.Run("G key goes to end", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 2 {
			t.Errorf("cursor = %d, want 2", model.cursor)
		}
	})
}

func TestHistoryManagerModel_Quit(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()

	t.Run("q key quits", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		updatedModel, cmd := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if !model.shouldQuit {
			t.Error("shouldQuit should be true")
		}
		if cmd == nil {
			t.Error("should return quit command")
		}
	})

	t.Run("esc key quits", func(t *testing.T) {
		m.shouldQuit = false
		msg := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, cmd := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if !model.shouldQuit {
			t.Error("shouldQuit should be true")
		}
		if cmd == nil {
			t.Error("should return quit command")
		}
	})

	t.Run("ctrl+c quits", func(t *testing.T) {
		m.shouldQuit = false
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		updatedModel, cmd := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if !model.shouldQuit {
			t.Error("shouldQuit should be true")
		}
		if cmd == nil {
			t.Error("should return quit command")
		}
	})
}

func TestHistoryManagerModel_SelectConversation(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()
	m.cursor = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)
	model := updatedModel.(HistoryManagerModel)

	if model.selectedConv == nil {
		t.Error("selectedConv should not be nil")
	}
	if model.selectedConv.ID != "conv-2" {
		t.Errorf("selectedConv.ID = %s, want conv-2", model.selectedConv.ID)
	}
	if cmd == nil {
		t.Error("should return quit command")
	}
}

func TestHistoryManagerModel_ToggleFavorite(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()
	m.cursor = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	updatedModel, cmd := m.Update(msg)
	model := updatedModel.(HistoryManagerModel)

	if store.toggledFavoriteID != "conv-1" {
		t.Errorf("toggledFavoriteID = %s, want conv-1", store.toggledFavoriteID)
	}
	if model.feedback == "" {
		t.Error("feedback should be set")
	}
	if cmd == nil {
		t.Error("should return reload command")
	}
}

func TestHistoryManagerModel_RenameMode(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()
	m.cursor = 0

	t.Run("r key enters rename mode", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeRename {
			t.Errorf("mode = %d, want ModeRename", model.mode)
		}
		if model.renameID != "conv-1" {
			t.Errorf("renameID = %s, want conv-1", model.renameID)
		}
	})

	t.Run("esc exits rename mode", func(t *testing.T) {
		m.mode = ModeRename
		m.renameID = "conv-1"
		msg := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeNormal {
			t.Errorf("mode = %d, want ModeNormal", model.mode)
		}
	})

	t.Run("enter confirms rename", func(t *testing.T) {
		m.mode = ModeRename
		m.renameID = "conv-1"
		m.renameInput.SetValue("New Title")
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeNormal {
			t.Errorf("mode = %d, want ModeNormal", model.mode)
		}
		if store.updatedTitleID != "conv-1" {
			t.Errorf("updatedTitleID = %s, want conv-1", store.updatedTitleID)
		}
		if store.updatedTitle != "New Title" {
			t.Errorf("updatedTitle = %s, want New Title", store.updatedTitle)
		}
		if cmd == nil {
			t.Error("should return reload command")
		}
	})
}

func TestHistoryManagerModel_SearchMode(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()

	t.Run("/ key enters search mode", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeSearch {
			t.Errorf("mode = %d, want ModeSearch", model.mode)
		}
	})

	t.Run("esc exits search mode", func(t *testing.T) {
		m.mode = ModeSearch
		m.searchActive = true
		m.searchQuery = "test"
		msg := tea.KeyMsg{Type: tea.KeyEscape}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeNormal {
			t.Errorf("mode = %d, want ModeNormal", model.mode)
		}
		if model.searchActive {
			t.Error("searchActive should be false")
		}
		if model.searchQuery != "" {
			t.Error("searchQuery should be empty")
		}
	})

	t.Run("enter confirms search", func(t *testing.T) {
		m.mode = ModeSearch
		m.searchInput.SetValue("First")
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeNormal {
			t.Errorf("mode = %d, want ModeNormal", model.mode)
		}
		if !model.searchActive {
			t.Error("searchActive should be true")
		}
		if model.searchQuery != "First" {
			t.Errorf("searchQuery = %s, want First", model.searchQuery)
		}
	})
}

func TestHistoryManagerModel_DeleteMode(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()
	m.cursor = 0

	t.Run("d key enters delete mode", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeConfirmDelete {
			t.Errorf("mode = %d, want ModeConfirmDelete", model.mode)
		}
		if model.deleteID != "conv-1" {
			t.Errorf("deleteID = %s, want conv-1", model.deleteID)
		}
	})

	t.Run("n cancels delete", func(t *testing.T) {
		m.mode = ModeConfirmDelete
		m.deleteID = "conv-1"
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeNormal {
			t.Errorf("mode = %d, want ModeNormal", model.mode)
		}
	})

	t.Run("y confirms delete", func(t *testing.T) {
		m.mode = ModeConfirmDelete
		m.deleteID = "conv-1"
		m.deleteTitle = "First Chat"
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		updatedModel, cmd := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.mode != ModeNormal {
			t.Errorf("mode = %d, want ModeNormal", model.mode)
		}
		if store.deletedID != "conv-1" {
			t.Errorf("deletedID = %s, want conv-1", store.deletedID)
		}
		if cmd == nil {
			t.Error("should return reload command")
		}
	})
}

func TestHistoryManagerModel_FilterToggle(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()

	t.Run("tab toggles filter", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyTab}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.filter != FilterFavorites {
			t.Errorf("filter = %d, want FilterFavorites", model.filter)
		}
		// Only the favorite conversation should be visible
		if len(model.filteredConversations) != 1 {
			t.Errorf("filteredConversations = %d, want 1", len(model.filteredConversations))
		}
	})

	t.Run("tab again toggles back", func(t *testing.T) {
		m.filter = FilterFavorites
		m.applyFilter()
		msg := tea.KeyMsg{Type: tea.KeyTab}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.filter != FilterAll {
			t.Errorf("filter = %d, want FilterAll", model.filter)
		}
	})
}

func TestHistoryManagerModel_ApplyFilter(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.conversations = store.conversations

	t.Run("filter all shows all", func(t *testing.T) {
		m.filter = FilterAll
		m.searchActive = false
		m.applyFilter()
		if len(m.filteredConversations) != 3 {
			t.Errorf("filteredConversations = %d, want 3", len(m.filteredConversations))
		}
	})

	t.Run("filter favorites shows only favorites", func(t *testing.T) {
		m.filter = FilterFavorites
		m.searchActive = false
		m.applyFilter()
		if len(m.filteredConversations) != 1 {
			t.Errorf("filteredConversations = %d, want 1", len(m.filteredConversations))
		}
	})

	t.Run("search filter works", func(t *testing.T) {
		m.filter = FilterAll
		m.searchActive = true
		m.searchQuery = "First"
		m.applyFilter()
		if len(m.filteredConversations) != 1 {
			t.Errorf("filteredConversations = %d, want 1", len(m.filteredConversations))
		}
		if m.filteredConversations[0].Title != "First Chat" {
			t.Errorf("filtered wrong conversation")
		}
	})

	t.Run("cursor adjusts when out of bounds", func(t *testing.T) {
		m.filter = FilterAll
		m.searchActive = false
		m.cursor = 10
		m.applyFilter()
		if m.cursor != 2 {
			t.Errorf("cursor = %d, want 2", m.cursor)
		}
	})
}

func TestHistoryManagerModel_View(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.width = 80
	m.height = 40

	t.Run("not ready shows initializing", func(t *testing.T) {
		m.ready = false
		view := m.View()
		if !strings.Contains(view, "Initializing") {
			t.Error("should show initializing")
		}
	})

	t.Run("loading shows loading", func(t *testing.T) {
		m.ready = true
		m.loading = true
		view := m.View()
		if !strings.Contains(view, "Loading") {
			t.Error("should show loading")
		}
	})

	t.Run("error shows error", func(t *testing.T) {
		m.ready = true
		m.loading = false
		m.err = errors.New("test error")
		view := m.View()
		if !strings.Contains(view, "Error") {
			t.Error("should show error")
		}
	})

	t.Run("normal view shows conversations", func(t *testing.T) {
		m.ready = true
		m.loading = false
		m.err = nil
		m.conversations = store.conversations
		m.applyFilter()
		view := m.View()
		if !strings.Contains(view, "First Chat") {
			t.Error("should show conversation title")
		}
		if !strings.Contains(view, "Conversation Manager") {
			t.Error("should show header")
		}
	})

	t.Run("feedback is shown", func(t *testing.T) {
		m.feedback = "Test feedback"
		view := m.View()
		if !strings.Contains(view, "Test feedback") {
			t.Error("should show feedback")
		}
	})

	t.Run("rename mode shows input", func(t *testing.T) {
		m.feedback = ""
		m.mode = ModeRename
		view := m.View()
		if !strings.Contains(view, "Rename") {
			t.Error("should show rename label")
		}
	})

	t.Run("search mode shows input", func(t *testing.T) {
		m.mode = ModeSearch
		view := m.View()
		if !strings.Contains(view, "Search") {
			t.Error("should show search label")
		}
	})

	t.Run("delete mode shows confirmation", func(t *testing.T) {
		m.mode = ModeConfirmDelete
		m.deleteTitle = "Test Title"
		view := m.View()
		if !strings.Contains(view, "Delete") {
			t.Error("should show delete confirmation")
		}
	})
}

func TestHistoryManagerModel_RenderStatusBar(t *testing.T) {
	store := &mockHistoryManagerStore{}
	m := NewHistoryManagerModel(store)
	m.width = 80
	m.height = 40

	t.Run("normal mode status bar", func(t *testing.T) {
		m.mode = ModeNormal
		bar := m.renderStatusBar(80)
		if !strings.Contains(bar, "Nav") {
			t.Error("should show Nav shortcut")
		}
		if !strings.Contains(bar, "Quit") {
			t.Error("should show Quit shortcut")
		}
	})

	t.Run("rename mode status bar", func(t *testing.T) {
		m.mode = ModeRename
		bar := m.renderStatusBar(80)
		if !strings.Contains(bar, "Save") {
			t.Error("should show Save shortcut")
		}
		if !strings.Contains(bar, "Cancel") {
			t.Error("should show Cancel shortcut")
		}
	})

	t.Run("search mode status bar", func(t *testing.T) {
		m.mode = ModeSearch
		bar := m.renderStatusBar(80)
		if !strings.Contains(bar, "Search") {
			t.Error("should show Search shortcut")
		}
	})

	t.Run("delete mode status bar", func(t *testing.T) {
		m.mode = ModeConfirmDelete
		bar := m.renderStatusBar(80)
		if !strings.Contains(bar, "Delete") {
			t.Error("should show Delete shortcut")
		}
	})
}

func TestHistoryManagerModel_Result(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)

	t.Run("no selection returns nil", func(t *testing.T) {
		conv, quit := m.Result()
		if conv != nil {
			t.Error("conv should be nil")
		}
		if quit {
			t.Error("quit should be false")
		}
	})

	t.Run("selection returns conversation", func(t *testing.T) {
		m.selectedConv = store.conversations[0]
		conv, quit := m.Result()
		if conv == nil {
			t.Fatal("conv should not be nil")
		}
		if conv.ID != "conv-1" {
			t.Errorf("conv.ID = %s, want conv-1", conv.ID)
		}
		if quit {
			t.Error("quit should be false")
		}
	})

	t.Run("quit returns quit flag", func(t *testing.T) {
		m.selectedConv = nil
		m.shouldQuit = true
		conv, quit := m.Result()
		if conv != nil {
			t.Error("conv should be nil")
		}
		if !quit {
			t.Error("quit should be true")
		}
	})
}

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		title  string
		maxLen int
		want   string
	}{
		{"Short", 10, "Short"},
		{"Exactly ten", 10, "Exactly te..."},
		{"A very long title that should be truncated", 20, "A very long title th..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := truncateTitle(tt.title, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateTitle(%q, %d) = %q, want %q", tt.title, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestHistoryManagerModel_MoveConversations(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()
	m.cursor = 1

	t.Run("ctrl+up moves conversation up", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyCtrlUp}
		updatedModel, cmd := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 0 {
			t.Errorf("cursor = %d, want 0", model.cursor)
		}
		if store.swappedIDs[0] != "conv-2" || store.swappedIDs[1] != "conv-1" {
			t.Error("wrong conversations swapped")
		}
		if cmd == nil {
			t.Error("should return reload command")
		}
	})

	t.Run("ctrl+down moves conversation down", func(t *testing.T) {
		m.cursor = 0
		store.swappedIDs = nil
		msg := tea.KeyMsg{Type: tea.KeyCtrlDown}
		updatedModel, cmd := m.Update(msg)
		model := updatedModel.(HistoryManagerModel)
		if model.cursor != 1 {
			t.Errorf("cursor = %d, want 1", model.cursor)
		}
		if cmd == nil {
			t.Error("should return reload command")
		}
	})
}

func TestHistoryManagerModel_HelpShortcut(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(HistoryManagerModel)

	if model.feedback == "" {
		t.Error("? should set feedback with help")
	}
	if !strings.Contains(model.feedback, "Nav") {
		t.Error("help should contain Nav")
	}
}

func TestHistoryManagerModel_ExportShortcut(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = false
	m.ready = true
	m.conversations = store.conversations
	m.applyFilter()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(HistoryManagerModel)

	if model.feedback == "" {
		t.Error("e should set feedback")
	}
	if !strings.Contains(model.feedback, "geminiweb history export") {
		t.Error("feedback should contain export command hint")
	}
}

func TestHistoryManagerModel_LoadingIgnoresKeys(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.loading = true // Still loading
	m.ready = true

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := m.Update(msg)
	model := updatedModel.(HistoryManagerModel)

	if model.cursor != 0 {
		t.Error("cursor should not change while loading")
	}
	if cmd != nil {
		t.Error("should not return command while loading")
	}
}

func TestHistoryManagerModel_RenderItem(t *testing.T) {
	store := &mockHistoryManagerStore{conversations: createTestConversations()}
	m := NewHistoryManagerModel(store)
	m.conversations = store.conversations
	m.applyFilter()
	m.cursor = 0

	t.Run("renders with cursor", func(t *testing.T) {
		item := m.renderItem(0, store.conversations[0], 60)
		if !strings.Contains(item, "▸") {
			t.Error("should show cursor indicator")
		}
		if !strings.Contains(item, "First Chat") {
			t.Error("should show title")
		}
	})

	t.Run("renders without cursor", func(t *testing.T) {
		item := m.renderItem(1, store.conversations[1], 60)
		if strings.Contains(item, "▸") {
			t.Error("should not show cursor indicator")
		}
	})

	t.Run("renders favorite star", func(t *testing.T) {
		item := m.renderItem(1, store.conversations[1], 60)
		if !strings.Contains(item, "★") {
			t.Error("should show favorite star")
		}
	})

	t.Run("truncates long title", func(t *testing.T) {
		longConv := &history.Conversation{
			ID:    "long",
			Title: "This is a very long title that should definitely be truncated because it is too long",
		}
		item := m.renderItem(0, longConv, 60)
		if !strings.Contains(item, "...") {
			t.Error("should truncate long title")
		}
	})
}
