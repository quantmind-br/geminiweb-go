package render

import (
	"sync"
	"testing"
)

func TestCacheKey(t *testing.T) {
	opts1 := DefaultOptions()
	opts2 := DefaultOptions().WithWidth(100)
	opts3 := DefaultOptions().WithStyle("light")

	key1 := cacheKey(opts1)
	key2 := cacheKey(opts2)
	key3 := cacheKey(opts3)

	if key1 == key2 {
		t.Error("Different widths should produce different keys")
	}
	if key1 == key3 {
		t.Error("Different styles should produce different keys")
	}

	// Same options should produce same key
	opts4 := DefaultOptions()
	if cacheKey(opts1) != cacheKey(opts4) {
		t.Error("Same options should produce same key")
	}
}

func TestPoolGetAndPut(t *testing.T) {
	ClearCache()
	defer ClearCache()

	opts := DefaultOptions()

	// First call should create pool and renderer
	renderer1, err := globalPool.get(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renderer1 == nil {
		t.Fatal("expected non-nil renderer")
	}

	if CacheSize() != 1 {
		t.Errorf("expected pool count 1, got %d", CacheSize())
	}

	// Return to pool
	globalPool.put(opts, renderer1)

	// Get again - should reuse from pool
	renderer2, err := globalPool.get(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renderer2 == nil {
		t.Fatal("expected non-nil renderer")
	}

	// Different options should create new pool
	opts2 := DefaultOptions().WithWidth(100)
	renderer3, err := globalPool.get(opts2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renderer3 == nil {
		t.Fatal("expected non-nil renderer")
	}

	if CacheSize() != 2 {
		t.Errorf("expected pool count 2, got %d", CacheSize())
	}

	globalPool.put(opts, renderer2)
	globalPool.put(opts2, renderer3)
}

func TestPoolConcurrency(t *testing.T) {
	ClearCache()
	defer ClearCache()

	opts := DefaultOptions()
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Launch 100 concurrent goroutines that get and put renderers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			renderer, err := globalPool.get(opts)
			if err != nil {
				errors <- err
				return
			}
			if renderer == nil {
				errors <- nil
				return
			}
			// Simulate some work
			_, err = renderer.Render("# Test")
			if err != nil {
				errors <- err
				return
			}
			globalPool.put(opts, renderer)
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("concurrent access error: %v", err)
		}
	}

	// Should still only have one pool entry
	if CacheSize() != 1 {
		t.Errorf("expected pool count 1 after concurrent access, got %d", CacheSize())
	}
}

func TestClearCache(t *testing.T) {
	ClearCache()

	opts := DefaultOptions()
	renderer, err := globalPool.get(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	globalPool.put(opts, renderer)

	if CacheSize() != 1 {
		t.Errorf("expected pool count 1, got %d", CacheSize())
	}

	ClearCache()

	if CacheSize() != 0 {
		t.Errorf("expected pool count 0 after clear, got %d", CacheSize())
	}
}

func TestCreateRenderer(t *testing.T) {
	opts := DefaultOptions()
	renderer, err := createRenderer(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renderer == nil {
		t.Fatal("expected non-nil renderer")
	}

	// Test rendering works
	output, err := renderer.Render("# Test")
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestCreateRendererWithInvalidStyle(t *testing.T) {
	opts := DefaultOptions().WithStyle("invalid_style_path")
	_, err := createRenderer(opts)
	if err == nil {
		t.Error("expected error for invalid style")
	}
}
