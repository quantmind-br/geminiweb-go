package toolexec

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// mockTool is a simple mock implementation of the Tool interface for testing.
type mockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, input *Input) (*Output, error)
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Execute(ctx context.Context, input *Input) (*Output, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return NewOutput().WithMessage("mock executed"), nil
}

func (m *mockTool) RequiresConfirmation(args map[string]any) bool {
	return false
}

// newMockTool creates a mock tool with the given name.
func newMockTool(name string) *mockTool {
	return &mockTool{
		name:        name,
		description: "Mock tool: " + name,
	}
}

// newMockToolWithDescription creates a mock tool with custom name and description.
func newMockToolWithDescription(name, description string) *mockTool {
	return &mockTool{
		name:        name,
		description: description,
	}
}

// TestNewRegistry tests that NewRegistry creates a valid empty registry.
func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if count := r.Count(); count != 0 {
		t.Errorf("NewRegistry() Count() = %d, want 0", count)
	}

	list := r.List()
	if len(list) != 0 {
		t.Errorf("NewRegistry() List() len = %d, want 0", len(list))
	}
}

// TestRegistryRegister tests the Register method.
func TestRegistryRegister(t *testing.T) {
	tests := []struct {
		name    string
		tool    Tool
		wantErr bool
		errType error
	}{
		{
			name:    "valid tool",
			tool:    newMockTool("test-tool"),
			wantErr: false,
		},
		{
			name:    "nil tool",
			tool:    nil,
			wantErr: true,
			errType: ErrNilTool,
		},
		{
			name:    "empty name tool",
			tool:    newMockTool(""),
			wantErr: true,
			errType: ErrValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			err := r.Register(tt.tool)

			if tt.wantErr {
				if err == nil {
					t.Error("Register() expected error but got none")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Register() error = %v, want error type %v", err, tt.errType)
				}
				return
			}

			if err != nil {
				t.Errorf("Register() unexpected error: %v", err)
				return
			}

			// Verify the tool was registered
			if !r.Has(tt.tool.Name()) {
				t.Errorf("Register() tool not found in registry after registration")
			}
		})
	}
}

// TestRegistryRegisterDuplicate tests that duplicate registrations fail.
func TestRegistryRegisterDuplicate(t *testing.T) {
	r := NewRegistry()

	tool1 := newMockTool("duplicate-tool")
	tool2 := newMockTool("duplicate-tool")

	// First registration should succeed
	if err := r.Register(tool1); err != nil {
		t.Fatalf("First Register() failed: %v", err)
	}

	// Second registration with same name should fail
	err := r.Register(tool2)
	if err == nil {
		t.Error("Register() duplicate should return error but got none")
		return
	}

	if !errors.Is(err, ErrDuplicateTool) {
		t.Errorf("Register() duplicate error = %v, want ErrDuplicateTool", err)
	}

	var dupErr *DuplicateToolError
	if !errors.As(err, &dupErr) {
		t.Error("Register() duplicate error should be DuplicateToolError")
		return
	}

	if dupErr.ToolName != "duplicate-tool" {
		t.Errorf("DuplicateToolError.ToolName = %q, want %q", dupErr.ToolName, "duplicate-tool")
	}
}

// TestRegistryGet tests the Get method.
func TestRegistryGet(t *testing.T) {
	r := NewRegistry()
	tool := newMockToolWithDescription("get-tool", "A tool for testing Get")

	if err := r.Register(tool); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	tests := []struct {
		name     string
		toolName string
		wantErr  bool
		errType  error
	}{
		{
			name:     "existing tool",
			toolName: "get-tool",
			wantErr:  false,
		},
		{
			name:     "non-existing tool",
			toolName: "non-existent",
			wantErr:  true,
			errType:  ErrToolNotFound,
		},
		{
			name:     "empty name",
			toolName: "",
			wantErr:  true,
			errType:  ErrToolNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Get(tt.toolName)

			if tt.wantErr {
				if err == nil {
					t.Error("Get() expected error but got none")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Get() error = %v, want error type %v", err, tt.errType)
				}
				if got != nil {
					t.Error("Get() returned non-nil tool on error")
				}
				return
			}

			if err != nil {
				t.Errorf("Get() unexpected error: %v", err)
				return
			}

			if got == nil {
				t.Error("Get() returned nil tool")
				return
			}

			if got.Name() != tt.toolName {
				t.Errorf("Get() tool name = %q, want %q", got.Name(), tt.toolName)
			}

			if got.Description() != "A tool for testing Get" {
				t.Errorf("Get() tool description = %q, want %q", got.Description(), "A tool for testing Get")
			}
		})
	}
}

// TestRegistryGetToolNotFoundError tests the ToolNotFoundError structure.
func TestRegistryGetToolNotFoundError(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("non-existent-tool")
	if err == nil {
		t.Fatal("Get() expected error but got none")
	}

	var notFoundErr *ToolNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("Get() error should be ToolNotFoundError, got %T", err)
	}

	if notFoundErr.ToolName != "non-existent-tool" {
		t.Errorf("ToolNotFoundError.ToolName = %q, want %q", notFoundErr.ToolName, "non-existent-tool")
	}

	// Test error message
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("ToolNotFoundError.Error() returned empty string")
	}
}

// TestRegistryList tests the List method.
func TestRegistryList(t *testing.T) {
	r := NewRegistry()

	// Empty registry
	list := r.List()
	if len(list) != 0 {
		t.Errorf("List() empty registry len = %d, want 0", len(list))
	}

	// Add tools
	tools := []Tool{
		newMockToolWithDescription("charlie", "Third tool"),
		newMockToolWithDescription("alpha", "First tool"),
		newMockToolWithDescription("bravo", "Second tool"),
	}

	for _, tool := range tools {
		if err := r.Register(tool); err != nil {
			t.Fatalf("Register() failed: %v", err)
		}
	}

	list = r.List()
	if len(list) != 3 {
		t.Fatalf("List() len = %d, want 3", len(list))
	}

	// Verify alphabetical ordering
	expectedOrder := []string{"alpha", "bravo", "charlie"}
	for i, info := range list {
		if info.Name != expectedOrder[i] {
			t.Errorf("List()[%d].Name = %q, want %q", i, info.Name, expectedOrder[i])
		}
	}
}

// TestRegistryHas tests the Has method.
func TestRegistryHas(t *testing.T) {
	r := NewRegistry()
	tool := newMockTool("has-tool")

	// Before registration
	if r.Has("has-tool") {
		t.Error("Has() returned true before registration")
	}

	if err := r.Register(tool); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// After registration
	if !r.Has("has-tool") {
		t.Error("Has() returned false after registration")
	}

	// Non-existent tool
	if r.Has("non-existent") {
		t.Error("Has() returned true for non-existent tool")
	}
}

// TestRegistryCount tests the Count method.
func TestRegistryCount(t *testing.T) {
	r := NewRegistry()

	// Empty registry
	if count := r.Count(); count != 0 {
		t.Errorf("Count() empty = %d, want 0", count)
	}

	// Add tools
	for i := 1; i <= 5; i++ {
		tool := newMockTool("tool-" + string(rune('0'+i)))
		if err := r.Register(tool); err != nil {
			t.Fatalf("Register() failed: %v", err)
		}

		if count := r.Count(); count != i {
			t.Errorf("Count() after %d registrations = %d, want %d", i, count, i)
		}
	}
}

// TestRegistryUnregister tests the Unregister method.
func TestRegistryUnregister(t *testing.T) {
	r := NewRegistry()

	tool := newMockTool("unregister-tool")
	if err := r.Register(tool); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Verify tool exists
	if !r.Has("unregister-tool") {
		t.Fatal("Tool not found after registration")
	}

	// Unregister
	err := r.Unregister("unregister-tool")
	if err != nil {
		t.Errorf("Unregister() unexpected error: %v", err)
	}

	// Verify tool is gone
	if r.Has("unregister-tool") {
		t.Error("Has() returned true after Unregister")
	}

	// Count should be 0
	if count := r.Count(); count != 0 {
		t.Errorf("Count() after Unregister = %d, want 0", count)
	}

	// Unregistering again should fail
	err = r.Unregister("unregister-tool")
	if err == nil {
		t.Error("Unregister() non-existent should return error")
	}
	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("Unregister() non-existent error = %v, want ErrToolNotFound", err)
	}
}

// TestRegistryClear tests the Clear method.
func TestRegistryClear(t *testing.T) {
	r := NewRegistry()

	// Add multiple tools
	for i := 0; i < 5; i++ {
		tool := newMockTool("clear-tool-" + string(rune('0'+i)))
		if err := r.Register(tool); err != nil {
			t.Fatalf("Register() failed: %v", err)
		}
	}

	if count := r.Count(); count != 5 {
		t.Fatalf("Count() before Clear = %d, want 5", count)
	}

	// Clear
	r.Clear()

	// Verify empty
	if count := r.Count(); count != 0 {
		t.Errorf("Count() after Clear = %d, want 0", count)
	}

	if list := r.List(); len(list) != 0 {
		t.Errorf("List() after Clear len = %d, want 0", len(list))
	}
}

// TestRegistryConcurrentAccess tests thread-safety of the registry.
func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	// Pre-register some tools
	for i := 0; i < 10; i++ {
		tool := newMockTool("initial-tool-" + string(rune('a'+i)))
		if err := r.Register(tool); err != nil {
			t.Fatalf("Register() failed: %v", err)
		}
	}

	var wg sync.WaitGroup
	numGoroutines := 100
	iterations := 50

	// Concurrent readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = r.Count()
				_ = r.List()
				_ = r.Has("initial-tool-a")
				_, _ = r.Get("initial-tool-a")
			}
		}()
	}

	// Concurrent writers (registering new tools)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				name := "concurrent-tool-" + string(rune('0'+id)) + "-" + string(rune('0'+j))
				tool := newMockTool(name)
				_ = r.Register(tool) // Ignore errors (duplicates are expected)
			}
		}(i)
	}

	wg.Wait()

	// Verify integrity
	if count := r.Count(); count < 10 {
		t.Errorf("Count() after concurrent access = %d, want >= 10", count)
	}
}

// TestDefaultRegistry tests the default global registry functions.
func TestDefaultRegistry(t *testing.T) {
	// Get the default registry
	dr := DefaultRegistry()
	if dr == nil {
		t.Fatal("DefaultRegistry() returned nil")
	}

	// Calling again should return the same registry
	dr2 := DefaultRegistry()
	if dr != dr2 {
		t.Error("DefaultRegistry() should return the same instance")
	}
}

// TestNewRegistryWithOptions tests creating a registry with options.
func TestNewRegistryWithOptions(t *testing.T) {
	tool1 := newMockTool("pre-tool-1")
	tool2 := newMockTool("pre-tool-2")

	r := NewRegistryWithOptions(
		WithTools(tool1, tool2),
	)

	if count := r.Count(); count != 2 {
		t.Errorf("NewRegistryWithOptions() Count() = %d, want 2", count)
	}

	if !r.Has("pre-tool-1") {
		t.Error("NewRegistryWithOptions() missing pre-tool-1")
	}

	if !r.Has("pre-tool-2") {
		t.Error("NewRegistryWithOptions() missing pre-tool-2")
	}
}

// TestWithToolsOption tests the WithTools registry option.
func TestWithToolsOption(t *testing.T) {
	t.Run("with valid tools", func(t *testing.T) {
		tools := []Tool{
			newMockTool("option-tool-1"),
			newMockTool("option-tool-2"),
		}

		r := NewRegistryWithOptions(WithTools(tools...))

		if count := r.Count(); count != 2 {
			t.Errorf("WithTools() Count() = %d, want 2", count)
		}
	})

	t.Run("with nil tool", func(t *testing.T) {
		// nil tools should be silently skipped
		tools := []Tool{
			newMockTool("valid-tool"),
			nil,
		}

		r := NewRegistryWithOptions(WithTools(tools...))

		if count := r.Count(); count != 1 {
			t.Errorf("WithTools() with nil tool Count() = %d, want 1", count)
		}
	})

	t.Run("with duplicate tools", func(t *testing.T) {
		// Duplicates should be silently skipped
		tools := []Tool{
			newMockTool("dup-tool"),
			newMockTool("dup-tool"),
		}

		r := NewRegistryWithOptions(WithTools(tools...))

		if count := r.Count(); count != 1 {
			t.Errorf("WithTools() with duplicate Count() = %d, want 1", count)
		}
	})

	t.Run("with empty name tool", func(t *testing.T) {
		// Empty name tools should be silently skipped
		tools := []Tool{
			newMockTool("valid-tool"),
			newMockTool(""),
		}

		r := NewRegistryWithOptions(WithTools(tools...))

		if count := r.Count(); count != 1 {
			t.Errorf("WithTools() with empty name Count() = %d, want 1", count)
		}
	})
}

// TestRegistrySnapshot tests the Snapshot method.
func TestRegistrySnapshot(t *testing.T) {
	r := NewRegistry().(*registry)

	// Add tools
	tools := []Tool{
		newMockTool("snap-z"),
		newMockTool("snap-a"),
		newMockTool("snap-m"),
	}

	for _, tool := range tools {
		if err := r.Register(tool); err != nil {
			t.Fatalf("Register() failed: %v", err)
		}
	}

	// Take snapshot
	snapshot := r.Snapshot()

	if snapshot == nil {
		t.Fatal("Snapshot() returned nil")
	}

	if len(snapshot.Tools) != 3 {
		t.Errorf("Snapshot().Tools len = %d, want 3", len(snapshot.Tools))
	}

	if len(snapshot.Infos) != 3 {
		t.Errorf("Snapshot().Infos len = %d, want 3", len(snapshot.Infos))
	}

	// Verify alphabetical ordering
	expectedOrder := []string{"snap-a", "snap-m", "snap-z"}
	for i, tool := range snapshot.Tools {
		if tool.Name() != expectedOrder[i] {
			t.Errorf("Snapshot().Tools[%d].Name() = %q, want %q", i, tool.Name(), expectedOrder[i])
		}
	}

	for i, info := range snapshot.Infos {
		if info.Name != expectedOrder[i] {
			t.Errorf("Snapshot().Infos[%d].Name = %q, want %q", i, info.Name, expectedOrder[i])
		}
	}
}

// TestSnapshotRegistryInterface tests that registry implements SnapshotRegistry.
func TestSnapshotRegistryInterface(t *testing.T) {
	r := NewRegistry()

	// Verify it implements SnapshotRegistry
	sr, ok := r.(SnapshotRegistry)
	if !ok {
		t.Fatal("NewRegistry() does not implement SnapshotRegistry")
	}

	// Should work without panic
	snapshot := sr.Snapshot()
	if snapshot == nil {
		t.Error("Snapshot() returned nil")
	}
}

// TestToolInfoFromTool tests the ToolInfoFromTool helper.
func TestToolInfoFromTool(t *testing.T) {
	tool := newMockToolWithDescription("info-tool", "Tool description")

	info := ToolInfoFromTool(tool)

	if info.Name != "info-tool" {
		t.Errorf("ToolInfoFromTool().Name = %q, want %q", info.Name, "info-tool")
	}

	if info.Description != "Tool description" {
		t.Errorf("ToolInfoFromTool().Description = %q, want %q", info.Description, "Tool description")
	}
}
