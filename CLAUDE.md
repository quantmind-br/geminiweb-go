# CLAUDE.md

**Note**: This project uses [bd (beads)](https://github.com/steveyegge/beads) for issue tracking. Use `bd` commands instead of markdown TODOs. See AGENTS.md for workflow details.

## Project Overview
**geminiweb-go** is a sophisticated CLI for interacting with Google Gemini via its web interface. It bypasses the need for official API keys by using cookie-based authentication and browser-like TLS fingerprinting (Chrome 133 profile) to emulate a real user session.

## Command Reference

### Build & Run
```bash
# Build (requires CGO_ENABLED=1 for TLS client)
make build                # Production build with version info
make build-dev            # Fast development build
make install              # Install to GOPATH/bin

# Run
make run ARGS="chat"      # Build and run with arguments
./build/geminiweb chat    # Launch TUI
geminiweb "Hello world"   # Single query
```

### Testing & Quality
```bash
make test                 # Run all tests
go test -v ./internal/api # Test specific package
make test-coverage        # Generate HTML coverage report
make lint                 # golangci-lint
make fmt                  # go fmt + gofumpt
make check                # Verify compilation
```

## Architecture Overview

### Package Roles
- **`cmd/geminiweb/`**: Entry point orchestrating the Cobra CLI.
- **`internal/api/`**: The core engine. `GeminiClient` handles TLS fingerprinting, `SNlM0e` token extraction, and background cookie rotation. `ChatSession` maintains conversation context (CID, RID, RCID).
- **`internal/tui/`**: Bubble Tea-based TUI following the Model-View-Update pattern.
- **`internal/browser/`**: Cross-platform cookie extraction (Chrome, Firefox, Edge, etc.) via `kooky`.
- **`internal/history/`**: JSON persistence for chat logs and metadata indexing.
- **`internal/render/`**: Markdown rendering using Glamour with custom theme support.
- **`internal/models/`**: Domain models and API endpoint definitions.

### Authentication Flow
1. **Cookie Load**: Retrieves `__Secure-1PSID` and `__Secure-1PSIDTS` from local config or browser.
2. **Token Fetch**: Performs `EndpointInit` to extract the `SNlM0e` session token via regex.
3. **Request Signing**: Injects cookies and the `at` (SNlM0e) parameter into every RPC call.
4. **Auto-Refresh**: On 401 errors, the client automatically triggers a browser cookie re-extraction and retries.
5. **Rotation**: A background goroutine refreshes `__Secure-1PSIDTS` periodically to prevent expiry.

## Code Style & Conventions

- **Programming Model**: Go 1.23+. Extensive use of **Functional Options** for constructors (e.g., `api.WithModel`).
- **JSON Handling**: Use `tidwall/gjson` for reading the complex, nested array-based Gemini RPC responses. Standard `encoding/json` is reserved for local configuration/history.
- **Networking**: NEVER use `net/http`. Use `bogdanfinn/tls-client` and `bogdanfinn/fhttp` to ensure consistent browser fingerprinting.
- **Error Handling**: Use custom error types from `internal/errors`. Wrap errors: `fmt.Errorf("failed to fetch gems: %w", err)`.
- **Concurrency**: Use `context.Context` for all network and long-running operations.
- **Imports**: Group as: stdlib, newline, external, newline, internal.

## Key Components & Gotchas

- **TUI Rendering**: We use `glamour.WithStylePath("dark")` instead of `AutoStyle` to prevent terminal escape sequence leaks (OSC 11) into stdin.
- **Multiline Input**: In the TUI, use `\ + Enter` to insert a newline.
- **Models**: `models.Model30Pro` (Gemini 3.0 Pro) is the recommended default.
- **Gems**: These are server-side personas. Access them via `/gems` in the TUI or the `gems` CLI command.
- **Persistence**: Data is stored in `~/.config/geminiweb/`.

## Testing Policy
Integration tests require active cookies. Set `SECURE_1PSID` and `SECURE_1PSIDTS` in your environment to run them. Use the interfaces defined in `internal/api` and `internal/tui` for mocking.
