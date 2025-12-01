# AGENTS.md

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
- `cmd/geminiweb/` - CLI entrypoint using Cobra
- `internal/api/` - GeminiClient: TLS client, token fetch, cookie rotation, content generation
- `internal/commands/` - Cobra commands: chat, query, config, import-cookies
- `internal/config/` - Cookie storage and config management (~/.geminiweb/)
- `internal/models/` - Types: Model, Response, Message, API constants/endpoints
- `internal/tui/` - Bubbletea interactive UI with Glamour markdown rendering
- `internal/errors/` - Custom error types

## Code Style
- Go 1.23+, use functional options pattern (see `ClientOption` in api/client.go)
- Errors: wrap with context using `fmt.Errorf("...: %w", err)`
- Imports: stdlib first, blank line, external deps, blank line, internal packages
- Use `tidwall/gjson` for JSON parsing, `bogdanfinn/tls-client` for HTTP
