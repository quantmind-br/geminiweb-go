# GeminiClient Lifecycle and Architecture

## Client Initialization

```go
import (
    "github.com/diogo/geminiweb/internal/api"
    "github.com/diogo/geminiweb/internal/config"
    "github.com/diogo/geminiweb/internal/models"
)

// Load cookies from file or browser export
cookies, err := config.LoadCookies()
if err != nil {
    log.Fatal(err)
}

// Create client with options
client, err := api.NewClient(
    cookies,
    api.WithModel(models.Model25Flash),
    api.WithAutoRefresh(true),
    api.WithRefreshInterval(9 * time.Minute),
    api.WithBrowserRefresh(browser.BrowserAuto), // Auto-refresh cookies from browser on auth failure
)
if err != nil {
    log.Fatal(err)
}

// Initialize: fetches access token (SNlM0e) from Gemini
err = client.Init()
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

## Single Query

```go
response, err := client.GenerateContent(ctx, "your prompt")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response.Text)
```

## Chat Session

```go
chat := client.StartChat()
response, err := chat.SendMessage(ctx, "hello")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response.Text)

// Continue conversation
response, err = chat.SendMessage(ctx, "follow up")
```

## Key Patterns

### Functional Options Pattern

The client uses functional options for configuration:

```go
type ClientOption func(*GeminiClient)

func WithModel(model models.Model) ClientOption {
    return func(c *GeminiClient) {
        c.model = model
    }
}
```

## Available Models

- `models.Model25Flash` - Fast model (gemini-2.5-flash)
- `models.Model25Pro` - Balanced model (gemini-2.5-pro)
- `models.Model30Pro` - Advanced model (gemini-3.0-pro)

## Key Components

- **GeminiClient**: Main client struct, manages HTTP client, cookies, access token
- **ChatSession**: Multi-turn conversation handler
- **CookieRotator**: Background goroutine for auto-refreshing cookies
- **Token extraction**: Parses SNlM0e token from Gemini homepage
- **Persona management**: Custom GPT-like personas with CRUD operations
- **BrowserExtractor**: Cookie extraction from installed browsers (Chrome, Firefox, Edge, etc.)

## Browser Cookie Refresh

The client supports automatic cookie refresh from the user's browser when authentication fails:

```go
// Enable browser refresh in client
client, err := api.NewClient(
    cookies,
    api.WithBrowserRefresh(browser.BrowserAuto), // or BrowserChrome, BrowserFirefox, etc.
)

// Manual refresh (if needed)
refreshed, err := client.RefreshFromBrowser()
if refreshed {
    fmt.Println("Cookies refreshed from browser")
}
```

When `GenerateContent` receives a 401 error and browser refresh is enabled, it automatically:
1. Extracts fresh cookies from the specified browser
2. Updates the client's cookies
3. Retries the failed request

Rate limiting: Minimum 30 seconds between refresh attempts to avoid excessive browser access.

## Error Types

- `errors.AuthError` - Authentication failures
- `errors.APIError` - API request failures
- `errors.TimeoutError` - Request timeouts
- `errors.UsageLimitError` - Rate limiting
- `errors.ModelError` - Model-related errors
- `errors.BlockedError` - IP/account blocks
- `errors.ParseError` - Response parsing failures
