# Product Guidelines

## Architectural Style
**Modular Monolith**: The project should be structured as a modular monolith. Functionality is divided into distinct packages within the `internal/` directory (e.g., `api`, `browser`, `tui`) with clear boundaries. This keeps the codebase unified in a single Go module and binary while maintaining separation of concerns.

## Error Handling & Logging
**Verbose & Interactive**: As a CLI tool, the user experience during failure is critical.
*   **User-Facing**: Errors presented to the user should be formatted, colorful, and actionable (using `lipgloss` styles). Avoid dumping raw stack traces to the user unless in debug mode.
*   **Debug Logs**: Detailed logs can be written to files or displayed in a debug view, but the primary console output should remain clean and helpful.

## Dependency Management
**Pragmatic**: We leverage the Go ecosystem's best tools to move fast.
*   **Adopt**: Well-maintained, industry-standard libraries like `cobra` for CLI commands, `bubbletea` for TUI, and `gjson` for complex JSON parsing.
*   **Avoid**: Unnecessary small dependencies that can be trivially implemented with the standard library.

## Testing Strategy
**Hybrid Approach**:
*   **Unit Tests**: Heavily test core logic that is deterministic, such as payload construction, JSON parsing (`gjson` paths), and internal state management (history, configuration).
*   **Integration/Manual**: Since the upstream API is undocumented and ephemeral, strictly mocked unit tests can become brittle. We rely on manual verification or optional integration tests for the actual API communication, while ensuring the *handling* of those responses is unit tested.
