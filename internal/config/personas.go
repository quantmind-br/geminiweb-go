package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Persona represents a system prompt configuration
type Persona struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	SystemPrompt string  `json:"system_prompt"`
	Model        string  `json:"model,omitempty"`       // Preferred model (optional)
	Temperature  float64 `json:"temperature,omitempty"` // For future use
}

// PersonaConfig stores all personas
type PersonaConfig struct {
	Personas       []Persona `json:"personas"`
	DefaultPersona string    `json:"default_persona,omitempty"`
}

// DefaultPersonas returns pre-configured personas
func DefaultPersonas() []Persona {
	return []Persona{
		{
			Name:         "default",
			Description:  "No system prompt",
			SystemPrompt: "",
		},
		{
			Name:        "coder",
			Description: "Expert programmer assistant",
			SystemPrompt: "You are a software development agent with access to tools.\n\n" +
				"When you need to use a tool, respond with exactly a JSON tool block and nothing else:\n\n" +
				"```tool\n" +
				"{\n" +
				"  \"name\": \"bash\",\n" +
				"  \"args\": {\"command\": \"ls -la\"},\n" +
				"  \"reason\": \"List directory contents\"\n" +
				"}\n" +
				"```\n\n" +
				"Wait for the tool result before continuing. Execute one tool per message.\n\n" +
				"Available tools:\n" +
				"- bash: Executes shell commands (requires confirmation)\n" +
				"  Args: {\"command\": \"string\"}\n" +
				"- file_read: Reads file contents\n" +
				"  Args: {\"path\": \"string\", \"lines\": number (optional)}\n" +
				"- file_write: Writes file contents (requires confirmation)\n" +
				"  Args: {\"path\": \"string\", \"content\": \"string\"}\n" +
				"- search: Searches files\n" +
				"  Args: {\"pattern\": \"string\", \"path\": \"string (optional)\", \"type\": \"regex|literal\"}\n\n" +
				"Guidelines:\n" +
				"- Prefer safe and reversible operations\n" +
				"- If a tool fails, analyze the error before retrying\n" +
				"- Avoid destructive commands unless explicitly requested",
		},
		{
			Name:        "writer",
			Description: "Creative writing assistant",
			SystemPrompt: `You are a creative writing assistant. Your goal is to:
- Help with creative writing, storytelling, and content creation
- Provide suggestions that enhance narrative flow
- Maintain consistent tone and style
- Offer multiple alternatives when asked
- Be concise but evocative in descriptions`,
		},
		{
			Name:        "analyst",
			Description: "Data and business analyst",
			SystemPrompt: `You are a data and business analyst. You should:
- Analyze information methodically
- Present findings in structured formats
- Use data to support conclusions
- Consider multiple perspectives
- Highlight key insights and actionable recommendations`,
		},
		{
			Name:        "teacher",
			Description: "Patient educational assistant",
			SystemPrompt: `You are a patient and thorough teacher. When explaining:
- Break down complex topics into simple parts
- Use analogies and examples
- Check understanding progressively
- Encourage questions
- Adapt explanations to the learner's level`,
		},
	}
}

// GetPersonasPath returns the path to the personas file
func GetPersonasPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "personas.json"), nil
}

// LoadPersonas loads the persona configuration
func LoadPersonas() (*PersonaConfig, error) {
	path, err := GetPersonasPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if file doesn't exist
			return &PersonaConfig{
				Personas:       DefaultPersonas(),
				DefaultPersona: "default",
			}, nil
		}
		return nil, fmt.Errorf("failed to read personas: %w", err)
	}

	var config PersonaConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse personas: %w", err)
	}

	// Merge with defaults (keep user customizations)
	config.Personas = mergePersonas(DefaultPersonas(), config.Personas)

	return &config, nil
}

// SavePersonas saves the persona configuration
func SavePersonas(config *PersonaConfig) error {
	path, err := GetPersonasPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal personas: %w", err)
	}

	// Use 0o600 for user data (personas may contain custom system prompts)
	return os.WriteFile(path, data, 0o600)
}

// GetPersona returns a persona by name
func GetPersona(name string) (*Persona, error) {
	config, err := LoadPersonas()
	if err != nil {
		return nil, err
	}

	for _, p := range config.Personas {
		if p.Name == name {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("persona '%s' not found", name)
}

// ListPersonaNames returns the names of all personas
func ListPersonaNames() ([]string, error) {
	config, err := LoadPersonas()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(config.Personas))
	for i, p := range config.Personas {
		names[i] = p.Name
	}
	return names, nil
}

// AddPersona adds a new persona
func AddPersona(persona Persona) error {
	config, err := LoadPersonas()
	if err != nil {
		return err
	}

	// Check if exists
	for _, p := range config.Personas {
		if p.Name == persona.Name {
			return fmt.Errorf("persona '%s' already exists", persona.Name)
		}
	}

	config.Personas = append(config.Personas, persona)
	return SavePersonas(config)
}

// UpdatePersona updates an existing persona
func UpdatePersona(persona Persona) error {
	config, err := LoadPersonas()
	if err != nil {
		return err
	}

	found := false
	for i, p := range config.Personas {
		if p.Name == persona.Name {
			config.Personas[i] = persona
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("persona '%s' not found", persona.Name)
	}

	return SavePersonas(config)
}

// DeletePersona removes a persona by name
func DeletePersona(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete the default persona")
	}

	config, err := LoadPersonas()
	if err != nil {
		return err
	}

	newPersonas := make([]Persona, 0, len(config.Personas))
	found := false
	for _, p := range config.Personas {
		if p.Name == name {
			found = true
			continue
		}
		newPersonas = append(newPersonas, p)
	}

	if !found {
		return fmt.Errorf("persona '%s' not found", name)
	}

	config.Personas = newPersonas

	// Reset default if deleted
	if config.DefaultPersona == name {
		config.DefaultPersona = "default"
	}

	return SavePersonas(config)
}

// SetDefaultPersona sets the default persona
func SetDefaultPersona(name string) error {
	// Verify persona exists
	_, err := GetPersona(name)
	if err != nil {
		return err
	}

	config, err := LoadPersonas()
	if err != nil {
		return err
	}

	config.DefaultPersona = name
	return SavePersonas(config)
}

// GetDefaultPersona returns the default persona
func GetDefaultPersona() (*Persona, error) {
	config, err := LoadPersonas()
	if err != nil {
		return nil, err
	}

	name := config.DefaultPersona
	if name == "" {
		name = "default"
	}

	return GetPersona(name)
}

func mergePersonas(defaults, custom []Persona) []Persona {
	result := make([]Persona, len(defaults))
	copy(result, defaults)

	// Add or replace with custom
	for _, cp := range custom {
		found := false
		for i, dp := range result {
			if dp.Name == cp.Name {
				result[i] = cp
				found = true
				break
			}
		}
		if !found {
			result = append(result, cp)
		}
	}

	return result
}

// FormatSystemPrompt formats the system prompt for inclusion in a message
func FormatSystemPrompt(persona *Persona, userMessage string) string {
	if persona == nil || persona.SystemPrompt == "" {
		return userMessage
	}

	return fmt.Sprintf(`[System Instructions]
%s

[User Message]
%s`, persona.SystemPrompt, userMessage)
}

// Validation constants
const (
	MaxNameLength        = 50
	MaxDescriptionLength = 200
	MaxPromptLength      = 32 * 1024 // 32KB
	MinNameLength        = 1
)

// ValidatePersona validates a persona's fields
func ValidatePersona(p Persona) error {
	fieldErrors := make(map[string]string)

	// Validate name
	if p.Name == "" {
		fieldErrors["name"] = "name is required"
	} else if len(p.Name) > MaxNameLength {
		fieldErrors["name"] = fmt.Sprintf("name too long (max %d characters)", MaxNameLength)
	} else if !isValidPersonaName(p.Name) {
		fieldErrors["name"] = "name must contain only alphanumeric characters, underscores, and hyphens"
	}

	// Validate description (optional but has max length)
	if len(p.Description) > MaxDescriptionLength {
		fieldErrors["description"] = fmt.Sprintf("description too long (max %d characters)", MaxDescriptionLength)
	}

	// Validate system prompt
	if len(p.SystemPrompt) > MaxPromptLength {
		fieldErrors["system_prompt"] = fmt.Sprintf("system prompt too long (max %d characters)", MaxPromptLength)
	}

	if len(fieldErrors) > 0 {
		return fmt.Errorf("validation failed: %v", fieldErrors)
	}

	return nil
}

// isValidPersonaName checks if a persona name contains only valid characters
func isValidPersonaName(name string) bool {
	if name == "default" {
		return true
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
