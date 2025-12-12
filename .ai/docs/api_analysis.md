
Based on my comprehensive analysis of the geminiweb-go project, I'll create detailed API documentation covering both served and consumed APIs.

# API Documentation

## APIs Served by This Project

### Overview
This project is a **CLI tool** that consumes Google Gemini's Web API rather than serving its own REST APIs. The application provides a command-line interface for interacting with Google Gemini through cookie-based authentication.

### Technology Stack
- **Language**: Go 1.23.10
- **CLI Framework**: Cobra
- **HTTP Client**: bogdanfinn/tls-client (for browser fingerprinting)
- **JSON Parsing**: tidwall/gjson
- **Browser Integration**: browserutils/kooky
- **TUI Framework**: Charm Bubbletea

### Endpoints

#### CLI Commands (Primary Interface)

**Root Command - Query Processing**
- **Method**: CLI execution
- **Path**: `geminiweb [prompt]`
- **Description**: Send a single query to Gemini and get response
- **Request**: 
  - Positional argument: prompt text
  - Flags: `--model`, `--output`, `--file`, `--image`, `--gem`
- **Response**: Text output to stdout or file
- **Authentication**: Cookie-based (auto-loaded from config)
- **Examples**:
  ```bash
  geminiweb "What is Go?"
  geminiweb -f prompt.md
  cat prompt.md | geminiweb
  geminiweb "Hello" -o response.md
  ```

**Interactive Chat**
- **Method**: CLI subcommand
- **Path**: `geminiweb chat`
- **Description**: Start interactive chat session with conversation context
- **Request**: Interactive input via TUI
- **Response**: Streaming responses in terminal
- **Authentication**: Cookie-based
- **Features**: Multi-turn conversations, file uploads, gem personas

**Configuration Management**
- **Method**: CLI subcommand
- **Path**: `geminiweb config`
- **Description**: Configure default model, themes, and settings
- **Request**: Interactive prompts or flags
- **Response**: Configuration saved to `~/.geminiweb/config.json`

**Cookie Management**
- **Method**: CLI subcommand
- **Path**: `geminiweb import-cookies <file>`
- **Description**: Import authentication cookies from browser export
- **Request**: Path to cookies.json file
- **Response**: Validation and storage of cookies

**Gems Management**
- **Method**: CLI subcommands
- **Path**: `geminiweb gems [list|create|update|delete]`
- **Description**: Manage server-side personas (Gems)
- **Request**: Gem operations via CLI flags
- **Response**: Gem information and operation results

**History Management**
- **Method**: CLI subcommand
- **Path**: `geminiweb history [list|export|search]`
- **Description**: Manage conversation history
- **Request**: History operations
- **Response**: Conversation data and exports

### Authentication & Security

#### Cookie-Based Authentication
- **Primary Method**: Browser cookies (`__Secure-1PSID`, `__Secure-1PSIDTS`)
- **Cookie Sources**:
  - Manual import from browser export
  - Automatic extraction from browsers (Chrome, Firefox, Edge, Chromium, Opera)
  - Auto-refresh on authentication failure

#### Access Token Management
- **Token Type**: SNlM0e token extracted from Gemini initialization page
- **Extraction**: Regex pattern matching from HTML response
- **Refresh**: Automatic token refresh on expiry

#### Browser Fingerprinting
- **Client Profile**: Chrome 133 emulation
- **Headers**: Complete browser header set for anti-bot detection
- **TLS Client**: Custom TLS fingerprinting to avoid detection

### Rate Limiting & Constraints

#### API Rate Limits
- **Cookie Rotation**: Maximum once per minute
- **Request Limits**: Enforced by Google's anti-bot systems
- **Error Codes**:
  - `1037`: Usage limit exceeded
  - `1060`: IP temporarily blocked
  - `2`: Anti-bot verification failed (new as of Dec 2024)

#### File Upload Constraints
- **Image Files**: Maximum 20MB
- **Text Files**: Maximum 50MB
- **Supported Formats**: JPEG, PNG, GIF, WebP, Plain Text, Markdown, JSON, CSV, HTML, XML

## External API Dependencies

### Services Consumed

#### Google Gemini Web API
- **Service Name**: Google Gemini Web Interface
- **Base URL**: `https://gemini.google.com`
- **Purpose**: AI content generation and conversation management

**Endpoints Used**:

1. **Initialization Endpoint**
   - **URL**: `https://gemini.google.com/app`
   - **Method**: GET
   - **Purpose**: Extract SNlM0e access token
   - **Authentication**: Required cookies
   - **Response**: HTML with embedded access token

2. **Content Generation Endpoint**
   - **URL**: `https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate`
   - **Method**: POST
   - **Purpose**: Generate AI responses
   - **Authentication**: Access token + cookies
   - **Request Format**: Form data with `at` and `f.req` parameters
   - **Response Format**: Streaming JSON chunks with end marker `[["e",`

3. **File Upload Endpoint**
   - **URL**: `https://content-push.googleapis.com/upload`
   - **Method**: POST
   - **Purpose**: Upload images and text files
   - **Authentication**: None required
   - **Request Format**: Multipart form data
   - **Response**: Plain text resource ID

4. **Cookie Rotation Endpoint**
   - **URL**: `https://accounts.google.com/RotateCookies`
   - **Method**: POST
   - **Purpose**: Refresh session cookies
   - **Authentication**: Current cookies
   - **Rate Limit**: Once per minute maximum

5. **Batch Execute Endpoint**
   - **URL**: `https://gemini.google.com/_/BardChatUi/data/batchexecute`
   - **Method**: POST
   - **Purpose**: Execute multiple RPC operations (Gems management)
   - **Authentication**: Access token + cookies
   **Request Format**: Batched RPC calls with identifiers

#### Browser Cookie Extraction
- **Service**: Local browser cookie stores
- **Browsers Supported**: Chrome, Firefox, Edge, Chromium, Opera
- **Purpose**: Automatic authentication cookie extraction
- **Method**: Direct database access via kooky library

### Authentication Method

#### Cookie-Based Flow
1. **Initial Setup**: Import cookies from browser or extract automatically
2. **Token Extraction**: GET request to Gemini app page to extract SNlM0e token
3. **Request Authentication**: Each API request includes:
   - `__Secure-1PSID` cookie
   - `__Secure-1PSIDTS` cookie (optional)
   - `at` form parameter (SNlM0e token)
   - Browser fingerprint headers

#### Browser Refresh Mechanism
- **Trigger**: Authentication failure (401/403 responses)
- **Process**: Extract fresh cookies from configured browser
- **Fallback**: Manual cookie import required if automatic extraction fails

### Error Handling

#### Centralized Error Types
- **Authentication Errors**: Cookie expiry, token invalid, IP blocked
- **Network Errors**: Connection failures, timeouts
- **API Errors**: Rate limiting, model unavailability, content policy violations
- **Parse Errors**: Invalid response format, missing data

#### Error Recovery Patterns
1. **Automatic Retry**: Browser cookie refresh on auth failure
2. **Graceful Degradation**: Continue with partial responses when possible
3. **User Guidance**: Clear error messages with resolution steps
4. **Rate Limiting**: Built-in delays for cookie rotation

#### Circuit Breaker Configuration
- **Cookie Rotation**: Rate limited to 1/minute
- **Browser Refresh**: Minimum 30 seconds between attempts
- **Request Timeouts**: 300 seconds default
- **Stream Detection**: Automatic termination on end marker

### Integration Patterns

#### Streaming Response Handling
- **Format**: Chunked JSON with size prefixes
- **End Detection**: `[["e",status,null,null,bytes]]` marker
- **Parsing**: Iterative chunk processing until valid response found

#### Session Management
- **Context Tracking**: CID, RID, RCID metadata for conversation continuity
- **Multi-turn Support**: Automatic metadata propagation between messages
- **Model Switching**: Dynamic model changes within sessions

#### File Integration
- **Upload Flow**: Local file → Google upload service → Resource ID → API reference
- **Type Detection**: MIME type inference from file extensions
- **Size Validation**: Pre-upload size checking with clear limits

#### Batch Operations
- **RPC Batching**: Multiple operations in single HTTP request
- **Identifier Matching**: Request-response correlation via custom identifiers
- **Error Isolation**: Individual operation failures don't affect batch

## Available Documentation

### Internal Documentation
- **API Breaking Changes**: `/docs/API_BREAKING_CHANGE_2024-12.md`
  - Details December 2024 anti-bot token changes
  - Explains new verification requirements
  - Provides migration guidance

### Code Documentation
- **Comprehensive Test Coverage**: Unit tests for all major components
- **Interface Definitions**: Clear separation of concerns with dependency injection
- **Error Documentation**: Detailed error codes and handling patterns

### Configuration Documentation
- **Default Settings**: Reasonable defaults for all configuration options
- **Environment Detection**: Automatic browser detection and cookie extraction
- **Theme Support**: Built-in themes for TUI and markdown rendering

### Documentation Quality Assessment
**Strengths**:
- Excellent error handling with detailed diagnostics
- Comprehensive test coverage
- Clear separation of concerns
- Well-documented breaking changes

**Areas for Improvement**:
- Could benefit from OpenAPI/Swagger specifications for external API contracts
- Missing integration examples for programmatic usage
- Limited documentation of internal data flow patterns

The project demonstrates mature API integration practices with robust error handling, authentication management, and clear architectural patterns. The main challenge is the reliance on Google's unofficial web API, which requires constant maintenance to handle anti-bot measures and API changes.