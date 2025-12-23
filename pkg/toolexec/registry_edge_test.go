package toolexec

import (
	"fmt"
	"sync"
	"testing"
)

func TestRegistryConcurrency(t *testing.T) {
	r := NewRegistry()
	const numGoroutines = 100
	const numTools = 50
	var wg sync.WaitGroup

	// Concurrent registration
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numTools; j++ {
				name := fmt.Sprintf("tool-%d-%d", id, j)
				tool := NewMockTool(name, "desc")
				_ = r.Register(tool)
			}
		}(i)
	}
	wg.Wait()

	if r.Count() != numGoroutines*numTools {
		t.Errorf("Expected %d tools, got %d", numGoroutines*numTools, r.Count())
	}

	// Concurrent get and unregister
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numTools; j++ {
				name := fmt.Sprintf("tool-%d-%d", id, j)
				_, _ = r.Get(name)
				_ = r.Unregister(name)
			}
		}(i)
	}
	wg.Wait()

	if r.Count() != 0 {
		t.Errorf("Expected 0 tools after concurrent unregister, got %d", r.Count())
	}
}

func TestRegistryMixedConcurrency(t *testing.T) {
	r := NewRegistry()
	const numGoroutines = 50
	const numOps = 100
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				name := fmt.Sprintf("tool-%d", (id+j)%20) // Overlapping names
				tool := NewMockTool(name, "desc")

				// Mix of operations
				switch (id + j) % 4 {
				case 0:
					_ = r.Register(tool)
				case 1:
					_ = r.Unregister(name)
				case 2:
					_, _ = r.Get(name)
				case 3:
					_ = r.List()
				}
			}
		}(i)
	}
	wg.Wait()
	// No panic or race means success
}

func TestRegistryEdgeCases(t *testing.T) {
	t.Run("WithToolsMixed", func(t *testing.T) {
		t1 := NewMockTool("t1", "d1")
		t2 := NewMockTool("t2", "d2")
		// t3 has empty name, should be skipped
		t3 := NewMockTool("", "d3")

		r := NewRegistryWithOptions(WithTools(t1, t2, nil, t3, t1))

		if r.Count() != 2 {
			t.Errorf("Expected 2 tools, got %d", r.Count())
		}
		if !r.Has("t1") || !r.Has("t2") {
			t.Error("Missing expected tools t1 or t2")
		}
	})

	t.Run("RegisterEmptyName", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(NewMockTool("", "desc"))
		if err == nil {
			t.Error("Expected error when registering tool with empty name")
		}
	})

	t.Run("UnregisterNonExistent", func(t *testing.T) {
		r := NewRegistry()
		err := r.Unregister("non-existent")
		if err == nil {
			t.Error("Expected error when unregistering non-existent tool")
		}
		if !IsToolNotFoundError(err) {
			t.Errorf("Expected ToolNotFoundError, got %v", err)
		}
	})
}
