# Development Tasks

Generated from: PLAN.md
Date: 2025-12-02

## Progress

### Phase 1: Conversation Persistence (Critical Path) ✅ COMPLETE

- [x] Task 1.1: Modify `ChatSession.SendMessage` to accept optional `[]*api.UploadedFile` parameter
- [x] Task 1.1-test: Write tests for ChatSession file support (80%+ coverage)
- [x] Task 1.2: Create `HistorySelectorModel` in `internal/tui/history_selector.go`
- [x] Task 1.2-test: Write tests for HistorySelectorModel (80%+ coverage)
- [x] Task 1.3: Refactor `commands/chat.go` to launch HistorySelectorModel first
- [x] Task 1.3-test: Write tests for refactored chat command (80%+ coverage)
- [x] Task 1.4: Update `Model.NewChatModel` to accept `*history.Conversation` for session initialization
- [x] Task 1.4-test: Write tests for NewChatModel conversation loading (80%+ coverage)
- [x] Task 1.5: Implement auto-save in `Model.Update` after responseMsg
- [x] Task 1.5-test: Write tests for auto-save functionality (80%+ coverage)
- [x] Task 1.6: Implement `/history` command to switch between conversations
- [x] Task 1.6-test: Write tests for /history command (80%+ coverage)

**Commits:**
- `feat(tui): implement auto-save for conversation persistence`
- `test(tui): add comprehensive tests for auto-save functionality`
- `feat(tui): implement /history command for conversation switching`
- `test(tui): add comprehensive tests for /history command`

**Coverage:** TUI package at 78.5%

### Phase 2: Input & Command Overhaul ✅ COMPLETE

- [x] Task 2.1: Configure multi-line input (Shift+Enter for newline, Enter to send)
- [x] Task 2.1-test: Write tests for multi-line input behavior (80%+ coverage)
- [x] Task 2.2: Implement command parsing in `Model.Update` for /file, /image, /history
- [x] Task 2.2-test: Write tests for command parsing (80%+ coverage)
- [x] Task 2.3: Implement `/file <path>` and `/image <path>` commands
- [x] Task 2.3-test: Write tests for file/image attachment commands (80%+ coverage)
- [x] Task 2.4: Clear attachments after sending message
- [x] Task 2.5: Update input area UX to show attached file count

**Implementation:**
- Added `createTextarea()` helper with Shift+Enter for newline, Enter to send
- Created `ParsedCommand` struct and `parseCommand()` function for command routing
- Implemented `/file` and `/image` commands with async file upload
- Added `attachments` field to Model, cleared after send
- Status bar shows attachment count with 📎 indicator

**Coverage:** TUI package at 79.1%

### Phase 3: Style and UX Polish

- [ ] Task 3.1: Display image URLs from ModelOutput in a styled format
- [ ] Task 3.1-test: Write tests for image URL display (80%+ coverage)
- [ ] Task 3.2: Refactor `internal/tui/styles.go` to use centralized theme struct
- [ ] Task 3.2-test: Write tests for theme system (80%+ coverage)
- [ ] Task 3.3: Extend ConfigModel for TUI theme selection
- [ ] Task 3.3-test: Write tests for theme configuration (80%+ coverage)
- [ ] Task 3.4: Create initial themes (Catppuccin/Nord)

### Final Validation

- [ ] Run full test suite and verify 80%+ coverage on new code
- [ ] Run linter and fix any issues
