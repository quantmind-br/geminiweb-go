# Technology Stack

## Core Language
*   **Go (Golang)**: Version 1.24.1 or higher. Chosen for its performance, concurrency support, and static binary compilation.

## CLI & User Interface
*   **Cobra**: The standard library for building modern Go CLI interactions. Used for command routing (`geminiweb chat`, `geminiweb gems`, etc.) and flag parsing.
*   **Bubble Tea**: A functional Elm Architecture (MVU) framework for Go terminal apps. Powers the interactive TUI.
*   **Lip Gloss**: Style definitions for the terminal. Used to create the "desktop-like" look and feel.

## Networking & Security
*   **tls-client (`github.com/bogdanfinn/tls-client`)**: A specialized HTTP client that mimics the TLS fingerprints (JA3/JA4) of real browsers (Chrome, Firefox). This is **critical** for avoiding bot detection by Google.
*   **Standard `net/http`**: Used only where fingerprinting is not required.

## Data & Persistence
*   **gjson (`github.com/tidwall/gjson`)**: A high-performance JSON parser. Essential for extracting data from Google's deeply nested, often unstructured RPC response arrays.
*   **File System**: No external database is used. Chat history and configuration are stored as local JSON/Markdown files in the user's config directory (e.g., `~/.config/geminiweb/`).

## Tool Execution
*   **Internal Framework**: A custom, modular tool execution engine (`pkg/toolexec`) designed for security and extensibility, minimizing external dependencies for this specific logic.
