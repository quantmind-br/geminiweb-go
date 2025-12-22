# Code Structure Analysis

## Architectural Overview
The `geminiweb-go` project is a CLI/TUI application designed to interface with the Google Gemini Web API. It follows a layered architectural pattern, primarily organized as a **modular monolith** with a clear separation between user interfaces, service logic, and domain models.

The system is structured into four main layers:
1.  **Presentation Layer (`internal/tui`, `internal/commands`)**: Uses Cobra for command-line argument parsing and Bubble Tea (Model-View-Update pattern) for an interactive terminal interface.
2.  **Service Layer (`internal/api`, `internal/history`, `pkg/toolexec`)**: Orchestrates business logic, including API communication, conversation persistence, and extensible tool execution.
3.  **Domain/Infrastructure Layer (`internal/models`, `internal/browser`, `internal/render`)**: Defines core data structures and provides utility services like browser cookie extraction and terminal rendering.
4.  **Application Entry (`cmd/geminiweb`, `cmd/debug`)**: Bootstraps the application, wires dependencies, and executes the selected interface.

## Core Components

| Component | Responsibility | Location |
| :--- | :--- | :--- |
| **Gemini Client** | Manages HTTP/RPC communication with Gemini Web, handles auth tokens, and facilitates content generation and file uploads. | `internal/api/` |
| **TUI Model** | Manages the interactive state, keyboard input, and UI rendering lifecycle for chat sessions. | `internal/tui/` |
| **Tool Execution** | A modular framework for executing external tools with security, middleware, and confirmation support. | `pkg/toolexec/` |
| **History Store** | Persists chat messages and conversation metadata locally using JSON files. | `internal/history/` |
| **Browser Bridge** | Extracts and decrypts session cookies from desktop browsers (Chrome, Firefox, etc.) to maintain authentication. | `internal/browser/` |
| **Render Engine** | Formats AI responses for terminal display using Markdown support and configurable themes. | `internal/render/` |

## Service Definitions

-   **`GeminiClient`**: The primary service for interacting with the Gemini Web API. It handles cookie rotation, "Gems" management (server-side personas), and content generation.
-   **`ChatSession`**: A stateful service that manages the context of an ongoing conversation, including conversation IDs (`CID`) and response IDs (`RID`).
-   **`History Store`**: Provides a high-level API for listing, retrieving, and searching saved conversations.
-   **`Tool Executor`**: A service in the `toolexec` package that looks up registered tools and executes them synchronously or asynchronously, applying security policies and middleware.

## Interface Contracts

The codebase utilizes interfaces to maintain loose coupling and facilitate testing:

-   **`GeminiClientInterface` (`internal/api/client.go`)**: Defines the full set of operations available via the Gemini Web API (GenerateContent, UploadImage, FetchGems, etc.).
-   **`Executor` (`pkg/toolexec/executor.go`)**: Defines the contract for tool execution (`Execute`, `ExecuteAsync`, `ExecuteMany`).
-   **`Tool` (`pkg/toolexec/tool.go`)**: The contract for any external function or capability that can be registered with the tool framework.
-   **`FullHistoryStore` (`internal/tui/model.go`)**: An extensive interface for managing the conversation lifecycle (List, Create, Delete, Favorite).
-   **`BrowserCookieExtractor` (`internal/api/client.go`)**: Abstracts the logic for stealing cookies from local browser profiles.

## Design Patterns Identified

-   **Model-View-Update (MVU)**: The core pattern for the TUI (via `charmbracelet/bubbletea`), ensuring predictable state transitions.
-   **Command Pattern**: Implemented via `spf13/cobra` for the CLI entry point and subcommands.
-   **Middleware Pattern**: Used in `pkg/toolexec` to wrap tool execution with logging, security validation, and panic recovery.
-   **Functional Options**: Used in `api.NewClient` and `toolexec.NewExecutor` to provide flexible, type-safe configuration.
-   **Registry Pattern**: Employed by the `toolexec` package to allow dynamic tool registration and discovery.
-   **Strategy Pattern**: Used for browser-specific cookie extraction logic.

## Component Relationships

-   **Application Flow**: `cmd/geminiweb` initializes the `GeminiClient`, which may use the `browser` package to refresh credentials. It then initializes the `tui.Model` or a CLI command.
-   **TUI & Services**: The `tui.Model` holds references to a `ChatSessionInterface` (for API calls) and a `FullHistoryStore` (for persistence).
-   **Tool Integration**: The `toolexec` package acts as a standalone engine that the `api` or `tui` layers can invoke to process LLM-driven actions (extensions).
-   **Model Dependency**: Almost all packages depend on `internal/models` for shared data structures like `ModelOutput`, `Message`, and `Gem`.

## Key Methods & Functions

-   **`api.NewClient(cookies, ...options)`**: Factory function for the main API client.
-   **`api.GeminiClient.GenerateContent(prompt, opts)`**: Core method for sending prompts and receiving AI responses.
-   **`toolexec.Executor.Execute(ctx, toolName, input)`**: Runs a tool through the middleware and security pipeline.
-   **`tui.Model.Update(msg)`**: Handles all events (key presses, API responses) in the TUI state machine.
-   **`history.Store.CreateConversation(model)`**: Generates a new conversation entry and metadata record.
-   **`browser.ExtractGeminiCookies(ctx, browserType)`**: Primary entry point for local authentication extraction.

## Available Documentation

| Document | Path | Quality Evaluation |
| :--- | :--- | :--- |
| **Structure Analysis** | `/.ai/docs/structure_analysis.md` | **High**: Excellent summary of package responsibilities and patterns. |
| **API Analysis** | `/.ai/docs/api_analysis.md` | **Excellent**: Technical deep dive into the internal Gemini Web RPC protocol. |
| **Tool Execution Doc** | `/pkg/toolexec/doc.go` | **Excellent**: Comprehensive package-level documentation for the tool framework. |
| **Project README** | `/README.md` | **Good**: Clear user-facing instructions and feature highlights. |
| **CLAUDE.md** | `/CLAUDE.md` | **Very Good**: Operational guide for developers (build/test/style). |