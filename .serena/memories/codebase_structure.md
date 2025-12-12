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
│   │   ├── gems.go             # Gems API (server-side personas)
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
│   │   ├── history.go          # History management (list, show, delete, export, search)
│   │   ├── persona.go          # Local persona management
│   │   └── gems.go             # Gems listing and selection
│   ├── config/                 # Configuration management
│   │   ├── config.go           # Settings storage
│   │   ├── cookies.go          # Cookie persistence
│   │   └── personas.go         # Local persona storage
│   ├── models/                 # Data types and constants
│   │   ├── constants.go        # Endpoints, Models, ErrorCodes, Headers
│   │   ├── response.go         # Response types (ModelOutput, Candidate, etc.)
│   │   └── message.go          # Message types
│   ├── tui/                    # Bubble Tea TUI
│   │   ├── model.go            # Main TUI model (chat interface, ~1600 lines)
│   │   ├── styles.go           # Lipgloss styling
│   │   ├── gems_model.go       # Gems selector TUI
│   │   ├── history_selector.go # Conversation history selector (/history command)
│   │   ├── history_manager.go  # Full history manager (/manage command)
│   │   ├── config_model.go     # Config editor TUI
│   │   └── *_test.go           # Tests for TUI components
│   ├── render/                 # Markdown rendering
│   │   ├── render.go           # Glamour markdown renderer
│   │   ├── cache.go            # LRU caching for rendered content
│   │   ├── themes.go           # Theme definitions
│   │   ├── tui_themes.go       # TUI-specific theme helpers
│   │   ├── config.go           # Render configuration
│   │   ├── options.go          # Functional options
│   │   └── themes/             # Custom theme files
│   ├── history/                # Conversation history
│   │   ├── store.go            # JSON-based persistence (Store, Conversation, Message)
│   │   ├── meta.go             # Metadata store (favorites, ordering, ConversationMeta)
│   │   ├── resolver.go         # Alias resolution (@last, @first, index numbers)
│   │   ├── export.go           # Export to Markdown/JSON with options
│   │   └── *_test.go           # Tests for history system
│   └── errors/                 # Custom error types
│       └── errors.go           # AuthError, APIError, TimeoutError, etc.
├── build/                      # Build output directory
├── Makefile                    # Build, test, lint commands
├── go.mod                      # Module definition
├── go.sum                      # Dependency checksums
└── CLAUDE.md                   # AI assistant instructions
```

## Key File Sizes (approximate)

- `internal/tui/model.go` - ~1600 lines (main chat TUI)
- `internal/api/client.go` - ~400 lines (API client)
- `internal/history/store.go` - ~300 lines (persistence)
