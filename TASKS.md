# Development Tasks

Generated from: PLAN.md
Date: 2025-12-02

## Progress

### Phase 1: Conversation Persistence (Critical Path)

- [ ] Task 1.1: Modify `ChatSession.SendMessage` to accept optional `[]*api.UploadedFile` parameter
- [ ] Task 1.1-test: Write tests for ChatSession file support (80%+ coverage)
- [ ] Task 1.2: Create `HistorySelectorModel` in `internal/tui/history_selector.go`
- [ ] Task 1.2-test: Write tests for HistorySelectorModel (80%+ coverage)
- [ ] Task 1.3: Refactor `commands/chat.go` to launch HistorySelectorModel first
- [ ] Task 1.3-test: Write tests for refactored chat command (80%+ coverage)
- [ ] Task 1.4: Update `Model.NewChatModel` to accept `*history.Conversation` for session initialization
- [ ] Task 1.4-test: Write tests for NewChatModel conversation loading (80%+ coverage)
- [ ] Task 1.5: Implement auto-save in `Model.Update` after responseMsg
- [ ] Task 1.5-test: Write tests for auto-save functionality (80%+ coverage)
- [ ] Task 1.6: Implement `/history` command to switch between conversations
- [ ] Task 1.6-test: Write tests for /history command (80%+ coverage)

### Phase 2: Input & Command Overhaul

- [ ] Task 2.1: Configure multi-line input (Shift+Enter for newline, Enter to send)
- [ ] Task 2.1-test: Write tests for multi-line input behavior (80%+ coverage)
- [ ] Task 2.2: Implement command parsing in `Model.Update` for /file, /image, /history
- [ ] Task 2.2-test: Write tests for command parsing (80%+ coverage)
- [ ] Task 2.3: Implement `/file <path>` and `/image <path>` commands
- [ ] Task 2.3-test: Write tests for file/image attachment commands (80%+ coverage)
- [ ] Task 2.4: Clear attachments after sending message
- [ ] Task 2.5: Update input area UX to show attached file count

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
