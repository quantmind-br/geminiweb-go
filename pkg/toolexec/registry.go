// Package toolexec provides a modular, extensible tool executor architecture.
// This file implements the Registry for tool registration, discovery, and retrieval.
// The registry is thread-safe and supports compile-time registration via init() functions.
package toolexec

import (
	"fmt"
	"sort"
	"sync"
)

// Registry defines the interface for tool registration and discovery.
// Implementations must be thread-safe for concurrent access.
type Registry interface {
	// Register adds a tool to the registry.
	// Returns ErrNilTool if tool is nil.
	// Returns ErrDuplicateTool if a tool with the same name is already registered.
	Register(tool Tool) error

	// Get retrieves a tool by name.
	// Returns ErrToolNotFound if no tool with that name is registered.
	Get(name string) (Tool, error)

	// List returns information about all registered tools.
	// The returned slice is sorted alphabetically by tool name.
	List() []ToolInfo

	// Has returns true if a tool with the given name is registered.
	Has(name string) bool

	// Count returns the number of registered tools.
	Count() int

	// Unregister removes a tool from the registry.
	// Returns ErrToolNotFound if no tool with that name is registered.
	Unregister(name string) error

	// Clear removes all tools from the registry.
	Clear()
}

// registry is the default thread-safe implementation of Registry.
// It uses a sync.RWMutex to allow concurrent reads with exclusive writes.
type registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new empty registry.
// The returned registry is thread-safe and ready for use.
func NewRegistry() Registry {
	return &registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
// Returns ErrNilTool if tool is nil.
// Returns ErrDuplicateTool if a tool with the same name is already registered.
// This method is thread-safe.
func (r *registry) Register(tool Tool) error {
	if tool == nil {
		return ErrNilTool
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("cannot register tool with empty name: %w", ErrValidationFailed)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		return NewDuplicateToolError(name)
	}

	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name.
// Returns ErrToolNotFound if no tool with that name is registered.
// This method is thread-safe for concurrent reads.
func (r *registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, NewToolNotFoundError(name)
	}

	return tool, nil
}

// List returns information about all registered tools.
// The returned slice is sorted alphabetically by tool name.
// This method is thread-safe for concurrent reads.
func (r *registry) List() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ToolInfo, 0, len(r.tools))
	for _, tool := range r.tools {
		infos = append(infos, ToolInfoFromTool(tool))
	}

	// Sort alphabetically by name for consistent ordering
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	return infos
}

// Has returns true if a tool with the given name is registered.
// This method is thread-safe for concurrent reads.
func (r *registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[name]
	return exists
}

// Count returns the number of registered tools.
// This method is thread-safe for concurrent reads.
func (r *registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// Unregister removes a tool from the registry.
// Returns ErrToolNotFound if no tool with that name is registered.
// This method is thread-safe.
func (r *registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return NewToolNotFoundError(name)
	}

	delete(r.tools, name)
	return nil
}

// Clear removes all tools from the registry.
// This method is thread-safe.
func (r *registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)
}

// defaultRegistry is the package-level global registry.
// It is initialized lazily on first access for safe use in init() functions.
var (
	defaultRegistry     Registry
	defaultRegistryOnce sync.Once
)

// getDefaultRegistry returns the default registry, initializing it if needed.
// This uses sync.Once to ensure thread-safe lazy initialization.
func getDefaultRegistry() Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// DefaultRegistry returns the package-level default registry.
// This registry is used by the package-level Register() function
// and can be used for tool discovery across the application.
func DefaultRegistry() Registry {
	return getDefaultRegistry()
}

// Register adds a tool to the default global registry.
// This is a convenience function that panics if registration fails,
// making it suitable for use in init() functions.
//
// Usage in tool implementations:
//
//	func init() {
//	    toolexec.Register(&MyTool{})
//	}
//
// Panics if tool is nil or if a tool with the same name is already registered.
func Register(tool Tool) {
	if err := getDefaultRegistry().Register(tool); err != nil {
		panic(fmt.Sprintf("toolexec.Register: failed to register tool: %v", err))
	}
}

// MustRegister is an alias for Register that emphasizes the panic behavior.
// It adds a tool to the default global registry, panicking on error.
func MustRegister(tool Tool) {
	Register(tool)
}

// Get retrieves a tool from the default global registry by name.
// Returns ErrToolNotFound if no tool with that name is registered.
func Get(name string) (Tool, error) {
	return getDefaultRegistry().Get(name)
}

// Has returns true if a tool with the given name is registered
// in the default global registry.
func Has(name string) bool {
	return getDefaultRegistry().Has(name)
}

// List returns information about all tools in the default global registry.
// The returned slice is sorted alphabetically by tool name.
func List() []ToolInfo {
	return getDefaultRegistry().List()
}

// Count returns the number of tools in the default global registry.
func Count() int {
	return getDefaultRegistry().Count()
}

// RegistryOption is a function that configures a registry.
// This allows for flexible registry configuration using the functional options pattern.
type RegistryOption func(*registry)

// NewRegistryWithOptions creates a new registry with the given options.
func NewRegistryWithOptions(opts ...RegistryOption) Registry {
	r := &registry{
		tools: make(map[string]Tool),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithTools pre-populates the registry with the given tools.
// Any nil tools or duplicates are silently skipped.
func WithTools(tools ...Tool) RegistryOption {
	return func(r *registry) {
		for _, tool := range tools {
			if tool != nil && tool.Name() != "" {
				if _, exists := r.tools[tool.Name()]; !exists {
					r.tools[tool.Name()] = tool
				}
			}
		}
	}
}

// RegistrySnapshot represents a point-in-time snapshot of registry contents.
// This is useful for safely iterating over tools without holding locks.
type RegistrySnapshot struct {
	// Tools is a slice of all registered tools at the time of the snapshot.
	Tools []Tool
	// Infos is a slice of ToolInfo for all registered tools.
	Infos []ToolInfo
}

// Snapshot creates a point-in-time snapshot of the registry contents.
// The snapshot contains copies of tool references (not deep copies of tools).
// This is useful for safely iterating over tools without holding locks.
func (r *registry) Snapshot() *RegistrySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := &RegistrySnapshot{
		Tools: make([]Tool, 0, len(r.tools)),
		Infos: make([]ToolInfo, 0, len(r.tools)),
	}

	for _, tool := range r.tools {
		snapshot.Tools = append(snapshot.Tools, tool)
		snapshot.Infos = append(snapshot.Infos, ToolInfoFromTool(tool))
	}

	// Sort for consistent ordering
	sort.Slice(snapshot.Tools, func(i, j int) bool {
		return snapshot.Tools[i].Name() < snapshot.Tools[j].Name()
	})
	sort.Slice(snapshot.Infos, func(i, j int) bool {
		return snapshot.Infos[i].Name < snapshot.Infos[j].Name
	})

	return snapshot
}

// SnapshotRegistry is an optional interface that registries can implement
// to support efficient point-in-time snapshots.
type SnapshotRegistry interface {
	Registry
	Snapshot() *RegistrySnapshot
}

// Ensure registry implements all interfaces
var (
	_ Registry         = (*registry)(nil)
	_ SnapshotRegistry = (*registry)(nil)
)
