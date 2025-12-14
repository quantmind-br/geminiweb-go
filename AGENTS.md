# AGENTS.md

## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**
```bash
bd ready --json
```

**Create new issues:**
```bash
bd create "Issue title" -t bug|feature|task -p 0-4 --json
bd create "Issue title" -p 1 --deps discovered-from:bd-123 --json
bd create "Subtask" --parent <epic-id> --json  # Hierarchical subtask (gets ID like epic-id.1)
```

**Claim and update:**
```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
```

**Complete work:**
```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`
6. **Commit together**: Always commit the `.beads/issues.jsonl` file together with the code changes so issue state stays in sync with code state

### Auto-Sync

bd automatically syncs with git:
- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### GitHub Copilot Integration

If using GitHub Copilot, also create `.github/copilot-instructions.md` for automatic instruction loading.
Run `bd onboard` to get the content, or see step 2 of the onboard instructions.

### MCP Server (Recommended)

If using Claude or MCP-compatible clients, install the beads MCP server:

```bash
pip install beads-mcp
```

Add to MCP config (e.g., `~/.config/claude/config.json`):
```json
{
  "beads": {
    "command": "beads-mcp",
    "args": []
  }
}
```

Then use `mcp__beads__*` functions instead of CLI commands.

### Managing AI-Generated Planning Documents

AI assistants often create planning and design documents during development:
- PLAN.md, IMPLEMENTATION.md, ARCHITECTURE.md
- DESIGN.md, CODEBASE_SUMMARY.md, INTEGRATION_PLAN.md
- TESTING_GUIDE.md, TECHNICAL_DESIGN.md, and similar files

**Best Practice: Use a dedicated directory for these ephemeral files**

**Recommended approach:**
- Create a `history/` directory in the project root
- Store ALL AI-generated planning/design docs in `history/`
- Keep the repository root clean and focused on permanent project files
- Only access `history/` when explicitly asked to review past planning

**Example .gitignore entry (optional):**
```
# AI planning documents (ephemeral)
history/
```

**Benefits:**
- ✅ Clean repository root
- ✅ Clear separation between ephemeral and permanent documentation
- ✅ Easy to exclude from version control if desired
- ✅ Preserves planning history for archeological research
- ✅ Reduces noise when browsing the project

### CLI Help

Run `bd <command> --help` to see all available flags for any command.
For example: `bd create --help` shows `--parent`, `--deps`, `--assignee`, etc.

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ✅ Store AI planning docs in `history/` directory
- ✅ Run `bd <cmd> --help` to discover available flags
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems
- ❌ Do NOT clutter repo root with planning documents

For more details, see README.md and QUICKSTART.md.

---

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