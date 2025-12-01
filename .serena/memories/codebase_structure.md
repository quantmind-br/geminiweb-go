# Codebase Structure

```
geminiweb-go/
├── cmd/geminiweb/
│   └── main.go                 # CLI entrypoint
├── internal/
│   ├── api/                    # Core API client
│   │   ├── client.go           # GeminiClient struct and initialization
│   │   ├── generate.go         # Content generation methods
│   │   ├── session.go          # ChatSession for multi-turn conversations
│   │   ├── token.go            # Access token (SNlM0e) extraction
│   │   ├── rotate.go           # Cookie rotation logic
│   │   ├── upload.go           # File upload support
│   │   ├── paths.go            # URL path construction
│   │   └── *_test.go           # Tests for each module
│   ├── browser/                # Browser cookie extraction
│   │   ├── browser.go          # Cookie extraction from Chrome, Firefox, Edge, etc.
│   │   └── browser_test.go     # Browser extraction tests
│   ├── commands/               # Cobra CLI commands
│   │   ├── root.go             # Main command, version flags, --browser-refresh
│   │   ├── chat.go             # Interactive chat command
│   │   ├── query.go            # Single query command
│   │   ├── config.go           # Configuration management
│   │   ├── import.go           # Cookie import command
│   │   ├── autologin.go        # Auto-login from browser command
│   │   ├── history.go          # History management
│   │   └── persona.go          # Persona management
│   ├── config/                 # Configuration management
│   │   ├── config.go           # Settings storage
│   │   ├── cookies.go          # Cookie persistence
│   │   └── personas.go         # Persona storage
│   ├── models/                 # Data types and constants
│   │   ├── constants.go        # Endpoints, Models, ErrorCodes, Headers
│   │   ├── response.go         # Response types (ModelOutput, Candidate, etc.)
│   │   └── message.go          # Message types
│   ├── tui/                    # Bubble Tea TUI
│   │   ├── model.go            # Main TUI model
│   │   ├── styles.go           # Lipgloss styling
│   │   └── config_model.go     # Config TUI model
│   ├── history/                # Conversation history
│   │   └── store.go            # JSON-based persistence
│   └── errors/                 # Custom error types
│       └── errors.go           # AuthError, APIError, TimeoutError, etc.
├── build/                      # Build output directory
├── Makefile                    # Build, test, lint commands
├── go.mod                      # Module definition
├── go.sum                      # Dependency checksums
└── CLAUDE.md                   # AI assistant instructions
```
