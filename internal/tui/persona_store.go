package tui

import (
	"fmt"
	"sync"

	"github.com/diogo/geminiweb/internal/config"
)

// PersonaStore defines the interface for persona CRUD operations.
// This abstraction enables testing with mock implementations.
type PersonaStore interface {
	// List returns all available personas
	List() ([]config.Persona, error)

	// Get retrieves a persona by name
	Get(name string) (*config.Persona, error)

	// Save creates or updates a persona
	Save(p config.Persona) error

	// Delete removes a persona by name
	Delete(name string) error

	// SetDefault sets the default persona
	SetDefault(name string) error

	// GetDefault returns the default persona
	GetDefault() (*config.Persona, error)
}

// personaStoreAdapter wraps the existing config functions to implement PersonaStore
type personaStoreAdapter struct{}

// NewPersonaStore creates a new PersonaStore backed by the config package
func NewPersonaStore() PersonaStore {
	return &personaStoreAdapter{}
}

// List returns all personas
func (s *personaStoreAdapter) List() ([]config.Persona, error) {
	cfg, err := config.LoadPersonas()
	if err != nil {
		return nil, err
	}
	return cfg.Personas, nil
}

// Get retrieves a persona by name
func (s *personaStoreAdapter) Get(name string) (*config.Persona, error) {
	return config.GetPersona(name)
}

// Save creates or updates a persona
func (s *personaStoreAdapter) Save(p config.Persona) error {
	// Check if it's an update or create
	existing, _ := config.GetPersona(p.Name)
	if existing != nil {
		return config.UpdatePersona(p)
	}
	return config.AddPersona(p)
}

// Delete removes a persona by name
func (s *personaStoreAdapter) Delete(name string) error {
	return config.DeletePersona(name)
}

// SetDefault sets the default persona
func (s *personaStoreAdapter) SetDefault(name string) error {
	return config.SetDefaultPersona(name)
}

// GetDefault returns the default persona
func (s *personaStoreAdapter) GetDefault() (*config.Persona, error) {
	return config.GetDefaultPersona()
}

// MockPersonaStore is an in-memory implementation for testing
type MockPersonaStore struct {
	mu             sync.RWMutex
	personas       map[string]*config.Persona
	defaultPersona string
	saveError      map[string]error // name -> error to return on Save
	deleteError    map[string]error // name -> error to return on Delete
}

// NewMockPersonaStore creates a new mock store with default personas
func NewMockPersonaStore() *MockPersonaStore {
	return &MockPersonaStore{
		personas:    make(map[string]*config.Persona),
		saveError:   make(map[string]error),
		deleteError: make(map[string]error),
	}
}

// NewMockPersonaStoreWithDefaults creates a mock store with default personas
func NewMockPersonaStoreWithDefaults() *MockPersonaStore {
	m := &MockPersonaStore{
		personas:    make(map[string]*config.Persona),
		deleteError: make(map[string]error),
		saveError:   make(map[string]error),
	}

	// Add default personas
	defaults := config.DefaultPersonas()
	for i := range defaults {
		p := defaults[i]
		m.personas[p.Name] = &p
	}
	m.defaultPersona = "default"

	return m
}

// List returns all personas
func (m *MockPersonaStore) List() ([]config.Persona, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]config.Persona, 0, len(m.personas))
	for _, p := range m.personas {
		result = append(result, *p)
	}
	return result, nil
}

// Get retrieves a persona by name
func (m *MockPersonaStore) Get(name string) (*config.Persona, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.personas[name]
	if !ok {
		return nil, fmt.Errorf("persona '%s' not found", name)
	}
	return p, nil
}

// Save creates or updates a persona
func (m *MockPersonaStore) Save(p config.Persona) error {
	// Check for forced error
	if err, ok := m.saveError[p.Name]; ok {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a copy and store
	persona := p
	m.personas[p.Name] = &persona
	return nil
}

// Delete removes a persona by name
func (m *MockPersonaStore) Delete(name string) error {
	// Check for forced error
	if err, ok := m.deleteError[name]; ok {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "default" {
		return fmt.Errorf("cannot delete the default persona")
	}

	if _, ok := m.personas[name]; !ok {
		return fmt.Errorf("persona '%s' not found", name)
	}

	delete(m.personas, name)

	// Reset default if deleted
	if m.defaultPersona == name {
		m.defaultPersona = "default"
	}

	return nil
}

// SetDefault sets the default persona
func (m *MockPersonaStore) SetDefault(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.personas[name]; !ok {
		return fmt.Errorf("persona '%s' not found", name)
	}

	m.defaultPersona = name
	return nil
}

// GetDefault returns the default persona
func (m *MockPersonaStore) GetDefault() (*config.Persona, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.defaultPersona == "" {
		m.defaultPersona = "default"
	}

	p, ok := m.personas[m.defaultPersona]
	if !ok {
		return nil, fmt.Errorf("default persona not found")
	}
	return p, nil
}

// SetSaveError sets an error to be returned when saving a specific persona
func (m *MockPersonaStore) SetSaveError(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveError[name] = err
}

// SetDeleteError sets an error to be returned when deleting a specific persona
func (m *MockPersonaStore) SetDeleteError(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteError[name] = err
}

// Clear clears all personas
func (m *MockPersonaStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.personas = make(map[string]*config.Persona)
	m.defaultPersona = ""
}
