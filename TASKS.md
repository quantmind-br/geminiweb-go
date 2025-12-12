# TASKS.md - Gems Interactive Management

## Project Briefing

**Feature:** Interactive Mode for Gem (Persona) Management in geminiweb-go CLI

**Objectives:**
1. Enable quick chat startup with selected Gem from TUI list (key `c`)
2. Improve UX with interactive list, real-time search, and detail viewing
3. Encapsulate selection logic for reuse between `gems list` and `/gems` command in chat

**Status:** Phases 1-3 completed. Phase 4 is optional backlog.

**Reference:** See `PLAN.md` for detailed design documentation.

---

## Current Sprint: Code Cleanup

### Pre-requisites
- [x] Interactive Gems selection in `gems list` (Phase 1)
- [x] `/gems` and `Ctrl+G` integration in chat TUI (Phase 2)
- [x] Real-time search/filter in Gems selector (Phase 3)

### Task 1: Errors Package Refactoring
**Description:** Simplify field access in error types by leveraging Go's embedded struct field promotion.

**Files:**
- `internal/errors/errors.go`
- `internal/errors/errors_test.go`

**Changes:**
- Replace `e.GeminiError.HTTPStatus` with `e.HTTPStatus`
- Replace `e.GeminiError.Code` with `e.Code`
- Replace `e.GeminiError.Cause` with `e.Cause`
- Replace `e.GeminiError.Message` with `e.Message`
- Replace `e.GeminiError.Endpoint` with `e.Endpoint`
- Replace `e.GeminiError.Body` with `e.Body`
- Replace `e.GeminiError.Operation` with `e.Operation`

| Subtask | Status |
| :--- | :--- |
| 1.1 Update `APIError` methods | [x] Completed |
| 1.2 Update `AuthError` methods | [x] Completed |
| 1.3 Update `NetworkError` methods | [x] Completed |
| 1.4 Update `TimeoutError` methods | [x] Completed |
| 1.5 Update `UsageLimitError` methods | [x] Completed |
| 1.6 Update `BlockedError` methods | [x] Completed |
| 1.7 Update `ParseError` methods | [x] Completed |
| 1.8 Update `PromptTooLongError` methods | [x] Completed |
| 1.9 Update `UploadError` methods | [x] Completed |
| 1.10 Update `DownloadError` methods | [x] Completed |
| 1.11 Update `GemError` methods | [x] Completed |
| 1.12 Update helper functions | [x] Completed |
| 1.13 Update tests | [x] Completed |

### Task 2: Test Linting Fixes
**Description:** Fix linting warnings for ignored return values in test files.

**Files:**
- `internal/commands/history_test.go`
- `internal/commands/import_test.go`
- `internal/commands/persona_test.go`
- `internal/commands/query_test.go`
- `internal/commands/root_test.go`

**Changes:**
- Replace `os.Setenv("HOME", tmpDir)` with `_ = os.Setenv("HOME", tmpDir)`
- Replace `defer os.Setenv("HOME", oldHome)` with `defer func() { _ = os.Setenv("HOME", oldHome) }()`

| Subtask | Status |
| :--- | :--- |
| 2.1 Update history_test.go | [x] Completed |
| 2.2 Update import_test.go | [x] Completed |
| 2.3 Update persona_test.go | [x] Completed |
| 2.4 Update query_test.go | [x] Completed |
| 2.5 Update root_test.go | [x] Completed |

### Task 3: Validation & Testing
| Subtask | Status |
| :--- | :--- |
| 3.1 Run `make test` - all tests pass | [x] Completed |
| 3.2 Run `make lint` - no warnings | [x] Completed |
| 3.3 Run `make build` - successful | [x] Completed |

### Task 4: Commit Changes
| Subtask | Status |
| :--- | :--- |
| 4.1 Stage changes | [~] In Progress |
| 4.2 Create commit with conventional message | [ ] Pending |
| 4.3 Push to remote | [ ] Pending |

---

## Phase 4 Backlog (Optional)

These items are enhancements that can be implemented after the current sprint.

| Task | Description | Effort | Status |
| :--- | :--- | :--- | :--- |
| 4.1 "No Gem" option | Add `<none>` item to chat selector to clear active persona | S | Backlog |
| 4.2 Auto-refresh for `gems list` chat | Enable `autoRefresh=true` when starting chat from gem listing | S | Backlog |
| 4.3 Hidden gems in chat selector | Add config/flag to include hidden gems in `/gems` selector | S | Backlog |
| 4.4 TUI filtering tests | Add tests for filter/transition logic in `internal/tui` | M | Backlog |

---

## Notes

- All test files currently pass (`make test` successful)
- Changes in errors package use Go's embedded struct field promotion for cleaner code
- No functional changes - pure refactoring for code quality
