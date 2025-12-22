package tui

import "github.com/diogo/geminiweb/internal/config"

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
