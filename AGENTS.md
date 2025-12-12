# AGENTS.md

## Project Overview
geminiweb-go is a CLI for interacting with Google Gemini's web API using cookie-based authentication and browser-like TLS fingerprinting to avoid detection.

## Build & Test Commands
```bash
make build          # Build binary to build/geminiweb (CGO_ENABLED=1 required)
make build-dev      # Fast dev build without optimizations
make test           # Run all tests: go test -v ./...
go test -v ./internal/api -run TestClientInit   # Run single test
make lint           # golangci-lint run ./...
make fmt            # go fmt + gofumpt
make check          # Verify build compiles
```

## Architecture
- **`cmd/geminiweb/`**: CLI entrypoint using Cobra.
- **`internal/api/`**: Core `GeminiClient` with TLS fingerprinting, token management, and auto-refresh logic.
- **`internal/browser/`**: Cross-platform browser cookie extraction (`kooky`).
- **`internal/commands/`**: Cobra command implementations (chat, query, config, history).
- **`internal/config/`**: Manages settings and cookie storage in `~/.geminiweb/`.
- **`internal/history/`**: JSON-based conversation persistence.
- **`internal/tui/`**: Interactive UI with Bubble Tea and Glamour for markdown.
- **`internal/models/`**: Shared data types, constants, and API endpoints.
- **`internal/errors/`**: Custom error types for authentication, API, and network issues.

## Code Style & Conventions
- Go 1.23+, use functional options pattern (see `ClientOption` in `api/client.go`).
- Errors: Use custom error types from `internal/errors/` with `errors.NewGeminiError`.
- Imports: stdlib, blank line, external deps, blank line, internal packages.
- Use `tidwall/gjson` for reading JSON responses.
- Use `bogdanfinn/tls-client` and `bogdanfinn/fhttp` for all HTTP requests.
- On auth failure (401), the client automatically tries to refresh cookies from the browser.
- Interfaces: Depend on `api.GeminiClientInterface`, `history.HistoryStoreInterface`, and `browser.BrowserCookieExtractor` for decoupling.
- Configuration: Access through `config.Config` struct, not direct environment variables.
- TUI: Use Bubble Tea with message-based state updates, not direct state mutation.
```,claude_md: