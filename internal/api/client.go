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

// BrowserCookieExtractor is an interface for extracting cookies from browsers
type BrowserCookieExtractor interface {
	ExtractGeminiCookies(ctx context.Context, browser browser.SupportedBrowser) (*browser.ExtractResult, error)
}

// GeminiClientInterface defines the interface for GeminiClient to enable dependency injection and mocking
type GeminiClientInterface interface {
	// Core client methods
	Init() error
	Close()
	GetAccessToken() string
	GetCookies() *config.Cookies
	GetModel() models.Model
	SetModel(model models.Model)
	IsClosed() bool

	// Chat methods
	StartChat(model ...models.Model) *ChatSession
	StartChatWithOptions(opts ...ChatOption) *ChatSession

	// Content generation
	GenerateContent(prompt string, opts *GenerateOptions) (*models.ModelOutput, error)
	UploadImage(filePath string) (*UploadedImage, error)
	UploadFile(filePath string) (*UploadedFile, error)

	// Image download
	DownloadImage(img models.WebImage, opts ImageDownloadOptions) (string, error)
	DownloadGeneratedImage(img models.GeneratedImage, opts ImageDownloadOptions) (string, error)
	DownloadAllImages(output *models.ModelOutput, opts ImageDownloadOptions) ([]string, error)
	DownloadSelectedImages(output *models.ModelOutput, indices []int, opts ImageDownloadOptions) ([]string, error)

	// Browser refresh
	RefreshFromBrowser() (bool, error)
	IsBrowserRefreshEnabled() bool

	// Gems methods
	FetchGems(includeHidden bool) (*models.GemJar, error)
	CreateGem(name, prompt, description string) (*models.Gem, error)
	UpdateGem(gemID, name, prompt, description string) (*models.Gem, error)
	DeleteGem(gemID string) error
	Gems() *models.GemJar
	GetGem(id, name string) *models.Gem

	// Batch RPC
	BatchExecute(requests []RPCData) ([]BatchResponse, error)
}

// RefreshFunc is a function type for dependency injection of refresh behavior
type RefreshFunc func() (bool, error)

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
	browserExtractor      BrowserCookieExtractor
	lastBrowserRefresh    time.Time
	browserRefreshMinWait time.Duration
	// Injected dependencies for testing
	refreshFunc  RefreshFunc
	cookieLoader CookieLoader
	// Gems cache
	gems *models.GemJar
	mu   sync.RWMutex
	closed bool
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

// WithBrowserCookieExtractor sets a custom browser cookie extractor
func WithBrowserCookieExtractor(extractor BrowserCookieExtractor) ClientOption {
	return func(c *GeminiClient) {
		c.browserExtractor = extractor
	}
}

// WithRefreshFunc sets a custom refresh function (for testing)
// This allows injecting a mock refresh function to test retry logic
func WithRefreshFunc(fn RefreshFunc) ClientOption {
	return func(c *GeminiClient) {
		c.refreshFunc = fn
	}
}

// WithCookieLoader sets a custom cookie loader function (for testing)
// This allows injecting a mock cookie loader for testing the initial auth flow
func WithCookieLoader(fn CookieLoader) ClientOption {
	return func(c *GeminiClient) {
		c.cookieLoader = fn
	}
}

// WithHTTPClient sets a custom HTTP client (for testing)
// This allows injecting a mock HTTP client to test HTTP interactions
func WithHTTPClient(client tls_client.HttpClient) ClientOption {
	return func(c *GeminiClient) {
		c.httpClient = client
	}
}

// CookieLoader is a function type for loading cookies (for dependency injection)
type CookieLoader func() (*config.Cookies, error)

// NewClient creates a new GeminiClient
// cookies can be nil - in this case, Init() will attempt to load cookies from disk
// or extract them from the browser if browserRefresh is enabled
func NewClient(cookies *config.Cookies, opts ...ClientOption) (*GeminiClient, error) {
	// Validate cookies only if provided (non-nil)
	if cookies != nil {
		if err := config.ValidateCookies(cookies); err != nil {
			return nil, err
		}
	}

	client := &GeminiClient{
		cookies:               cookies,
		model:                 models.DefaultModel, // Default: gemini-2.5-flash (widely available)
		autoRefresh:           true,
		refreshInterval:       9 * time.Minute,  // Default: 9 minutes
		browserRefreshMinWait: 30 * time.Second, // Minimum wait between browser refreshes
		cookieLoader:          config.LoadCookies,
	}

	// Apply options first (allows injecting custom HTTP client)
	for _, opt := range opts {
		opt(client)
	}

	// Create default TLS client only if not injected via options
	if client.httpClient == nil {
		// Create TLS client with Chrome profile for browser emulation
		// Using Chrome_133 (latest available) for better fingerprint compatibility
		options := []tls_client.HttpClientOption{
			tls_client.WithTimeoutSeconds(300),
			tls_client.WithClientProfile(profiles.Chrome_133),
			tls_client.WithNotFollowRedirects(),
		}

		httpClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client: %w", err)
		}
		client.httpClient = httpClient
	}

	return client, nil
}

// Init initializes the client by:
// 1. Attempting to authenticate (load cookies from disk or browser)
// 2. Fetching the access token (with browser fallback on auth failure)
// 3. Starting cookie rotation if enabled
func (c *GeminiClient) Init() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	// Step 1: Ensure we have valid cookies (from constructor, disk, or browser)
	if err := c.attemptInitialAuth(); err != nil {
		return err
	}

	// Step 2: Get access token
	token, err := GetAccessToken(c.httpClient, c.cookies)
	if err != nil {
		// If access token fails (likely expired cookies), try browser refresh as fallback
		// This handles the case where cookies exist on disk but are expired
		browserType := c.browserRefreshType
		if browserType == "" {
			browserType = browser.BrowserAuto
		}

		if refreshErr := c.initialBrowserRefresh(browserType); refreshErr != nil {
			// Return original error if browser refresh also fails
			return fmt.Errorf("authentication failed: %w (browser refresh also failed: %v)", err, refreshErr)
		}

		// Retry getting access token with fresh cookies
		token, err = GetAccessToken(c.httpClient, c.cookies)
		if err != nil {
			return fmt.Errorf("authentication failed after browser refresh: %w", err)
		}
	}
	c.accessToken = token

	// Step 3: Start cookie rotation if enabled
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

// attemptInitialAuth tries to authenticate the client by:
// 1. First, trying to load cookies from disk (if cookies are nil)
// 2. If that fails or cookies are invalid, trying browser extraction (if enabled)
// This method does NOT use rate-limiting for browser refresh as it's part of initialization
// The caller must hold c.mu.Lock()
func (c *GeminiClient) attemptInitialAuth() error {
	// If cookies are already provided and valid, just return
	if c.cookies != nil && c.cookies.Secure1PSID != "" {
		return nil
	}

	// Step 1: Try to load cookies from disk
	if c.cookieLoader != nil {
		cookies, err := c.cookieLoader()
		if err == nil && cookies != nil && cookies.Secure1PSID != "" {
			c.cookies = cookies
			return nil
		}
		// LoadCookies failed, continue to browser refresh
	}

	// Step 2: Try browser extraction (without rate limiting - it's initialization)
	// Use browserRefreshType if set, otherwise use "auto"
	browserType := c.browserRefreshType
	if browserType == "" {
		browserType = browser.BrowserAuto
	}

	err := c.initialBrowserRefresh(browserType)
	if err != nil {
		return fmt.Errorf("authentication failed: cookies not found and browser extraction failed: %w", err)
	}

	return nil
}

// initialBrowserRefresh extracts cookies from the browser during initialization
// This method does NOT enforce rate limiting (unlike RefreshFromBrowser)
// because it's part of the initial authentication flow
// The caller must hold c.mu.Lock()
func (c *GeminiClient) initialBrowserRefresh(browserType browser.SupportedBrowser) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use injected extractor if available, otherwise use default implementation
	var result *browser.ExtractResult
	var err error

	if c.browserExtractor != nil {
		result, err = c.browserExtractor.ExtractGeminiCookies(ctx, browserType)
	} else {
		result, err = browser.ExtractGeminiCookies(ctx, browserType)
	}

	if err != nil {
		return fmt.Errorf("failed to extract cookies from browser: %w", err)
	}

	// Update cookies
	c.cookies = result.Cookies

	// Save cookies to disk for next time
	if err := config.SaveCookies(c.cookies); err != nil {
		// Log but don't fail - cookies are updated in memory
		fmt.Printf("Warning: failed to save cookies to disk: %v\n", err)
	}

	return nil
}

// RefreshFromBrowser attempts to refresh cookies by extracting them from the browser
// Returns true if cookies were successfully refreshed
func (c *GeminiClient) RefreshFromBrowser() (bool, error) {
	// PHASE 1: Check preconditions with read lock
	c.mu.RLock()
	if !c.browserRefresh {
		c.mu.RUnlock()
		return false, fmt.Errorf("browser refresh is not enabled")
	}
	if time.Since(c.lastBrowserRefresh) < c.browserRefreshMinWait {
		waitTime := c.browserRefreshMinWait - time.Since(c.lastBrowserRefresh)
		c.mu.RUnlock()
		return false, fmt.Errorf("browser refresh attempted too recently, wait %v", waitTime)
	}
	browserType := c.browserRefreshType
	extractor := c.browserExtractor
	c.mu.RUnlock()

	// PHASE 2: Network operations WITHOUT lock
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var result *browser.ExtractResult
	var err error
	if extractor != nil {
		result, err = extractor.ExtractGeminiCookies(ctx, browserType)
	} else {
		result, err = browser.ExtractGeminiCookies(ctx, browserType)
	}

	if err != nil {
		// Update timestamp even on error for rate limiting
		c.mu.Lock()
		c.lastBrowserRefresh = time.Now()
		c.mu.Unlock()
		return false, fmt.Errorf("failed to extract cookies from browser: %w", err)
	}

	// PHASE 3: Update state with write lock
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check rate limit (double-check locking pattern)
	if time.Since(c.lastBrowserRefresh) < c.browserRefreshMinWait {
		return false, fmt.Errorf("browser refresh completed by another goroutine")
	}

	// Update cookies atomically
	c.cookies.SetBoth(result.Cookies.Secure1PSID, result.Cookies.Secure1PSIDTS)
	c.lastBrowserRefresh = time.Now()

	// Save updated cookies to disk
	if err := config.SaveCookies(c.cookies); err != nil {
		// Log but don't fail - cookies are updated in memory
		fmt.Printf("Warning: failed to save refreshed cookies to disk: %v\n", err)
	}

	// Re-fetch access token with new cookies
	// Note: This HTTP request is still under lock, but it's short
	token, err := GetAccessToken(c.httpClient, c.cookies)
	if err != nil {
		return false, fmt.Errorf("failed to get access token with new cookies: %w", err)
	}
	c.accessToken = token

	return true, nil
}

// ChatOption configura uma ChatSession
type ChatOption func(*ChatSession)

// WithChatModel define o modelo para a sessão
func WithChatModel(model models.Model) ChatOption {
	return func(s *ChatSession) {
		s.model = model
	}
}

// WithGem define o gem para a sessão (usando objeto Gem)
func WithGem(gem *models.Gem) ChatOption {
	return func(s *ChatSession) {
		if gem != nil {
			s.gemID = gem.ID
		}
	}
}

// WithGemID define o gem para a sessão (usando ID direto)
func WithGemID(gemID string) ChatOption {
	return func(s *ChatSession) {
		s.gemID = gemID
	}
}

// StartChatWithOptions cria uma nova sessão de chat com opções
func (c *GeminiClient) StartChatWithOptions(opts ...ChatOption) *ChatSession {
	session := &ChatSession{
		client: c,
		model:  c.GetModel(),
	}
	for _, opt := range opts {
		opt(session)
	}
	return session
}
