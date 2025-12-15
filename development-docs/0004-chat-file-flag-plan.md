# Implementation Plan: Chat Initial Prompt from File

## Overview

Add a `--file/-f` flag to the `geminiweb chat` command that allows users to start an interactive chat session with an initial prompt loaded from a file. The prompt is sent as the first message, and the chat continues interactively.

---

## Motivation

### Current Behavior
- `geminiweb -f prompt.md` - Sends a single query (non-interactive) and exits
- `geminiweb chat` - Starts interactive chat with empty input

### Desired Behavior
- `geminiweb chat -f context.md` - Starts interactive chat, sends file content as first message, continues interactively
- Combines the convenience of file-based prompts with persistent interactive sessions

### Use Cases
1. **Context Loading**: Start chat with project context/documentation
2. **Task Continuation**: Resume complex tasks defined in files
3. **Code Review**: Load code files as initial context for review sessions
4. **Template Prompts**: Use predefined prompts stored in files

---

## Technical Analysis

### Files to Modify

| File | Changes |
|------|---------|
| `internal/commands/chat.go` | Add flag, file reading, pass to TUI |
| `internal/tui/model.go` | Accept initial prompt, auto-send on start |

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        chat.go (command)                         │
│  1. Parse --file flag                                           │
│  2. Read file content (if specified)                            │
│  3. Validate content                                            │
│  4. Pass initialPrompt to RunChatWithPersona()                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        model.go (TUI)                           │
│  1. Receive initialPrompt in constructor                        │
│  2. Store in Model struct                                       │
│  3. On Init(), if initialPrompt set:                            │
│     - Add user message to messages                              │
│     - Set loading state                                         │
│     - Return sendMessage command                                │
│  4. Process response normally                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Detailed Implementation

### Phase 1: Command Layer (`internal/commands/chat.go`)

#### 1.1 Add Flag Variable

```go
// Location: After line 23 (chatPersonaFlag)
// chatFileFlag is the --file flag for providing initial prompt from file
var chatFileFlag string
```

#### 1.2 Register Flag

```go
// Location: In init() function, after line 62
chatCmd.Flags().StringVarP(&chatFileFlag, "file", "f", "", "Read initial prompt from file")
```

#### 1.3 Update Command Documentation

```go
// Location: Update Long description (lines 28-52)
// Add new section:

INITIAL PROMPT FROM FILE:
  Use --file to start the chat with content from a file:
    geminiweb chat --file context.md
    geminiweb chat -f prompt.txt --new

  The file content is sent as the first message, and the chat
  continues interactively. Combine with other flags:
    geminiweb chat -f task.md --gem "Code Helper"
    geminiweb chat -f context.md --persona coder
```

#### 1.4 Modify runChat() Function

```go
// Location: In runChat(), after line 65 (before model resolution)

// Read initial prompt from file if specified
var initialPrompt string
if chatFileFlag != "" {
    data, err := os.ReadFile(chatFileFlag)
    if err != nil {
        return fmt.Errorf("failed to read file '%s': %w", chatFileFlag, err)
    }
    initialPrompt = strings.TrimSpace(string(data))
    if initialPrompt == "" {
        return fmt.Errorf("file '%s' is empty", chatFileFlag)
    }
}
```

#### 1.5 Update TUI Call

```go
// Location: Line 165 - Replace existing call
// Current:
return tui.RunChatWithPersona(client, session, modelName, selectedConv, store, resolvedGem.Name, persona)

// New:
return tui.RunChatWithInitialPrompt(client, session, modelName, selectedConv, store, resolvedGem.Name, persona, initialPrompt)
```

#### 1.6 Add Required Import

```go
// Location: imports section
import (
    // ... existing imports ...
    "os"
    "strings"
)
```

---

### Phase 2: TUI Layer (`internal/tui/model.go`)

#### 2.1 Add Field to Model Struct

```go
// Location: In Model struct (around line 101-157)
// Add after line 152 (persona field):

// Initial prompt to send automatically on start
initialPrompt string
```

#### 2.2 Create New Public Function

```go
// Location: After RunChatWithPersona (around line 1435)

// RunChatWithInitialPrompt starts the chat TUI with all options including an initial prompt
// If initialPrompt is non-empty, it will be sent automatically when the TUI starts
func RunChatWithInitialPrompt(
    client api.GeminiClientInterface,
    session ChatSessionInterface,
    modelName string,
    conv *history.Conversation,
    store HistoryStoreInterface,
    gemName string,
    persona *config.Persona,
    initialPrompt string,
) error {
    m := NewChatModelWithConversation(client, session, modelName, conv, store)
    m.activeGemName = gemName
    m.persona = persona
    m.initialPrompt = initialPrompt

    p := tea.NewProgram(
        m,
        tea.WithAltScreen(),
    )

    _, err := p.Run()
    return err
}
```

#### 2.3 Modify Init() Function

```go
// Location: Init() function (lines 211-216)
// Current:
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        textarea.Blink,
        m.spinner.Tick,
    )
}

// New:
func (m Model) Init() tea.Cmd {
    cmds := []tea.Cmd{
        textarea.Blink,
        m.spinner.Tick,
    }

    // If there's an initial prompt, send it automatically
    if m.initialPrompt != "" {
        cmds = append(cmds, m.sendInitialPrompt())
    }

    return tea.Batch(cmds...)
}
```

#### 2.4 Add sendInitialPrompt Method

```go
// Location: After sendMessageWithAttachments (around line 806)

// sendInitialPrompt creates a command to send the initial prompt from file
// This is called automatically on Init() when initialPrompt is set
func (m *Model) sendInitialPrompt() tea.Cmd {
    prompt := m.initialPrompt
    m.initialPrompt = "" // Clear to prevent re-sending

    return func() tea.Msg {
        // Return a message that triggers the send flow
        return initialPromptMsg{prompt: prompt}
    }
}

// initialPromptMsg is sent when an initial prompt needs to be processed
type initialPromptMsg struct {
    prompt string
}
```

#### 2.5 Handle initialPromptMsg in Update()

```go
// Location: In Update() switch statement (around line 245)
// Add new case after existing message types:

case initialPromptMsg:
    // Process initial prompt as if user typed it
    prompt := msg.prompt

    // Apply persona system prompt if set
    finalPrompt := prompt
    if m.persona != nil && m.persona.SystemPrompt != "" {
        finalPrompt = config.FormatSystemPrompt(m.persona, prompt)
    }

    // Add user message to chat
    m.messages = append(m.messages, chatMessage{
        role:    "user",
        content: prompt, // Show original prompt, not with system prompt
    })

    // Save to history if available
    if m.historyStore != nil && m.conversation != nil {
        _ = m.historyStore.AddMessage(m.conversation.ID, "user", prompt, "")
    }

    // Set loading state and send message
    m.loading = true
    m.updateViewport()
    return m, tea.Batch(
        m.sendMessage(finalPrompt),
        animationTick(),
    )
```

---

### Phase 3: Maintain Backward Compatibility

#### 3.1 Update RunChatWithPersona to Call New Function

```go
// Location: RunChatWithPersona function (lines 1423-1435)
// Modify to delegate to new function:

func RunChatWithPersona(
    client api.GeminiClientInterface,
    session ChatSessionInterface,
    modelName string,
    conv *history.Conversation,
    store HistoryStoreInterface,
    gemName string,
    persona *config.Persona,
) error {
    return RunChatWithInitialPrompt(client, session, modelName, conv, store, gemName, persona, "")
}
```

This ensures all existing callers continue to work without modification.

---

## Testing Strategy

### Unit Tests

#### 3.1 `internal/commands/chat_test.go`

```go
func TestChatCommand_FileFlag(t *testing.T) {
    // Verify flag is registered
    flag := chatCmd.Flags().Lookup("file")
    if flag == nil {
        t.Error("file flag not found")
    }
    if flag.Shorthand != "f" {
        t.Errorf("Expected shorthand 'f', got '%s'", flag.Shorthand)
    }
}

func TestChatCommand_FileFlag_ReadFile(t *testing.T) {
    // Create temp file
    tmpFile, err := os.CreateTemp("", "test_prompt_*.md")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpFile.Name())

    content := "Test prompt content\nWith multiple lines"
    if _, err := tmpFile.WriteString(content); err != nil {
        t.Fatal(err)
    }
    tmpFile.Close()

    // Test reading
    data, err := os.ReadFile(tmpFile.Name())
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }

    if strings.TrimSpace(string(data)) != content {
        t.Errorf("Content mismatch")
    }
}

func TestChatCommand_FileFlag_EmptyFile(t *testing.T) {
    tmpFile, err := os.CreateTemp("", "test_empty_*.md")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpFile.Name())
    tmpFile.Close()

    data, _ := os.ReadFile(tmpFile.Name())
    if strings.TrimSpace(string(data)) != "" {
        t.Error("Expected empty content")
    }
}

func TestChatCommand_FileFlag_NonExistent(t *testing.T) {
    _, err := os.ReadFile("/nonexistent/path/file.md")
    if err == nil {
        t.Error("Expected error for non-existent file")
    }
}
```

#### 3.2 `internal/tui/model_test.go`

```go
func TestModel_InitialPrompt(t *testing.T) {
    // Create model with initial prompt
    m := Model{
        initialPrompt: "Test initial prompt",
        messages:      []chatMessage{},
    }

    if m.initialPrompt == "" {
        t.Error("initialPrompt should be set")
    }
}

func TestInitialPromptMsg(t *testing.T) {
    msg := initialPromptMsg{prompt: "test prompt"}
    if msg.prompt != "test prompt" {
        t.Error("prompt mismatch")
    }
}

func TestSendInitialPrompt_ClearsPrompt(t *testing.T) {
    m := &Model{
        initialPrompt: "test",
    }

    _ = m.sendInitialPrompt()

    // After calling sendInitialPrompt, the field should be cleared
    if m.initialPrompt != "" {
        t.Error("initialPrompt should be cleared after sendInitialPrompt")
    }
}
```

### Integration Tests (Manual)

```bash
# Test 1: Basic file prompt
echo "Hello, tell me about Go programming" > /tmp/test.md
geminiweb chat -f /tmp/test.md --new

# Test 2: File with gem
echo "Review this code for bugs" > /tmp/review.md
geminiweb chat -f /tmp/review.md --gem "Code Helper"

# Test 3: File with persona
echo "Write a short story" > /tmp/story.md
geminiweb chat -f /tmp/story.md --persona writer

# Test 4: Large file (multi-page prompt)
cat large_context.md | wc -c  # Should handle large files
geminiweb chat -f large_context.md

# Test 5: Error cases
geminiweb chat -f /nonexistent/file.md  # Should show clear error
touch /tmp/empty.md && geminiweb chat -f /tmp/empty.md  # Should error on empty
```

---

## Edge Cases and Error Handling

### 4.1 File Not Found
```go
if chatFileFlag != "" {
    data, err := os.ReadFile(chatFileFlag)
    if err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("file not found: %s", chatFileFlag)
        }
        return fmt.Errorf("failed to read file '%s': %w", chatFileFlag, err)
    }
    // ...
}
```

### 4.2 Empty File
```go
initialPrompt = strings.TrimSpace(string(data))
if initialPrompt == "" {
    return fmt.Errorf("file '%s' is empty", chatFileFlag)
}
```

### 4.3 Permission Denied
```go
// os.ReadFile already returns permission errors in err
// The error message will include "permission denied"
```

### 4.4 Binary Files
```go
// Binary files will be read as-is
// Gemini will likely fail to process them meaningfully
// Consider adding validation:
if !utf8.Valid(data) {
    return fmt.Errorf("file '%s' appears to be binary, not text", chatFileFlag)
}
```

### 4.5 Very Large Files
```go
// Consider adding size limit
const maxFileSize = 1 * 1024 * 1024 // 1MB
info, _ := os.Stat(chatFileFlag)
if info.Size() > maxFileSize {
    return fmt.Errorf("file '%s' is too large (max 1MB)", chatFileFlag)
}
```

---

## Documentation Updates

### 5.1 Command Help (`chat.go` Long description)

Add section:
```
INITIAL PROMPT FROM FILE:
  Use --file to start the chat with content from a file:
    geminiweb chat --file context.md
    geminiweb chat -f prompt.txt --new

  The file content is sent as the first message, and the chat
  continues interactively. Useful for:
    - Loading project context
    - Starting with predefined prompts
    - Code review sessions

  Combine with other flags:
    geminiweb chat -f task.md --gem "Code Helper"
    geminiweb chat -f context.md --persona coder
```

### 5.2 Root Command Examples (`root.go`)

Add example:
```
  geminiweb chat -f prompt.md          Start chat with file as initial prompt
```

### 5.3 README.md (if exists)

Add usage example in chat section.

---

## Implementation Checklist

### Phase 1: Command Layer
- [ ] Add `chatFileFlag` variable
- [ ] Register flag in `init()`
- [ ] Update command Long description
- [ ] Add file reading logic in `runChat()`
- [ ] Add validation (empty file, non-existent)
- [ ] Add required imports (`os`, `strings`)
- [ ] Update TUI call to pass `initialPrompt`

### Phase 2: TUI Layer
- [ ] Add `initialPrompt` field to `Model` struct
- [ ] Create `RunChatWithInitialPrompt()` function
- [ ] Update `RunChatWithPersona()` to delegate
- [ ] Modify `Init()` to check for initial prompt
- [ ] Add `sendInitialPrompt()` method
- [ ] Add `initialPromptMsg` type
- [ ] Handle `initialPromptMsg` in `Update()`
- [ ] Ensure history recording works

### Phase 3: Testing
- [ ] Add unit tests for flag registration
- [ ] Add unit tests for file reading
- [ ] Add unit tests for error cases
- [ ] Add unit tests for TUI initial prompt handling
- [ ] Manual integration testing

### Phase 4: Documentation
- [ ] Update chat command help text
- [ ] Update root command examples

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking existing chat flow | Low | High | Backward-compatible API, extensive testing |
| Memory issues with large files | Low | Medium | Add file size limit (1MB) |
| Race condition on Init | Low | Medium | Clear initialPrompt before returning cmd |
| History not recording | Medium | Medium | Test history integration |
| Persona not applied | Medium | Low | Test with persona flag |

---

## Success Criteria

1. `geminiweb chat -f file.md` sends file content as first message
2. Chat continues interactively after initial response
3. All existing flags (`--gem`, `--persona`, `--new`) work with `--file`
4. History records the initial message correctly
5. Error messages are clear and helpful
6. No regression in existing chat functionality
7. Tests pass with >80% coverage for new code
