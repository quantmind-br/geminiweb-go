# Plan: Stabilize Core Components

## Phase 1: Setup and Audit
- [x] Task: Audit current test coverage and identify specific functions with 0% coverage in `internal/commands` and `pkg/toolexec`. 094f90d
- [x] Task: Create a reusable mock for `GeminiClient` and `TUI` interfaces to be used in command tests. 3a011df

## Phase 2: Harden `pkg/toolexec`
- [x] Task: Implement unit tests for `pkg/toolexec/security.go` (PathValidator, BlacklistValidator). f3f9076
- [x] Task: Implement unit tests for `pkg/toolexec/registry.go` (Edge cases in Register/Unregister). 36ad02b
- [x] Task: Implement unit tests for `pkg/toolexec/result.go` (Error wrapping and unwrapping). a5bbdc1
- [x] Task: Verify and fix any race conditions in `pkg/toolexec/executor.go` concurrency tests. b595abb
## Phase 3: Refactor and Test `internal/commands`

- [x] Task: Refactor `internal/commands/root.go` to support dependency injection for client creation. 696bbd3
- [x] Task: Implement table-driven tests for `internal/commands/gems.go` (List, Create, Delete). 1592a7c
- [x] Task: Implement tests for `internal/commands/chat.go` (Session initialization flags). 2afd5c1
- [x] Task: Implement tests for `internal/commands/autologin.go` (Browser detection logic). 0aea272
## Phase 4: Error Handling Standardization

- [x] Task: Create a centralized error printing helper in `internal/tui/styles` using `lipgloss`. f0b9d7a
- [x] Task: Refactor `cmd/geminiweb/main.go` to use the new error helper for top-level command errors. f4a97ec
- [ ] Task: Conductor - User Manual Verification 'Error Handling Standardization' (Protocol in workflow.md)
