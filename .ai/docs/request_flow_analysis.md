# Request Flow Analysis

## Entry Points Overview
The application is primarily a CLI-based tool with multiple entry points defined in the `internal/commands` package using the Cobra library.
- **Root Command (`geminiweb`)**: Handles single queries from positional arguments, standard input, or files.
- **Chat Command (`geminiweb chat`)**: Launches an interactive TUI-based chat session.
- **Config Command (`geminiweb config`)**: Manages local settings and personas.
- **Utility Commands**: Commands like `import-cookies`, `auto-login`, `history`, `persona`, and `gems` for session and preference management.

## Request Routing Map
Request routing follows a path from the CLI layer to the API client layer:
1. **CLI Layer (`internal/commands`)**: Parses flags and arguments, then calls appropriate methods on the `GeminiClient` or `ChatSession`.
2. **Session Layer (`internal/api/session.go`)**: Manages conversation state (metadata) for multi-turn chats.
3. **API Layer (`internal/api/client.go`)**: Centralizes logic for authentication, cookie rotation, and model selection.
4. **Implementation Layer (`internal/api/generate.go`, `upload.go`, `batch.go`)**: Handles the specific HTTP request formation for different Gemini operations.

## Middleware Pipeline
The "middleware" in this Go-based client is implemented as a sequence of lifecycle hooks and configuration options:
- **Initialization Pipeline**: `Init()` method handles auth checks, cookie loading, and token retrieval before any request.
- **Activity Monitoring**: `resetIdleTimer()` is called on every request to manage auto-close logic for background resources.
- **Request Pre-processing**: `ensureRunning()` checks client state and re-initializes if necessary.
- **Cookie Rotation**: A background `CookieRotator` goroutine periodically refreshes `__Secure-1PSIDTS` to maintain session validity.

## Controller/Handler Analysis
- **`GeminiClient`**: The primary controller that manages the `tls_client.HttpClient`, cookies, and authentication state.
- **`ChatSession`**: A stateful wrapper around the client that tracks `cid` (conversation ID), `rid` (reply ID), and `rcid` (reply candidate ID).
- **`runQuery` (internal/commands/query.go)**: The functional handler for one-off requests, managing UI elements like spinners and formatted output.
- **`doGenerateContent`**: The low-level handler that constructs the multipart/form-data payload, sets security headers, and processes the streaming response.

## Authentication & Authorization Flow
1. **Cookie Discovery**: Tries loading from `cookies.json` or extracting from browsers (Chrome, Firefox, etc.).
2. **Session Initialization**: Calls `EndpointInit` (`/app`) to verify cookies and extract the `SNlM0e` access token using regex.
3. **Token Propagation**: The `SNlM0e` token is passed as the `at` form parameter in subsequent POST requests.
4. **Header Injection**: Requests include `__Secure-1PSID` and `__Secure-1PSIDTS` cookies and specific `x-goog-ext-*` headers.
5. **Fallback Auth**: If a 401/Auth error occurs, the system triggers `RefreshFromBrowser` if enabled, automatically re-extracting fresh cookies.

## Error Handling Pathways
- **Structured Errors**: Uses a custom `GeminiError` struct that wraps HTTP status codes, internal Gemini codes (e.g., 1037 for rate limit), and the response body.
- **Predicate Checks**: Provides helper methods like `IsAuth()`, `IsRateLimit()`, and `IsBlocked()` for logical branching.
- **Diagnostic Capture**: Automatically captures and truncates response bodies during failures to provide context for "blocked" or "sorry" redirects.
- **Retry Logic**: Explicit retry loop in `GenerateContent` that attempts a browser-based cookie refresh upon detecting authentication failure.

## Request Lifecycle Diagram
1. **User Input**: CLI command (e.g., `geminiweb "Hello"`) is executed.
2. **Client Setup**: `api.NewClient()` is called with options; `Init()` loads cookies and fetches the `SNlM0e` token.
3. **Payload Construction**: `buildPayloadWithGem()` converts the prompt and files into the complex nested JSON format required by Gemini.
4. **Network Request**: `tls_client.Do(POST)` sends the request to the `StreamGenerate` endpoint with appropriate security headers.
5. **Stream Processing**: The response is read as a stream, stopping at the end marker `[["e",...]`.
6. **Data Extraction**: `parseResponse()` uses GJSON paths (defined in `paths.go`) to extract text, metadata, and images.
7. **Presentation**: The output is rendered via `glamour` (markdown) and displayed to the user with TUI styling.