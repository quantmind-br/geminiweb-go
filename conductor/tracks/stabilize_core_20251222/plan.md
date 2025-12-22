# Plan: Stabilize Core Components

## Phase 1: Setup and Audit
- [x] Task: Audit current test coverage and identify specific functions with 0% coverage in `internal/commands` and `pkg/toolexec`. 094f90d
- [x] Task: Create a reusable mock for `GeminiClient` and `TUI` interfaces to be used in command tests. 3a011df

## Phase 2: Harden `pkg/toolexec`
- [ ] Task: Implement unit tests for `pkg/toolexec/security.go` (PathValidator, BlacklistValidator).
- [ ] Task: Implement unit tests for `pkg/toolexec/registry.go` (Edge cases in Register/Unregister).
- [ ] Task: Implement unit tests for `pkg/toolexec/result.go` (Error wrapping and unwrapping).
- [ ] Task: Verify and fix any race conditions in `pkg/toolexec/executor.go` concurrency tests.

## Phase 3: Refactor and Test `internal/commands`
- [ ] Task: Refactor `internal/commands/root.go` to support dependency injection for client creation.
- [ ] Task: Implement table-driven tests for `internal/commands/gems.go` (List, Create, Delete).
- [ ] Task: Implement tests for `internal/commands/chat.go` (Session initialization flags).
- [ ] Task: Implement tests for `internal/commands/autologin.go` (Browser detection logic).

## Phase 4: Error Handling Standardization
- [ ] Task: Create a centralized error printing helper in `internal/tui/styles` using `lipgloss`.
- [ ] Task: Refactor `cmd/geminiweb/main.go` to use the new error helper for top-level command errors.
- [ ] Task: Conductor - User Manual Verification 'Error Handling Standardization' (Protocol in workflow.md)
