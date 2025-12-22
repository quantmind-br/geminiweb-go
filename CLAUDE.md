# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Note**: This project uses [bd (beads)](https://github.com/steveyegge/beads) for issue tracking. Use `bd` commands instead of markdown TODOs. See AGENTS.md for workflow details.

## Project Overview
**geminiweb-go** is a CLI/TUI for interacting with Google Gemini via its web interface. It uses cookie-based authentication and browser-like TLS fingerprinting (Chrome 133) to emulate real browser behavior, bypassing the need for official API keys.

## Command Reference

### Build & Run
```bash
make build                # Production build (requires CGO_ENABLED=1)
make build-dev            # Fast development build
make install              # Install to GOPATH/bin
make run ARGS="chat"      # Build and run with arguments
./build/geminiweb chat    # Launch TUI
geminiweb "Hello world"   # Single query
```

### Testing & Quality
```bash
make test                            # Run all tests
go test -v ./internal/api            # Test specific package
go test -v -run TestFunctionName ./internal/api  # Run single test
make test-coverage                   # Generate HTML coverage report
make cover                           # Show coverage breakdown by function
make lint                            # golangci-lint
make fmt                             # go fmt + gofumpt
make check                           # Verify compilation
```

## Architecture Overview

### Package Roles
- **`internal/api/`**: Core engine. `GeminiClient` handles TLS fingerprinting, `SNlM0e` token extraction, cookie rotation. `ChatSession` maintains conversation context (CID, RID, RCID).
- **`internal/tui/`**: Bubble Tea TUI (~1600 lines in `model.go`). State machine with modes: input, loading, viewing, gems selection, history management.
- **`internal/browser/`**: Cookie extraction from Chrome, Firefox, Edge via `kooky`.
- **`internal/history/`**: JSON persistence with `Store`, `Resolver` (aliases: `@last`, `@first`, index numbers), `Export` (markdown/JSON).
- **`internal/config/`**: Settings, cookie persistence, local personas.
- **`internal/render/`**: Glamour markdown rendering with LRU cache.
- **`internal/models/`**: Domain models, endpoint constants, response types.

### Authentication Flow
1. **Cookie Load**: Retrieves `__Secure-1PSID` and `__Secure-1PSIDTS` from local config or browser.
2. **Token Fetch**: `EndpointInit` extracts the `SNlM0e` session token via regex.
3. **Request Signing**: Injects cookies and `at` (SNlM0e) into every RPC call.
4. **Auto-Refresh**: On 401, triggers browser cookie re-extraction and retries.
5. **Rotation**: Background goroutine refreshes `__Secure-1PSIDTS` every 9 minutes.

## Code Style & Conventions

- **Go 1.23+**: Use **Functional Options** for constructors (e.g., `api.WithModel`).
- **JSON Parsing**: Use `tidwall/gjson` for Gemini API responses; `encoding/json` for local files.
- **HTTP**: NEVER use `net/http`. Use `bogdanfinn/tls-client` and `bogdanfinn/fhttp`.
- **Errors**: Use `internal/errors` types (`AuthError`, `APIError`, `TimeoutError`, etc.). Wrap: `fmt.Errorf("context: %w", err)`.
- **Context**: Pass `context.Context` for all network/long-running operations.
- **Imports**: Group as: stdlib → external → internal (with blank lines between).

## Key Gotchas

- **TUI Rendering**: Use `glamour.WithStylePath("dark")` instead of `AutoStyle` to prevent OSC 11 escape sequence leaks.
- **Multiline Input**: `\ + Enter` inserts newline in TUI.
- **Models**: Default is `models.ModelPro`. Available: `fast`, `pro`, `thinking`. Legacy: `Model25Flash`, `Model30Pro`.
- **Stream Completion**: Responses end with `[["e",...]]` marker.
- **Storage**: Data in `~/.config/geminiweb/`. History in `~/.config/geminiweb/history/`.

## TUI Slash Commands
- `/gems` - Select server-side persona
- `/history` - Open conversation selector
- `/manage` - Full history manager (reorder, delete, export)
- `/favorite` - Toggle favorite on current conversation
- `/new` - Start new conversation
- `/model <name>` - Switch model (fast/pro/thinking)
- `/persona <name>` - Switch local persona

## API Error Codes
- **1037**: Usage limit exceeded (rate limit).
- **Error 2**: Manual verification (CAPTCHA) required.

## Testing
Integration tests require cookies. Set `SECURE_1PSID` and `SECURE_1PSIDTS` in environment. Use interfaces (`GeminiClientInterface`, `HistoryStoreInterface`) for mocking.
