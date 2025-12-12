# History System

The history system provides conversation persistence, management, and export functionality.

## Components

### Store (`internal/history/store.go`)

Core persistence layer for conversations.

```go
type Message struct {
    Role      string    // "user" or "assistant"
    Content   string
    Thoughts  string    // Model's thinking (if available)
    Timestamp time.Time
}

type Conversation struct {
    ID         string
    Title      string
    Model      string
    CreatedAt  time.Time
    UpdatedAt  time.Time
    Messages   []Message
    CID        string    // Gemini conversation ID
    RID        string    // Gemini response ID
    RCID       string    // Gemini response candidate ID
    IsFavorite bool
    OrderIndex int
}
```

**Key Methods:**
- `NewStore(baseDir string)` - Create store instance
- `DefaultStore()` - Create store at `~/.geminiweb/history/`
- `CreateConversation(model string)` - Create new conversation
- `GetConversation(id string)` - Load conversation
- `ListConversations()` - List all (sorted by order)
- `AddMessage(id string, msg Message)` - Add message
- `UpdateMetadata(id, cid, rid, rcid string)` - Update Gemini IDs
- `DeleteConversation(id string)` - Delete
- `UpdateTitle(id, title string)` - Rename

### Metadata (`internal/history/meta.go`)

Manages favorites and conversation ordering.

```go
type ConversationMeta struct {
    ID         string
    Title      string
    IsFavorite bool
}

type HistoryMeta struct {
    Version int
    Order   []string                    // Ordered conversation IDs
    Meta    map[string]ConversationMeta // ID -> metadata
}
```

**Key Methods:**
- `ToggleFavorite(id string) (bool, error)` - Toggle favorite
- `SetFavorite(id string, fav bool)` - Set favorite
- `MoveConversation(id string, newIndex int)` - Reorder
- `SwapConversations(id1, id2 string)` - Swap positions

### Resolver (`internal/history/resolver.go`)

Resolves conversation references (aliases, indices).

```go
type Resolver struct {
    store *Store
}
```

**Supported Aliases:**
- `@last` - Most recent conversation
- `@first` - Oldest conversation
- `1`, `2`, ... - Index in list (1-based)
- Full UUID - Direct ID

**Key Methods:**
- `Resolve(ref string) (string, error)` - Get ID from reference
- `ResolveWithInfo(ref string) (*Conversation, error)` - Get full conversation
- `ValidateRef(ref string) error` - Check if valid

### Export (`internal/history/export.go`)

Export conversations to various formats.

```go
type ExportFormat string
const (
    ExportFormatMarkdown ExportFormat = "markdown"
    ExportFormatJSON     ExportFormat = "json"
)

type ExportOptions struct {
    Format          ExportFormat
    IncludeMetadata bool
    IncludeThoughts bool
}
```

**Key Methods:**
- `ExportToMarkdown(id string) (string, error)` - Export as markdown
- `ExportToMarkdownWithOptions(id string, opts ExportOptions) (string, error)`
- `ExportToJSON(id string) ([]byte, error)` - Export as JSON
- `ExportToJSONWithOptions(id string, opts ExportOptions) ([]byte, error)`
- `SearchConversations(query string, searchContent bool) ([]SearchResult, error)`

## CLI Commands (`internal/commands/history.go`)

```bash
geminiweb history list              # List all conversations
geminiweb history list --favorites  # List only favorites
geminiweb history show @last        # Show most recent
geminiweb history show 1            # Show by index
geminiweb history delete @last      # Delete with confirmation
geminiweb history delete 1 --force  # Delete without confirmation
geminiweb history rename 1 "Title"  # Rename conversation
geminiweb history favorite @last    # Toggle favorite
geminiweb history export @last -o chat.md   # Export as markdown
geminiweb history export @last -o chat.json # Export as JSON
geminiweb history search "API"      # Search titles
geminiweb history search "error" --content  # Search content too
```

## TUI Integration

### FullHistoryStore Interface (`internal/tui/model.go`)

```go
type FullHistoryStore interface {
    HistoryStoreInterface
    ListConversations() ([]*history.Conversation, error)
    GetConversation(id string) (*history.Conversation, error)
    CreateConversation(model string) (*history.Conversation, error)
    DeleteConversation(id string) error
    ToggleFavorite(id string) (bool, error)
    MoveConversation(id string, newIndex int) error
    SwapConversations(id1, id2 string) error
    ExportToMarkdown(id string) (string, error)
    // Note: ExportToJSON not in interface yet
}
```

### TUI Commands
- `/history` - Open history selector
- `/manage` - Open full history manager  
- `/favorite` - Toggle favorite on current conversation
- `/new` - Start new conversation

## Storage Location

Conversations stored in: `~/.geminiweb/history/`
- Each conversation: `<uuid>.json`
- Metadata: `meta.json`
