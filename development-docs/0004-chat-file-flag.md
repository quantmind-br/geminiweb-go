# Implementation: Chat Initial Prompt from File

## Project Briefing

Add a `--file/-f` flag to the `geminiweb chat` command that allows users to start an interactive chat session with an initial prompt loaded from a file.

## Task Checklist

### Phase 1: Command Layer
- [x] Add `chatFileFlag` variable to chat.go
- [x] Register `--file/-f` flag in `init()`
- [x] Update command Long description with documentation
- [x] Add file reading logic in `runChat()`
  - [x] File existence check
  - [x] File size validation (max 1MB)
  - [x] Binary file detection (utf8 validation)
  - [x] Empty file detection
- [x] Add required imports (`os`, `strings`, `unicode/utf8`)
- [x] Update TUI call to pass `initialPrompt`

### Phase 2: TUI Layer
- [x] Add `initialPrompt` field to `Model` struct
- [x] Create `RunChatWithInitialPrompt()` function
- [x] Update `RunChatWithPersona()` to delegate to new function
- [x] Add `sendInitialPrompt()` method
- [x] Add `initialPromptMsg` type
- [x] Handle `initialPromptMsg` in `Update()`
- [x] Modify `Init()` to check for initial prompt

### Phase 3: Testing
- [x] Add unit tests for flag registration
- [x] Add unit tests for file reading scenarios
- [x] Add unit tests for TUI initial prompt handling
- [x] Run full test suite to verify no regressions

### Phase 4: Documentation
- [x] Update chat command help text

## Implementation Summary

### Files Modified

1. **`internal/commands/chat.go`**
   - Added `chatFileFlag` variable
   - Added `maxFileSize` constant (1MB)
   - Registered `--file/-f` flag
   - Added file reading logic with validation:
     - File existence check
     - File size limit (1MB)
     - Binary file detection via UTF-8 validation
     - Empty file detection
   - Updated TUI call to use `RunChatWithInitialPrompt`

2. **`internal/tui/model.go`**
   - Added `initialPrompt` field to `Model` struct
   - Added `initialPromptMsg` type
   - Added `sendInitialPrompt()` method
   - Modified `Init()` to trigger initial prompt if set
   - Added handler for `initialPromptMsg` in `Update()`
   - Created `RunChatWithInitialPrompt()` function
   - Updated `RunChatWithPersona()` to delegate

3. **`internal/commands/chat_test.go`**
   - Added `TestChatCommand_FileFlag` - flag registration
   - Added `TestChatCommand_FileFlag_ReadFile` - file reading
   - Added `TestChatCommand_FileFlag_EmptyFile` - empty file handling
   - Added `TestChatCommand_FileFlag_NonExistent` - missing file
   - Added `TestChatCommand_FileFlag_MaxFileSize` - size constant

4. **`internal/tui/model_test.go`**
   - Added `TestModel_InitialPrompt` - field test
   - Added `TestInitialPromptMsg` - type test
   - Added `TestSendInitialPrompt_ClearsPrompt` - prompt clearing
   - Added `TestSendInitialPrompt_ReturnsMessage` - message creation
   - Added `TestModel_Init_WithInitialPrompt` - Init behavior
   - Added `TestModel_Init_WithoutInitialPrompt` - Init behavior
   - Added `TestModel_Update_InitialPromptMsg` - Update handling

### Usage Examples

```bash
# Start chat with file as initial prompt
geminiweb chat -f prompt.md

# Combine with gem
geminiweb chat -f task.md --gem "Code Helper"

# Combine with persona
geminiweb chat -f context.md --persona coder

# Skip history selector
geminiweb chat -f prompt.txt --new
```

### Error Handling

- File not found: "file not found: <path>"
- Permission denied: "failed to access file '<path>': <error>"
- File too large: "file '<path>' is too large (max 1MB)"
- Binary file: "file '<path>' appears to be binary, not text"
- Empty file: "file '<path>' is empty"

## Notes

- No deviations from the original plan
- All tests pass
- Backward compatible - existing callers continue to work
