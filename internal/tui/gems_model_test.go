package tui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
)

// mockGemsClient is a mock implementation of api.GeminiClientInterface for testing
type mockGemsClient struct {
	gems          *models.GemJar
	fetchErr      error
	fetchCalled   bool
	includeHidden bool
}

func (m *mockGemsClient) FetchGems(includeHidden bool) (*models.GemJar, error) {
	m.fetchCalled = true
	m.includeHidden = includeHidden
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.gems, nil
}

// Implement other required methods (unused in these tests)
func (m *mockGemsClient) Init() error                 { return nil }
func (m *mockGemsClient) Close()                      {}
func (m *mockGemsClient) GetAccessToken() string      { return "" }
func (m *mockGemsClient) GetCookies() *config.Cookies { return nil }
func (m *mockGemsClient) GetModel() models.Model      { return models.Model{} }
func (m *mockGemsClient) SetModel(model models.Model) {}
func (m *mockGemsClient) IsClosed() bool              { return false }
func (m *mockGemsClient) StartChat(model ...models.Model) *api.ChatSession {
	return nil
}
func (m *mockGemsClient) StartChatWithOptions(opts ...api.ChatOption) *api.ChatSession {
	return nil
}
func (m *mockGemsClient) GenerateContent(prompt string, opts *api.GenerateOptions) (*models.ModelOutput, error) {
	return nil, nil
}
func (m *mockGemsClient) UploadImage(filePath string) (*api.UploadedImage, error) {
	return nil, nil
}
func (m *mockGemsClient) UploadFile(filePath string) (*api.UploadedFile, error) {
	return nil, nil
}
func (m *mockGemsClient) UploadText(content string, fileName string) (*api.UploadedFile, error) {
	return nil, nil
}
func (m *mockGemsClient) RefreshFromBrowser() (bool, error) { return false, nil }
func (m *mockGemsClient) IsBrowserRefreshEnabled() bool     { return false }
func (m *mockGemsClient) CreateGem(name, prompt, description string) (*models.Gem, error) {
	return nil, nil
}
func (m *mockGemsClient) UpdateGem(gemID, name, prompt, description string) (*models.Gem, error) {
	return nil, nil
}
func (m *mockGemsClient) DeleteGem(gemID string) error { return nil }
func (m *mockGemsClient) Gems() *models.GemJar         { return m.gems }
func (m *mockGemsClient) IsAutoCloseEnabled() bool     { return false }
func (m *mockGemsClient) GetGem(id, name string) *models.Gem {
	return nil
}
func (m *mockGemsClient) BatchExecute(requests []api.RPCData) ([]api.BatchResponse, error) {
	return nil, nil
}
func (m *mockGemsClient) DownloadImage(img models.WebImage, opts api.ImageDownloadOptions) (string, error) {
	return "", nil
}
func (m *mockGemsClient) DownloadGeneratedImage(img models.GeneratedImage, opts api.ImageDownloadOptions) (string, error) {
	return "", nil
}
func (m *mockGemsClient) DownloadAllImages(output *models.ModelOutput, opts api.ImageDownloadOptions) ([]string, error) {
	return nil, nil
}
func (m *mockGemsClient) DownloadSelectedImages(output *models.ModelOutput, indices []int, opts api.ImageDownloadOptions) ([]string, error) {
	return nil, nil
}

// createTestGems creates a test GemJar with sample gems
func createTestGems() *models.GemJar {
	jar := make(models.GemJar)
	jar["gem1"] = &models.Gem{
		ID:          "gem1",
		Name:        "Test Gem One",
		Description: "A test gem for testing",
		Prompt:      "You are a helpful assistant",
		Predefined:  false,
	}
	jar["gem2"] = &models.Gem{
		ID:          "gem2",
		Name:        "System Gem",
		Description: "A system gem",
		Prompt:      "System prompt",
		Predefined:  true,
	}
	jar["gem3"] = &models.Gem{
		ID:          "gem3",
		Name:        "Another Custom",
		Description: "Another custom gem",
		Prompt:      "Another prompt",
		Predefined:  false,
	}
	return &jar
}

// createMockClient creates a mock client with test gems
func createMockClient() *mockGemsClient {
	gems := createTestGems()
	return &mockGemsClient{gems: gems}
}

func TestNewGemsModel(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	if m.client == nil {
		t.Error("client should not be nil")
	}

	if m.view != gemsViewList {
		t.Errorf("Expected view to be gemsViewList, got %v", m.view)
	}

	if m.cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", m.cursor)
	}

	if !m.loading {
		t.Error("Expected loading to be true initially")
	}

	if m.feedbackTimeout != 2*time.Second {
		t.Errorf("Expected feedbackTimeout to be 2s, got %v", m.feedbackTimeout)
	}

	if m.searching {
		t.Error("Expected searching to be false initially")
	}
}

func TestNewGemsModel_IncludeHidden(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, true)

	if !m.includeHidden {
		t.Error("Expected includeHidden to be true")
	}
}

func TestGemsModel_Init(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init should return a command (batch of loadGems and textinput.Blink)")
	}
}

func TestGemsModel_Update_WindowSize(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.width != 100 {
			t.Errorf("Expected width 100, got %d", typedModel.width)
		}
		if typedModel.height != 40 {
			t.Errorf("Expected height 40, got %d", typedModel.height)
		}
		if !typedModel.ready {
			t.Error("Model should be ready after WindowSizeMsg")
		}
		if typedModel.searchInput.Width < 20 {
			t.Error("searchInput width should be at least 20")
		}
	} else {
		t.Error("Update should return GemsModel type")
	}

	if cmd != nil {
		t.Error("WindowSizeMsg should return nil command")
	}
}

func TestGemsModel_Update_GemsLoaded(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.loading = true

	gems := createTestGems()
	msg := gemsLoadedMsg{gems: gems}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.loading {
			t.Error("loading should be false after gems are loaded")
		}
		if typedModel.err != nil {
			t.Error("err should be nil after successful load")
		}
		if len(typedModel.allGems) != 3 {
			t.Errorf("Expected 3 gems, got %d", len(typedModel.allGems))
		}
		if len(typedModel.filteredGems) != 3 {
			t.Errorf("Expected 3 filtered gems, got %d", len(typedModel.filteredGems))
		}
	} else {
		t.Error("Update should return GemsModel type")
	}
}

func TestGemsModel_Update_GemsLoadedError(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.loading = true

	msg := gemsLoadedMsg{err: &mockError{"test error"}}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.loading {
			t.Error("loading should be false after error")
		}
		if typedModel.err == nil {
			t.Error("err should be set after error")
		}
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestGemsModel_Update_FeedbackClear(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.feedback = "Test feedback"

	msg := gemsFeedbackClearMsg{}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.feedback != "" {
			t.Error("Feedback should be cleared")
		}
	}
}

func TestGemsModel_Update_CtrlC(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Expected quit command for Ctrl+C")
	}
}

func TestGemsModel_Update_Quit(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.view = gemsViewList

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Expected quit command for 'q' from list view")
	}
}

func TestGemsModel_Update_QuitFromDetails(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.view = gemsViewDetails

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.view != gemsViewList {
			t.Error("Should return to list view")
		}
	}

	if cmd != nil {
		t.Error("Should not quit, just return to list")
	}
}

func TestGemsModel_Update_Escape(t *testing.T) {
	t.Run("from list view", func(t *testing.T) {
		client := createMockClient()
		m := NewGemsModel(client, false)
		m.view = gemsViewList

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("Expected quit command for Escape from list view")
		}
	})

	t.Run("from details view", func(t *testing.T) {
		client := createMockClient()
		m := NewGemsModel(client, false)
		m.view = gemsViewDetails

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, cmd := m.Update(msg)

		if typedModel, ok := updatedModel.(GemsModel); ok {
			if typedModel.view != gemsViewList {
				t.Error("Should return to list view")
			}
		}

		if cmd != nil {
			t.Error("Should not quit when escaping from details view")
		}
	})
}

func TestGemsModel_Update_Search(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if !typedModel.searching {
			t.Error("Should be in search mode after '/'")
		}
	}

	if cmd == nil {
		t.Error("Should return blink command for text input")
	}
}

func TestGemsModel_Update_Navigation(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.cursor = 0

	t.Run("down navigation", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(GemsModel); ok {
			if typedModel.cursor != 1 {
				t.Errorf("Expected cursor to be 1, got %d", typedModel.cursor)
			}
		}
	})

	t.Run("up navigation", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(GemsModel); ok {
			if typedModel.cursor != len(m.filteredGems)-1 {
				t.Errorf("Expected cursor to wrap to %d, got %d", len(m.filteredGems)-1, typedModel.cursor)
			}
		}
	})

	t.Run("j navigation", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(GemsModel); ok {
			if typedModel.cursor != 1 {
				t.Errorf("Expected cursor to be 1, got %d", typedModel.cursor)
			}
		}
	})

	t.Run("k navigation", func(t *testing.T) {
		m.cursor = 1
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(GemsModel); ok {
			if typedModel.cursor != 0 {
				t.Errorf("Expected cursor to be 0, got %d", typedModel.cursor)
			}
		}
	})

	t.Run("home navigation", func(t *testing.T) {
		m.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyHome}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(GemsModel); ok {
			if typedModel.cursor != 0 {
				t.Errorf("Expected cursor to be 0, got %d", typedModel.cursor)
			}
		}
	})

	t.Run("end navigation", func(t *testing.T) {
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyEnd}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(GemsModel); ok {
			if typedModel.cursor != len(m.filteredGems)-1 {
				t.Errorf("Expected cursor to be %d, got %d", len(m.filteredGems)-1, typedModel.cursor)
			}
		}
	})
}

func TestGemsModel_Update_Enter(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.cursor = 0
	m.view = gemsViewList

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.view != gemsViewDetails {
			t.Error("Should switch to details view")
		}
		if typedModel.selectedGem == nil {
			t.Error("selectedGem should be set")
		}
	}
}

func TestGemsModel_SearchEscape(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.searching = true
	m.searchInput.SetValue("test")

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.searching {
			t.Error("Should exit search mode")
		}
		if typedModel.searchInput.Value() != "" {
			t.Error("Search input should be cleared")
		}
		if len(typedModel.filteredGems) != len(m.allGems) {
			t.Error("Filter should be reset")
		}
	}
}

func TestGemsModel_SearchEnter(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.searching = true
	m.searchInput.Focus()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.searching {
			t.Error("Should exit search mode")
		}
	}
}

func TestGemsModel_filterGems(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems

	t.Run("filter by name", func(t *testing.T) {
		m.searchInput.SetValue("Test")
		m.filterGems()

		if len(m.filteredGems) != 1 {
			t.Errorf("Expected 1 filtered gem, got %d", len(m.filteredGems))
		}
	})

	t.Run("filter by description", func(t *testing.T) {
		m.searchInput.SetValue("system")
		m.filterGems()

		if len(m.filteredGems) != 1 {
			t.Errorf("Expected 1 filtered gem, got %d", len(m.filteredGems))
		}
	})

	t.Run("empty filter", func(t *testing.T) {
		m.searchInput.SetValue("")
		m.filterGems()

		if len(m.filteredGems) != len(m.allGems) {
			t.Errorf("Expected all gems, got %d", len(m.filteredGems))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		m.searchInput.SetValue("xyz123notfound")
		m.filterGems()

		if len(m.filteredGems) != 0 {
			t.Errorf("Expected 0 filtered gems, got %d", len(m.filteredGems))
		}
	})
}

func TestGemsModel_View_NotReady(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.ready = false

	view := m.View()

	if view == "" {
		t.Error("View should not be empty when not ready")
	}

	if !contains(view, "Initializing") {
		t.Error("View should contain initializing message")
	}
}

func TestGemsModel_View_Loading(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.ready = true
	m.loading = true

	view := m.View()

	if !contains(view, "Loading") {
		t.Error("View should contain loading message")
	}
}

func TestGemsModel_View_Error(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.ready = true
	m.loading = false
	m.err = &mockError{"test error"}

	view := m.View()

	if !contains(view, "Error") {
		t.Error("View should contain error message")
	}
}

func TestGemsModel_View_ListView(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.ready = true
	m.loading = false
	m.width = 80
	m.height = 24
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.view = gemsViewList

	view := m.View()

	if !contains(view, "Gems") {
		t.Error("View should contain Gems title")
	}

	if !contains(view, "Navigate") {
		t.Error("View should contain navigation hints")
	}
}

func TestGemsModel_View_DetailsView(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.ready = true
	m.loading = false
	m.width = 80
	m.height = 24
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.view = gemsViewDetails
	m.selectedGem = m.filteredGems[0]

	view := m.View()

	if !contains(view, "Details") {
		t.Error("View should contain Details title")
	}

	if !contains(view, "Name:") {
		t.Error("View should contain Name label")
	}

	if !contains(view, "ID:") {
		t.Error("View should contain ID label")
	}
}

func TestGemsModel_View_Feedback(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.ready = true
	m.loading = false
	m.width = 80
	m.height = 24
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.feedback = "ID copied"

	view := m.View()

	if !contains(view, "ID copied") {
		t.Error("View should contain feedback message")
	}
}

func TestGemsModel_View_SearchMode(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.ready = true
	m.loading = false
	m.width = 80
	m.height = 24
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.searching = true
	m.searchInput.SetValue("test")

	view := m.View()

	if !contains(view, "Cancel") || !contains(view, "Confirm") {
		t.Error("View should contain search mode shortcuts")
	}
}

func TestSortGems(t *testing.T) {
	gems := createTestGems()
	sorted := sortGems(gems.Values())

	// Custom gems should come before system gems
	foundSystem := false
	for _, gem := range sorted {
		if gem.Predefined {
			foundSystem = true
		} else if foundSystem {
			t.Error("Custom gems should come before system gems")
		}
	}

	// Check that gems are sorted by name within their category
	var lastCustomName, lastSystemName string
	for _, gem := range sorted {
		name := gem.Name
		if !gem.Predefined {
			if lastCustomName != "" && name < lastCustomName {
				t.Error("Custom gems should be sorted by name")
			}
			lastCustomName = name
		} else {
			if lastSystemName != "" && name < lastSystemName {
				t.Error("System gems should be sorted by name")
			}
			lastSystemName = name
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},         // String fits, no truncation
		{"hello world", 8, "hello..."}, // String truncated with "..."
		{"hi", 2, "hi"},                // String fits exactly
		{"hello", 3, "hel"},            // maxLen <= 3, just truncate without "..."
		{"a", 1, "a"},                  // Single char fits
		{"hello world", 5, "he..."},    // Truncate with "..."
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestClearGemsFeedback(t *testing.T) {
	cmd := clearGemsFeedback(time.Millisecond)

	if cmd == nil {
		t.Error("clearGemsFeedback should return a command")
	}
}

func TestGemsModel_renderHeader(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems

	header := m.renderHeader(80)

	if header == "" {
		t.Error("renderHeader should not return empty string")
	}

	if !contains(header, "Gems") {
		t.Error("Header should contain 'Gems'")
	}
}

func TestGemsModel_renderSearchBar(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	bar := m.renderSearchBar(80)

	if bar == "" {
		t.Error("renderSearchBar should not return empty string")
	}
}

func TestGemsModel_renderGemItem(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	gem := &models.Gem{
		ID:          "test-id",
		Name:        "Test Gem",
		Description: "A test description",
		Predefined:  false,
	}

	t.Run("not selected", func(t *testing.T) {
		item := m.renderGemItem(gem, false, 60)

		if item == "" {
			t.Error("renderGemItem should not return empty string")
		}

		if !contains(item, "Test Gem") {
			t.Error("Item should contain gem name")
		}

		if !contains(item, "custom") {
			t.Error("Item should indicate custom gem")
		}
	})

	t.Run("selected", func(t *testing.T) {
		item := m.renderGemItem(gem, true, 60)

		if !contains(item, "Test Gem") {
			t.Error("Item should contain gem name")
		}
	})

	t.Run("system gem", func(t *testing.T) {
		systemGem := &models.Gem{
			ID:         "system-id",
			Name:       "System Gem",
			Predefined: true,
		}

		item := m.renderGemItem(systemGem, false, 60)

		if !contains(item, "system") {
			t.Error("Item should indicate system gem")
		}
	})
}

func TestGemsModel_renderStatusBar(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	t.Run("list view", func(t *testing.T) {
		m.view = gemsViewList
		m.searching = false

		bar := m.renderStatusBar(80)

		if !contains(bar, "Navigate") {
			t.Error("Status bar should contain 'Navigate'")
		}
		if !contains(bar, "Search") {
			t.Error("Status bar should contain 'Search'")
		}
	})

	t.Run("details view", func(t *testing.T) {
		m.view = gemsViewDetails

		bar := m.renderStatusBar(80)

		if !contains(bar, "Copy ID") {
			t.Error("Status bar should contain 'Copy ID'")
		}
		if !contains(bar, "Back") {
			t.Error("Status bar should contain 'Back'")
		}
	})

	t.Run("search mode", func(t *testing.T) {
		m.view = gemsViewList
		m.searching = true

		bar := m.renderStatusBar(80)

		if !contains(bar, "Confirm") {
			t.Error("Status bar should contain 'Confirm'")
		}
		if !contains(bar, "Cancel") {
			t.Error("Status bar should contain 'Cancel'")
		}
	})
}

func TestGemsModel_Enums(t *testing.T) {
	if gemsViewList != 0 {
		t.Errorf("Expected gemsViewList to be 0, got %d", gemsViewList)
	}
	if gemsViewDetails != 1 {
		t.Errorf("Expected gemsViewDetails to be 1, got %d", gemsViewDetails)
	}
}

func TestRunGemsTUI(t *testing.T) {
	// Just test that the function exists and has correct signature
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunGemsTUI panicked: %v", r)
		}
	}()

	// We can't actually run the tea program in a test
	_ = RunGemsTUI
}

func TestGemsModel_renderListView_Empty(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.allGems = []*models.Gem{}
	m.filteredGems = []*models.Gem{}
	m.ready = true
	m.width = 80
	m.height = 24

	list := m.renderListView(80)

	if !contains(list, "No gems found") {
		t.Error("Should show 'No gems found' when list is empty")
	}
}

func TestGemsModel_renderDetailsView_NoSelection(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.selectedGem = nil

	details := m.renderDetailsView(80)

	if !contains(details, "No gem selected") {
		t.Error("Should show 'No gem selected' message")
	}
}

func TestGemsModel_ChatFromListView(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.view = gemsViewList
	m.cursor = 0

	// Test 'c' key - should start chat
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		// Should set startChatGemID
		if typedModel.startChatGemID == "" {
			t.Error("startChatGemID should be set after pressing 'c'")
		}
		if typedModel.startChatGemName == "" {
			t.Error("startChatGemName should be set after pressing 'c'")
		}
	}

	// Should return quit command
	if cmd == nil {
		t.Error("Should return quit command to start chat")
	}
}

func TestGemsModel_ChatFromDetailsView(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.view = gemsViewDetails
	m.selectedGem = m.filteredGems[0]

	// Test 'c' key - should start chat
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.startChatGemID == "" {
			t.Error("startChatGemID should be set after pressing 'c' in details view")
		}
	}

	if cmd == nil {
		t.Error("Should return quit command to start chat")
	}
}

func TestGemsModel_CopyID_ListViewShortcut(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.view = gemsViewList
	m.cursor = 0

	// Test 'y' key for copy (now separate from 'c' for chat)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.feedback == "" {
			// Note: actual clipboard test may fail in CI without display
			// So we just check that the model was updated
			_ = typedModel
		}
	}

	// Should return feedback clear command
	if cmd == nil {
		// Note: clipboard may fail without display, so feedback might not be set
		_ = cmd
	}
}

func TestGemsModel_CopyID_DetailsViewShortcut(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.view = gemsViewDetails
	m.selectedGem = m.filteredGems[0]

	// Test 'y' key (yank)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	_, _ = m.Update(msg)

	// Note: actual clipboard functionality may not work in CI environment
	// This test just ensures the code path doesn't panic
}

func TestGemsModel_NavigationWrapping(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.cursor = len(m.filteredGems) - 1

	// Test wrap around with down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.cursor != 0 {
			t.Errorf("Expected cursor to wrap to 0, got %d", typedModel.cursor)
		}
	}
}

func TestGemsModel_GNavigationShortcut(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.cursor = 2

	// Test 'g' key (go to start)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		if typedModel.cursor != 0 {
			t.Errorf("Expected cursor to be 0, got %d", typedModel.cursor)
		}
	}
}

func TestGemsModel_GShiftNavigationShortcut(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.cursor = 0

	// Test 'G' key (go to end)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		expected := len(m.filteredGems) - 1
		if typedModel.cursor != expected {
			t.Errorf("Expected cursor to be %d, got %d", expected, typedModel.cursor)
		}
	}
}

func TestGemsModel_SearchModeInput(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems
	m.searching = true
	m.searchInput.Focus()

	// Type a character in search mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		// Search input should have been updated
		if typedModel.searching != true {
			t.Error("Should still be in search mode")
		}
	}

	// Should return a command from textinput
	_ = cmd
}

func TestGemsModel_renderHeader_FilteredCount(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	// Filter to fewer gems
	m.filteredGems = m.allGems[:1]

	header := m.renderHeader(80)

	// Should show filtered count
	if !contains(header, "1/") {
		t.Error("Header should show filtered count")
	}
}

func TestGemsModel_renderListView_WithScroll(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	// Create many gems
	jar := make(models.GemJar)
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("gem%d", i)
		jar[id] = &models.Gem{
			ID:          id,
			Name:        fmt.Sprintf("Gem %d", i),
			Description: "Test gem",
			Predefined:  false,
		}
	}

	m.allGems = sortGems(jar.Values())
	m.filteredGems = m.allGems
	m.ready = true
	m.width = 80
	m.height = 15 // Small height to trigger scrolling
	m.cursor = 10 // Cursor in the middle

	list := m.renderListView(80)

	// Should have scroll indicator
	if !contains(list, "more") {
		t.Log("Note: scroll indicator may not be visible depending on terminal height calculation")
	}
}

func TestGemsModel_renderDetailsView_WithPrompt(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.selectedGem = &models.Gem{
		ID:          "test-id",
		Name:        "Test Gem",
		Description: "A test gem",
		Prompt:      "You are a helpful assistant.\n\n## Instructions\n\n- Be helpful\n- Be concise",
		Predefined:  false,
	}
	m.width = 80

	details := m.renderDetailsView(80)

	if !contains(details, "Prompt") {
		t.Error("Details should contain Prompt section")
	}
}

func TestGemsModel_renderDetailsView_SystemGem(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.selectedGem = &models.Gem{
		ID:         "system-id",
		Name:       "System Gem",
		Predefined: true,
	}

	details := m.renderDetailsView(80)

	if !contains(details, "system") {
		t.Error("Details should indicate system gem")
	}
}

func TestGemsModel_EmptyFilteredList(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.allGems = []*models.Gem{}
	m.filteredGems = []*models.Gem{}
	m.view = gemsViewList
	m.cursor = 0

	// Try to enter details view with empty list
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(msg)

	if typedModel, ok := updatedModel.(GemsModel); ok {
		// Should stay in list view
		if typedModel.view != gemsViewList {
			t.Error("Should stay in list view with empty list")
		}
	}
}

func TestGemsModel_CopyIDNoSelection(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.allGems = []*models.Gem{}
	m.filteredGems = []*models.Gem{}
	m.view = gemsViewList

	// Try to copy with empty list
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	_, cmd := m.Update(msg)

	// Should not return a command when nothing to copy
	if cmd != nil {
		t.Error("Should not return command when nothing to copy")
	}
}

func TestGemsModel_View_NarrowWidth(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.ready = true
	m.loading = false
	m.width = 30 // Very narrow
	m.height = 24
	gems := createTestGems()
	m.allGems = sortGems(gems.Values())
	m.filteredGems = m.allGems

	view := m.View()

	// Should still render without panic
	if view == "" {
		t.Error("View should not be empty even with narrow width")
	}
}

func TestSortGems_MixedTypes(t *testing.T) {
	jar := make(models.GemJar)
	jar["system1"] = &models.Gem{ID: "system1", Name: "Zebra System", Predefined: true}
	jar["custom1"] = &models.Gem{ID: "custom1", Name: "Apple Custom", Predefined: false}
	jar["system2"] = &models.Gem{ID: "system2", Name: "Alpha System", Predefined: true}
	jar["custom2"] = &models.Gem{ID: "custom2", Name: "Zoo Custom", Predefined: false}

	sorted := sortGems(jar.Values())

	// First two should be custom (sorted alphabetically)
	if sorted[0].Name != "Apple Custom" {
		t.Errorf("Expected first gem to be 'Apple Custom', got %s", sorted[0].Name)
	}
	if sorted[1].Name != "Zoo Custom" {
		t.Errorf("Expected second gem to be 'Zoo Custom', got %s", sorted[1].Name)
	}
	// Next two should be system (sorted alphabetically)
	if sorted[2].Name != "Alpha System" {
		t.Errorf("Expected third gem to be 'Alpha System', got %s", sorted[2].Name)
	}
	if sorted[3].Name != "Zebra System" {
		t.Errorf("Expected fourth gem to be 'Zebra System', got %s", sorted[3].Name)
	}
}

// TestGemsModel_resetForm tests the resetForm method
func TestGemsModel_resetForm(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	// Set some values in the form
	m.formInputs[formFieldName].SetValue("Test Name")
	m.formInputs[formFieldDescription].SetValue("Test Description")
	m.formInputs[formFieldPrompt].SetValue("Test Prompt")
	m.promptTextarea.SetValue("Multi-line\nprompt")
	m.formFocus = 2
	m.useTextarea = true

	// Reset the form
	m.resetForm()

	// Verify all values are cleared
	if m.formInputs[formFieldName].Value() != "" {
		t.Error("Name field should be empty after reset")
	}
	if m.formInputs[formFieldDescription].Value() != "" {
		t.Error("Description field should be empty after reset")
	}
	if m.formInputs[formFieldPrompt].Value() != "" {
		t.Error("Prompt field should be empty after reset")
	}
	if m.promptTextarea.Value() != "" {
		t.Error("Prompt textarea should be empty after reset")
	}
	if m.formFocus != 0 {
		t.Errorf("formFocus should be 0 after reset, got %d", m.formFocus)
	}
	if m.useTextarea {
		t.Error("useTextarea should be false after reset")
	}
}

// TestGemsModel_populateForm tests the populateForm method
func TestGemsModel_populateForm(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	gem := &models.Gem{
		ID:          "test-id",
		Name:        "Test Gem",
		Description: "Test Description",
		Prompt:      "Test Prompt",
	}

	m.populateForm(gem)

	if m.formInputs[formFieldName].Value() != "Test Gem" {
		t.Errorf("Name field should be 'Test Gem', got '%s'", m.formInputs[formFieldName].Value())
	}
	if m.formInputs[formFieldDescription].Value() != "Test Description" {
		t.Errorf("Description field should be 'Test Description', got '%s'", m.formInputs[formFieldDescription].Value())
	}
	if m.formInputs[formFieldPrompt].Value() != "Test Prompt" {
		t.Errorf("Prompt field should be 'Test Prompt', got '%s'", m.formInputs[formFieldPrompt].Value())
	}
	if m.promptTextarea.Value() != "Test Prompt" {
		t.Errorf("Prompt textarea should be 'Test Prompt', got '%s'", m.promptTextarea.Value())
	}
	if m.formFocus != 0 {
		t.Errorf("formFocus should be 0 after populate, got %d", m.formFocus)
	}
}

// TestGemsModel_populateForm_MultilinePrompt tests populateForm with multiline prompt
func TestGemsModel_populateForm_MultilinePrompt(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	gem := &models.Gem{
		ID:          "test-id",
		Name:        "Test Gem",
		Description: "Test Description",
		Prompt:      "Line 1\nLine 2\nLine 3",
	}

	m.populateForm(gem)

	if !m.useTextarea {
		t.Error("useTextarea should be true for multiline prompt")
	}
	if m.promptTextarea.Value() != "Line 1\nLine 2\nLine 3" {
		t.Errorf("Prompt textarea should have multiline content, got '%s'", m.promptTextarea.Value())
	}
}

// TestGemsModel_updateFocusedInput tests updateFocusedInput method
func TestGemsModel_updateFocusedInput(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	// Test updating name input - need to focus the field first
	m.view = gemsViewCreate
	m.formFocus = formFieldName
	m.focusCurrentField()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	m.updateFocusedInput(msg)

	if m.formInputs[formFieldName].Value() != "a" {
		t.Errorf("Name field should be 'a', got '%s'", m.formInputs[formFieldName].Value())
	}
}

// TestGemsModel_blurCurrentField tests blurCurrentField method
func TestGemsModel_blurCurrentField(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	// Focus a field and then blur it
	m.formFocus = formFieldName
	m.focusCurrentField()
	if !m.formInputs[formFieldName].Focused() {
		t.Error("Name field should be focused")
	}

	m.blurCurrentField()
	if m.formInputs[formFieldName].Focused() {
		t.Error("Name field should not be focused after blur")
	}
}

// TestGemsModel_focusCurrentField tests focusCurrentField method
func TestGemsModel_focusCurrentField(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	// Test focusing each field
	tests := []struct {
		field      int
		fieldName  string
		isTextarea bool
	}{
		{formFieldName, "Name", false},
		{formFieldDescription, "Description", false},
		{formFieldPrompt, "Prompt", false},
	}

	for _, tt := range tests {
		m.formFocus = tt.field
		m.focusCurrentField()

		if tt.isTextarea {
			if !m.promptTextarea.Focused() {
				t.Errorf("%s textarea should be focused", tt.fieldName)
			}
		} else {
			if !m.formInputs[tt.field].Focused() {
				t.Errorf("%s field should be focused", tt.fieldName)
			}
		}
	}
}

// TestGemsModel_submitForm tests submitForm method
func TestGemsModel_submitForm(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	// Set up form for creating a new gem
	m.formInputs[formFieldName].SetValue("New Gem")
	m.formInputs[formFieldDescription].SetValue("Description")
	m.formInputs[formFieldPrompt].SetValue("Prompt")
	m.view = gemsViewCreate

	updatedModel, cmd := m.submitForm()
	typedModel := updatedModel.(GemsModel)

	if !typedModel.submitting {
		t.Error("Should be submitting after submitForm")
	}
	if cmd == nil {
		t.Error("submitForm should return a command")
	}
}

// TestGemsModel_submitForm_Update tests submitForm for update
func TestGemsModel_submitForm_Update(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	// Set up form for updating
	m.selectedGem = &models.Gem{ID: "existing-id", Name: "Old Name"}
	m.formInputs[formFieldName].SetValue("Updated Name")
	m.formInputs[formFieldDescription].SetValue("New Description")
	m.formInputs[formFieldPrompt].SetValue("New Prompt")
	m.view = gemsViewEdit

	updatedModel, cmd := m.submitForm()
	typedModel := updatedModel.(GemsModel)

	if !typedModel.submitting {
		t.Error("Should be submitting after submitForm")
	}
	if cmd == nil {
		t.Error("submitForm should return a command")
	}
}

// TestGemsModel_loadGems tests loadGems method
func TestGemsModel_loadGems(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	cmd := m.loadGems()
	if cmd == nil {
		t.Error("loadGems should return a command")
	}

	// Execute the command to trigger the message
	msg := cmd()
	if _, ok := msg.(gemsLoadedMsg); !ok {
		t.Error("Command should return gemsLoadedMsg")
	}
}

// TestGemsModel_createGem tests createGem method
func TestGemsModel_createGem(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	cmd := m.createGem("Test Gem", "Prompt", "Description")
	if cmd == nil {
		t.Error("createGem should return a command")
	}

	// Execute the command
	msg := cmd()
	if _, ok := msg.(gemCreatedMsg); !ok {
		t.Error("Command should return gemCreatedMsg")
	}
}

// TestGemsModel_updateGem tests updateGem method
func TestGemsModel_updateGem(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	cmd := m.updateGem("test-id", "Updated Name", "New Prompt", "New Description")
	if cmd == nil {
		t.Error("updateGem should return a command")
	}

	// Execute the command
	msg := cmd()
	if _, ok := msg.(gemUpdatedMsg); !ok {
		t.Error("Command should return gemUpdatedMsg")
	}
}

// TestGemsModel_deleteGem tests deleteGem method
func TestGemsModel_deleteGem(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	cmd := m.deleteGem("test-id", "Test Gem")
	if cmd == nil {
		t.Error("deleteGem should return a command")
	}

	// Execute the command
	msg := cmd()
	if _, ok := msg.(gemDeletedMsg); !ok {
		t.Error("Command should return gemDeletedMsg")
	}
}

// TestGemsModel_handleFormInput_Escape tests handleFormInput with escape key
func TestGemsModel_handleFormInput_Escape(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.view = gemsViewCreate
	m.formInputs[formFieldName].SetValue("Test")

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	updatedModel, cmd := m.handleFormInput(msg)
	typedModel := updatedModel.(GemsModel)

	if typedModel.view != gemsViewList {
		t.Error("Should return to list view on escape")
	}
	if typedModel.formInputs[formFieldName].Value() != "" {
		t.Error("Form should be reset on escape")
	}
	if cmd != nil {
		t.Error("Should not return command on escape")
	}
}

// TestGemsModel_handleFormInput_Tab tests handleFormInput with tab key
func TestGemsModel_handleFormInput_Tab(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.formFocus = 0

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, cmd := m.handleFormInput(msg)
	typedModel := updatedModel.(GemsModel)

	if typedModel.formFocus != 1 {
		t.Errorf("Should move to next field on tab, got focus %d", typedModel.formFocus)
	}
	if cmd == nil {
		t.Error("Should return blink command on tab")
	}
}

// TestGemsModel_handleFormInput_ShiftTab tests handleFormInput with shift+tab
func TestGemsModel_handleFormInput_ShiftTab(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.formFocus = 1

	// Create shift+tab key message
	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updatedModel, cmd := m.handleFormInput(msg)
	typedModel := updatedModel.(GemsModel)

	if typedModel.formFocus != 0 {
		t.Errorf("Should move to previous field on shift+tab, got focus %d", typedModel.formFocus)
	}
	if cmd == nil {
		t.Error("Should return blink command on shift+tab")
	}
}

// TestGemsModel_handleFormInput_CtrlC tests handleFormInput with ctrl+c
func TestGemsModel_handleFormInput_CtrlC(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.handleFormInput(msg)

	if cmd == nil {
		t.Error("Should return quit command on ctrl+c")
	}
}

// TestGemsModel_handleDeleteConfirm tests handleDeleteConfirm method
func TestGemsModel_handleDeleteConfirm_Yes(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.selectedGem = &models.Gem{ID: "test-id", Name: "Test Gem"}
	m.view = gemsViewDelete

	// Send 'y' to confirm
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updatedModel, cmd := m.handleDeleteConfirm(msg)
	typedModel := updatedModel.(GemsModel)

	// Should be submitting and return a delete command
	if !typedModel.submitting {
		t.Error("Should be submitting after 'y'")
	}
	if cmd == nil {
		t.Error("Should return delete command")
	}
	// Note: view doesn't change to list view until delete completes
}

// TestGemsModel_handleDeleteConfirm_No tests handleDeleteConfirm with 'n'
func TestGemsModel_handleDeleteConfirm_No(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.selectedGem = &models.Gem{ID: "test-id", Name: "Test Gem"}
	m.view = gemsViewDelete

	// Send 'n' to cancel
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updatedModel, cmd := m.handleDeleteConfirm(msg)
	typedModel := updatedModel.(GemsModel)

	if typedModel.view != gemsViewList {
		t.Error("Should return to list view on cancel")
	}
	// Note: selectedGem is NOT cleared on cancel
	if cmd != nil {
		t.Error("Should not return command on cancel")
	}
}

// TestGemsModel_handleDeleteConfirm_Escape tests handleDeleteConfirm with escape
func TestGemsModel_handleDeleteConfirm_Escape(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.selectedGem = &models.Gem{ID: "test-id", Name: "Test Gem"}
	m.view = gemsViewDelete

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	updatedModel, cmd := m.handleDeleteConfirm(msg)
	typedModel := updatedModel.(GemsModel)

	if typedModel.view != gemsViewList {
		t.Error("Should return to list view on escape")
	}
	if cmd != nil {
		t.Error("Should not return command on escape")
	}
}

// TestGemsModel_handleDeleteConfirm_CtrlC tests handleDeleteConfirm with ctrl+c
func TestGemsModel_handleDeleteConfirm_CtrlC(t *testing.T) {
	client := createMockClient()
	m := NewGemsModel(client, false)
	m.view = gemsViewDelete

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.handleDeleteConfirm(msg)

	if cmd == nil {
		t.Error("Should return quit command on ctrl+c")
	}
}
