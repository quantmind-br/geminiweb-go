package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"github.com/diogo/geminiweb/internal/browser"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
)

// GeminiClient is the main client for interacting with Gemini Web API
type GeminiClient struct {
	httpClient      tls_client.HttpClient
	cookies         *config.Cookies
	accessToken     string
	model           models.Model
	rotator         *CookieRotator
	autoRefresh     bool
	refreshInterval time.Duration
	// Browser-based cookie refresh
	browserRefresh        bool
	browserRefreshType    browser.SupportedBrowser
	lastBrowserRefresh    time.Time
	browserRefreshMinWait time.Duration
	mu                    sync.RWMutex
	closed                bool
}

// ClientOption is a function that configures the client
type ClientOption func(*GeminiClient)

// WithModel sets the default model for the client
func WithModel(model models.Model) ClientOption {
	return func(c *GeminiClient) {
		c.model = model
	}
}

// WithAutoRefresh enables automatic cookie refresh
func WithAutoRefresh(enabled bool) ClientOption {
	return func(c *GeminiClient) {
		c.autoRefresh = enabled
	}
}

// WithRefreshInterval sets the cookie refresh interval
func WithRefreshInterval(interval time.Duration) ClientOption {
	return func(c *GeminiClient) {
		c.refreshInterval = interval
	}
}

// WithBrowserRefresh enables automatic cookie refresh from browser when auth fails
// browserType can be "auto", "chrome", "firefox", "edge", "chromium", "opera"
func WithBrowserRefresh(browserType browser.SupportedBrowser) ClientOption {
	return func(c *GeminiClient) {
		c.browserRefresh = true
		c.browserRefreshType = browserType
	}
}

// NewClient creates a new GeminiClient
func NewClient(cookies *config.Cookies, opts ...ClientOption) (*GeminiClient, error) {
	// Validate cookies
	if err := config.ValidateCookies(cookies); err != nil {
		return nil, err
	}

	// Create TLS client with Chrome profile for browser emulation
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(300),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithNotFollowRedirects(),
	}

	httpClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	client := &GeminiClient{
		httpClient:            httpClient,
		cookies:               cookies,
		model:                 models.Model25Flash,
		autoRefresh:           true,
		refreshInterval:       9 * time.Minute,  // Default: 9 minutes
		browserRefreshMinWait: 30 * time.Second, // Minimum wait between browser refreshes
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// Init initializes the client by fetching the access token
func (c *GeminiClient) Init() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	// Get access token
	token, err := GetAccessToken(c.httpClient, c.cookies)
	if err != nil {
		return err
	}
	c.accessToken = token

	// Start cookie rotation if enabled
	if c.autoRefresh {
		c.rotator = NewCookieRotator(c.httpClient, c.cookies, c.refreshInterval)
		c.rotator.Start()
	}

	return nil
}

// Close shuts down the client and stops background tasks
func (c *GeminiClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true

	if c.rotator != nil {
		c.rotator.Stop()
	}
}

// GetAccessToken returns the current access token
func (c *GeminiClient) GetAccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

// GetCookies returns the current cookies
func (c *GeminiClient) GetCookies() *config.Cookies {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cookies
}

// GetHTTPClient returns the underlying HTTP client
func (c *GeminiClient) GetHTTPClient() tls_client.HttpClient {
	return c.httpClient
}

// GetModel returns the default model
func (c *GeminiClient) GetModel() models.Model {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.model
}

// SetModel sets the default model
func (c *GeminiClient) SetModel(model models.Model) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.model = model
}

// IsClosed returns whether the client is closed
func (c *GeminiClient) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// StartChat creates a new chat session
func (c *GeminiClient) StartChat(model ...models.Model) *ChatSession {
	m := c.GetModel()
	if len(model) > 0 {
		m = model[0]
	}

	return &ChatSession{
		client: c,
		model:  m,
	}
}

// IsBrowserRefreshEnabled returns whether browser refresh is enabled
func (c *GeminiClient) IsBrowserRefreshEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.browserRefresh
}

// RefreshFromBrowser attempts to refresh cookies by extracting them from the browser
// Returns true if cookies were successfully refreshed
func (c *GeminiClient) RefreshFromBrowser() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.browserRefresh {
		return false, fmt.Errorf("browser refresh is not enabled")
	}

	// Rate limit browser refresh attempts
	if time.Since(c.lastBrowserRefresh) < c.browserRefreshMinWait {
		return false, fmt.Errorf("browser refresh attempted too recently, wait %v", c.browserRefreshMinWait-time.Since(c.lastBrowserRefresh))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := browser.ExtractGeminiCookies(ctx, c.browserRefreshType)
	if err != nil {
		c.lastBrowserRefresh = time.Now()
		return false, fmt.Errorf("failed to extract cookies from browser: %w", err)
	}

	// Update cookies
	c.cookies.Secure1PSID = result.Cookies.Secure1PSID
	c.cookies.Secure1PSIDTS = result.Cookies.Secure1PSIDTS
	c.lastBrowserRefresh = time.Now()

	// Save updated cookies to disk
	if err := config.SaveCookies(c.cookies); err != nil {
		// Log but don't fail - cookies are updated in memory
		fmt.Printf("Warning: failed to save refreshed cookies to disk: %v\n", err)
	}

	// Re-fetch access token with new cookies
	token, err := GetAccessToken(c.httpClient, c.cookies)
	if err != nil {
		return false, fmt.Errorf("failed to get access token with new cookies: %w", err)
	}
	c.accessToken = token

	return true, nil
}
