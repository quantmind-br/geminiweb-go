# Specification: Stabilize Core Components

## 1. Goal
The primary goal of this track is to improve the stability and maintainability of the `geminiweb-go` codebase by increasing unit test coverage in critical low-coverage areas (`internal/commands`, `pkg/toolexec`) and standardizing user-facing error reporting to match the new "Interactive Richness" design philosophy.

## 2. Scope
### In Scope
*   **Coverage Improvement**:
    *   `pkg/toolexec`: Target >80% coverage. Focus on `security.go`, `registry.go`, and error mapping in `result.go`.
    *   `internal/commands`: Target >80% coverage. Focus on `chat.go`, `gems.go`, and `root.go`.
*   **Refactoring**:
    *   Refactor CLI commands to use dependency injection for easier mocking of `GeminiClient` and `TUI` components.
*   **Error Handling**:
    *   Audit current error outputs in the CLI.
    *   Implement consistent `lipgloss`-styled error rendering for common failures (Auth, Network, Rate Limit).

### Out of Scope
*   New feature development.
*   Changes to the core `internal/api` logic (unless required for testing).
*   UI redesign beyond error messages.

## 3. Detailed Requirements

### 3.1 `pkg/toolexec` Hardening
*   **Security Policy Tests**: Add comprehensive test cases for `PathValidator` and `BlacklistValidator` to ensure no escapes are possible.
*   **Concurrency Tests**: Verify `ExecuteMany` and `ExecuteAsync` behavior under load.
*   **Mocking**: Ensure the `Executor` interface is cleanly mocked in consuming packages.

### 3.2 `internal/commands` Refactoring
*   **Dependency Injection**: Move `GeminiClient` initialization out of `init()` or global state and into a factory or struct that can be mocked.
*   **Table Driven Tests**: Use table-driven tests for CLI flag parsing and command execution flows.

### 3.3 Error Standardization
*   Create a shared `styles` helper for error messages.
*   Ensure all `cobra` command `RunE` returns are intercepted and formatted before exit.

## 4. Success Criteria
*   `go test -cover ./...` shows >80% coverage for target packages.
*   All existing tests pass.
*   Manual verification confirms error messages are styled correctly.
