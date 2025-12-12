
Based on my analysis of the geminiweb-go codebase, I can now provide a comprehensive request flow analysis. This is a CLI application that interacts with Google Gemini's web API, not a traditional web server, so the "request flow" refers to how user commands are processed through the CLI to API interactions.

# Request Flow Analysis

## Entry Points Overview

The application has a single main entry point through the CLI:

**Primary Entry Point**: `cmd/geminiweb/main.go`
- Calls `commands.Execute()` which initializes the Cobra CLI framework
- Routes to different command handlers based on user input

**Command Entry Points**:
- **Root Command** (`internal/commands/root.go`): Handles single queries, file input, stdin input
- **Chat Command** (`internal/commands/chat.go`): Interactive chat sessions
- **Config Command** (`internal/commands/config.go`): Configuration management
- **Import Cookies** (`internal/commands/autologin.go`): Browser cookie extraction
- **History Command** (`internal/commands/history.go`): Conversation history management
- **Persona Command** (`internal/commands/persona.go`): Persona management
- **Gems Command** (`internal/commands/gems.go`): Server-side persona management

## Request Routing Map

The application uses Cobra for CLI routing with the following flow:

```
CLI Input → Cobra Router → Command Handler → API Client → Gemini Web API
```

**Routing Logic**:
1. **Root Command**: Direct query execution via `runQuery()`
2. **Subcommands**: Each has dedicated handlers (e.g., `runChat()` for chat)
3. **Input Sources**: Command line args, files (`-f`), stdin, or interactive TUI

**Key Routing Functions**:
- `commands.Execute()`: Main router entry point
- `runQuery()`: Handles single prompt requests
- `runChat()`: Manages interactive chat sessions
- TUI models handle internal routing for interactive features

## Middleware Pipeline

The application implements several middleware-like layers:

**Authentication Layer**:
- Cookie validation and loading (`internal/config/cookies.go`)
- Browser-based cookie extraction (`internal/browser/browser.go`)
- Access token fetching (`internal/api/token.go`)
- Automatic cookie rotation (`internal/api/rotate.go`)

**HTTP Client Middleware**:
- TLS client with browser fingerprinting (`tls_client` with Chrome_133 profile)
- Request/response header management
- Cookie injection and management
- Error handling and retry logic

**Configuration Layer**:
- Config loading and validation (`internal/config/config.go`)
- Model selection and validation
- Browser refresh settings
- Theme and rendering preferences

## Controller/Handler Analysis

**Primary Controllers**:

1. **Query Controller** (`internal/commands/query.go`):
   - Handles single prompt processing
   - Manages file uploads and image attachments
   - Coordinates response rendering and output

2. **Chat Controller** (`internal/commands/chat.go`):
   - Manages conversation sessions
   - Handles history persistence
   - Coordinates TUI interactions

3. **API Client Controller** (`internal/api/client.go`):
   - Central API interaction hub
   - Manages authentication state
   - Handles content generation requests
   - Manages file uploads and gems operations

4. **TUI Controllers** (`internal/tui/`):
   - `model.go`: Main chat interface
   - `config_model.go`: Configuration management
   - `gems_model.go`: Gems selection interface
   - `history_selector.go`: Conversation history management

**Handler Flow**:
```
User Input → Command Handler → API Client → HTTP Request → Gemini API
                ↓
         Response Processing → Rendering → Output
```

## Authentication & Authorization Flow

**Multi-layer Authentication**:

1. **Cookie-based Authentication**:
   - Loads cookies from `~/.geminiweb/cookies.json`
   - Falls back to browser extraction if needed
   - Supports Chrome, Firefox, Edge, Chromium, Opera

2. **Access Token Management**:
   - Fetches SNlM0e token from `https://gemini.google.com/app`
   - Token extraction via regex pattern matching
   - Automatic token refresh on authentication failures

3. **Browser Refresh Mechanism**:
   - Auto-extracts fresh cookies on auth failures
   - Rate-limited to prevent abuse
   - Silent fallback for seamless user experience

4. **Cookie Rotation**:
   - Background rotation of `__Secure-1PSIDTS` cookies
   - Configurable interval (default: 9 minutes)
   - Prevents session expiration

**Authentication Flow**:
```
Client Init → Load Cookies → Get Access Token → Validate → Ready
      ↓ (if failed)
Browser Extract → New Cookies → Get Access Token → Validate → Ready
```

## Error Handling Pathways

**Centralized Error System** (`internal/errors/errors.go`):

**Error Types**:
- `GeminiError`: Base error type with rich context
- Authentication errors (401, expired cookies)
- Network errors (timeouts, connection failures)
- API errors (rate limiting, model inconsistencies)
- Validation errors (file size, format issues)

**Error Handling Flow**:
1. **Detection**: HTTP status codes, response parsing, network failures
2. **Classification**: Categorize as auth, network, API, or validation error
3. **Recovery**: Automatic retry with browser refresh for auth errors
4. **User Feedback**: Clear error messages with actionable guidance

**Recovery Mechanisms**:
- Automatic browser cookie refresh on authentication failures
- Rate limiting with exponential backoff
- Graceful degradation for non-critical errors
- Detailed error diagnostics for debugging

## Request Lifecycle Diagram

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Input     │───▶│  Cobra Router    │───▶│ Command Handler │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   User Output   │◀───│ Response Renderer│◀───│  API Client     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Gemini Web API  │◀───│ HTTP Request     │◀───│ Auth Middleware │
└─────────────────┘    └──────────────────┘    └─────────────────┘

Detailed Flow:
1. User enters command/prompt
2. Cobra routes to appropriate handler
3. Handler creates API client with configuration
4. Client initializes authentication (cookies + token)
5. Request is built with proper headers and payload
6. HTTP request sent to Gemini API
7. Response parsed and validated
8. Error handling and recovery if needed
9. Response rendered and displayed to user
```

**Key Request Patterns**:

**Single Query Request**:
```
runQuery() → GenerateContent() → HTTP POST → Parse Response → Render Output
```

**Chat Session Request**:
```
Chat TUI → SendMessage() → GenerateContent() → Update Context → Render
```

**File Upload Request**:
```
UploadFile() → Multipart Form → Upload Endpoint → Resource ID → Include in Prompt
```

**Gems Management Request**:
```
BatchExecute() → Multiple RPC Calls → Parse Batch Response → Update Cache
```

The request flow is designed for resilience with automatic authentication recovery, comprehensive error handling, and seamless user experience through intelligent fallbacks and retries.