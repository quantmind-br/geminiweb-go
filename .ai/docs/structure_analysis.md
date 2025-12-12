
# Code Structure Analysis

## Architectural Overview

geminiweb-go is a CLI application for interacting with Google Gemini's web API using cookie-based authentication. The architecture follows a clean, modular design with clear separation of concerns:

- **Entry Point**: Single binary entry point via `cmd/geminiweb/main.go`
- **Core Layer**: API client with TLS fingerprinting and browser emulation
- **Command Layer**: Cobra-based CLI with subcommands for different operations
- **UI Layer**: Bubble Tea TUI for interactive chat with Glamour markdown rendering
- **Data Layer**: JSON-based persistence for configuration, cookies, and conversation history
- **Infrastructure**: Browser cookie extraction, error handling, and rendering utilities

The application uses a hexagonal architecture pattern with the API client as the core domain component, surrounded by adapters for browser integration, CLI commands, and user interfaces.

## Core Components

### API Client (`internal/api/`)
- **GeminiClient**: Main client with Chrome 133 TLS fingerprinting for browser emulation
- **ChatSession**: Maintains conversation context across multi-turn interactions
- **CookieRotator**: Background token rotation with configurable intervals
- **File Upload**: Support for image and file attachments
- **Gems Integration**: Server-side persona management
- **Batch RPC**: Low-level API communication

### Browser Integration (`internal/browser/`)
- **Multi-Browser Support**: Chrome, Firefox, Edge, Chromium, Opera
- **Auto-Detection**: Automatic browser discovery and cookie extraction
- **Cross-Platform**: Uses kooky library for encrypted cookie database access
- **Rate-Limited Refresh**: 30-second minimum between browser refresh attempts

### TUI Framework (`internal/tui/`)
- **Bubble Tea Model**: Reactive state management for chat interface
- **Multi-Mode Support**: Chat, gem selection, history browsing, configuration
- **Markdown Rendering**: Glamour-based rendering with custom themes
- **Keyboard Shortcuts**: Comprehensive key bindings for navigation

### Configuration System (`internal/config/`)
- **JSON Persistence**: User settings stored in `~/.geminiweb/`
- **Cookie Management**: Secure storage and validation of authentication cookies
- **Persona Storage**: Local persona definitions and management
- **Theme Configuration**: Markdown and TUI theme customization

### History Management (`internal/history/`)
- **Conversation Persistence**: JSON-based storage with metadata
- **Search & Filter**: Conversation browsing with search capabilities
- **Export Support**: Markdown export of conversations
- **Metadata Management**: Conversation ordering, favorites, and timestamps

## Service Definitions

### GeminiClient Service
Primary service for API communication with responsibilities:
- Authentication via cookie-based access tokens
- Content generation with multiple model support
- File upload and attachment handling
- Background cookie rotation and browser refresh
- Gems (server-side personas) management
- Batch RPC execution for API operations

### BrowserCookieExtractor Service
Handles browser integration with responsibilities:
- Cross-browser cookie extraction with decryption
- Automatic browser detection and fallback
- Rate-limited refresh to avoid browser database locks
- Support for multiple browser profiles

### HistoryStore Service
Manages conversation persistence with responsibilities:
- JSON-based conversation storage and retrieval
- Metadata management for ordering and favorites
- Conversation search and filtering
- Export functionality for external use

### RenderService
Handles content presentation with responsibilities:
- Markdown rendering with Glamour
- Theme management and customization
- Performance optimization via LRU caching
- Terminal-specific formatting

## Interface Contracts

### GeminiClientInterface
```go
type GeminiClientInterface interface {
    // Core client lifecycle
    Init() error
    Close()
    GetAccessToken() string
    GetCookies() *config.Cookies
    GetModel() models.Model
    SetModel(model models.Model)
    IsClosed() bool
    
    // Chat operations
    StartChat(model ...models.Model) *ChatSession
    StartChatWithOptions(opts ...ChatOption) *ChatSession
    
    // Content generation
    GenerateContent(prompt string, opts *GenerateOptions) (*models.ModelOutput, error)
    UploadImage(filePath string) (*UploadedImage, error)
    UploadFile(filePath string) (*UploadedFile, error)
    
    // Browser refresh
    RefreshFromBrowser() (bool, error)
    IsBrowserRefreshEnabled() bool
    
    // Gems management
    FetchGems(includeHidden bool) (*models.GemJar, error)
    CreateGem(name, prompt, description string) (*models.Gem, error)
    UpdateGem(gemID, name, prompt, description string) (*models.Gem, error)
    DeleteGem(gemID string) error
    Gems() *models.GemJar
    GetGem(id, name string) *models.Gem
    
    // Batch operations
    BatchExecute(requests []RPCData) ([]BatchResponse, error)
}
```

### BrowserCookieExtractor Interface
```go
type BrowserCookieExtractor interface {
    ExtractGeminiCookies(ctx context.Context, browser browser.SupportedBrowser) (*browser.ExtractResult, error)
}
```

### ChatSessionInterface
```go
type ChatSessionInterface interface {
    SendMessage(prompt string, files []*api.UploadedFile) (*models.ModelOutput, error)
    SetMetadata(cid, rid, rcid string)
    GetMetadata() []string
    CID() string
    RID() string
    RCID() string
    GetModel() models.Model
    SetModel(model models.Model)
    LastOutput() *models.ModelOutput
    ChooseCandidate(index int) error
    SetGem(gemID string)
    GetGemID() string
}
```

## Design Patterns Identified

### Dependency Injection
- Functional options pattern for client configuration (`WithModel`, `WithAutoRefresh`, `WithBrowserRefresh`)
- Interface-based design for testability and mocking
- Service locator pattern for component initialization

### Repository Pattern
- HistoryStore implements repository pattern for conversation persistence
- Configuration management follows repository pattern with JSON storage
- Cookie management with validation and persistence

### Command Pattern
- Cobra CLI commands implement command pattern for different operations
- Each command encapsulates specific functionality (chat, config, import, etc.)

### Observer Pattern
- Bubble Tea's message-driven architecture for TUI updates
- Event-driven state management for reactive UI updates

### Strategy Pattern
- Multiple browser extraction strategies with fallback mechanisms
- Different rendering strategies for various content types
- Model selection strategies for different use cases

### Factory Pattern
- Client factory with configurable options
- Theme factory for different rendering styles
- Model factory for different Gemini models

## Component Relationships

### Core Flow
1. **CLI Entry** → Commands → API Client → Gemini Web API
2. **TUI Flow** → Bubble Tea Model → API Client → Response Rendering
3. **Browser Integration** → Cookie Extractor → API Client Authentication
4. **History Flow** → TUI → History Store → JSON Persistence

### Data Flow
- **Configuration**: JSON files → Config Service → Client Options
- **Authentication**: Browser → Cookies → Access Token → API Requests
- **Conversations**: User Input → API Client → Response → History Store
- **Rendering**: API Response → Markdown Renderer → TUI Display

### Dependency Graph
- Commands depend on API Client, Config, Browser, History
- TUI depends on API Client, Render Service, History Store
- API Client depends on Browser Extractor, Config, Models
- All components depend on shared Models and Error types

## Key Methods & Functions

### Authentication Methods
- `ExtractGeminiCookies()`: Browser cookie extraction with decryption
- `RotateCookies()`: Background token rotation
- `RefreshFromBrowser()`: Rate-limited browser refresh on auth failure
- `ValidateCookies()`: Cookie format and validity checking

### Content Generation Methods
- `GenerateContent()`: Core API communication with metadata support
- `SendMessage()`: Chat session message handling with context
- `UploadFile()`/`UploadImage()`: File attachment processing
- `BatchExecute()`: Low-level RPC batch operations

### UI Management Methods
- `Update()`: Bubble Tea state management and message handling
- `View()`: Terminal rendering with Lipgloss styling
- `Markdown()`: Glamour-based markdown rendering with caching
- `CreateTextarea()`: Multi-line input component configuration

### History Management Methods
- `CreateConversation()`: New conversation initialization
- `ListConversations()`: Conversation retrieval with metadata
- `ExportToMarkdown()`: Conversation export functionality
- `ToggleFavorite()`: Conversation bookmarking

## Available Documentation

### Existing Documentation
- `./.serena/memories/codebase_structure.md`: Comprehensive structural overview
- `./.serena/memories/project_overview.md`: High-level project description
- `./README.md`: User-facing documentation with usage examples
- `./docs/API_BREAKING_CHANGE_2024-12.md`: API change documentation

### Documentation Quality Assessment
- **Excellent**: The existing documentation provides comprehensive coverage of the codebase structure, with detailed component descriptions and clear architectural patterns
- **Well-Maintained**: Recent updates reflect current codebase state
- **Developer-Focused**: Technical documentation is thorough with good separation between user and developer documentation
- **Complete Coverage**: All major components are documented with their responsibilities and relationships

The documentation quality is high and provides excellent foundation for understanding the system architecture and component interactions.