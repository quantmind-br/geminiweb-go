# Persona Management System

## Overview

The persona management system allows users to create, store, and use custom GPT-like personas for Gemini interactions. Personas are stored in `~/.geminiweb/personas.json` and provide a way to customize the AI's behavior and expertise.

## Persona Structure

```go
type Persona struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    SystemPrompt string   `json:"system_prompt"`
    CreatedAt   time.Time `json:"created_at"`
}
```

## Key Components

### Storage (`internal/config/personas.go`)
- **LoadPersonas()**: Load all personas from JSON file
- **SavePersonas()**: Persist personas to JSON
- **GetPersona()**: Retrieve a specific persona by ID
- **CreatePersona()**: Add new persona to storage
- **UpdatePersona()**: Modify existing persona
- **DeletePersona()**: Remove persona from storage

### Commands (`internal/commands/persona.go`)
- `geminiweb persona create <name>` - Interactive persona creation
- `geminiweb persona list` - Show all personas
- `geminiweb persona use <id>` - Use persona for current session
- `geminiweb persona delete <id>` - Remove persona
- `geminiweb persona edit <id>` - Modify persona

### API Integration (`internal/api/components/gem_mixin.go`)
- Mixes persona CRUD operations into GeminiClient
- Integrates persona system prompt with user queries
- Maintains persona context across chat sessions

## Usage Examples

### Create a Persona

```bash
./build/geminiweb persona create "Technical Writer"
```

Interactive prompts:
- Name: "Technical Writer"
- Description: "Expert technical writer for API documentation"
- System prompt: "You are a technical writing expert with 10+ years experience..."

### List Personas

```bash
./build/geminiweb persona list
```

Output:
```
ID: 550e8400-e29b-41d4-a716-446655440000
Name: Technical Writer
Description: Expert technical writer for API documentation
Created: 2024-01-15 10:30:00
```

### Use in Chat

```bash
./build/geminiweb chat --persona 550e8400-e29b-41d4-a716-446655440000
```

### Single Query with Persona

```bash
./build/geminiweb --persona 550e8400-e29b-41d4-a716-446655440000 "Explain TLS"
```

## Best Practices

1. **Descriptive Names**: Use clear, descriptive names for personas
2. **Detailed System Prompts**: Include specific instructions about behavior, expertise, and style
3. **Security**: Always validate and sanitize persona system prompts
4. **Context Injection**: System prompts are prepended to user queries automatically
5. **Backup**: Personas are stored as JSON in config directory

## Storage Location

Personas are stored in: `~/.geminiweb/personas.json`

Example content:
```json
{
  "personas": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Technical Writer",
      "description": "Expert technical writer for API documentation",
      "system_prompt": "You are a technical writing expert with 10+ years experience in API documentation...",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

## Security Considerations

- System prompts should not contain sensitive information
- No automatic execution of commands or code
- Prevents prompt injection by validating persona inputs
- Sanitizes user-provided persona content
