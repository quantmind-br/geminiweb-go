
Based on my analysis of the geminiweb-go codebase, I can now provide a comprehensive data flow analysis. Here's the complete documentation:

# Data Flow Analysis

## Data Models Overview

The application uses several core data models that represent different aspects of the system:

### Core Models
- **Message**: Simple chat message structure with `Role` ("user"/"assistant") and `Content` fields
- **ModelOutput**: Complete API response containing metadata, multiple candidates, and chosen selection
- **Candidate**: Individual response with text, thoughts, and images (web and generated)
- **Gem**: Server-side persona with ID, name, description, prompt, and predefined flag
- **GemJar**: Collection container for gems with filtering and retrieval methods

### Configuration Models
- **Config**: User configuration including default model, TUI theme, and markdown settings
- **Cookies**: Authentication credentials with Secure1PSID and Secure1PSIDTS
- **MarkdownConfig**: Rendering options for markdown output

### History Models
- **Conversation**: Complete chat session with messages, metadata, timestamps, and Gemini API identifiers
- **ConversationMeta**: Lightweight metadata for conversation listing (ID, title, favorite status)
- **HistoryMeta**: Global ordering and favorites management for all conversations

### File Models
- **UploadedFile**: File reference with resource ID, filename, MIME type, and size
- **WebImage/GeneratedImage**: Image structures with URL, title, and alt text

## Data Transformation Map

### API Response Transformation
1. **Raw API Response** → **ModelOutput**: JSON parsing extracts metadata, candidates, and images
2. **ModelOutput** → **chatMessage**: TUI-specific transformation for display
3. **Candidate Selection**: User can choose different candidates, updating chosen index

### Configuration Transformation
1. **JSON Config File** → **Config Struct**: Unmarshaling with defaults fallback
2. **Config** → **Render Options**: Mapping of markdown settings to renderer parameters
3. **Environment Variables** → **Config Override**: Runtime configuration precedence

### History Transformation
1. **Conversation** → **JSON Storage**: Serialization with metadata preservation
2. **JSON Files** → **Conversation List**: Loading with metadata resolution and ordering
3. **Message Addition**: Real-time conversation updates with timestamp tracking

### File Upload Transformation
1. **Local File** → **Multipart Form**: File encoding with MIME type detection
2. **Upload Response** → **UploadedFile**: Resource ID extraction and metadata creation
3. **File References** → **API Payload**: Integration into generation requests

## Storage Interactions

### Local File System Storage
- **Configuration**: `~/.geminiweb/config.json` - User settings and preferences
- **Cookies**: `~/.geminiweb/cookies.json` - Authentication credentials with restricted permissions (0600)
- **History**: `~/.geminiweb/history/*.json` - Individual conversation files
- **Metadata**: `~/.geminiweb/history/meta.json` - Global conversation ordering and favorites

### Database Patterns
- **JSON File-based Storage**: Each conversation stored as separate JSON file
- **Metadata Index**: Separate meta.json file for efficient listing and ordering
- **Atomic Operations**: File locking with sync.RWMutex for concurrent access
- **Cleanup Mechanisms**: Automatic orphaned metadata removal and consistency checks

### Remote API Storage
- **Gemini Web API**: Primary data source for model responses and gems
- **Content Push Service**: File upload endpoint with resource ID generation
- **Batch Execute**: Multiple RPC operations in single HTTP request

## Validation Mechanisms

### Input Validation
- **Cookie Validation**: Required Secure1PSID presence and format checking
- **File Upload Validation**: Size limits (20MB images, 50MB text), MIME type verification
- **Model Validation**: Header format verification and availability checking
- **Prompt Validation**: Empty string prevention and length limits

### API Response Validation
- **Status Code Checking**: HTTP 200/201 validation for successful operations
- **JSON Format Validation**: Response structure verification with gjson parsing
- **Error Code Mapping**: Gemini API error codes to structured error types
- **Stream End Detection**: Special marker detection for streaming responses

### Configuration Validation
- **Default Fallbacks**: Graceful degradation when config files are missing
- **Theme Validation**: Built-in theme verification and custom theme loading
- **Browser Type Validation**: Supported browser checking for auto-refresh

## State Management Analysis

### Client State
- **Authentication State**: Cookie management with automatic rotation
- **Session State**: Chat context with metadata tracking (CID, RID, RCID)
- **Model State**: Current model selection and header configuration
- **Gem Cache**: In-memory gem collection with lazy loading

### TUI State
- **Message History**: In-memory message buffer for display
- **UI Component State**: Viewport, textarea, spinner state management
- **Selection State**: Cursor position, active modes, and user interactions
- **Loading States**: Async operation tracking with visual feedback

### History State
- **Conversation Ordering**: Persistent ordering with metadata.json
- **Favorite Management**: Boolean flag per conversation with persistence
- **Search State**: Filtered conversation lists with query matching
- **Export State**: Markdown generation with conversation reconstruction

## Serialization Processes

### JSON Serialization
- **Configuration**: Structured config with indentation and error handling
- **History**: Conversation serialization with timestamp preservation
- **Metadata**: Compact JSON for efficient loading and updating
- **API Payloads**: Nested array structures for Gemini API compatibility

### Form Data Serialization
- **Generation Requests**: URL-encoded form data with access tokens
- **File Uploads**: Multipart form encoding with proper MIME boundaries
- **Batch Operations**: JSON arrays with RPC identifiers and payloads

### Response Parsing
- **Streaming Responses**: Chunk-based parsing with end marker detection
- **Batch Responses**: Multi-part response parsing with identifier matching
- **Error Responses**: Structured error extraction with body truncation

## Data Lifecycle Diagrams

### Chat Session Lifecycle
1. **Initialization**: Load config → Authenticate → Create session
2. **Message Flow**: User input → File upload → API request → Response parsing → Display
3. **Persistence**: Message addition → Conversation save → Metadata update
4. **Session Management**: Context tracking → Cookie rotation → Error recovery

### File Upload Lifecycle
1. **Validation**: File existence → Size check → MIME type detection
2. **Upload**: Multipart encoding → HTTP POST → Resource ID extraction
3. **Integration**: File reference creation → API payload inclusion → Response generation

### Configuration Lifecycle
1. **Loading**: Default creation → File read → JSON parsing → Validation
2. **Runtime Use**: Environment override → Option mapping → Component configuration
3. **Persistence**: User changes → Validation → JSON marshaling → Atomic write

### History Management Lifecycle
1. **Creation**: New conversation → Initial metadata → File creation
2. **Updates**: Message addition → Timestamp update → Metadata sync
3. **Operations**: Listing → Filtering → Reordering → Export → Deletion

The data flow architecture emphasizes separation of concerns with clear transformation boundaries, robust error handling, and efficient caching strategies. The system maintains data consistency through atomic operations and comprehensive validation at each transformation stage.