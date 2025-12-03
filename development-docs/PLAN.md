# Refactoring/Design Plan: TUI Enhancement - Conversation History, Input Area & UX

## 1\. Executive Summary & Goals

This plan outlines the refactoring and new feature implementation required to significantly enhance the `geminiweb` interactive TUI. The primary focus is integrating local conversation persistence, improving user input capabilities, and polishing the overall user experience.

### Key Goals:

1.  **Integrate Local Persistence:** Fully connect the `internal/history` package to the `internal/tui/model.go` for automatic real-time saving and conversation resumption.
2.  **Enhance Input Usability:** Enable multi-line input and implement command-based file attachment (VIM-style command mode).
3.  **Improve UX/Styling:** Implement URL display for images and introduce TUI color theme customization.

-----

## 2\. Current Situation Analysis

### Overview of Relevant Components

| Component | Status | Role | Required Change |
| :--- | :--- | :--- | :--- |
| `internal/tui/model.go` | Exists | Main TUI loop (`Update`, `View`) and state management. | Major refactoring needed to handle pre-chat selection, history loading, saving logic, new commands (`/file`, `/history`), and enhanced input (multi-line, command parsing). |
| `internal/api/session.go` | Exists | Manages chat context (`metadata`: `[cid, rid, rcid]`). | Needs modification to accept `UploadedFile` for `SendMessage` to support file attachment. |
| `internal/history/store.go` | Exists | Persistence layer (JSON files). | Already provides necessary public methods (`DefaultStore()`, `ListConversations()`, `GetConversation()`, `AddMessage()`, `UpdateMetadata()`). |
| `internal/commands/chat.go` | Exists | Entry point for `geminiweb chat`. | Needs refactoring to handle conversation ID passed from a TUI selection menu, instead of immediately starting a session. |
| `internal/api/generate.go` | Exists | Builds the API payload (`buildPayloadWithGem`). | Needs modification to correctly handle `UploadedFile` objects in `GenerateOptions`. |
| `internal/api/upload.go` | Exists | Contains `UploadedFile` type and `UploadFile`/`UploadText` methods. | The `UploadedFile` type should be leveraged directly. |
| `internal/render/themes.go`| Exists | Manages markdown themes. | Extend to provide TUI colors. |
| `internal/tui/styles.go`| Exists | TUI Lipgloss styles. | Refactor to load colors from a central theme provider. |

### Key Pain Points/Limitations

  * **Lack of Persistence:** Conversations are ephemeral; context is lost upon exiting the TUI.
  * **Input Area:** The single-line input (`textarea.New()`) limits complex, multi-paragraph prompts.
  * **Feature Gaps:** Missing basic interactive TUI features like history switching and file attachment.
  * **Image URLs:** Image response (URL) is currently not presented in a user-friendly manner.

-----

## 3\. Proposed Solution / Refactoring Strategy

The strategy involves a **phased approach** focused on injecting persistence, overhauling input, and applying visual improvements.

### 3.1. High-Level Design / Architectural Overview

The core architectural change is the introduction of a **Pre-Chat Selection Model** and a direct **History Persister** layer inside the main `Model`.

  * **TUI Flow Change:** `geminiweb chat` $\rightarrow$ **`HistorySelectorModel`** $\rightarrow$ `ChatModel` (loaded session or new session).
  * **History Persister:** `ChatModel.Update` will interact directly with `history.Store` after an assistant's `responseMsg` is received.
  * **Input Area Refactoring:** The existing `textarea.Model` will be configured for multi-line. New internal logic in `ChatModel.Update` will parse commands (e.g., `/file`, `/history`) *before* sending the message to the session.

### 3.2. Key Components / Modules

| Component | Responsibility | Change Summary |
| :--- | :--- | :--- |
| `internal/tui/history_selector.go` **(New)** | Handles the pre-chat menu to select *New* or *Resume* conversation. | New file, using logic similar to `gems_model.go`. |
| `internal/tui/model.go` | Manages chat state, now including: `currentConvID`, `attachments []*api.UploadedFile`. | New state/logic to save/load on `Init`/`Update`, and manage attached files. |
| `internal/api/session.go` | Handle API call context. | Modify `SendMessage(prompt string, files []*api.UploadedFile)` to incorporate files into `GenerateOptions`. |
| `internal/api/generate.go` | API payload construction. | Update `buildPayloadWithGem` signature to accept `[]*api.UploadedFile` and modify JSON payload structure when files are present. |
| `internal/tui/styles.go` | TUI Visuals. | Extract color constants into a configurable theme struct, allowing runtime updates. |

-----

### 3.3. Detailed Action Plan / Phases

#### 📦 Phase 1: Conversation Persistence (Critical Path)

**Objective(s):** Implement automatic saving and the conversation selection menu.
**Priority:** High (Foundation for long-term chat)

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| **1.1:** Modify `ChatSession` for File Support | Prepare session logic for file attachments (for Phase 2). | S | Update `session.SendMessage` to take an optional `[]*api.UploadedFile` and update `GenerateOptions` accordingly. |
| **1.2:** Create `HistorySelectorModel` | Enable user to select existing or new conversation at startup. | M | New file `internal/tui/history_selector.go` with list/select logic. |
| **1.3:** Refactor `commands/chat.go` | Launch `HistorySelectorModel` instead of immediately creating a `ChatSession`. | S | `runChat()` now calls `RunHistorySelector()`; logic moved to TUI. |
| **1.4:** Update `Model.NewChatModel` | Accept `*history.Conversation` to initialize `ChatSession` and `messages`. | M | Load conversation data (messages, metadata) into the TUI Model and session. |
| **1.5:** Implement Auto-Save in `Model.Update` | Save user message, then save assistant response *and* metadata (`CID`, `RID`, `RCID`). | L | Successful `history.Store.AddMessage` and `history.Store.UpdateMetadata` call after every `responseMsg`. |
| **1.6:** Implement `/history` Command | Allow dynamic switching between saved conversations within the chat TUI. | M | New state/sub-view in `Model.Update` to show a temporary list (using existing `gems_model` rendering logic as a base). |

-----

#### 🛠️ Phase 2: Input & Command Overhaul

**Objective(s):** Enable multi-line text input and implement file/image attachment commands.
**Priority:** Medium

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| **2.1:** Configure Multi-line Input | Improve prompt complexity. | S | In `Model.NewChatModel`, set `textarea.Newline` to `Shift+Enter` or `Ctrl+Enter`, making `Enter` the default submission key. |
| **2.2:** Command Parsing in `Model.Update` | Enable handling of `/file`, `/image`, `/history` (from Phase 1). | M | New internal `m.parseInput(input string) (command, args, cleanPrompt)` function. |
| **2.3:** Implement `/file <path>` & `/image <path>` | Allow multimodal input. | M | `Model` must track a list of `[]*api.UploadedFile`. The parsing logic calls `client.UploadFile` and stores the result in state *before* sending the message (Task 1.1 dependency). |
| **2.4:** Clear Attachments on Send | Prevent accidental re-sending of files. | S | Reset `Model.attachments` to `nil` immediately after successfully sending the message. |
| **2.5:** Update Input Area UX | Show active attached files (e.g., `[1 file attached]`) above the textarea. | S | Modify `Model.View` to render attachment count. |
| **2.6:** (Future) Implement Gem Autocomplete | Provide a smoother UX for typing `/gem <name>`. | L | Requires integrating a text-input `bubble` for suggestions/state management (out of scope for MVP). |

-----

#### 🎨 Phase 3: Style and UX Polish

**Objective(s):** Improve the display of image URLs and enable TUI color customization.
**Priority:** Medium/Low

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| **3.1:** Display Image URLs from `ModelOutput` | Provide useful links when images are returned. | M | In `Model.updateViewport`, detect image URLs in `models.ModelOutput` and append a styled list (e.g., `[Image: Title] (URL)`) below the main markdown content. |
| **3.2:** Refactor `internal/tui/styles.go` | Centralize color definitions for easy theming. | M | Define a theme struct (e.g., `TUITheme`) in a new file (e.g., `internal/render/tui_themes.go`) and initialize `internal/tui/styles.go` with the current theme. |
| **3.3:** Extend `ConfigModel` for Themes | Allow user selection of a new TUI theme. | S | Integrate the theme selection logic into `internal/tui/config_model.go`, updating the theme and saving the name to `config.json`. |
| **3.4:** Create New Themes (Catppuccin/Nord) | Provide initial theme choices for users. | S | New theme definitions in `internal/render/tui_themes.go`. |

-----

### 3.4. Data Model Changes

  * **`internal/api/session.go`:**

    ```go
    // Modified SendMessage signature (before: prompt string)
    func (s *ChatSession) SendMessage(prompt string, files []*UploadedFile) (*models.ModelOutput, error)
    ```

  * **`internal/api/generate.go`:**

    ```go
    // Modified GenerateOptions struct
    type GenerateOptions struct {
        Model    models.Model
        Metadata []string
        // Previously Images - now consolidated/unified to Files
        Files   []*UploadedFile // <-- CHANGE: Renamed field to be more generic, assuming UploadedImage is aliased to UploadedFile
        GemID    string
    }
    // Update all internal usages of GenerateOptions (including client.go, session.go)
    // Update buildPayloadWithGem signature
    func buildPayloadWithGem(prompt string, metadata []string, files []*UploadedFile, gemID string) (string, error)
    ```

  * **`internal/tui/model.go`:**

    ```go
    // New field to track attached files
    attachments []*api.UploadedFile
    // New field to track conversation ID
    currentConvID string
    ```

-----

## 4\. Key Considerations & Risk Mitigation

### 4.1. Technical Risks & Challenges

| Risk/Challenge | Impact | Mitigation Strategy |
| :--- | :--- | :--- |
| **R1:** Race conditions in `Model.Update` (Auto-Save) | Corrupted history file or lost metadata/messages. | **Mutex in `history.Store` (already exists).** Ensure `Model.Update` only attempts to save *after* `responseMsg` is fully processed and `m.session.metadata` is updated. |
| **R2:** Complex `textarea` configuration for multi-line/send | Poor UX, accidental message sending. | Use `bubbles/textarea` built-in `SetSubmitKeys` or `SetNewlineKeys` functionality. Clearly communicate key bindings in the status bar. |
| **R3:** File Upload Failure & Retries | Loss of user context, frustration. | **Implement file upload as part of the command parsing step (Task 2.3), *before* `SendMessage`.** If upload fails, alert user with error message (`errMsg`) but retain prompt and state for retry. |
| **R4:** `glamour` markdown renderer re-initialization on theme change | Performance bottleneck or memory leak. | Leverage `internal/render`'s existing `rendererPool` and `cacheKey` logic. Ensure the TUI theme selection triggers a cache flush or key update for the renderer pool. |

### 4.2. Dependencies

  * Phase 1.5 (Auto-Save) depends on a stable `history.Store` interface.
  * Phase 2.3 (`/file` command) depends on successful completion of Phase 1.1 (modifying `ChatSession.SendMessage` to accept files).
  * Phase 3.1 (Image URL Display) depends on Phase 3.2 (Style Refactor) to ensure the image link styling is consistent with the new theme logic.

### 4.3. Non-Functional Requirements (NFRs) Addressed

  * **Reliability:** Auto-save greatly increases session reliability. (Phase 1)
  * **Usability:** Multi-line input and command-based file attachment drastically improve user workflow efficiency. (Phase 2)
  * **Maintainability:** Centralizing TUI styles and abstracting theme logic makes future visual updates simpler. (Phase 3)
  * **Extensibility:** Introducing command parsing in `Model.Update` creates a robust foundation for adding future TUI commands (e.g., `/model`, `/persona`). (Phase 2)

-----

## 5\. Success Metrics / Validation Criteria

| Metric | Validation Criteria |
| :--- | :--- |
| **Persistence Integrity** | 1. A completed chat session can be resumed, accurately restoring all messages and the correct `CID`, `RID`, and `RCID`. |
| **Input Usability** | 2. Users can input multi-line prompts using `Shift+Enter` (or `Ctrl+Enter`) and submit the message using `Enter`. |
| **File Attachment** | 3. The `/file <path>` command successfully uploads a file and includes it in the *next* outgoing API request. |
| **UX Improvement** | 4. Image URLs in model output are extracted from the markdown and displayed in a visually distinct, styled format below the main response. |
| **Theming** | 5. Users can change the TUI's primary color palette via the `geminiweb config` menu, and the change is immediately reflected in the chat border/accents. |

-----

## 6\. Assumptions Made

1.  **Data Consistency:** It is assumed that the existing `internal/history/store.go` logic is robust enough to handle concurrent read/write operations (due to its internal `sync.RWMutex`).
2.  **API Compatibility:** The current understanding of the Gemini API payload structure for multi-modal input (files and text in `buildPayloadWithGem`) is correct and will not change unexpectedly.
3.  **TUI Framework Stability:** The `charmbracelet/bubbles` and `bubbletea` libraries will handle the complex state transitions (switching between chat, `/history` command list, and `/gems` command list) without major visual glitches or performance issues.

-----

## 7\. Open Questions / Areas for Further Investigation

1.  **HistorySelector Implementation:** Should the `HistorySelectorModel` be a separate `tea.Model` run before the chat, or a sub-view within the main `ChatModel`? (Decision: Separate `tea.Model` for cleaner separation, followed by refactoring `commands/chat.go`).
2.  **Error Handling for File Attachments:** How should the `ChatModel` handle *multiple* file upload failures from a single `/file` command chain? (Decision: Group all upload failures into a single descriptive `errMsg` and prevent message send, retaining the input text.)
3.  **TUI Theme Scope:** Will the TUI theme apply only to the `lipgloss` styles in `internal/tui/styles.go`, or should it also attempt to modify the `glamour` markdown theme (via `internal/render/themes.go`) to match? (Decision: TUI theme applies to `lipgloss` styles; markdown theme remains configurable but separate unless explicitly linked by a new config field.)