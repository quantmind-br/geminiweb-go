# Recent Changes and Project Status

## Latest Commits (as of 2025-12-02)

Recent development has focused on TUI improvements and conversation persistence:

1. **TUI Input Overhaul**
   - Replaced Shift+Enter with `\ + Enter` for multiline input
   - Phase 2 & 3 style and UX polish
   - Improved input handling and command system

2. **Conversation Persistence**
   - `/history` command for switching between conversations
   - Auto-save functionality for conversation persistence
   - History selector model for conversation selection
   - Conversation loading from JSON files

3. **API Enhancements**
   - File support added to ChatSession.SendMessage
   - HTTP client injection for improved testability

## Current Git Status

- Modified: `CLAUDE.md`, `PLAN.md`
- Deleted: `TASKS.md`
- New directory: `development-docs/`

## Key Architectural Features

### TUI Components
- **Main chat model**: `internal/tui/model.go`
- **Gems selector**: `internal/tui/gems_model.go`
- **History selector**: `internal/tui/history_selector.go`
- **Config editor**: `internal/tui/config_model.go`
- **Styles**: `internal/tui/styles.go`

### Render System
- Pooled Glamour renderers for markdown
- Theme system (dark/light/dracula/nord/custom)
- LRU caching for rendered content
- Located in `internal/render/`

### Browser Cookie Extraction
- Package: `internal/browser/`
- Supported browsers: Chrome, Chromium, Firefox, Edge, Opera
- `auto-login` command for direct extraction
- `--browser-refresh` flag for auto-refresh on 401
- Rate limiting: 30 second minimum between refresh attempts

## Testing

Integration tests require:
```bash
export SECURE_1PSID="your_cookie_value"
export SECURE_1PSIDTS="optional_cookie_value"
make test
```

## Known Working State

- All TUI features functional
- Conversation persistence working
- Gems integration complete
- Browser cookie extraction stable
