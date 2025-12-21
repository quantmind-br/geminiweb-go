# AGENTS.md

## Project Overview
`geminiweb-go` is a modular CLI/TUI application for interacting with Google Gemini's web API. It uses cookie-based authentication and browser TLS fingerprinting (Chrome 133) to emulate real browser behavior, enabling features like Gems management and file uploads.

## Build & Test Commands
```bash
make build          # Production build (requires CGO_ENABLED=1)
make build-dev      # Fast development build
make test           # Run all tests: go test -v ./...
make lint           # golangci-lint run ./...
make fmt            # go fmt + gofumpt
make check          # Verify build compiles
```

## Architecture & Key Components
- **`internal/api`**: `GeminiClient` (auth, TLS fingerprinting, token management) and `ChatSession` (stateful context).
- **`internal/browser`**: Extracts session cookies from local browsers (Chrome, Firefox, etc.).
- **`internal/tui`**: Terminal UI using Bubble Tea (The Elm Architecture).
- **`internal/history`**: Persistence for conversations in `~/.config/geminiweb/history/`.
- **`internal/models`**: Shared data structures and Gemini RPC endpoint constants.

## Code Style & Conventions
- **Go 1.23+**: Use functional options pattern for component configuration.
- **JSON Parsing**: Use `tidwall/gjson` for API responses; `encoding/json` for local persistence.
- **HTTP**: Use `bogdanfinn/tls-client` and `fhttp` to maintain browser fingerprints.
- **Errors**: Wrap with context: `fmt.Errorf("context: %w", err)`. Use `internal/errors` types.
- **DI**: Depend on interfaces (`GeminiClientInterface`, etc.) for testability.

## Issue Tracking: bd (beads)
**STRICT RULE**: Use `bd` for all task tracking. No markdown TODOs.
- `bd ready --json`: Check for unblocked work.
- `bd create "Title" -t bug|feature|task -p 1`: Create new issue.
- `bd update <id> --status in_progress`: Claim a task.
- `bd close <id> --reason "Done"`: Complete work.
- Always commit `.beads/issues.jsonl` with your code changes.

## AI Planning & History
Store all AI-generated planning docs (PLAN.md, DESIGN.md, etc.) in the `history/` directory to keep the root clean.

## Git Workflow
- Commits: Use conventional commits (feat:, fix:, chore:).
- Sync: Always `git pull` before running `bd` commands to ensure issue state is current.
