package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/diogo/geminiweb/internal/models"
)

func TestNewImageSelectorModel(t *testing.T) {
	images := []models.WebImage{
		{URL: "https://example.com/1.jpg", Title: "Image 1"},
		{URL: "https://example.com/2.jpg", Title: "Image 2"},
	}
	targetDir := "/tmp/images"

	m := NewImageSelectorModel(images, targetDir)

	if len(m.images) != 2 {
		t.Errorf("expected 2 images, got %d", len(m.images))
	}
	if m.targetDir != targetDir {
		t.Errorf("expected targetDir %s, got %s", targetDir, m.targetDir)
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	if len(m.selected) != 0 {
		t.Error("expected empty selection map")
	}
	if m.confirmed {
		t.Error("should not be confirmed initially")
	}
	if m.cancelled {
		t.Error("should not be cancelled initially")
	}
}

func TestImageSelectorModel_Init(t *testing.T) {
	m := ImageSelectorModel{}
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestImageSelectorModel_Update_WindowSize(t *testing.T) {
	m := ImageSelectorModel{}
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}

	updated, cmd := m.Update(msg)

	if updated.width != 100 {
		t.Errorf("expected width 100, got %d", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("expected height 40, got %d", updated.height)
	}
	if !updated.ready {
		t.Error("model should be ready after WindowSizeMsg")
	}
	if cmd != nil {
		t.Error("Update should return nil command")
	}
}

func TestImageSelectorModel_Update_CancelKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"ctrl+c", "ctrl+c"},
		{"esc", "esc"},
		{"q", "q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ImageSelectorModel{
				images: []models.WebImage{{URL: "test.jpg"}},
			}

			var msg tea.KeyMsg
			switch tt.key {
			case "ctrl+c":
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			case "esc":
				msg = tea.KeyMsg{Type: tea.KeyEscape}
			case "q":
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
			}

			updated, _ := m.Update(msg)

			if !updated.cancelled {
				t.Error("should be cancelled")
			}
			if !updated.confirmed {
				t.Error("should be confirmed when cancelled")
			}
		})
	}
}

func TestImageSelectorModel_Update_Navigation(t *testing.T) {
	images := []models.WebImage{
		{URL: "1.jpg"},
		{URL: "2.jpg"},
		{URL: "3.jpg"},
	}

	t.Run("down moves cursor forward", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

		updated, _ := m.Update(msg)
		if updated.cursor != 1 {
			t.Errorf("expected cursor 1, got %d", updated.cursor)
		}
	})

	t.Run("down wraps around", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		m.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

		updated, _ := m.Update(msg)
		if updated.cursor != 0 {
			t.Errorf("expected cursor 0 (wrapped), got %d", updated.cursor)
		}
	})

	t.Run("up moves cursor backward", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		m.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}

		updated, _ := m.Update(msg)
		if updated.cursor != 1 {
			t.Errorf("expected cursor 1, got %d", updated.cursor)
		}
	})

	t.Run("up wraps around", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		m.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}

		updated, _ := m.Update(msg)
		if updated.cursor != 2 {
			t.Errorf("expected cursor 2 (wrapped), got %d", updated.cursor)
		}
	})

	t.Run("arrow keys work", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updated, _ := m.Update(msg)
		if updated.cursor != 1 {
			t.Errorf("expected cursor 1 with KeyDown, got %d", updated.cursor)
		}

		m = NewImageSelectorModel(images, "")
		m.cursor = 1
		msg = tea.KeyMsg{Type: tea.KeyUp}
		updated, _ = m.Update(msg)
		if updated.cursor != 0 {
			t.Errorf("expected cursor 0 with KeyUp, got %d", updated.cursor)
		}
	})
}

func TestImageSelectorModel_Update_Selection(t *testing.T) {
	images := []models.WebImage{
		{URL: "1.jpg"},
		{URL: "2.jpg"},
	}

	t.Run("space toggles selection", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}

		updated, _ := m.Update(msg)
		if !updated.selected[0] {
			t.Error("cursor 0 should be selected after space")
		}

		// Toggle again
		updated, _ = updated.Update(msg)
		if updated.selected[0] {
			t.Error("cursor 0 should be deselected after second space")
		}
	})

	t.Run("select all", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

		updated, _ := m.Update(msg)
		if len(updated.selected) != 2 {
			t.Errorf("expected 2 selected, got %d", len(updated.selected))
		}
		if !updated.selected[0] || !updated.selected[1] {
			t.Error("both images should be selected")
		}
	})

	t.Run("select none", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		m.selected[0] = true
		m.selected[1] = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}

		updated, _ := m.Update(msg)
		if len(updated.selected) != 0 {
			t.Errorf("expected 0 selected, got %d", len(updated.selected))
		}
	})

	t.Run("home goes to first", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		m.cursor = 1
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}

		updated, _ := m.Update(msg)
		if updated.cursor != 0 {
			t.Errorf("expected cursor 0, got %d", updated.cursor)
		}
	})

	t.Run("end goes to last", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}

		updated, _ := m.Update(msg)
		if updated.cursor != 1 {
			t.Errorf("expected cursor 1, got %d", updated.cursor)
		}
	})
}

func TestImageSelectorModel_Update_Confirm(t *testing.T) {
	images := []models.WebImage{{URL: "1.jpg"}}
	m := NewImageSelectorModel(images, "")
	msg := tea.KeyMsg{Type: tea.KeyEnter}

	updated, _ := m.Update(msg)

	if !updated.confirmed {
		t.Error("should be confirmed after Enter")
	}
	if updated.cancelled {
		t.Error("should not be cancelled after Enter")
	}
}

func TestImageSelectorModel_View(t *testing.T) {
	images := []models.WebImage{
		{URL: "https://example.com/1.jpg", Title: "Image 1"},
		{URL: "https://example.com/2.jpg", Title: "Image 2"},
	}

	t.Run("shows initializing when not ready", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		view := m.View()
		if !strings.Contains(view, "Initializing") {
			t.Error("should show initializing message")
		}
	})

	t.Run("renders correctly when ready", func(t *testing.T) {
		m := NewImageSelectorModel(images, "/tmp")
		m.ready = true
		m.width = 80
		m.height = 40

		view := m.View()

		if !strings.Contains(view, "Select images to download") {
			t.Error("should contain header")
		}
		if !strings.Contains(view, "Image 1") {
			t.Error("should contain first image title")
		}
		if !strings.Contains(view, "Image 2") {
			t.Error("should contain second image title")
		}
		if !strings.Contains(view, "Space: toggle") {
			t.Error("should contain help text")
		}
	})

	t.Run("shows selection count", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 80
		m.height = 40
		m.selected[0] = true

		view := m.View()
		if !strings.Contains(view, "1 of 2 selected") {
			t.Error("should show selection count")
		}
	})

	t.Run("shows cursor indicator", func(t *testing.T) {
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 80
		m.height = 40
		m.cursor = 0

		view := m.View()
		// Cursor should be visible (either > or highlighted)
		if !strings.Contains(view, ">") && !strings.Contains(view, "Image 1") {
			t.Error("should show cursor indicator")
		}
	})

	t.Run("uses alt text when title is empty", func(t *testing.T) {
		images := []models.WebImage{
			{URL: "1.jpg", Alt: "Alt text"},
		}
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 80
		m.height = 40

		view := m.View()
		if !strings.Contains(view, "Alt text") {
			t.Error("should use alt text when title is empty")
		}
	})

	t.Run("uses fallback when title and alt are empty", func(t *testing.T) {
		images := []models.WebImage{
			{URL: "1.jpg"},
		}
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 80
		m.height = 40

		view := m.View()
		if !strings.Contains(view, "Image 1") {
			t.Error("should use fallback 'Image 1'")
		}
	})

	t.Run("truncates long titles", func(t *testing.T) {
		images := []models.WebImage{
			{URL: "1.jpg", Title: strings.Repeat("A", 100)},
		}
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 50
		m.height = 40

		view := m.View()
		if !strings.Contains(view, "...") {
			t.Error("should truncate long titles with ellipsis")
		}
	})

	t.Run("truncates long URLs", func(t *testing.T) {
		images := []models.WebImage{
			{URL: "https://example.com/" + strings.Repeat("a", 100)},
		}
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 50
		m.height = 40

		view := m.View()
		if !strings.Contains(view, "...") {
			t.Error("should truncate long URLs with ellipsis")
		}
	})

	t.Run("shows selected checkbox", func(t *testing.T) {
		images := []models.WebImage{{URL: "1.jpg"}}
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 80
		m.height = 40
		m.selected[0] = true

		view := m.View()
		if !strings.Contains(view, "[x]") {
			t.Error("should show [x] for selected items")
		}
	})

	t.Run("shows unselected checkbox", func(t *testing.T) {
		images := []models.WebImage{{URL: "1.jpg"}}
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 80
		m.height = 40

		view := m.View()
		if !strings.Contains(view, "[ ]") {
			t.Error("should show [ ] for unselected items")
		}
	})

	t.Run("handles small height", func(t *testing.T) {
		images := []models.WebImage{{URL: "1.jpg"}, {URL: "2.jpg"}, {URL: "3.jpg"}}
		m := NewImageSelectorModel(images, "")
		m.ready = true
		m.width = 80
		m.height = 5 // Very small height

		view := m.View()
		if view == "" {
			t.Error("should still render with small height")
		}
	})
}

func TestImageSelectorModel_SelectedCount(t *testing.T) {
	images := []models.WebImage{{URL: "1.jpg"}, {URL: "2.jpg"}, {URL: "3.jpg"}}
	m := NewImageSelectorModel(images, "")

	if m.SelectedCount() != 0 {
		t.Errorf("expected 0, got %d", m.SelectedCount())
	}

	m.selected[0] = true
	m.selected[2] = true

	if m.SelectedCount() != 2 {
		t.Errorf("expected 2, got %d", m.SelectedCount())
	}
}

func TestImageSelectorModel_SelectedIndices(t *testing.T) {
	images := []models.WebImage{{URL: "1.jpg"}, {URL: "2.jpg"}, {URL: "3.jpg"}}
	m := NewImageSelectorModel(images, "")

	indices := m.SelectedIndices()
	if len(indices) != 0 {
		t.Errorf("expected empty, got %v", indices)
	}

	m.selected[0] = true
	m.selected[2] = true

	indices = m.SelectedIndices()
	if len(indices) != 2 {
		t.Errorf("expected 2 indices, got %d", len(indices))
	}
	if indices[0] != 0 || indices[1] != 2 {
		t.Errorf("expected [0, 2], got %v", indices)
	}
}

func TestImageSelectorModel_IsConfirmed(t *testing.T) {
	t.Run("returns true when confirmed and not cancelled", func(t *testing.T) {
		m := ImageSelectorModel{confirmed: true, cancelled: false}
		if !m.IsConfirmed() {
			t.Error("should be confirmed")
		}
	})

	t.Run("returns false when not confirmed", func(t *testing.T) {
		m := ImageSelectorModel{confirmed: false, cancelled: false}
		if m.IsConfirmed() {
			t.Error("should not be confirmed")
		}
	})

	t.Run("returns false when cancelled", func(t *testing.T) {
		m := ImageSelectorModel{confirmed: true, cancelled: true}
		if m.IsConfirmed() {
			t.Error("should not be confirmed when cancelled")
		}
	})
}

func TestImageSelectorModel_IsCancelled(t *testing.T) {
	t.Run("returns true when cancelled", func(t *testing.T) {
		m := ImageSelectorModel{cancelled: true}
		if !m.IsCancelled() {
			t.Error("should be cancelled")
		}
	})

	t.Run("returns false when not cancelled", func(t *testing.T) {
		m := ImageSelectorModel{cancelled: false}
		if m.IsCancelled() {
			t.Error("should not be cancelled")
		}
	})
}

func TestImageSelectorModel_TargetDir(t *testing.T) {
	expectedDir := "/tmp/images"
	m := ImageSelectorModel{targetDir: expectedDir}

	if m.TargetDir() != expectedDir {
		t.Errorf("expected %s, got %s", expectedDir, m.TargetDir())
	}
}

func TestImageSelectorModel_Scrolling(t *testing.T) {
	// Create many images to test scrolling
	images := make([]models.WebImage, 20)
	for i := 0; i < 20; i++ {
		images[i] = models.WebImage{URL: fmt.Sprintf("%d.jpg", i), Title: fmt.Sprintf("Image %d", i)}
	}

	m := NewImageSelectorModel(images, "")
	m.ready = true
	m.width = 80
	m.height = 10 // Small height to force scrolling

	// Move cursor down several times
	for i := 0; i < 15; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	// Should scroll to show cursor
	view := m.View()
	if !strings.Contains(view, "Image 15") {
		t.Error("should show image 15 after scrolling")
	}

	// Move to end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	view = m.View()
	if !strings.Contains(view, "Image 19") {
		t.Error("should show last image")
	}

	// Move to beginning
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	view = m.View()
	if !strings.Contains(view, "Image 0") {
		t.Error("should show first image")
	}
}

func TestImageSelectorModel_EmptyImages(t *testing.T) {
	m := NewImageSelectorModel([]models.WebImage{}, "")
	m.ready = true
	m.width = 80
	m.height = 40

	view := m.View()
	if !strings.Contains(view, "0 of 0 selected") {
		t.Error("should handle empty image list")
	}

	// Test navigation with empty list
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 0 {
		t.Error("cursor should stay at 0 with empty list")
	}
}

func TestImageSelectorModel_Update_OtherKeys(t *testing.T) {
	images := []models.WebImage{{URL: "1.jpg"}}
	m := NewImageSelectorModel(images, "")

	// Test keys that should do nothing
	keys := []tea.KeyType{
		tea.KeyTab,
		tea.KeyBackspace,
		tea.KeyInsert,
		tea.KeyDelete,
	}

	for _, keyType := range keys {
		msg := tea.KeyMsg{Type: keyType}
		updated, _ := m.Update(msg)
		if updated.cursor != m.cursor || updated.confirmed || updated.cancelled {
			t.Errorf("key %v should not change state", keyType)
		}
	}
}
