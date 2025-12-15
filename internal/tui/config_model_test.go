package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diogo/geminiweb/internal/config"
)

func TestNewConfigModel(t *testing.T) {
	// Test that NewConfigModel creates a model without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewConfigModel panicked: %v", r)
		}
	}()

	m := NewConfigModel()

	if m.configDir == "" {
		t.Error("configDir should not be empty")
	}

	if m.cookiesPath == "" {
		t.Error("cookiesPath should not be empty")
	}

	if m.view != viewMain {
		t.Errorf("Expected view to be viewMain, got %v", m.view)
	}

	if m.cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", m.cursor)
	}

	if m.modelCursor < 0 {
		t.Error("modelCursor should be non-negative")
	}

	if m.feedbackTimeout != 2*time.Second {
		t.Errorf("Expected feedbackTimeout to be 2s, got %v", m.feedbackTimeout)
	}
}

func TestConfigModel_Init(t *testing.T) {
	m := NewConfigModel()
	cmd := m.Init()

	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestClearFeedback(t *testing.T) {
	cmd := clearFeedback(time.Millisecond)

	if cmd == nil {
		t.Error("clearFeedback should return a command")
	}
}

func TestConfigModel_Update_WindowSize(t *testing.T) {
	m := NewConfigModel()

	// Simulate WindowSizeMsg
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(ConfigModel); ok {
		if typedModel.width != 100 {
			t.Errorf("Expected width 100, got %d", typedModel.width)
		}
		if typedModel.height != 40 {
			t.Errorf("Expected height 40, got %d", typedModel.height)
		}
		if !typedModel.ready {
			t.Error("Model should be ready after WindowSizeMsg")
		}
	} else {
		t.Error("Update should return ConfigModel type")
	}

	if cmd != nil {
		t.Error("WindowSizeMsg should return nil command")
	}
}

func TestConfigModel_Update_feedbackClearMsg(t *testing.T) {
	m := NewConfigModel()
	m.feedback = "Test feedback"

	// Simulate feedbackClearMsg
	msg := feedbackClearMsg{}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(ConfigModel); ok {
		if typedModel.feedback != "" {
			t.Error("Feedback should be cleared")
		}
	} else {
		t.Error("Update should return ConfigModel type")
	}

	// Should return nil command for feedback clear
	if cmd != nil {
		t.Error("feedbackClearMsg should return nil command")
	}
}

func TestConfigModel_Update_CtrlC(t *testing.T) {
	m := NewConfigModel()

	// Simulate Ctrl+C
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd := m.Update(msg)

	// Should return tea.Quit command
	if cmd == nil {
		t.Error("Expected quit command for Ctrl+C")
	}

	// Model should remain unchanged
	if typedModel, ok := updatedModel.(ConfigModel); ok {
		if typedModel.view != m.view {
			t.Error("Model should remain unchanged")
		}
	}
}

func TestConfigModel_Update_Escape(t *testing.T) {
	t.Run("from main view", func(t *testing.T) {
		m := NewConfigModel()

		// Simulate Escape from main view
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, cmd := m.Update(msg)

		// Should return tea.Quit command
		if cmd == nil {
			t.Error("Expected quit command for Escape from main view")
		}
	})

	t.Run("from model select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewModelSelect

		// Simulate Escape from model select view
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, cmd := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			if typedModel.view != viewMain {
				t.Error("Should return to main view")
			}
		}

		// Should return nil command (not quit)
		if cmd != nil {
			t.Errorf("Should not quit when escaping from model select view, got cmd: %v", cmd)
		}
	})
}

func TestConfigModel_Update_Up(t *testing.T) {
	t.Run("from main view", func(t *testing.T) {
		m := NewConfigModel()
		m.cursor = 0

		// Simulate Up key
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			// Should wrap to last item
			if typedModel.cursor != menuItemCount-1 {
				t.Errorf("Expected cursor to wrap to %d, got %d", menuItemCount-1, typedModel.cursor)
			}
		}
	})

	t.Run("from model select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewModelSelect
		m.modelCursor = 0

		// Simulate Up key
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			models := config.AvailableModels()
			// Should wrap to last model
			if typedModel.modelCursor != len(models)-1 {
				t.Errorf("Expected modelCursor to wrap to %d, got %d", len(models)-1, typedModel.modelCursor)
			}
		}
	})
}

func TestConfigModel_Update_Down(t *testing.T) {
	t.Run("from main view", func(t *testing.T) {
		m := NewConfigModel()
		m.cursor = menuItemCount - 1

		// Simulate Down key
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			// Should wrap to first item
			if typedModel.cursor != 0 {
				t.Errorf("Expected cursor to wrap to 0, got %d", typedModel.cursor)
			}
		}
	})

	t.Run("from model select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewModelSelect
		models := config.AvailableModels()
		m.modelCursor = len(models) - 1

		// Simulate Down key
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			// Should wrap to first model
			if typedModel.modelCursor != 0 {
				t.Errorf("Expected modelCursor to wrap to 0, got %d", typedModel.modelCursor)
			}
		}
	})
}

func TestConfigModel_Update_Enter(t *testing.T) {
	t.Run("on default model", func(t *testing.T) {
		m := NewConfigModel()
		m.cursor = menuDefaultModel

		// Simulate Enter
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			if typedModel.view != viewModelSelect {
				t.Error("Should switch to model select view")
			}
		}

		if cmd != nil {
			t.Error("Enter on model select should return nil command")
		}
	})

	t.Run("on verbose", func(t *testing.T) {
		m := NewConfigModel()
		m.cursor = menuVerbose
		originalVerbose := m.config.Verbose

		// Simulate Enter
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			if typedModel.config.Verbose == originalVerbose {
				t.Error("Verbose should be toggled")
			}
			if typedModel.feedback == "" {
				t.Error("Should set feedback message")
			}
		}

		// Should return clear feedback command
		if cmd == nil {
			t.Error("Should return clear feedback command")
		}
	})

	t.Run("on auto close", func(t *testing.T) {
		m := NewConfigModel()
		m.cursor = menuAutoClose
		originalAutoClose := m.config.AutoClose

		// Simulate Enter
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			if typedModel.config.AutoClose == originalAutoClose {
				t.Error("AutoClose should be toggled")
			}
			if typedModel.feedback == "" {
				t.Error("Should set feedback message")
			}
		}

		// Should return clear feedback command
		if cmd == nil {
			t.Error("Should return clear feedback command")
		}
	})

	t.Run("on exit", func(t *testing.T) {
		m := NewConfigModel()
		m.cursor = menuExit

		// Simulate Enter
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		// Should return tea.Quit command
		if cmd == nil {
			t.Error("Expected quit command for exit")
		}
	})
}

func TestConfigModel_Update_ModelSelect(t *testing.T) {
	m := NewConfigModel()
	m.view = viewModelSelect
	m.modelCursor = 0

	// Simulate Enter on a model
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)

	if typedModel, ok := updatedModel.(ConfigModel); ok {
		// Should return to main view
		if typedModel.view != viewMain {
			t.Error("Should return to main view after model selection")
		}

		// Should set default model
		models := config.AvailableModels()
		if typedModel.config.DefaultModel != models[0] {
			t.Error("Should update default model")
		}

		// Should set feedback
		if typedModel.feedback == "" {
			t.Error("Should set feedback message")
		}
	}

	// Should return clear feedback command
	if cmd == nil {
		t.Error("Should return clear feedback command")
	}
}

func TestConfigModel_View_NotReady(t *testing.T) {
	m := NewConfigModel()
	m.ready = false

	view := m.View()

	if view == "" {
		t.Error("View should not be empty when not ready")
	}

	// Should contain loading message
	if !contains(view, "Initializing") {
		t.Error("View should contain initializing message")
	}
}

func TestConfigModel_View_Ready(t *testing.T) {
	m := NewConfigModel()
	m.ready = true
	m.width = 80
	m.height = 24

	view := m.View()

	if view == "" {
		t.Error("View should not be empty when ready")
	}

	// Should contain config title
	if !contains(view, "Configuration") {
		t.Error("View should contain configuration title")
	}
}

func TestConfigModel_renderMainMenu(t *testing.T) {
	m := NewConfigModel()

	menu := m.renderMainMenu(80)

	if menu == "" {
		t.Error("renderMainMenu should not return empty string")
	}

	// Should contain menu items
	if !contains(menu, "Default Model") {
		t.Error("Menu should contain Default Model item")
	}
	if !contains(menu, "Verbose Logging") {
		t.Error("Menu should contain Verbose Logging item")
	}
	if !contains(menu, "Auto Close") {
		t.Error("Menu should contain Auto Close item")
	}
	if !contains(menu, "Exit") {
		t.Error("Menu should contain Exit item")
	}
}

func TestConfigModel_renderModelSelect(t *testing.T) {
	m := NewConfigModel()

	menu := m.renderModelSelect(80)

	if menu == "" {
		t.Error("renderModelSelect should not return empty string")
	}

	// Should contain model list
	models := config.AvailableModels()
	for _, model := range models {
		if !contains(menu, model) {
			t.Errorf("Menu should contain model: %s", model)
		}
	}
}

func TestConfigModel_renderBoolValue(t *testing.T) {
	t.Run("true value", func(t *testing.T) {
		m := NewConfigModel()
		result := m.renderBoolValue(true)

		if result == "" {
			t.Error("renderBoolValue should not return empty string")
		}

		// Should contain "enabled"
		if !contains(result, "enabled") {
			t.Error("renderBoolValue(true) should contain 'enabled'")
		}
	})

	t.Run("false value", func(t *testing.T) {
		m := NewConfigModel()
		result := m.renderBoolValue(false)

		if result == "" {
			t.Error("renderBoolValue should not return empty string")
		}

		// Should contain "disabled"
		if !contains(result, "disabled") {
			t.Error("renderBoolValue(false) should contain 'disabled'")
		}
	})
}

func TestConfigModel_renderStatusBar(t *testing.T) {
	t.Run("main view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewMain

		bar := m.renderStatusBar(80)

		if bar == "" {
			t.Error("renderStatusBar should not return empty string")
		}

		// Should contain navigation hints
		if !contains(bar, "Navigate") {
			t.Error("Status bar should contain 'Navigate'")
		}
		if !contains(bar, "Select") {
			t.Error("Status bar should contain 'Select'")
		}
		if !contains(bar, "Exit") {
			t.Error("Status bar should contain 'Exit'")
		}
	})

	t.Run("model select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewModelSelect

		bar := m.renderStatusBar(80)

		if bar == "" {
			t.Error("renderStatusBar should not return empty string")
		}

		// Should contain navigation hints
		if !contains(bar, "Back") {
			t.Error("Status bar should contain 'Back'")
		}
	})
}


func TestConfigModel_ThemeSelection(t *testing.T) {
	t.Run("escape from theme select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewThemeSelect

		// Simulate Escape from theme select view
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, cmd := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			if typedModel.view != viewMain {
				t.Error("Should return to main view")
			}
		}

		// Should return nil command (not quit)
		if cmd != nil {
			t.Errorf("Should not quit when escaping from theme select view, got cmd: %v", cmd)
		}
	})

	t.Run("navigate up in theme select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewThemeSelect
		m.themeCursor = 0

		// Simulate Up key
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			// Should wrap to last theme
			if typedModel.themeCursor == 0 {
				t.Error("Expected themeCursor to wrap to last theme")
			}
		}
	})

	t.Run("navigate down in theme select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewThemeSelect
		m.themeCursor = 6 // Last theme index (7 themes: 0-6)

		// Simulate Down key
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			// Should wrap to first theme
			if typedModel.themeCursor != 0 {
				t.Errorf("Expected themeCursor to wrap to 0, got %d", typedModel.themeCursor)
			}
		}
	})

	t.Run("enter on theme menu item", func(t *testing.T) {
		m := NewConfigModel()
		m.cursor = menuTheme

		// Simulate Enter
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			if typedModel.view != viewThemeSelect {
				t.Error("Should switch to theme select view")
			}
		}

		if cmd != nil {
			t.Error("Enter on theme select should return nil command")
		}
	})

	t.Run("select theme", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewThemeSelect
		m.themeCursor = 1 // Second theme (tokyonight)

		// Simulate Enter on a theme
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, cmd := m.Update(msg)

		if typedModel, ok := updatedModel.(ConfigModel); ok {
			// Should return to main view
			if typedModel.view != viewMain {
				t.Error("Should return to main view after theme selection")
			}

			// Should set feedback
			if typedModel.feedback == "" {
				t.Error("Should set feedback message")
			}

			// Should set markdown style
			if typedModel.config.Markdown.Style == "" {
				t.Error("Should set markdown style")
			}
		}

		// Should return clear feedback command
		if cmd == nil {
			t.Error("Should return clear feedback command")
		}
	})
}

func TestConfigModel_renderThemeSelect(t *testing.T) {
	m := NewConfigModel()

	menu := m.renderThemeSelect(80)

	if menu == "" {
		t.Error("renderThemeSelect should not return empty string")
	}

	// Should contain markdown theme title
	if !contains(menu, "Select Markdown Theme") {
		t.Error("Menu should contain 'Select Markdown Theme' title")
	}

	// Should contain at least dark theme
	if !contains(menu, "dark") {
		t.Error("Menu should contain 'dark' theme")
	}

	// Should contain tokyonight theme
	if !contains(menu, "tokyonight") {
		t.Error("Menu should contain 'tokyonight' theme")
	}
}

func TestConfigModel_renderMainMenu_Theme(t *testing.T) {
	m := NewConfigModel()

	menu := m.renderMainMenu(80)

	// Should contain Theme item
	if !contains(menu, "Theme") {
		t.Error("Menu should contain Theme item")
	}
}

func TestConfigModel_ThemeCursorInit(t *testing.T) {
	m := NewConfigModel()

	// Should have non-negative theme cursor
	if m.themeCursor < 0 {
		t.Error("themeCursor should be non-negative")
	}
}

func TestConfigModel_ConfigView_Enum(t *testing.T) {
	// Test that configView constants are properly defined
	if viewMain != 0 {
		t.Errorf("Expected viewMain to be 0, got %d", viewMain)
	}
	if viewModelSelect != 1 {
		t.Errorf("Expected viewModelSelect to be 1, got %d", viewModelSelect)
	}
}

func TestConfigModel_MenuConstants(t *testing.T) {
	// Test that menu constants are properly defined
	if menuDefaultModel != 0 {
		t.Errorf("Expected menuDefaultModel to be 0, got %d", menuDefaultModel)
	}
	if menuVerbose != 1 {
		t.Errorf("Expected menuVerbose to be 1, got %d", menuVerbose)
	}
	if menuAutoClose != 2 {
		t.Errorf("Expected menuAutoClose to be 2, got %d", menuAutoClose)
	}
	if menuCloseDelay != 3 {
		t.Errorf("Expected menuCloseDelay to be 3, got %d", menuCloseDelay)
	}
	if menuAutoReInit != 4 {
		t.Errorf("Expected menuAutoReInit to be 4, got %d", menuAutoReInit)
	}
	if menuCopyToClipboard != 5 {
		t.Errorf("Expected menuCopyToClipboard to be 5, got %d", menuCopyToClipboard)
	}
	if menuTheme != 6 {
		t.Errorf("Expected menuTheme to be 6, got %d", menuTheme)
	}
	if menuTUITheme != 7 {
		t.Errorf("Expected menuTUITheme to be 7, got %d", menuTUITheme)
	}
	if menuExit != 8 {
		t.Errorf("Expected menuExit to be 8, got %d", menuExit)
	}
	if menuItemCount != 9 {
		t.Errorf("Expected menuItemCount to be 9, got %d", menuItemCount)
	}
}

func TestConfigModel_cookiesExistDetection(t *testing.T) {
	// Create a temporary cookies file
	tmpDir := t.TempDir()
	tmpCookies := tmpDir + "/cookies.json"

	// Create the file
	f, err := os.Create(tmpCookies)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	_ = f.Close()

	// Note: Testing cookiesExist detection requires mocking file system operations
	// which is beyond the scope of this unit test. The functionality is tested
	// indirectly through NewConfigModel which reads the actual file system.
	// This test verifies that NewConfigModel can be called without panicking
	// when cookies file exists.
	_ = tmpCookies // Use variable to avoid unused error

	// The test passes if NewConfigModel doesn't panic
	m := NewConfigModel()
	if m.configDir == "" {
		t.Error("ConfigModel should have valid configDir")
	}
}

func TestConfigModel_feedbackClearMsg(t *testing.T) {
	// Test that feedbackClearMsg type is properly defined
	msg := feedbackClearMsg{}

	// The message should be instantiatable without panic
	// Zero value is valid - just verify type exists
	_ = msg
}

func TestRunConfig(t *testing.T) {
	// Just test that the function exists and doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunConfig panicked: %v", r)
		}
	}()

	// We can't actually run the tea program in a test
	// So we'll just test function signature
	_ = RunConfig
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// Simple substring search implementation
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════════════
// TUI THEME CONFIGURATION TESTS (Phase 3)
// ═══════════════════════════════════════════════════════════════════════════════

func TestConfigModel_TUIThemeCursorInit(t *testing.T) {
	m := NewConfigModel()

	// Should have non-negative TUI theme cursor
	if m.tuiThemeCursor < 0 {
		t.Error("tuiThemeCursor should be non-negative")
	}
}

func TestConfigModel_TUIThemeSelection(t *testing.T) {
	t.Run("escape from TUI theme select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewTUIThemeSelect

		newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		updatedM := newM.(ConfigModel)

		if updatedM.view != viewMain {
			t.Errorf("Expected view to be viewMain after Esc, got %d", updatedM.view)
		}
	})

	t.Run("navigate up in TUI theme select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewTUIThemeSelect
		m.tuiThemeCursor = 1

		newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		updatedM := newM.(ConfigModel)

		if updatedM.tuiThemeCursor != 0 {
			t.Errorf("Expected tuiThemeCursor to be 0, got %d", updatedM.tuiThemeCursor)
		}
	})

	t.Run("navigate down in TUI theme select view", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewTUIThemeSelect
		m.tuiThemeCursor = 0

		newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		updatedM := newM.(ConfigModel)

		if updatedM.tuiThemeCursor != 1 {
			t.Errorf("Expected tuiThemeCursor to be 1, got %d", updatedM.tuiThemeCursor)
		}
	})

	t.Run("enter on TUI theme menu item", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewMain
		m.cursor = menuTUITheme

		newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		updatedM := newM.(ConfigModel)

		if updatedM.view != viewTUIThemeSelect {
			t.Errorf("Expected view to be viewTUIThemeSelect, got %d", updatedM.view)
		}
	})

	t.Run("select TUI theme", func(t *testing.T) {
		m := NewConfigModel()
		m.view = viewTUIThemeSelect
		m.tuiThemeCursor = 1 // Select second theme (catppuccin)

		newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		updatedM := newM.(ConfigModel)

		if updatedM.view != viewMain {
			t.Errorf("Expected view to return to viewMain after selection")
		}

		if cmd == nil {
			t.Error("Should return clear feedback command")
		}
	})
}

func TestConfigModel_renderTUIThemeSelect(t *testing.T) {
	m := NewConfigModel()

	menu := m.renderTUIThemeSelect(80)

	if menu == "" {
		t.Error("renderTUIThemeSelect should not return empty string")
	}

	// Should contain TUI theme title
	if !contains(menu, "Select TUI Theme") {
		t.Error("Menu should contain 'Select TUI Theme' title")
	}

	// Should contain tokyonight theme
	if !contains(menu, "tokyonight") {
		t.Error("Menu should contain 'tokyonight' theme")
	}

	// Should contain catppuccin theme
	if !contains(menu, "catppuccin") {
		t.Error("Menu should contain 'catppuccin' theme")
	}

	// Should contain nord theme
	if !contains(menu, "nord") {
		t.Error("Menu should contain 'nord' theme")
	}
}

func TestConfigModel_renderMainMenu_TUITheme(t *testing.T) {
	m := NewConfigModel()

	menu := m.renderMainMenu(80)

	// Should contain TUI Theme item
	if !contains(menu, "TUI Theme") {
		t.Error("Menu should contain TUI Theme item")
	}
}

func TestConfigModel_viewTUIThemeSelect_Constant(t *testing.T) {
	// Test that viewTUIThemeSelect is properly defined
	if viewTUIThemeSelect != 3 {
		t.Errorf("Expected viewTUIThemeSelect to be 3, got %d", viewTUIThemeSelect)
	}
}
