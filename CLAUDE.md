# CLAUDE.md

Context for working in the **geminiweb-go** repository.

**Important**: This project uses [bd (beads)](https://github.com/steveyegge/beads) for issue tracking. **Do not use markdown TODOs.** Use `bd` commands for all tasks.

## Project Overview
**geminiweb-go** is a feature-rich CLI and TUI client for the Google Gemini Web interface. It bypasses official API keys by emulating a browser session using cookie-based authentication, TLS fingerprinting, and automated session management.

## Development Commands

### Build & Install
```bash
make build                # Production build (CGO_ENABLED=1 for kooky/sqlite)
make build-dev            # Faster development build
make install              # Install binary to $GOPATH/bin
make run ARGS="chat"      # Build and launch interactive TUI
```

### Testing & Quality
```bash
make test                 # Run all tests
go test -v ./internal/api # Test API package specifically
make lint                 # Run golangci-lint
make fmt                  # Run gofumpt (strict formatting)
make check                # Verify compilation across all packages
```

### Issue Tracking (Beads Workflow)
```bash
bd ready --json           # View unblocked tasks
bd create "Title" -p 1    # Create a new prioritized task
bd update <id> --status in_progress  # Claim a task
bd close <id> --reason "Fixed"       # Complete a task
```

## Architecture & Core Components

### Subsystems
- **`internal/api/`**: The core engine.
    - `GeminiClient`: Manages `tls-client`, `SNlM0e` (at) tokens, and cookie rotation.
    - `ChatSession`: Tracks conversation state (`cid`, `rid`, `rcid`).
    - `generate.go`: Implements the complex `StreamGenerate` RPC parsing via `gjson`.
- **`internal/tui/`**: Bubble Tea implementation using **Model-View-Update (MVU)**.
- **`pkg/toolexec/`**: Standalone framework for tool execution with **Registry** and **Middleware**.
- **`internal/history/`**: JSON persistence maintaining `meta.json` (index) and conversation files.
- **`internal/browser/`**: Handles session cookie extraction from local browser profiles (Chrome, Firefox, Edge).

## Technical Conventions

### Code Style
- **Functional Options**: Preferred for configuration (e.g., `api.NewClient(api.WithModel("thinking"))`).
- **Networking**: **MANDATORY**: Use `bogdanfinn/tls-client` for Gemini API calls. Standard `net/http` will be blocked.
- **JSON Parsing**: Use `tidwall/gjson` for the deeply nested array-based Gemini web responses.
- **Concurrency**: Pass `context.Context` through all service and API layers.
- **Interfaces**: Define interfaces for services to facilitate TUI testing with mocks.

### Error Handling
- Use structured errors from `internal/errors`.
- `AuthError`: Triggered on 401 or login redirects.
- `RateLimitError`: Detects Google code `1037`.
- `BlockedError`: Triggered when safety filters or "Sorry" redirects occur.

## UI Specifics
- **Input**: `\ + Enter` for newlines; `Enter` sends the prompt.
- **Slash Commands**: `/gems` (persona), `/history` (selector), `/new` (reset), `/model <name>`.
- **Styling**: Use `lipgloss` for components. `internal/render` wraps `glamour` with fixed dark-mode styling.

## Development Gotchas
- **CGO**: Required for `kooky` to read certain encrypted browser SQLite databases.
- **Cookie Rotation**: `__Secure-1PSIDTS` is short-lived; the client uses a background goroutine to refresh it.
- **RPC Indices**: The protocol is positional. Update `internal/api/paths.go` if the Google Web API structure changes.
Use 'bd' for task tracking
