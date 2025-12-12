# TASKS: TUI Export Command Implementation

## Overview

Implement a `/export` command in the TUI to export conversations to `.md` and `.json` formats.

**Reference:** `PLAN.md` - Detailed analysis and design decisions.

---

## Phase 1: Infrastructure

### 1.1 Add `ExportToJSON` to `FullHistoryStore` interface
- **File:** `internal/tui/model.go:70-80`
- **Action:** Add `ExportToJSON(id string) ([]byte, error)` to the interface
- **Notes:** The method already exists on `*history.Store` (in `internal/history/export.go:106`)

### 1.2 Create `exportResultMsg` struct
- **File:** `internal/tui/model.go` (near other message types)
- **Fields:**
  ```go
  type exportResultMsg struct {
      path      string  // Absolute path of exported file
      format    string  // "markdown" or "json"
      size      int64   // File size in bytes
      overwrite bool    // If file was overwritten
      err       error   // Error, if any
  }
  ```

### 1.3 Create `parseExportArgs` function
- **File:** `internal/tui/model.go`
- **Signature:** `func parseExportArgs(args string) (path, format string, err error)`
- **Behavior:**
  - `/export chat.md` -> path="chat.md", format="markdown"
  - `/export chat.json` -> path="chat.json", format="json"
  - `/export chat` -> path="chat.md", format="markdown" (default)
  - `/export chat -f json` -> path="chat.json", format="json"
  - `/export -f json` -> error (need path)
- **Format inference:** `.json` -> json, everything else -> markdown

### 1.4 Create `validateExportPath` function
- **File:** `internal/tui/model.go`
- **Signature:** `func validateExportPath(path string) (string, error)`
- **Behavior:**
  1. Expand `~` to home directory
  2. Convert to absolute path
  3. Verify parent directory exists
  4. Return clean absolute path

### 1.5 Create `sanitizeFilename` function
- **File:** `internal/tui/model.go`
- **Signature:** `func sanitizeFilename(title string) string`
- **Behavior:**
  1. Replace invalid chars (`/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`) with `_`
  2. Truncate to 200 characters
  3. Return safe filename (without extension)

---

## Phase 2: Core Implementation

### 2.1 Create `exportCommand` function (async I/O)
- **File:** `internal/tui/model.go`
- **Signature:** `func exportCommand(store FullHistoryStore, convID, format, path string) tea.Cmd`
- **Behavior:**
  1. Check if file exists (for overwrite flag)
  2. Call `store.ExportToMarkdown` or `store.ExportToJSON`
  3. Write to file
  4. Return `exportResultMsg`

### 2.2 Create `exportFromMemory` function
- **File:** `internal/tui/model.go`
- **Signature:** `func exportFromMemory(messages []chatMessage, format, path string) tea.Cmd`
- **Behavior:** Export `m.messages` directly when conversation not persisted
- **Notes:** For unsaved conversations (no ID yet)

### 2.3 Create `handleExportCommand` method
- **File:** `internal/tui/model.go`
- **Signature:** `func (m Model) handleExportCommand(args string) (tea.Model, tea.Cmd)`
- **Behavior:**
  1. Parse args with `parseExportArgs`
  2. If no path given, use sanitized conversation title
  3. Validate path with `validateExportPath`
  4. Check for conversation:
     - `m.conversation.ID != ""` -> use store
     - `len(m.messages) > 0` -> use memory export
     - else -> error "no conversation to export"
  5. Return appropriate tea.Cmd

### 2.4 Add `exportResultMsg` handler in `Update`
- **File:** `internal/tui/model.go` (Update function)
- **Behavior:**
  ```go
  case exportResultMsg:
      if msg.err != nil {
          m.err = msg.err
      } else {
          feedback := fmt.Sprintf("✓ Exported to %s", msg.path)
          if msg.overwrite {
              feedback += " (overwritten)"
          }
          m.err = fmt.Errorf(feedback)  // Using err for feedback (existing pattern)
      }
      return m, nil
  ```

### 2.5 Register `/export` command in switch
- **File:** `internal/tui/model.go` (in command handling switch)
- **Add:**
  ```go
  case "export":
      return m.handleExportCommand(parsed.Args)
  ```

---

## Phase 3: Robustness & UX

### 3.1 Add export concurrency protection
- **File:** `internal/tui/model.go`
- **Action:** Use `m.loading` flag or add `m.exporting` flag
- **Notes:** Prevent double exports

### 3.2 Update help text in status bar
- **File:** `internal/tui/model.go` (View function or wherever status bar is rendered)
- **Action:** Add `/export` to the list of available commands

### 3.3 Add default filename generation
- **File:** `internal/tui/model.go` (in `handleExportCommand`)
- **Behavior:** When no path given:
  - If conversation has title: use `sanitizeFilename(title) + extension`
  - Else: use `conversation_TIMESTAMP.extension`

---

## Phase 4: Tests

### 4.1 Test `parseExportArgs`
- **File:** `internal/tui/model_test.go` (new or existing)
- **Cases:**
  - Valid path with `.md` extension
  - Valid path with `.json` extension
  - Valid path without extension (defaults to markdown)
  - Path with `-f json` flag
  - Empty args (error)

### 4.2 Test `validateExportPath`
- **File:** `internal/tui/model_test.go`
- **Cases:**
  - Expand `~` to home
  - Convert relative to absolute
  - Parent directory exists
  - Parent directory doesn't exist (error)

### 4.3 Test `sanitizeFilename`
- **File:** `internal/tui/model_test.go`
- **Cases:**
  - Normal title -> unchanged
  - Title with invalid chars -> replaced with `_`
  - Very long title -> truncated

### 4.4 Test `handleExportCommand` integration
- **File:** `internal/tui/model_test.go`
- **Cases:**
  - Export with conversation ID (uses store)
  - Export without conversation ID but with messages (uses memory)
  - Export with no messages (error)
  - Export with invalid path (error)

---

## Technical Decisions (from PLAN.md)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| File exists | Overwrite with feedback | Simple, avoids file proliferation |
| Default directory | CWD | Unix convention |
| Unsaved conversation | Export from memory | Better UX |
| Default format | Markdown | More readable |
| `-f` vs extension | Flag takes priority | Explicit over implicit |

---

## Checklist

- [x] Phase 1: Infrastructure
  - [x] 1.1 Add ExportToJSON to interface
  - [x] 1.2 Create exportResultMsg
  - [x] 1.3 Create parseExportArgs
  - [x] 1.4 Create validateExportPath
  - [x] 1.5 Create sanitizeFilename
- [x] Phase 2: Core Implementation
  - [x] 2.1 Create exportCommand
  - [x] 2.2 Create exportFromMemory
  - [x] 2.3 Create handleExportCommand
  - [x] 2.4 Add exportResultMsg handler
  - [x] 2.5 Register /export command
- [x] Phase 3: Robustness & UX
  - [x] 3.1 Concurrency protection
  - [x] 3.2 Update help text
  - [x] 3.3 Default filename generation
- [x] Phase 4: Tests
  - [x] 4.1 Test parseExportArgs
  - [x] 4.2 Test validateExportPath
  - [x] 4.3 Test sanitizeFilename
  - [x] 4.4 Integration tests
