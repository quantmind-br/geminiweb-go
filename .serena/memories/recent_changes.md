# Recent Changes and Project Status

## Latest Commits (as of 2025-12-10)

Recent development focused on conversation management and TUI enhancements:

1. **Conversation Management (ffc561c)**
   - Implemented `internal/history/meta.go` for metadata store
   - Added favorites system and conversation ordering
   - `HistoryMeta` struct tracks order and metadata

2. **Gems and History Selectors (7a99171)**
   - Added `internal/tui/gems_model.go` for gem selection
   - Added `internal/tui/history_selector.go` for conversation switching
   - Both support fuzzy filtering

3. **TUI Input Overhaul (8e86a8a, 94e8337, 0ab3f27)**
   - Replaced Shift+Enter with `\ + Enter` for multiline input
   - Phase 2 & 3 style and UX polish
   - Improved input handling and command system

4. **Conversation Persistence Features**
   - `/history` command for switching between conversations
   - `/manage` command for full history management
   - `/favorite` command to toggle favorites
   - Auto-save functionality
   - History selector with filtering

## TUI Commands Available

```
/exit, /quit    - Exit chat
/gems, /gem     - Open gems selector
/history, /hist - Open history selector
/manage         - Open full history manager
/favorite, /fav - Toggle favorite on current conversation
/file <path>    - Attach file
/image <path>   - Attach image
/clear          - Clear attachments
/new            - Start new conversation
/help           - Show help
```

## Key Architectural Components

### History System (`internal/history/`)
- **store.go**: `Store`, `Conversation`, `Message` - Core persistence
- **meta.go**: `HistoryMeta`, `ConversationMeta` - Favorites and ordering
- **resolver.go**: `Resolver` - Alias resolution (@last, @first, indices)
- **export.go**: Export to Markdown/JSON with options

### TUI Components (`internal/tui/`)
- **model.go**: Main chat model (~1600 lines)
- **gems_model.go**: Gems selector with filtering
- **history_selector.go**: Conversation picker
- **history_manager.go**: Full management interface
- **config_model.go**: Config editor

## Current Git Status

- Modified: `CLAUDE.md`, `PLAN.md`
- New directory: `.ai/`, `.cursor/`

## Testing Requirements

Integration tests require environment variables:
```bash
export SECURE_1PSID="your_cookie_value"
export SECURE_1PSIDTS="optional_cookie_value"
make test
```
