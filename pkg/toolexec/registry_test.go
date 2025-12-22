package toolexec

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	t.Run("Register", func(t *testing.T) {
		tool := NewMockTool("test", "Mock Tool")
		
		// 1. Success
		if err := r.Register(tool); err != nil {
			t.Errorf("Register failed: %v", err)
		}

		// 2. Duplicate
		if err := r.Register(tool); err == nil {
			t.Error("Expected error for duplicate registration, got nil")
		} else if !IsDuplicateToolError(err) {
			// passed
		}

		// 3. Nil
		if err := r.Register(nil); err != ErrNilTool {
			t.Errorf("Expected ErrNilTool, got %v", err)
		}

		// 4. Empty Name
		if err := r.Register(NewMockTool("", "Empty")); err == nil {
			t.Error("Expected error for empty name")
		}
	})

	t.Run("Get", func(t *testing.T) {
		if _, err := r.Get("test"); err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if _, err := r.Get("missing"); err == nil {
			t.Error("Expected error for missing tool")
		} else if !IsToolNotFoundError(err) {
			// passed
		}
	})

	t.Run("Has", func(t *testing.T) {
		if !r.Has("test") {
			t.Error("Has returned false for existing tool")
		}
		if r.Has("missing") {
			t.Error("Has returned true for missing tool")
		}
	})

	t.Run("List", func(t *testing.T) {
		list := r.List()
		if len(list) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(list))
		}
		if list[0].Name != "test" {
			t.Errorf("Expected tool name 'test', got '%s'", list[0].Name)
		}
	})

	t.Run("Count", func(t *testing.T) {
		if r.Count() != 1 {
			t.Errorf("Expected count 1, got %d", r.Count())
		}
	})

	t.Run("Snapshot", func(t *testing.T) {
		snap := r.(SnapshotRegistry).Snapshot()
		if len(snap.Tools) != 1 {
			t.Errorf("Expected 1 tool in snapshot, got %d", len(snap.Tools))
		}
	})

	t.Run("Unregister", func(t *testing.T) {
		if err := r.Unregister("test"); err != nil {
			t.Errorf("Unregister failed: %v", err)
		}
		if r.Has("test") {
			t.Error("Tool still exists after unregister")
		}
		if err := r.Unregister("test"); err == nil {
			t.Error("Expected error unregistering missing tool")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		r.Register(NewMockTool("t1", "t1"))
		r.Clear()
		if r.Count() != 0 {
			t.Error("Clear failed not empty")
		}
	})
}

func TestDefaultRegistryWrappers(t *testing.T) {
	// Ensure clean state
	DefaultRegistry().Clear()

	tool := NewMockTool("global", "global")
	
	// Test MustRegister
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustRegister panicked unexpectedly: %v", r)
			}
		}()
		MustRegister(tool)
	}()

	// Test Has
	if !Has("global") {
		t.Error("Global Has failed")
	}

	// Test Get
	if _, err := Get("global"); err != nil {
		t.Error("Global Get failed")
	}

	// Test List
	if len(List()) != 1 {
		t.Error("Global List failed")
	}

	// Test Count
	if Count() != 1 {
		t.Error("Global Count failed")
	}

	// Test panic on duplicate
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustRegister did not panic on duplicate")
			}
		}()
		MustRegister(tool)
	}()
}

func TestNewRegistryWithOptions(t *testing.T) {
	tool := NewMockTool("opt", "opt")
	r := NewRegistryWithOptions(WithTools(tool))
	if !r.Has("opt") {
		t.Error("WithTools failed")
	}
}
