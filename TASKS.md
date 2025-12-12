# TASKS - Keyboard Shortcuts for Chat TUI

## Overview
Add keyboard shortcuts `Ctrl+E` (Export) and `Ctrl+G` (Gems) to the Chat TUI.

## Tasks

### Phase 1: Implementation
- [x] 1.1: Add `ctrl+g` case in `Model.Update` to open gem selector
- [x] 1.2: Add `ctrl+e` case in `Model.Update` to export conversation
- [x] 1.3: Update `renderStatusBar` to display new shortcuts

### Phase 2: Testing & Validation
- [x] 2.1: Write unit tests for new keyboard shortcuts
- [x] 2.2: Build and verify
- [x] 2.3: Increase test coverage to 80%+

## Validation Criteria
1. `Ctrl+E` exports conversation to Markdown with default filename
2. `Ctrl+G` opens Gems selector immediately
3. Status bar shows `^E (Export)` and `^G (Gems)`
4. Existing shortcuts continue working (`Enter`, `Ctrl+C`, `Esc`, `\+Enter`)
5. `Ctrl+E` without conversation shows appropriate error
6. Test coverage >= 80%

## Implementation Summary

### Changes Made
1. **`internal/tui/model.go:263-274`** - Added `ctrl+g` and `ctrl+e` cases in the keyboard handling switch
2. **`internal/tui/model.go:698-699`** - Added `^E` and `^G` shortcuts to the status bar
3. **`internal/tui/model_test.go`** - Added unit tests for new shortcuts and additional coverage tests
4. **`internal/tui/history_manager_test.go`** - Created comprehensive test suite (new file)

### Test Coverage Results

| Package | Before | After |
|---------|--------|-------|
| `internal/tui` | 65.8% | **83.5%** |

### Tests Added
- `TestModel_Update_CtrlG` - Tests that Ctrl+G opens gem selector
- `TestModel_Update_CtrlE` - Tests export with messages and error without messages
- `TestRenderStatusBar_ShowsNewShortcuts` - Tests status bar shows new shortcuts
- `TestHistoryManagerModel_*` - Comprehensive tests for history manager (25+ test cases)
- `TestModel_FormatError` - Tests error formatting
- `TestModel_UpdateGemSelection` - Tests gem selection mode (12 test cases)
- `TestModel_ExportFromMemory` - Tests in-memory export
- `TestJsonMarshalIndent` - Tests JSON helper
- `TestNewChatModel_WithClient` - Tests model constructor
