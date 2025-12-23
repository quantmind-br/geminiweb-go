# AGENTS.md

## Project Overview
`geminiweb-go` is a high-performance CLI/TUI client for the Google Gemini Web API. It emulates browser behavior via Chrome 133 TLS fingerprinting and cookie-based auth to enable unofficial features like Gems, thinking models, and tool execution.

## Build & Test Commands
```bash
make build          # Production build (CGO_ENABLED=1 for kooky/sqlite)
make build-dev      # Development build
make test           # Full test suite (requires SECURE_1PSID env for integration)
make lint           # Run golangci-lint
make fmt            # Run gofumpt (strict formatting)
make check          # Verify compilation across all packages
```

## Architecture Overview
- **Presentation Layer**: CLI via `spf13/cobra` and TUI via `charmbracelet/bubbletea` (MVU pattern).
- **Service Layer**: `internal/api` (GeminiClient/ChatSession) and `pkg/toolexec` (modular tool execution).
- **Persistence**: Local JSON-based history in `~/.geminiweb/history/`.
- **Infrastructure**: `internal/browser` for cookie extraction (kooky) and `internal/render` for Glamour-based markdown.

## Code Style & Tech Stack
- **Language**: Go 1.23+ with Functional Options pattern for constructors.
- **Networking**: STRICT: Use `bogdanfinn/tls-client` + `fhttp` for API calls to avoid 403s.
- **JSON**: Use `tidwall/gjson` for parsing nested Gemini RPC arrays; `encoding/json` for local files.
- **Concurrency**: Always pass `context.Context` through service and API layers.
- **Errors**: Use structured types in `internal/errors`. Handle code 1037 (Rate Limit).

## Key Conventions
- **Interfaces**: Define interfaces (e.g., `GeminiClientInterface`) to facilitate UI mocking.
- **UI Styling**: Use `lipgloss` for components; `internal/render` for markdown.
- **Tool Execution**: `pkg/toolexec` uses a Registry and Middleware for security policies.
- **RPC Mapping**: See `internal/api/paths.go` for positional index mappings.

## Issue Tracking (BEADS)
**STRICT RULE**: Use [bd (beads)](https://github.com/steveyegge/beads) for all tasks. No markdown TODOs.
- `bd ready`: View next unblocked tasks.
- `bd create "Title" -t bug|feature|task`: Create a new issue.
- `bd update <id> --status in_progress`: Claim a task.
- `bd close <id>`: Mark task as complete.

## Git & Workflow
- **Commits**: Conventional Commits format (`feat:`, `fix:`, `chore:`).
- **Planning**: Store AI design/analysis docs in `.ai/docs/`.
- **Tests**: Integration tests require `SECURE_1PSID` and `SECURE_1PSIDTS` env vars.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
