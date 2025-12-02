# CLAUDE.md

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

# Quality
make lint                 # golangci-lint
make fmt                  # go fmt + gofumpt
```

## Architecture

### Package Structure

- **`cmd/geminiweb/`** - CLI entrypoint
- **`internal/api/`** - GeminiClient: TLS client, token extraction (SNlM0e), cookie rotation, content generation, browser refresh
- **`internal/browser/`** - Browser cookie extraction using `browserutils/kooky` (Chrome, Firefox, Edge, Chromium, Opera)
- **`internal/commands/`** - Cobra commands: root (query), chat, config, import-cookies, auto-login, history, persona
- **`internal/config/`** - Settings and cookie storage in `~/.geminiweb/`
- **`internal/models/`** - Types (ModelOutput, Candidate, WebImage), model definitions, API constants/endpoints
- **`internal/tui/`** - Bubble Tea TUI with Glamour markdown rendering and Lipgloss styling
- **`internal/render/`** - Markdown rendering with Glamour, pooled renderers, configurable themes and caching
- **`internal/history/`** - JSON-based conversation history persistence
- **`internal/errors/`** - Custom error types

### Key Dependencies

- **TLS/HTTP**: `bogdanfinn/tls-client` (Chrome fingerprinting), `bogdanfinn/fhttp`
- **CLI**: `spf13/cobra`
- **TUI**: `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `charmbracelet/glamour`
- **JSON**: `tidwall/gjson`
- **Browser Cookies**: `browserutils/kooky` (cross-platform cookie extraction with decryption)

### Client Lifecycle

```go
client, err := api.NewClient(cookies,
    api.WithModel(models.Model25Flash),
    api.WithBrowserRefresh(browser.BrowserAuto), // Optional: auto-refresh from browser on auth failure
)
err := client.Init()              // Fetches access token (SNlM0e)
response, err := client.GenerateContent("prompt", opts)  // Auto-retries with fresh cookies on 401
chat := client.StartChat()
response, err := chat.SendMessage("hello")
client.Close()
```

### Key Patterns

1. **Functional Options** - `ClientOption` functions configure GeminiClient (WithModel, WithAutoRefresh, WithRefreshInterval, WithBrowserRefresh)
2. **TLS Fingerprinting** - Chrome 133 profile via `bogdanfinn/tls-client` to appear as real browser
3. **Auto Cookie Rotation** - Background goroutine refreshes tokens at `/accounts.google.com/RotateCookies` (default 9 min interval)
4. **Browser Cookie Refresh** - On auth failure (401), automatically extracts fresh cookies from browser and retries (rate-limited to 30s)
5. **Bubble Tea Architecture** - TUI uses Model/Update/View pattern; messages flow through Update, never mutate state directly
6. **Dependency Injection** - Key components use interfaces (`GeminiClientInterface`, `ChatSessionInterface`, `BrowserCookieExtractor`) and option functions (`WithRefreshFunc`, `WithCookieLoader`) for testability

### TUI Notes

- **Glamour markdown**: Use `glamour.WithStylePath("dark")` instead of `glamour.WithAutoStyle()` to avoid OSC 11 terminal query escape sequence leaks into stdin
- **Textarea input filtering**: Only pass `tea.KeyMsg` to textarea.Update() to prevent escape sequences from appearing as garbage characters
- **Viewport**: Always updated with all messages for scrolling support

## Code Style

- Go 1.23+, functional options pattern
- Errors: wrap with context using `fmt.Errorf("...: %w", err)`
- Imports: stdlib → blank line → external deps → blank line → internal packages
- Use `tidwall/gjson` for JSON parsing (not encoding/json for reads)
- Use `bogdanfinn/fhttp` for HTTP requests (not net/http)

## Models

Default model is `models.DefaultModel` which points to `models.Model30Pro` (gemini-3.0-pro).

- `models.Model25Flash` - Fast model (gemini-2.5-flash)
- `models.Model25Pro` - Balanced model (gemini-2.5-pro)
- `models.Model30Pro` - Advanced model (gemini-3.0-pro) - **recommended default**
- `models.ModelUnspecified` - Server's default model (no model header sent)
