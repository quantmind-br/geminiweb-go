# Code Style and Conventions

## Go Version
- Go 1.23+

## Code Formatting
- Use `go fmt` and `gofumpt` for formatting
- Run `make fmt` before committing

## Import Organization

Standard order with blank lines:
1. Standard library
2. External dependencies
3. Internal packages

```go
import (
    "context"
    "fmt"
    "sync"
    "time"

    tls_client "github.com/bogdanfinn/tls-client"
    "github.com/tidwall/gjson"

    "github.com/diogo/geminiweb/internal/config"
    "github.com/diogo/geminiweb/internal/models"
)
```

## Error Handling

- Wrap errors with context using `fmt.Errorf("...: %w", err)`
- Use custom error types in `internal/errors/` for domain-specific errors
- Error types: `AuthError`, `APIError`, `TimeoutError`, `UsageLimitError`, `ModelError`, `BlockedError`, `ParseError`
- Constructor pattern: `NewAuthError(message string) *AuthError`

## Struct Definitions

```go
type GeminiClient struct {
    httpClient      tls_client.HttpClient
    cookies         *config.Cookies
    accessToken     string
    model           models.Model
    rotator         *CookieRotator
    autoRefresh     bool
    refreshInterval time.Duration
    mu              sync.RWMutex
    closed          bool
}
```

## Functional Options Pattern

Use for configurable constructors:

```go
type ClientOption func(*GeminiClient)

func WithModel(model models.Model) ClientOption {
    return func(c *GeminiClient) {
        c.model = model
    }
}

func WithAutoRefresh(enabled bool) ClientOption {
    return func(c *GeminiClient) {
        c.autoRefresh = enabled
    }
}

func WithRefreshInterval(interval time.Duration) ClientOption {
    return func(c *GeminiClient) {
        c.refreshInterval = interval
    }
}

func NewClient(cookies *config.Cookies, opts ...ClientOption) (*GeminiClient, error) {
    client := &GeminiClient{
        cookies:         cookies,
        model:           models.Model25Flash,
        autoRefresh:     true,
        refreshInterval: 9 * time.Minute,
    }
    for _, opt := range opts {
        opt(client)
    }
    return client, nil
}
```

## JSON Parsing

Use `tidwall/gjson` for JSON parsing:

```go
result := gjson.Get(jsonString, "path.to.value")
if result.Exists() {
    value := result.String()
}

// Get nested values
safetyRatings := gjson.Get(jsonString, "candidates.0.content.parts.0.safetyRatings")
safetyRatings.ForEach(func(key, value gjson.Result) bool {
    // Process each safety rating
    return true
})
```

## HTTP Requests

Use `bogdanfinn/tls-client` (NOT standard http client) for browser-like requests:

```go
httpClient, err := tls_client.NewHttpClient(
    tls_client.NewNoopLogger(),
    tls_client.WithTimeoutSeconds(300),
    tls_client.WithClientProfile(profiles.Chrome_120),
)
```

## Struct Tags and JSON

```go
type Cookies struct {
    Secure1PSID    string `json:"secure_1psid,omitempty"`
    Secure1PSIDTS  string `json:"secure_1psidts,omitempty"`
}

type Persona struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    SystemPrompt string   `json:"system_prompt"`
    CreatedAt   time.Time `json:"created_at"`
}
```

## Context Usage

Always pass context explicitly:

```go
func (c *GeminiClient) GenerateContent(ctx context.Context, prompt string) (*models.ModelOutput, error) {
    ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
    defer cancel()
    
    // Use ctx in requests
}
```

## Naming Conventions

- Use MixedCaps or mixedCaps for multi-word names
- Acronyms should be all caps: `HTTP`, `URL`, `ID`
- Constants: `EndpointGoogle`, `Model25Flash`, `ErrAuthFailed`
- Structs: `GeminiClient`, `AuthError`, `ModelOutput`
- Functions: `NewClient`, `GenerateContent`, `StartChat`
- Private fields: `httpClient`, `accessToken`, `mu`
- Public methods: `Init()`, `GenerateContent()`, `Close()`

## Test Patterns

Tests use `unittest.IsolatedAsyncioTestCase`:

```go
func TestGeminiClientInit(t *testing.T) {
    cookies := &config.Cookies{
        Secure1PSID:   os.Getenv("SECURE_1PSID"),
        Secure1PSIDTS: os.Getenv("SECURE_1PSIDTS"),
    }
    
    client, err := api.NewClient(cookies)
    require.NoError(t, err)
    defer client.Close()
    
    err = client.Init()
    require.NoError(t, err)
    assert.NotEmpty(t, client.GetAccessToken())
}
```
