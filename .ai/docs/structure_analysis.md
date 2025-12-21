# Code Structure Analysis

## Architectural Overview
`geminiweb-go` is designed as a modular CLI application that interacts with the Google Gemini web API by emulating browser behavior. The architecture follows a **Clean/Hexagonal** approach, prioritizing separation of concerns and testability through interfaces.

*   **Entry Points**: `cmd/` contains the binary main functions, delegating logic to the `internal/` packages.
*   **Domain Logic**: The core logic resides in `internal/api` (communication) and `internal/models` (data structures).
*   **Infrastructure**: `internal/browser` and `internal/config` handle external dependencies like cookie extraction and local file persistence.
*   **UI Layer**: A rich terminal user interface (TUI) implemented in `internal/tui` using the Bubble Tea (The Elm Architecture) pattern.
*   **Command Layer**: Cobra-based CLI commands in `internal/commands` that orchestrate services to fulfill user requests.

## Core Components
*   **GeminiClient (`internal/api`)**: The central engine that manages authentication, TLS fingerprinting (Chrome emulation), and API requests.
*   **ChatSession (`internal/api`)**: A stateful wrapper around the client that tracks conversation IDs (`cid`, `rid`, `rcid`) to maintain multi-turn context.
*   **History Store (`internal/history`)**: Manages the persistence of conversations to local JSON files, including metadata for ordering and favorites.
*   **TUI Model (`internal/tui`)**: A complex state machine managing viewport rendering, user input, and asynchronous API communication within the terminal.
*   **Cookie Rotator (`internal/api`)**: A background service that ensures session tokens (1PSIDTS) are refreshed to prevent session expiry.

## Service Definitions
*   **API Service**: Encapsulated by `GeminiClient`, responsible for `GenerateContent`, `UploadFile`, and `FetchGems`. It abstracts the complexity of the Gemini internal RPC protocol.
*   **Browser Integration Service**: Handles the extraction and decryption of cookies from local browser profiles (Chrome, Firefox, etc.) to automate login.
*   **Persistence Service**: Managed by `internal/config` and `internal/history`, handling user settings, personas, and chat logs.
*   **Rendering Service**: `internal/render` uses Glamour and custom themes to convert Gemini's Markdown responses into terminal-optimized visual output.

## Interface Contracts
The codebase makes extensive use of interfaces to decouple the TUI and Commands from the concrete API implementation:

*   **`GeminiClientInterface` (`internal/api/client.go`)**: Defines the capabilities of the Gemini client, allowing for easy mocking during UI testing.
*   **`ChatSessionInterface` (`internal/tui/model.go`)**: Defines how the UI interacts with a specific conversation thread.
*   **`BrowserCookieExtractor` (`internal/api/client.go`)**: Abstracts the browser-specific logic for cookie retrieval.
*   **`FullHistoryStore` (`internal/tui/model.go`)**: An interface used by the TUI to manage list, search, and export operations on chat history.

## Design Patterns Identified
*   **Functional Options**: Used in `internal/api` (`NewClient`, `WithModel`, `WithAutoRefresh`) for clean and extensible component configuration.
*   **The Elm Architecture (MVU)**: The `internal/tui` package follows the Model-View-Update pattern via the `charmbracelet/bubbletea` framework.
*   **Repository Pattern**: `internal/history` acts as a repository for `Conversation` models, hiding the filesystem complexity.
*   **Dependency Injection**: Interfaces are passed into constructors (e.g., `NewChatModel(client GeminiClientInterface, ...)`), facilitating unit testing.
*   **Strategy Pattern**: The browser extraction logic uses different strategies based on the selected browser type.
*   **Proxy/Wrapper**: `ChatSession` wraps the `GeminiClient` to append context metadata automatically to requests.

## Component Relationships
1.  **CLI Command** → **GeminiClient**: Commands initialize the client with user-provided or auto-detected credentials.
2.  **TUI Model** → **ChatSession**: The interactive chat UI uses a session to track the specific thread.
3.  **GeminiClient** → **Browser**: The client calls the browser package when authentication tokens are missing or expired.
4.  **TUI Model** → **History Store**: Every assistant response is automatically persisted to the local history store via the TUI's update loop.
5.  **Render Service** ← **TUI Model**: The TUI passes raw API responses to the render service before displaying them in the viewport.

## Key Methods & Functions
*   **`api.NewClient`**: Initializes the core client with TLS fingerprinting.
*   **`api.GenerateContent`**: The primary method for sending prompts and receiving (and parsing) stream-like responses.
*   **`api.RefreshFromBrowser`**: Automates the "login" process by pulling cookies from the user's browser.
*   **`history.Store.ListConversations`**: Efficiently retrieves chat history using a `meta.json` index.
*   **`tui.Model.Update`**: The main event loop for the terminal interface, handling keyboard input and async messages.

## Available Documentation
*   **`/.ai/docs/structure_analysis.md`**: Provides a high-level overview of the package layout and architecture (Quality: High, recently updated).
*   **`/.ai/docs/api_analysis.md`**: Deep dive into the Gemini RPC protocol and payload structures (Quality: Technical/Deep).
*   **`/.cursor/rules/project-overview.mdc`**: Contains core architectural rules and naming conventions for the project (Quality: Essential for contributors).
*   **`/README.md`**: User-facing documentation for installation and basic usage.
*   **`/.serena/memories/`**: Contains various markdown files tracking project history and specific system designs like the history system and client lifecycle.