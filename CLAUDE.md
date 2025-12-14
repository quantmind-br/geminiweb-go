# CLAUDE.md

**Note**: This project uses [bd (beads)](https://github.com/steveyegge/beads)
for issue tracking. Use `bd` commands instead of markdown TODOs.
See AGENTS.md for workflow details.

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**geminiweb-go** is a CLI for interacting with Google Gemini via the web API. It uses cookie-based authentication (not API keys) and browser-like TLS fingerprinting (Chrome 133 profile) to communicate with Gemini's web interface.

## Build & Development Commands

```bash
# Build (requires CGO_ENABLED=1 for TLS client)
make build                # Production build with version info
make build-dev            # Fast development build
make install              # Install to GOPATH/bin

# Run
make run ARGS="chat"      # Build and run with arguments
./build/geminiweb         # Direct execution

# Testing
make test                 # Run all tests: go test -v ./...
go test -v ./internal/api -run TestClientInit   # Single test
make test-coverage        # Tests with HTML coverage report
make cover                # Show coverage breakdown by function

# Quality
make lint                 # golangci-lint
make fmt                  # go fmt + gofumpt
make check                # Verify build compiles without output
```

## Architecture

### Package Structure

- **`cmd/geminiweb/`** - CLI entrypoint.
- **`internal/api/`** - `GeminiClient`: Core component handling TLS client (Chrome 133 fingerprint), access token extraction (SNlM0e), background cookie rotation, content generation, browser refresh logic, and Gems API interaction.
- **`internal/browser/`** - Browser cookie extraction using `browserutils/kooky` (supports Chrome, Firefox, Edge, Chromium, Opera).
- **`internal/commands/`** - Cobra commands: root (single query), chat, config, import-cookies, auto-login, history, persona, gems.
- **`internal/config/`** - Manages user settings, cookie storage, and local personas in `~/.geminiweb/`.
- **`internal/models/`** - Data types (ModelOutput, Candidate, Gem), model definitions, and API constants/endpoints.
- **`internal/tui/`** - Bubble Tea TUI: main chat model, gems selector, history selector, config editor.
- **`internal/render/`** - Markdown rendering with Glamour, featuring pooled renderers, a theme system (dark/light/dracula/nord/custom), and LRU caching for performance.
- **`internal/history/`** - JSON-based conversation history persistence with metadata management.
- **`internal/errors/`** - Custom error types (AuthError, APIError, TimeoutError, UsageLimitError, BlockedError, ParseError) for robust error handling.

### Key Dependencies

- **TLS/HTTP**: `bogdanfinn/tls-client` (Chrome fingerprinting), `bogdanfinn/fhttp`
- **CLI**: `spf13/cobra`
- **TUI**: `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `charmbracelet/glamour`
- **JSON**: `tidwall/gjson`
- **Browser Cookies**: `browserutils/kooky` (cross-platform cookie extraction with decryption)

### Client Lifecycle

```go
client, err := api.NewClient(cookies,
    api.WithModel(models.Model30Pro),
    api.WithBrowserRefresh(browser.BrowserAuto), // Optional: auto-refresh from browser on auth failure
)
err := client.Init()              // Fetches access token (SNlM0e)
response, err := client.GenerateContent("prompt", opts)  // Auto-retries with fresh cookies on 401
chat := client.StartChat()
response, err := chat.SendMessage("hello")
client.Close()
```

### Key Patterns & Authentication Flow

1.  **Functional Options** - `ClientOption` functions configure `GeminiClient` (`WithModel`, `WithAutoRefresh`, `WithBrowserRefresh`).
2.  **TLS Fingerprinting** - A Chrome 133 profile via `bogdanfinn/tls-client` makes requests appear as a real browser to avoid anti-bot detection.
3.  **Multi-Layered Auth**:
    *   **Cookie-based**: Loads `__Secure-1PSID` and `__Secure-1PSIDTS` from `~/.geminiweb/cookies.json`.
    *   **Access Token**: Fetches a temporary `SNlM0e` token required for API requests.
    *   **Auto Cookie Rotation**: A background goroutine refreshes the `__Secure-1PSIDTS` cookie via `/accounts.google.com/RotateCookies` (default 9 min interval) to prevent session expiry.
    *   **Browser Cookie Refresh**: On an authentication failure (401), the client automatically extracts fresh cookies from a local browser and retries the request. This is rate-limited to 30 seconds.
4.  **Bubble Tea Architecture** - The TUI uses the Model/Update/View pattern. Messages flow through the `Update` function; state is never mutated directly.
5.  **Dependency Injection** - Key components use interfaces (`GeminiClientInterface`, `ChatSessionInterface`, `BrowserCookieExtractor`) and option functions (`WithRefreshFunc`, `WithCookieLoader`) for testability.
6.  **Context Propagation** - Always pass `context.Context` explicitly; use `context.WithTimeout` for request deadlines.

### TUI Notes

- **Glamour markdown**: Use `glamour.WithStylePath("dark")` instead of `glamour.WithAutoStyle()` to avoid OSC 11 terminal query escape sequence leaks into stdin.
- **Textarea input filtering**: Only pass `tea.KeyMsg` to `textarea.Update()` to prevent escape sequences from appearing as garbage characters.
- **Viewport**: Always updated with the full message history to support scrolling.
- **Input**: Use `\ + Enter` for multiline input (a backslash followed by Enter inserts a newline).

## Code Style

- Go 1.23+, functional options pattern.
- Errors: Wrap with context using `fmt.Errorf("...: %w", err)`.
- Imports: stdlib → blank line → external deps → blank line → internal packages.
- Use `tidwall/gjson` for parsing JSON responses (not `encoding/json` for reads).
- Use `bogdanfinn/fhttp` and `bogdanfinn/tls-client` for HTTP requests (not `net/http`).

## Models

The default model is `models.DefaultModel` which points to `models.Model30Pro`.

- `models.Model25Flash` - Fast model (gemini-2.5-flash)
- `models.Model30Pro` - Advanced model (gemini-3.0-pro) - **recommended default**
- `models.ModelUnspecified` - Server's default model (no model header is sent)

## Gems (Server-side Personas)

Gems are Google's custom personas stored on their servers. Unlike local personas, gems sync across devices.

```bash
geminiweb gems list              # Browse gems with interactive TUI
geminiweb chat --gem "Code Helper"  # Start chat with a gem
geminiweb chat -g code           # Partial name matching
```

In the chat TUI, type `/gems` to switch gems without leaving the session.

## Testing

Integration tests require valid cookies set as environment variables:
```bash
export SECURE_1PSID="your_cookie_value"
export SECURE_1PSIDTS="optional_cookie_value"  # Some accounts require this
make test
```

For mocking in tests, use the provided interfaces:
- `GeminiClientInterface` - Mock the API client
- `ChatSessionInterface` - Mock chat sessions
- `BrowserCookieExtractor` - Mock browser cookie extraction
```