
Now I have a comprehensive understanding of the project structure and dependencies. Let me create the dependency analysis based on the codebase examination.

# Dependency Analysis

## Internal Dependencies Map

### Core Package Dependencies
- **cmd/geminiweb** → **internal/commands**: Entry point delegates to command execution
- **internal/commands** → **internal/api**: Client creation and API interactions
- **internal/commands** → **internal/config**: Configuration loading and management
- **internal/commands** → **internal/browser**: Browser cookie extraction
- **internal/commands** → **internal/models**: Data types and constants
- **internal/commands** → **internal/render**: Markdown rendering for output
- **internal/commands** → **internal/tui**: Interactive chat interface
- **internal/commands** → **internal/history**: Conversation history management
- **internal/commands** → **internal/errors**: Error handling

### API Package Dependencies
- **internal/api** → **internal/browser**: Browser cookie extraction interface
- **internal/api** → **internal/config**: Cookie validation and loading
- **internal/api** → **internal/models**: API data structures and constants
- **internal/api** → **internal/errors**: Error types and handling

### TUI Package Dependencies
- **internal/tui** → **internal/api**: Chat session interface and client
- **internal/tui** → **internal/history**: History store interface
- **internal/tui** → **internal/models**: Message and response types
- **internal/tui** → **internal/render**: Markdown rendering in chat

### Render Package Dependencies
- **internal/render** → **internal/config**: Configuration-based rendering options

### History Package Dependencies
- **internal/history** → **internal/models**: Conversation and message types

### Cross-Cutting Dependencies
- **internal/models** → **internal/errors**: Error code constants (backward compatibility)
- All packages → **internal/models**: Shared data types and constants

## External Libraries Analysis

### Core HTTP/TLS Libraries
- **github.com/bogdanfinn/tls-client v1.9.1**: HTTP client with Chrome 133 fingerprinting for browser emulation
- **github.com/bogdanfinn/fhttp v0.5.34**: HTTP utilities for advanced request handling

### CLI Framework
- **github.com/spf13/cobra v1.8.1**: Command-line interface framework for command structure and flag management

### TUI Framework Stack
- **github.com/charmbracelet/bubbletea v1.3.4**: Core TUI framework for interactive interfaces
- **github.com/charmbracelet/bubbles v0.21.0**: Reusable TUI components (spinner, textarea, viewport, etc.)
- **github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834**: Terminal styling and layout
- **github.com/charmbracelet/glamour v0.10.0**: Markdown rendering for terminal output

### Browser Integration
- **github.com/browserutils/kooky v0.2.4**: Cross-platform browser cookie extraction with decryption support

### Data Processing
- **github.com/tidwall/gjson v1.18.0**: JSON parsing and extraction for API responses

### System Integration
- **github.com/atotto/clipboard v0.1.4**: Cross-platform clipboard access
- **golang.org/x/term v0.33.0**: Terminal handling and TTY detection

### Indirect Dependencies
- **golang.org/x/crypto v0.40.0**: Cryptographic functions for cookie decryption
- **golang.org/x/net v0.42.0**: Network utilities and HTTP extensions
- **golang.org/x/sys v0.34.0**: System-level interfaces for cross-platform support

## Service Integrations

### Google Gemini Web API
- **Primary Integration**: Direct communication with gemini.google.com web interface
- **Authentication**: Cookie-based using `__Secure-1PSID` and `__Secure-1PSIDTS`
- **Endpoints**: 
  - Content generation: `https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate`
  - Cookie rotation: `https://accounts.google.com/RotateCookies`
  - File upload: `https://content-push.googleapis.com/upload`
  - Batch operations: `https://gemini.google.com/_/BardChatUi/data/batchexecute`

### Browser Cookie Extraction
- **Chrome/Chromium**: Cookie extraction with SQLite database decryption
- **Firefox**: Cookie extraction with profile support
- **Edge**: Cookie extraction with profile support
- **Opera**: Cookie extraction with profile support
- **Auto-detection**: Sequential browser detection with fallback

### File System Integration
- **Configuration**: `~/.geminiweb/config.json` for user settings
- **Cookies**: `~/.geminiweb/cookies.json` for persistent authentication
- **History**: `~/.geminiweb/history/` for conversation persistence
- **Themes**: Built-in theme files for markdown rendering

## Dependency Injection Patterns

### Functional Options Pattern
The `api` package uses functional options for client configuration:
```go
type ClientOption func(*GeminiClient)

func WithModel(model models.Model) ClientOption
func WithAutoRefresh(enabled bool) ClientOption
func WithBrowserRefresh(browserType browser.SupportedBrowser) ClientOption
func WithBrowserCookieExtractor(extractor BrowserCookieExtractor) ClientOption
func WithRefreshFunc(fn RefreshFunc) ClientOption
func WithCookieLoader(fn CookieLoader) ClientOption
func WithHTTPClient(client tls_client.HttpClient) ClientOption
```

### Interface-Based Dependency Injection
- **GeminiClientInterface**: Enables mocking and testing of API client
- **BrowserCookieExtractor**: Allows custom cookie extraction implementations
- **ChatSessionInterface**: Decouples TUI from API implementation
- **HistoryStoreInterface**: Enables different storage backends

### Constructor Injection
- **NewClient()**: Accepts optional dependencies via functional options
- **NewStore()**: Creates history store with configurable base directory
- **NewChatModel()**: Creates TUI model with injected client and session

### Factory Pattern
- **ModelFromName()**: Creates model instances from string names
- **ParseBrowser()**: Creates browser type enums from strings
- **DefaultConfig()**: Creates default configuration instances

## Module Coupling Assessment

### High Cohesion Areas
- **API Package**: Well-contained HTTP client logic with clear interfaces
- **Models Package**: Centralized data types and constants
- **Config Package**: Focused configuration management
- **History Package**: Self-contained persistence layer

### Moderate Coupling
- **Commands → API**: Necessary coupling for CLI functionality
- **TUI → API**: Required for interactive chat features
- **API → Browser**: Cookie extraction dependency

### Low Coupling Areas
- **Render Package**: Minimal dependencies, primarily configuration-driven
- **Errors Package**: Standalone error types with minimal coupling
- **Browser Package**: Focused cookie extraction with clear interface

### Potential Tight Coupling
- **Commands Package**: Acts as orchestrator, couples many subsystems
- **TUI Model**: Large struct with multiple responsibilities (chat, history, gems)

## Dependency Graph

```
cmd/geminiweb
    ↓
internal/commands
    ├── internal/api
    │   ├── internal/browser
    │   ├── internal/config
    │   └── internal/models
    ├── internal/config
    ├── internal/tui
    │   ├── internal/api
    │   ├── internal/history
    │   └── internal/models
    ├── internal/history
    │   └── internal/models
    ├── internal/render
    │   └── internal/config
    ├── internal/models
    │   └── internal/errors
    └── internal/errors
```

## Potential Dependency Issues

### Circular Dependencies
- **models → errors**: Backward compatibility error code aliases create circular reference
- **Impact**: Minimal, but could complicate future refactoring

### Large Command Package
- **Issue**: commands package has too many responsibilities
- **Impact**: Makes testing and maintenance difficult
- **Recommendation**: Split into domain-specific command packages

### TUI Model Complexity
- **Issue**: Model struct handles chat, history, gems, and file attachments
- **Impact**: Violates single responsibility principle
- **Recommendation**: Extract separate models for different TUI modes

### Configuration Coupling
- **Issue**: Multiple packages directly access configuration
- **Impact**: Configuration changes affect many modules
- **Recommendation**: Use configuration interfaces to reduce coupling

### Browser Extraction Coupling
- **Issue**: API package directly depends on browser package
- **Impact**: Makes API client less portable
- **Recommendation**: Use interface abstraction for cookie extraction

### Testing Challenges
- **Issue**: External dependencies (browsers, file system) make integration testing complex
- **Impact**: Reduced test coverage and reliability
- **Recommendation**: Increase use of dependency injection for testability