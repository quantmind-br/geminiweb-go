## Context

You are an AI assistant helping with software development tasks.

**Current Date:** 2025-12-02 13:01:40

---

## Task Description

Otimize o código da aplicação removendo dead code, código legado, modifique o que for necessário para melhorar o desempenho e organização do código.

---

## Project Constraints & Rules



---

## Project Structure

└── geminiweb-go/
    ├── build/
    ├── cmd/
    ├── internal/
    │   ├── api/
    │   │   ├── batch.go [5.3KB]
    │   │   ├── client.go [13.2KB]
    │   │   ├── gems.go [6.9KB]
    │   │   ├── generate.go [10.7KB]
    │   │   ├── paths.go [1009B]
    │   │   ├── rotate.go [3.3KB]
    │   │   ├── session.go [2.8KB]
    │   │   ├── token.go [2.3KB]
    │   │   └── upload.go [10.9KB]
    │   ├── browser/
    │   │   └── browser.go [6.4KB]
    │   ├── commands/
    │   │   ├── autologin.go [3.9KB]
    │   │   ├── chat.go [1.5KB]
    │   │   ├── config.go [332B]
    │   │   ├── gems.go [8.3KB]
    │   │   ├── history.go [3.5KB]
    │   │   ├── import.go [919B]
    │   │   ├── persona.go [4.3KB]
    │   │   ├── query.go [12.1KB]
    │   │   └── root.go [4.2KB]
    │   ├── config/
    │   │   ├── config.go [3.8KB]
    │   │   ├── cookies.go [3.8KB]
    │   │   └── personas.go [6.9KB]
    │   ├── errors/
    │   │   └── errors.go [24.9KB]
    │   ├── history/
    │   │   └── store.go [6.4KB]
    │   ├── models/
    │   │   ├── constants.go [3.5KB]
    │   │   ├── gems.go [1.6KB]
    │   │   ├── message.go [150B]
    │   │   └── response.go [2.3KB]
    │   ├── render/
    │   │   ├── themes/
    │   │   │   ├── catppuccin.json [3.5KB]
    │   │   │   ├── dark.json [3.5KB]
    │   │   │   └── tokyonight.json [3.5KB]
    │   │   ├── cache.go [3.1KB]
    │   │   ├── config.go [1.1KB]
    │   │   ├── options.go [1.8KB]
    │   │   ├── render.go [633B]
    │   │   └── themes.go [3.4KB]
    │   └── tui/
    │       ├── config_model.go [14.8KB]
    │       ├── model.go [14.6KB]
    │       └── styles.go [6.2KB]
    ├── AGENTS.md [1.2KB]
    ├── CLAUDE.md [4.5KB]
    ├── Makefile [2.5KB]
    ├── README.md [3.7KB]
    ├── go.mod [3.2KB]
    ├── go.sum [17.3KB]
    └── plan.md [40.7KB]

<file path="internal/api/batch.go">
package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/tidwall/gjson"

	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// RPCData representa uma chamada RPC individual para batch execute
type RPCData struct {
	RPCID      string // ID do método RPC (ex: "CNgdBe" para listar gems)
	Payload    string // JSON payload como string
	Identifier string // Identificador para match na resposta
}

// Serialize converte RPCData para o formato esperado pela API Google
// Formato: [rpcid, payload, null, identifier]
func (r *RPCData) Serialize() []interface{} {
	return []interface{}{r.RPCID, r.Payload, nil, r.Identifier}
}

// BatchResponse representa uma resposta individual do batch execute
type BatchResponse struct {
	Identifier string // Identifier que foi enviado na requisição
	Data       string // JSON string com os dados da resposta
	Error      error  // Erro se houver falha nesta operação específica
}

// BatchExecute executa múltiplas chamadas RPC em uma única requisição HTTP
// Este é o método central para todas as operações de Gems
func (c *GeminiClient) BatchExecute(requests []RPCData) ([]BatchResponse, error) {
	if c.IsClosed() {
		return nil, fmt.Errorf("client is closed")
	}

	if len(requests) == 0 {
		return nil, fmt.Errorf("no requests provided")
	}

	// Construir array de requisições serializadas
	// Formato final: [[[rpc1], [rpc2], ...]] - nota: 3 níveis de colchetes
	var serialized []interface{}
	for _, req := range requests {
		serialized = append(serialized, req.Serialize())
	}

	// Wrap in outer array: [[...]] -> [[[...]]]
	outerPayload := []interface{}{serialized}

	payload, err := json.Marshal(outerPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch payload: %w", err)
	}

	// Criar form data (igual ao generate)
	form := url.Values{}
	form.Set("at", c.GetAccessToken())
	form.Set("f.req", string(payload))

	req, err := http.NewRequest(
		http.MethodPost,
		models.EndpointBatchExec,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Usar mesmos headers do generate
	for key, value := range models.DefaultHeaders() {
		req.Header.Set(key, value)
	}

	// Set cookies
	cookies := c.GetCookies()
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
	if cookies.Secure1PSIDTS != "" {
		req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, apierrors.NewNetworkErrorWithEndpoint("batch execute", models.EndpointBatchExec, err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		// Read response body for error diagnostics
		errorBody := make([]byte, 0, 4096)
		buf := make([]byte, 1024)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				errorBody = append(errorBody, buf[:n]...)
				if len(errorBody) >= 4096 {
					break
				}
			}
			if readErr != nil {
				break
			}
		}
		return nil, apierrors.NewAPIErrorWithBody(resp.StatusCode, models.EndpointBatchExec, "batch execute failed", string(errorBody))
	}

	// Ler body completo
	body := make([]byte, 0, 65536)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return parseBatchResponse(body, requests)
}

// parseBatchResponse analisa a resposta do batch execute
// Formato da resposta:
// )]}'
// [["wrb.fr","RPCID","data_json",null,null,null,"identifier"],...]
func parseBatchResponse(body []byte, requests []RPCData) ([]BatchResponse, error) {
	lines := strings.Split(string(body), "\n")
	var jsonLine string

	// Pular linhas de lixo (como ")]}'" ou vazias) e encontrar JSON válido
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == ")]}" || line == ")]}'" {
			continue
		}
		if gjson.Valid(line) {
			jsonLine = line
			break
		}
	}

	if jsonLine == "" {
		return nil, apierrors.NewParseError("no valid JSON in batch response", "")
	}

	parsed := gjson.Parse(jsonLine)

	// Criar respostas iniciais
	responses := make([]BatchResponse, len(requests))
	for i, req := range requests {
		responses[i] = BatchResponse{Identifier: req.Identifier}
	}

	// Iterar sobre as partes da resposta e fazer match por identifier
	parsed.ForEach(func(_, part gjson.Result) bool {
		if !part.IsArray() {
			return true
		}

		arr := part.Array()
		if len(arr) < 3 {
			return true
		}

		// Extrair dados (posição 2 contém o JSON string)
		data := ""
		if arr[2].Type == gjson.String {
			data = arr[2].String()
		}

		// Encontrar identifier (procurar nas últimas posições)
		var identifier string
		for i := len(arr) - 1; i >= 3; i-- {
			if arr[i].Type == gjson.String && arr[i].String() != "" {
				candidateID := arr[i].String()
				// Verificar se é um identifier conhecido
				for _, req := range requests {
					if candidateID == req.Identifier {
						identifier = candidateID
						break
					}
				}
				if identifier != "" {
					break
				}
			}
		}

		// Atualizar resposta correspondente
		if identifier != "" {
			for i, resp := range responses {
				if resp.Identifier == identifier {
					responses[i].Data = data
					break
				}
			}
		}

		return true
	})

	return responses, nil
}
</file>
<file path="internal/api/client.go">
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
		model:                 models.Model30Pro, // Default model: gemini-3.0-pro
		autoRefresh:           true,
		refreshInterval:       9 * time.Minute,  // Default: 9 minutes
		browserRefreshMinWait: 30 * time.Second, // Minimum wait between browser refreshes
		cookieLoader:          config.LoadCookies,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
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

	// Use injected extractor if available, otherwise use default implementation
	var result *browser.ExtractResult
	var err error

	if c.browserExtractor != nil {
		result, err = c.browserExtractor.ExtractGeminiCookies(ctx, c.browserRefreshType)
	} else {
		result, err = browser.ExtractGeminiCookies(ctx, c.browserRefreshType)
	}

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
</file>
<file path="internal/api/gems.go">
package api

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/diogo/geminiweb/internal/models"
)

// FetchGems carrega todos os gems do servidor Google
// includeHidden: se true, inclui gems de sistema ocultos (não visíveis na UI web)
func (c *GeminiClient) FetchGems(includeHidden bool) (*models.GemJar, error) {
	// Determinar parâmetro para gems de sistema
	systemParam := models.ListGemsNormal
	if includeHidden {
		systemParam = models.ListGemsIncludeHidden
	}

	// Duas requisições RPC em batch:
	// 1. Gems de sistema (predefinidos pelo Google)
	// 2. Gems customizados (criados pelo usuário)
	requests := []RPCData{
		{
			RPCID:      models.RPCListGems,
			Payload:    fmt.Sprintf("[%d]", systemParam),
			Identifier: "system",
		},
		{
			RPCID:      models.RPCListGems,
			Payload:    fmt.Sprintf("[%d]", models.ListGemsCustom),
			Identifier: "custom",
		},
	}

	responses, err := c.BatchExecute(requests)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gems: %w", err)
	}

	jar := make(models.GemJar)

	for _, resp := range responses {
		if resp.Error != nil || resp.Data == "" {
			continue
		}

		predefined := resp.Identifier == "system"
		gems, err := parseGemsResponse(resp.Data, predefined)
		if err != nil {
			// Log mas não falha - pode ter dados parciais
			continue
		}

		for _, gem := range gems {
			jar[gem.ID] = gem
		}
	}

	// Atualizar cache no client
	c.mu.Lock()
	c.gems = &jar
	c.mu.Unlock()

	return &jar, nil
}

// parseGemsResponse analisa a resposta JSON de listagem de gems
func parseGemsResponse(data string, predefined bool) ([]*models.Gem, error) {
	parsed := gjson.Parse(data)
	if !parsed.IsArray() {
		return nil, fmt.Errorf("invalid gems response: not an array")
	}

	// Gems estão na posição [2] do array de resposta
	gemsArray := parsed.Get("2")
	if !gemsArray.Exists() || !gemsArray.IsArray() {
		// Pode não ter gems - não é erro
		return nil, nil
	}

	var gems []*models.Gem
	gemsArray.ForEach(func(_, gemData gjson.Result) bool {
		gem := parseGemData(gemData, predefined)
		if gem != nil {
			gems = append(gems, gem)
		}
		return true
	})

	return gems, nil
}

// parseGemData extrai dados de um gem individual da resposta
// Estrutura do gem no array:
// [0] = ID (string)
// [1][0] = Nome (string)
// [1][1] = Descrição (string)
// [2][0] = Prompt (string, pode não existir)
func parseGemData(data gjson.Result, predefined bool) *models.Gem {
	id := data.Get("0").String()
	if id == "" {
		return nil
	}

	name := data.Get("1.0").String()
	description := data.Get("1.1").String()

	// Prompt pode não existir (posição [2] pode ser null)
	prompt := ""
	promptData := data.Get("2.0")
	if promptData.Exists() {
		prompt = promptData.String()
	}

	return &models.Gem{
		ID:          id,
		Name:        name,
		Description: description,
		Prompt:      prompt,
		Predefined:  predefined,
	}
}

// CreateGem cria um novo gem customizado no servidor
func (c *GeminiClient) CreateGem(name, prompt, description string) (*models.Gem, error) {
	// Payload estruturado com padding específico exigido pela API
	// Formato: [[name, description, prompt, null x5, 0, null, 1, null x3, []]]
	inner := []interface{}{
		name,            // 0: nome
		description,     // 1: descrição
		prompt,          // 2: system prompt
		nil,             // 3
		nil,             // 4
		nil,             // 5
		nil,             // 6
		nil,             // 7
		0,               // 8: flag
		nil,             // 9
		1,               // 10: flag
		nil,             // 11
		nil,             // 12
		nil,             // 13
		[]interface{}{}, // 14: array vazio
	}

	payload, err := json.Marshal([]interface{}{inner})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create payload: %w", err)
	}

	requests := []RPCData{
		{
			RPCID:      models.RPCCreateGem,
			Payload:    string(payload),
			Identifier: "create",
		},
	}

	responses, err := c.BatchExecute(requests)
	if err != nil {
		return nil, fmt.Errorf("failed to create gem: %w", err)
	}

	if len(responses) == 0 || responses[0].Error != nil {
		return nil, fmt.Errorf("failed to create gem: no valid response")
	}

	// Extrair ID do gem criado (posição [0] do response data)
	respData := gjson.Parse(responses[0].Data)
	gemID := respData.Get("0").String()
	if gemID == "" {
		return nil, fmt.Errorf("failed to create gem: no ID in response")
	}

	gem := &models.Gem{
		ID:          gemID,
		Name:        name,
		Description: description,
		Prompt:      prompt,
		Predefined:  false,
	}

	// Atualizar cache
	c.mu.Lock()
	if c.gems != nil {
		(*c.gems)[gemID] = gem
	}
	c.mu.Unlock()

	return gem, nil
}

// UpdateGem atualiza um gem existente
// IMPORTANTE: Deve fornecer todos os campos, mesmo que só queira atualizar um
func (c *GeminiClient) UpdateGem(gemID, name, prompt, description string) (*models.Gem, error) {
	// Payload similar ao create, mas com gem_id na frente e um campo extra
	// Formato: [gem_id, [name, description, prompt, null x5, 0, null, 1, null x3, [], 0]]
	inner := []interface{}{
		name,            // 0
		description,     // 1
		prompt,          // 2
		nil,             // 3
		nil,             // 4
		nil,             // 5
		nil,             // 6
		nil,             // 7
		0,               // 8
		nil,             // 9
		1,               // 10
		nil,             // 11
		nil,             // 12
		nil,             // 13
		[]interface{}{}, // 14
		0,               // 15: flag extra (diferencia de create)
	}

	payload, err := json.Marshal([]interface{}{gemID, inner})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update payload: %w", err)
	}

	requests := []RPCData{
		{
			RPCID:      models.RPCUpdateGem,
			Payload:    string(payload),
			Identifier: "update",
		},
	}

	_, err = c.BatchExecute(requests)
	if err != nil {
		return nil, fmt.Errorf("failed to update gem: %w", err)
	}

	gem := &models.Gem{
		ID:          gemID,
		Name:        name,
		Description: description,
		Prompt:      prompt,
		Predefined:  false,
	}

	// Atualizar cache
	c.mu.Lock()
	if c.gems != nil {
		(*c.gems)[gemID] = gem
	}
	c.mu.Unlock()

	return gem, nil
}

// DeleteGem remove um gem customizado do servidor
func (c *GeminiClient) DeleteGem(gemID string) error {
	payload, err := json.Marshal([]interface{}{gemID})
	if err != nil {
		return fmt.Errorf("failed to marshal delete payload: %w", err)
	}

	requests := []RPCData{
		{
			RPCID:      models.RPCDeleteGem,
			Payload:    string(payload),
			Identifier: "delete",
		},
	}

	_, err = c.BatchExecute(requests)
	if err != nil {
		return fmt.Errorf("failed to delete gem: %w", err)
	}

	// Remover do cache
	c.mu.Lock()
	if c.gems != nil {
		delete(*c.gems, gemID)
	}
	c.mu.Unlock()

	return nil
}

// Gems retorna o cache de gems (nil se FetchGems nunca foi chamado)
func (c *GeminiClient) Gems() *models.GemJar {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gems
}

// GetGem retorna um gem por ID ou nome do cache
func (c *GeminiClient) GetGem(id, name string) *models.Gem {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.gems == nil {
		return nil
	}
	return c.gems.Get(id, name)
}
</file>
<file path="internal/api/generate.go">
package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/tidwall/gjson"

	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// GenerateOptions contains options for content generation
type GenerateOptions struct {
	Model    models.Model
	Metadata []string         // [cid, rid, rcid] for chat context
	Images   []*UploadedImage // Images to include in the prompt
	GemID    string           // ID do gem a usar (server-side persona)
}

// GenerateContent sends a prompt to Gemini and returns the response
func (c *GeminiClient) GenerateContent(prompt string, opts *GenerateOptions) (*models.ModelOutput, error) {
	result, err := c.doGenerateContent(prompt, opts)

	// If auth error and browser refresh is enabled, try to refresh and retry
	if err != nil && c.IsBrowserRefreshEnabled() && isAuthError(err) {
		// Use injected refresh function if available (for testing)
		var refreshed bool
		var refreshErr error

		if c.refreshFunc != nil {
			// Use injected function for testing
			refreshed, refreshErr = c.refreshFunc()
		} else {
			// Use default implementation
			refreshed, refreshErr = c.RefreshFromBrowser()
		}

		if refreshErr == nil && refreshed {
			// Retry the request with new cookies
			return c.doGenerateContent(prompt, opts)
		}
	}

	return result, err
}

// isAuthError checks if an error is an authentication error
// using the centralized error checking function
func isAuthError(err error) bool {
	return apierrors.IsAuthError(err)
}

// doGenerateContent performs the actual content generation request
func (c *GeminiClient) doGenerateContent(prompt string, opts *GenerateOptions) (*models.ModelOutput, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	if c.IsClosed() {
		return nil, fmt.Errorf("client is closed")
	}

	model := c.GetModel()
	var metadata []string
	var images []*UploadedImage
	var gemID string

	if opts != nil {
		if opts.Model.Name != "" {
			model = opts.Model
		}
		metadata = opts.Metadata
		images = opts.Images
		gemID = opts.GemID
	}

	// Build the request payload
	payload, err := buildPayloadWithGem(prompt, metadata, images, gemID)
	if err != nil {
		return nil, fmt.Errorf("failed to build payload: %w", err)
	}

	// Create form data
	form := url.Values{}
	form.Set("at", c.GetAccessToken())
	form.Set("f.req", payload)

	req, err := http.NewRequest(
		http.MethodPost,
		models.EndpointGenerate,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range models.DefaultHeaders() {
		req.Header.Set(key, value)
	}

	// Set model-specific headers
	for key, value := range model.Header {
		req.Header.Set(key, value)
	}

	// Set cookies
	cookies := c.GetCookies()
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
	if cookies.Secure1PSIDTS != "" {
		req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, apierrors.NewNetworkErrorWithEndpoint("generate content", models.EndpointGenerate, err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		// Read response body for error diagnostics
		errorBody := make([]byte, 0, 4096)
		buf := make([]byte, 1024)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				errorBody = append(errorBody, buf[:n]...)
				// Limit error body to 4KB for safety
				if len(errorBody) >= 4096 {
					break
				}
			}
			if readErr != nil {
				break
			}
		}
		return nil, apierrors.NewAPIErrorWithBody(resp.StatusCode, models.EndpointGenerate, "generate content failed", string(errorBody))
	}

	// Read response body
	body := make([]byte, 0, 65536)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return parseResponse(body, model.Name)
}

// buildPayload creates the f.req payload for the generate request
func buildPayload(prompt string, metadata []string) (string, error) {
	return buildPayloadWithGem(prompt, metadata, nil, "")
}

// buildPayloadWithImages creates the f.req payload including file references
// Based on the Python Gemini-API implementation
func buildPayloadWithImages(prompt string, metadata []string, images []*UploadedImage) (string, error) {
	return buildPayloadWithGem(prompt, metadata, images, "")
}

// buildPayloadWithGem creates the f.req payload including file references and gem
// Based on the Python Gemini-API implementation
func buildPayloadWithGem(prompt string, metadata []string, images []*UploadedImage, gemID string) (string, error) {
	// Inner payload structure depends on whether files are included
	var inner []interface{}

	if len(images) > 0 {
		// Build file parts: [[file_id], filename] for each file
		var fileParts []interface{}
		for _, img := range images {
			fileParts = append(fileParts, []interface{}{
				[]interface{}{img.ResourceID}, // File ID wrapped in array
				img.FileName,                  // Original filename
			})
		}

		// With files: [prompt, 0, None, files_array], None, metadata
		inner = []interface{}{
			[]interface{}{
				prompt, // Prompt directly (not in array)
				0,      // Flags/mode
				nil,    // Reserved
				fileParts,
			},
			nil,      // Reserved
			metadata, // Chat metadata [cid, rid, rcid]
		}
	} else {
		// Without files: [[prompt]], None, metadata
		inner = []interface{}{
			[]interface{}{prompt},
			nil,
			metadata,
		}
	}

	// Add gem_id if provided
	// Format: 16 nulls followed by gem_id (position 19 total)
	if gemID != "" {
		for i := 0; i < 16; i++ {
			inner = append(inner, nil)
		}
		inner = append(inner, gemID)
	}

	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return "", err
	}

	// Outer payload: [null, innerJSON]
	outer := []interface{}{
		nil,
		string(innerJSON),
	}

	outerJSON, err := json.Marshal(outer)
	if err != nil {
		return "", err
	}

	return string(outerJSON), nil
}

// parseResponse parses the Gemini API response
func parseResponse(body []byte, modelName string) (*models.ModelOutput, error) {
	// Response has garbage prefix - find first valid JSON line
	var jsonLine string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if gjson.Valid(line) {
			jsonLine = line
			break
		}
	}

	if jsonLine == "" {
		return nil, apierrors.NewParseError("no valid JSON found in response", "")
	}

	parsed := gjson.Parse(jsonLine)

	// Check for alternative error format first
	// Format: [["wrb.fr",null,null,null,null,[3]],...]
	// Error code at position 0.5.0 (first element of the array at position 5)
	altErrorCode := parsed.Get(PathAltErrorCode)
	if altErrorCode.Exists() && !altErrorCode.IsArray() && altErrorCode.Int() > 0 {
		return nil, handleErrorCode(models.ErrorCode(altErrorCode.Int()), modelName)
	}

	// Find the response body
	var responseBody gjson.Result
	var bodyIndex int

	parsed.ForEach(func(key, value gjson.Result) bool {
		bodyData := value.Get(PathBody)
		if !bodyData.Exists() {
			return true
		}

		// Try to parse the body data as JSON
		bodyJSON := gjson.Parse(bodyData.String())
		if bodyJSON.Get(PathCandList).Exists() {
			responseBody = bodyJSON
			bodyIndex = int(key.Int())
			return false
		}
		return true
	})

	if !responseBody.Exists() {
		// Check for error codes in the standard path
		errorCode := parsed.Get(PathErrorCode)
		if errorCode.Exists() {
			return nil, handleErrorCode(models.ErrorCode(errorCode.Int()), modelName)
		}
		return nil, apierrors.NewParseError("no response body found", PathBody)
	}

	// Extract metadata
	metadataResult := responseBody.Get(PathMetadata)
	var metadata []string
	if metadataResult.IsArray() {
		metadataResult.ForEach(func(_, v gjson.Result) bool {
			metadata = append(metadata, v.String())
			return true
		})
	}

	// Extract candidates
	candidateList := responseBody.Get(PathCandList)
	if !candidateList.Exists() || !candidateList.IsArray() {
		return nil, apierrors.NewParseError("no candidates found", PathCandList)
	}

	candidates := []models.Candidate{}
	candidateList.ForEach(func(candIdx, candValue gjson.Result) bool {
		rcid := candValue.Get(PathCandRCID).String()
		if rcid == "" {
			return true // Skip candidates without RCID
		}

		// Extract text
		text := candValue.Get(PathCandText).String()

		// Handle special URL-based text
		if matched, _ := regexp.MatchString(`^http://googleusercontent\.com/card_content/\d+`, text); matched {
			altText := candValue.Get(PathCandTextAlt).String()
			if altText != "" {
				text = altText
			}
		}

		// Extract thoughts
		thoughts := candValue.Get(PathCandThoughts).String()

		// Extract web images
		var webImages []models.WebImage
		candValue.Get(PathCandWebImages).ForEach(func(_, imgValue gjson.Result) bool {
			imgURL := imgValue.Get(PathWebImgURL).String()
			if imgURL == "" {
				return true
			}
			webImages = append(webImages, models.WebImage{
				URL:   imgURL,
				Title: imgValue.Get(PathWebImgTitle).String(),
				Alt:   imgValue.Get(PathWebImgAlt).String(),
			})
			return true
		})

		// Extract generated images
		var generatedImages []models.GeneratedImage
		candValue.Get(PathCandGenImages).ForEach(func(imgIdx, imgValue gjson.Result) bool {
			imgURL := imgValue.Get(PathGenImgURL).String()
			if imgURL == "" {
				return true
			}

			imgNum := imgValue.Get(PathGenImgNum).String()
			title := "[Generated Image]"
			if imgNum != "" {
				title = fmt.Sprintf("[Generated Image %s]", imgNum)
			}

			alts := imgValue.Get(PathGenImgAlts)
			alt := ""
			if alts.IsArray() {
				if altVal := alts.Get(fmt.Sprintf("%d", imgIdx.Int())); altVal.Exists() {
					alt = altVal.String()
				} else if altVal := alts.Get("0"); altVal.Exists() {
					alt = altVal.String()
				}
			}

			generatedImages = append(generatedImages, models.GeneratedImage{
				URL:   imgURL,
				Title: title,
				Alt:   alt,
			})
			return true
		})

		candidates = append(candidates, models.Candidate{
			RCID:            rcid,
			Text:            text,
			Thoughts:        thoughts,
			WebImages:       webImages,
			GeneratedImages: generatedImages,
		})
		return true
	})

	if len(candidates) == 0 {
		return nil, apierrors.NewParseError("no valid candidates found", PathCandList)
	}

	_ = bodyIndex // Used for generated image parsing in extended version

	return &models.ModelOutput{
		Metadata:   metadata,
		Candidates: candidates,
		Chosen:     0,
	}, nil
}

// handleErrorCode converts API error codes to appropriate errors
// using the centralized error handling function
func handleErrorCode(code models.ErrorCode, modelName string) error {
	return apierrors.HandleErrorCode(code, models.EndpointGenerate, modelName)
}
</file>
<file path="internal/api/paths.go">
// Package api provides the Gemini Web API client implementation.
package api

// GJSON paths for extracting values from Gemini responses.
// These centralize the "magic indices" from the Python implementation.
const (
	// Response body paths
	PathBody      = "2"
	PathCandList  = "4"
	PathMetadata  = "1"
	PathErrorCode = "0.5.2.0.1.0"

	// Alternative error path - used when API returns simple error format
	// e.g., [["wrb.fr",null,null,null,null,[3]],...]  - error code at position 0.5.0
	PathAltErrorCode = "0.5.0"

	// Candidate paths (relative to candidate object)
	PathCandRCID      = "0"
	PathCandText      = "1.0"
	PathCandTextAlt   = "22.0"
	PathCandThoughts  = "37.0.0"
	PathCandWebImages = "12.1"
	PathCandGenImages = "12.7.0"

	// Web image paths (relative to web image object)
	PathWebImgURL   = "0.0.0"
	PathWebImgTitle = "7.0"
	PathWebImgAlt   = "0.4"

	// Generated image paths (relative to generated image object)
	PathGenImgURL  = "0.3.3"
	PathGenImgNum  = "3.6"
	PathGenImgAlts = "3.5"
)
</file>
<file path="internal/api/rotate.go">
package api

import (
	"fmt"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"

	"github.com/diogo/geminiweb/internal/config"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// Rate limiting for cookie rotation (1 call per minute max)
var (
	lastRotateTime time.Time
	rotateMutex    sync.Mutex
)

// RotateCookies refreshes the __Secure-1PSIDTS cookie
func RotateCookies(client tls_client.HttpClient, cookies *config.Cookies) (string, error) {
	rotateMutex.Lock()
	defer rotateMutex.Unlock()

	// Rate limit: don't call more than once per minute
	if time.Since(lastRotateTime) < time.Minute {
		return "", nil // Skip if called too recently
	}

	req, err := http.NewRequest(
		http.MethodPost,
		models.EndpointRotateCookies,
		strings.NewReader(`[000,"-0000000000000000000"]`),
	)
	if err != nil {
		return "", apierrors.NewGeminiErrorWithCause("create rotate request", err)
	}

	// Set headers
	for key, value := range models.RotateCookiesHeaders() {
		req.Header.Set(key, value)
	}

	// Set cookies
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
	if cookies.Secure1PSIDTS != "" {
		req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", apierrors.NewNetworkErrorWithEndpoint("rotate cookies", models.EndpointRotateCookies, err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode == 401 {
		return "", apierrors.NewAuthErrorWithEndpoint("unauthorized during cookie rotation", models.EndpointRotateCookies)
	}

	if resp.StatusCode != 200 {
		return "", apierrors.NewAPIError(resp.StatusCode, models.EndpointRotateCookies,
			fmt.Sprintf("cookie rotation failed with status: %d", resp.StatusCode))
	}

	// Update last rotate time
	lastRotateTime = time.Now()

	// Extract new __Secure-1PSIDTS from response cookies
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "__Secure-1PSIDTS" {
			return cookie.Value, nil
		}
	}

	return "", nil
}

// CookieRotator manages background cookie rotation
type CookieRotator struct {
	client   tls_client.HttpClient
	cookies  *config.Cookies
	interval time.Duration
	stopCh   chan struct{}
	running  bool
	mu       sync.Mutex
}

// NewCookieRotator creates a new cookie rotator
func NewCookieRotator(client tls_client.HttpClient, cookies *config.Cookies, interval time.Duration) *CookieRotator {
	return &CookieRotator{
		client:   client,
		cookies:  cookies,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins background cookie rotation
func (r *CookieRotator) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.mu.Unlock()

	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				newToken, err := RotateCookies(r.client, r.cookies)
				if err != nil {
					// Log error but continue
					continue
				}
				if newToken != "" {
					r.cookies.Update1PSIDTS(newToken)
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}

// Stop halts background cookie rotation
func (r *CookieRotator) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		close(r.stopCh)
		r.running = false
	}
}
</file>
<file path="internal/api/session.go">
package api

import (
	"github.com/diogo/geminiweb/internal/models"
)

// ChatSession maintains conversation context across messages
type ChatSession struct {
	client     *GeminiClient
	model      models.Model
	metadata   []string // [cid, rid, rcid]
	lastOutput *models.ModelOutput
	gemID      string // ID do gem associado à sessão (server-side persona)
}

// SendMessage sends a message in the chat session and updates context
func (s *ChatSession) SendMessage(prompt string) (*models.ModelOutput, error) {
	opts := &GenerateOptions{
		Model:    s.model,
		Metadata: s.metadata,
		GemID:    s.gemID,
	}

	output, err := s.client.GenerateContent(prompt, opts)
	if err != nil {
		return nil, err
	}

	// Update session state
	s.lastOutput = output
	s.updateMetadata(output)

	return output, nil
}

// updateMetadata updates the session metadata from the response
func (s *ChatSession) updateMetadata(output *models.ModelOutput) {
	if len(output.Metadata) > 0 {
		s.metadata = make([]string, len(output.Metadata))
		copy(s.metadata, output.Metadata)
	}

	// Update rcid with the chosen candidate's RCID
	if len(s.metadata) >= 3 {
		s.metadata[2] = output.RCID()
	} else if len(s.metadata) == 2 {
		s.metadata = append(s.metadata, output.RCID())
	}
}

// SetMetadata allows setting metadata directly (for resuming conversations)
func (s *ChatSession) SetMetadata(cid, rid, rcid string) {
	s.metadata = []string{cid, rid, rcid}
}

// GetMetadata returns the current session metadata
func (s *ChatSession) GetMetadata() []string {
	return s.metadata
}

// CID returns the conversation ID
func (s *ChatSession) CID() string {
	if len(s.metadata) > 0 {
		return s.metadata[0]
	}
	return ""
}

// RID returns the reply ID
func (s *ChatSession) RID() string {
	if len(s.metadata) > 1 {
		return s.metadata[1]
	}
	return ""
}

// RCID returns the reply candidate ID
func (s *ChatSession) RCID() string {
	if len(s.metadata) > 2 {
		return s.metadata[2]
	}
	return ""
}

// GetModel returns the session's model
func (s *ChatSession) GetModel() models.Model {
	return s.model
}

// SetModel changes the session's model
func (s *ChatSession) SetModel(model models.Model) {
	s.model = model
}

// LastOutput returns the last response from the session
func (s *ChatSession) LastOutput() *models.ModelOutput {
	return s.lastOutput
}

// ChooseCandidate selects a different candidate from the last output
func (s *ChatSession) ChooseCandidate(index int) error {
	if s.lastOutput == nil {
		return nil
	}
	if index >= len(s.lastOutput.Candidates) {
		return nil
	}

	s.lastOutput.Chosen = index
	s.updateMetadata(s.lastOutput)
	return nil
}

// SetGem define o gem para a sessão
func (s *ChatSession) SetGem(gemID string) {
	s.gemID = gemID
}

// GetGemID retorna o gem ID da sessão
func (s *ChatSession) GetGemID() string {
	return s.gemID
}
</file>
<file path="internal/api/token.go">
package api

import (
	"fmt"
	"regexp"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"

	"github.com/diogo/geminiweb/internal/config"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

// SNlM0e pattern for extracting access token from HTML
var snlm0ePattern = regexp.MustCompile(`"SNlM0e":"([^"]+)"`)

// GetAccessToken fetches the SNlM0e access token from gemini.google.com
func GetAccessToken(client tls_client.HttpClient, cookies *config.Cookies) (string, error) {
	req, err := http.NewRequest(http.MethodGet, models.EndpointInit, nil)
	if err != nil {
		return "", apierrors.NewGeminiErrorWithCause("create access token request", err)
	}

	// Set headers
	for key, value := range models.DefaultHeaders() {
		req.Header.Set(key, value)
	}

	// Set cookies
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
	if cookies.Secure1PSIDTS != "" {
		req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", apierrors.NewNetworkErrorWithEndpoint("fetch access token", models.EndpointInit, err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		// Read response body for diagnostics
		errorBody := make([]byte, 0, 2048)
		buf := make([]byte, 512)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				errorBody = append(errorBody, buf[:n]...)
				if len(errorBody) >= 2048 {
					break
				}
			}
			if readErr != nil {
				break
			}
		}

		authErr := apierrors.NewAuthErrorWithEndpoint(
			fmt.Sprintf("failed to fetch access token, status: %d", resp.StatusCode),
			models.EndpointInit,
		)
		authErr.GeminiError.HTTPStatus = resp.StatusCode
		authErr.GeminiError.WithBody(string(errorBody))
		return "", authErr
	}

	// Read response body
	body := make([]byte, 0)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Extract SNlM0e token using regex
	matches := snlm0ePattern.FindSubmatch(body)
	if len(matches) < 2 {
		return "", apierrors.NewAuthErrorWithEndpoint(
			"SNlM0e token not found in response. Cookies may be expired.",
			models.EndpointInit,
		)
	}

	return string(matches[1]), nil
}
</file>
<file path="internal/api/upload.go">
package api

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"

	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
)

const (
	MaxImageSize = 20 * 1024 * 1024 // 20MB
	MaxFileSize  = 50 * 1024 * 1024 // 50MB for text files
)

// SupportedImageTypes returns the list of supported MIME types for image upload
func SupportedImageTypes() []string {
	return []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
}

// SupportedTextTypes returns the list of supported MIME types for text file upload
func SupportedTextTypes() []string {
	return []string{
		"text/plain",
		"text/markdown",
		"text/x-markdown",
		"application/json",
		"text/csv",
		"text/html",
		"text/xml",
		"application/xml",
	}
}

// UploadedFile represents an uploaded file ready for use in prompts
// This can be an image or text file - the API treats them similarly
type UploadedFile struct {
	ResourceID string
	FileName   string
	MIMEType   string
	Size       int64
}

// UploadedImage represents an uploaded image ready for use in prompts
// Deprecated: Use UploadedFile instead
type UploadedImage = UploadedFile

// FileUploader handles file uploads to Gemini (images, text, etc.)
type FileUploader struct {
	client *GeminiClient
}

// NewFileUploader creates a new file uploader
func NewFileUploader(client *GeminiClient) *FileUploader {
	return &FileUploader{
		client: client,
	}
}

// UploadFile uploads any supported file from disk (images or text)
func (u *FileUploader) UploadFile(filePath string) (*UploadedFile, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Detect MIME type
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Determine max size based on file type
	maxSize := int64(MaxFileSize)
	if u.isImageType(mimeType) {
		maxSize = MaxImageSize
	}

	if fileInfo.Size() > maxSize {
		return nil, fmt.Errorf("file size (%d bytes) exceeds maximum (%d bytes)", fileInfo.Size(), maxSize)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	return u.uploadStream(file, filepath.Base(filePath), mimeType, fileInfo.Size())
}

// UploadText uploads text content as a file
func (u *FileUploader) UploadText(content string, fileName string) (*UploadedFile, error) {
	if fileName == "" {
		fileName = "prompt.txt"
	}

	// Ensure .txt extension for proper MIME detection
	if filepath.Ext(fileName) == "" {
		fileName += ".txt"
	}

	data := []byte(content)
	if int64(len(data)) > MaxFileSize {
		return nil, fmt.Errorf("content size (%d bytes) exceeds maximum (%d bytes)", len(data), MaxFileSize)
	}

	mimeType := "text/plain"
	ext := filepath.Ext(fileName)
	if detectedType := mime.TypeByExtension(ext); detectedType != "" {
		mimeType = detectedType
	}

	return u.uploadStream(bytes.NewReader(data), fileName, mimeType, int64(len(data)))
}

// uploadStream executes the actual upload using Google's content-push service
// Based on the Python Gemini-API implementation
func (u *FileUploader) uploadStream(
	reader io.Reader,
	fileName string,
	mimeType string,
	size int64,
) (*UploadedFile, error) {
	// Create multipart body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create form file: %v", err))
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to write file data: %v", err))
	}

	_ = writer.Close()

	// Simple POST to upload endpoint (no URL parameters)
	req, err := fhttp.NewRequest(fhttp.MethodPost, models.EndpointUpload, &body)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create request: %v", err))
	}

	// Headers - only Content-Type and Push-ID are needed
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range models.UploadHeaders() {
		req.Header.Set(key, value)
	}

	// No cookies needed for upload endpoint

	resp, err := u.client.httpClient.Do(req)
	if err != nil {
		return nil, apierrors.NewUploadNetworkError(fileName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, apierrors.NewUploadErrorWithStatus(fileName, resp.StatusCode, string(bodyBytes))
	}

	// Response is plain text containing the file identifier
	// Example: /contrib_service/ttl_1d/1709764705i7wdlyx3mdzndme3a767pluckv4flj
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to read response: %v", err))
	}

	resourceID := strings.TrimSpace(string(respBody))
	if resourceID == "" {
		return nil, apierrors.NewUploadError(fileName, "empty resource ID in upload response")
	}

	return &UploadedFile{
		ResourceID: resourceID,
		FileName:   fileName,
		MIMEType:   mimeType,
		Size:       size,
	}, nil
}

func (u *FileUploader) isImageType(mimeType string) bool {
	for _, supported := range SupportedImageTypes() {
		if strings.HasPrefix(mimeType, supported) {
			return true
		}
	}
	return false
}

func (u *FileUploader) isTextType(mimeType string) bool {
	for _, supported := range SupportedTextTypes() {
		if strings.HasPrefix(mimeType, supported) {
			return true
		}
	}
	return false
}

// ImageUploader handles image uploads to Gemini
// Deprecated: Use FileUploader instead
type ImageUploader struct {
	client *GeminiClient
}

// NewImageUploader creates a new image uploader
// Deprecated: Use NewFileUploader instead
func NewImageUploader(client *GeminiClient) *ImageUploader {
	return &ImageUploader{
		client: client,
	}
}

// UploadFile uploads an image file from disk
func (u *ImageUploader) UploadFile(filePath string) (*UploadedImage, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() > MaxImageSize {
		return nil, fmt.Errorf("file size exceeds maximum %d bytes", MaxImageSize)
	}

	// Detect MIME type
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	if !u.isSupportedType(mimeType) {
		return nil, fmt.Errorf("unsupported image type: %s", mimeType)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	return u.uploadStream(file, filepath.Base(filePath), mimeType, fileInfo.Size())
}

// UploadFromReader uploads from an io.Reader
func (u *ImageUploader) UploadFromReader(
	reader io.Reader,
	fileName string,
	mimeType string,
) (*UploadedImage, error) {
	// Read all content into buffer (needed for multipart)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	if int64(len(data)) > MaxImageSize {
		return nil, fmt.Errorf("data size exceeds maximum %d bytes", MaxImageSize)
	}

	return u.uploadStream(bytes.NewReader(data), fileName, mimeType, int64(len(data)))
}

// uploadStream executes the actual upload using Google's content-push service
// Based on the Python Gemini-API implementation
func (u *ImageUploader) uploadStream(
	reader io.Reader,
	fileName string,
	mimeType string,
	size int64,
) (*UploadedImage, error) {
	// Create multipart body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create form file: %v", err))
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to write file data: %v", err))
	}

	_ = writer.Close()

	// Simple POST to upload endpoint (no URL parameters)
	req, err := fhttp.NewRequest(fhttp.MethodPost, models.EndpointUpload, &body)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create request: %v", err))
	}

	// Headers - only Content-Type and Push-ID are needed
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range models.UploadHeaders() {
		req.Header.Set(key, value)
	}

	// No cookies needed for upload endpoint

	resp, err := u.client.httpClient.Do(req)
	if err != nil {
		return nil, apierrors.NewUploadNetworkError(fileName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, apierrors.NewUploadErrorWithStatus(fileName, resp.StatusCode, string(bodyBytes))
	}

	// Response is plain text containing the file identifier
	// Example: /contrib_service/ttl_1d/1709764705i7wdlyx3mdzndme3a767pluckv4flj
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to read response: %v", err))
	}

	resourceID := strings.TrimSpace(string(respBody))
	if resourceID == "" {
		return nil, apierrors.NewUploadError(fileName, "empty resource ID in upload response")
	}

	return &UploadedImage{
		ResourceID: resourceID,
		FileName:   fileName,
		MIMEType:   mimeType,
		Size:       size,
	}, nil
}

func (u *ImageUploader) isSupportedType(mimeType string) bool {
	for _, supported := range SupportedImageTypes() {
		if strings.HasPrefix(mimeType, supported) {
			return true
		}
	}
	return false
}

func generateUploadID() string {
	return fmt.Sprintf("geminiweb-%d", time.Now().UnixNano())
}

// UploadImage is a convenience method on GeminiClient for uploading images
func (c *GeminiClient) UploadImage(filePath string) (*UploadedImage, error) {
	uploader := NewImageUploader(c)
	return uploader.UploadFile(filePath)
}

// UploadImageFromReader is a convenience method for uploading from a reader
func (c *GeminiClient) UploadImageFromReader(
	reader io.Reader,
	fileName string,
	mimeType string,
) (*UploadedImage, error) {
	uploader := NewImageUploader(c)
	return uploader.UploadFromReader(reader, fileName, mimeType)
}

// UploadFile is a convenience method on GeminiClient for uploading any file
func (c *GeminiClient) UploadFile(filePath string) (*UploadedFile, error) {
	uploader := NewFileUploader(c)
	return uploader.UploadFile(filePath)
}

// UploadText is a convenience method for uploading text content as a file
func (c *GeminiClient) UploadText(content string, fileName string) (*UploadedFile, error) {
	uploader := NewFileUploader(c)
	return uploader.UploadText(content, fileName)
}

// LargePromptThreshold is the size (in bytes) above which prompts should be uploaded as files
const LargePromptThreshold = 100 * 1024 // 100KB
</file>
<file path="internal/browser/browser.go">
// Package browser provides functionality to extract cookies from web browsers.
package browser

import (
	"context"
	"fmt"
	"strings"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/chrome"
	_ "github.com/browserutils/kooky/browser/chromium"
	_ "github.com/browserutils/kooky/browser/edge"
	_ "github.com/browserutils/kooky/browser/firefox"
	_ "github.com/browserutils/kooky/browser/opera"

	"github.com/diogo/geminiweb/internal/config"
)

// SupportedBrowser represents a supported browser type
type SupportedBrowser string

const (
	BrowserAuto     SupportedBrowser = "auto"
	BrowserChrome   SupportedBrowser = "chrome"
	BrowserChromium SupportedBrowser = "chromium"
	BrowserFirefox  SupportedBrowser = "firefox"
	BrowserEdge     SupportedBrowser = "edge"
	BrowserOpera    SupportedBrowser = "opera"
)

// AllSupportedBrowsers returns a list of all supported browsers
func AllSupportedBrowsers() []SupportedBrowser {
	return []SupportedBrowser{
		BrowserChrome,
		BrowserChromium,
		BrowserFirefox,
		BrowserEdge,
		BrowserOpera,
	}
}

// String returns the string representation of the browser
func (b SupportedBrowser) String() string {
	return string(b)
}

// ParseBrowser parses a browser string into a SupportedBrowser
func ParseBrowser(s string) (SupportedBrowser, error) {
	switch strings.ToLower(s) {
	case "auto", "":
		return BrowserAuto, nil
	case "chrome", "google-chrome":
		return BrowserChrome, nil
	case "chromium":
		return BrowserChromium, nil
	case "firefox", "mozilla", "mozilla-firefox":
		return BrowserFirefox, nil
	case "edge", "microsoft-edge", "msedge":
		return BrowserEdge, nil
	case "opera":
		return BrowserOpera, nil
	default:
		return "", fmt.Errorf("unsupported browser: %s. Supported: chrome, chromium, firefox, edge, opera", s)
	}
}

// ExtractResult contains the result of cookie extraction
type ExtractResult struct {
	Cookies     *config.Cookies
	BrowserName string
	StorePath   string
}

// ExtractGeminiCookies extracts Gemini authentication cookies from browsers
func ExtractGeminiCookies(ctx context.Context, browser SupportedBrowser) (*ExtractResult, error) {
	if browser == BrowserAuto {
		return extractFromAllBrowsers(ctx)
	}
	return extractFromBrowser(ctx, browser)
}

// extractFromAllBrowsers tries to extract cookies from all supported browsers
func extractFromAllBrowsers(ctx context.Context) (*ExtractResult, error) {
	// Try browsers in order of popularity
	browsers := []SupportedBrowser{
		BrowserChrome,
		BrowserFirefox,
		BrowserEdge,
		BrowserChromium,
		BrowserOpera,
	}

	var lastErr error
	for _, browser := range browsers {
		result, err := extractFromBrowser(ctx, browser)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("could not find Gemini cookies in any browser: %w", lastErr)
	}
	return nil, fmt.Errorf("could not find Gemini cookies in any supported browser")
}

// extractFromBrowser extracts cookies from a specific browser
// It tries all profiles of the browser until it finds the cookies
func extractFromBrowser(ctx context.Context, browser SupportedBrowser) (*ExtractResult, error) {
	stores := kooky.FindAllCookieStores(ctx)

	var matchingStores []kooky.CookieStore
	var browserName string

	// Collect all stores that match the browser
	for _, store := range stores {
		name := store.Browser()
		nameLower := strings.ToLower(name)

		if matchesBrowser(nameLower, browser) {
			matchingStores = append(matchingStores, store)
			if browserName == "" {
				browserName = name
			}
		} else {
			store.Close()
		}
	}

	if len(matchingStores) == 0 {
		return nil, fmt.Errorf("browser %s not found or no cookie store available", browser)
	}

	// Try each store/profile until we find the cookies
	var lastErr error
	for _, store := range matchingStores {
		result, err := extractCookiesFromStore(ctx, store, browserName, store.Profile())
		store.Close()
		if err == nil {
			// Close remaining stores
			for _, s := range matchingStores {
				s.Close()
			}
			return result, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("browser %s not found or no cookie store available", browser)
}

// matchesBrowser checks if a browser name matches the target browser
func matchesBrowser(browserName string, target SupportedBrowser) bool {
	browserName = strings.ToLower(browserName)

	switch target {
	case BrowserChrome:
		return strings.Contains(browserName, "chrome") && !strings.Contains(browserName, "chromium")
	case BrowserChromium:
		return strings.Contains(browserName, "chromium")
	case BrowserFirefox:
		return strings.Contains(browserName, "firefox")
	case BrowserEdge:
		return strings.Contains(browserName, "edge")
	case BrowserOpera:
		return strings.Contains(browserName, "opera")
	default:
		return false
	}
}

// extractCookiesFromStore extracts Gemini cookies from a specific cookie store
func extractCookiesFromStore(ctx context.Context, store kooky.CookieStore, browserName, profile string) (*ExtractResult, error) {
	// Extract cookies for google.com domain (includes .google.com, .google.com.br, etc.)
	cookies := store.TraverseCookies(
		kooky.Valid,
		kooky.DomainContains("google.com"),
	).OnlyCookies()

	var secure1PSID, secure1PSIDTS string

	for cookie := range cookies {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		switch cookie.Name {
		case "__Secure-1PSID":
			// Prefer .google.com over regional domains
			if secure1PSID == "" || cookie.Domain == ".google.com" {
				secure1PSID = cookie.Value
			}
		case "__Secure-1PSIDTS":
			if secure1PSIDTS == "" || cookie.Domain == ".google.com" {
				secure1PSIDTS = cookie.Value
			}
		}
	}

	displayName := browserName
	if profile != "" {
		displayName = fmt.Sprintf("%s (profile: %s)", browserName, profile)
	}

	if secure1PSID == "" {
		return nil, fmt.Errorf("cookie __Secure-1PSID not found in %s. Please ensure you are logged into gemini.google.com", displayName)
	}

	return &ExtractResult{
		Cookies: &config.Cookies{
			Secure1PSID:   secure1PSID,
			Secure1PSIDTS: secure1PSIDTS,
		},
		BrowserName: displayName,
	}, nil
}

// ListAvailableBrowsers returns a list of browsers that have cookie stores
func ListAvailableBrowsers() []string {
	ctx := context.Background()
	stores := kooky.FindAllCookieStores(ctx)
	var browsers []string

	seen := make(map[string]bool)
	for _, store := range stores {
		name := store.Browser()
		if !seen[name] {
			browsers = append(browsers, name)
			seen[name] = true
		}
		store.Close()
	}

	return browsers
}
</file>
<file path="internal/commands/autologin.go">
package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/browser"
	"github.com/diogo/geminiweb/internal/config"
)

var (
	autoLoginBrowser string
	autoLoginList    bool
)

var autoLoginCmd = &cobra.Command{
	Use:   "auto-login",
	Short: "Extract authentication cookies from browser",
	Long: `Automatically extract Gemini authentication cookies from your browser.

This command reads cookies directly from your browser's cookie store,
eliminating the need to manually export and import cookies.

Supported browsers: chrome, chromium, firefox, edge, opera

IMPORTANT:
- Close the browser before running this command to avoid database locks
- You must be logged into gemini.google.com in the browser
- On macOS, you may be prompted for keychain access (Chrome uses Keychain to encrypt cookies)

Examples:
  geminiweb auto-login              # Auto-detect browser
  geminiweb auto-login -b chrome    # Extract from Chrome
  geminiweb auto-login -b firefox   # Extract from Firefox
  geminiweb auto-login --list       # List available browsers`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if autoLoginList {
			return runListBrowsers()
		}
		return runAutoLogin(autoLoginBrowser)
	},
}

func init() {
	autoLoginCmd.Flags().StringVarP(&autoLoginBrowser, "browser", "b", "auto",
		"Browser to extract cookies from (chrome, chromium, firefox, edge, opera, auto)")
	autoLoginCmd.Flags().BoolVarP(&autoLoginList, "list", "l", false,
		"List available browsers with cookie stores")
}

func runAutoLogin(browserName string) error {
	targetBrowser, err := browser.ParseBrowser(browserName)
	if err != nil {
		return err
	}

	fmt.Println("Extracting cookies from browser...")
	fmt.Println("Note: If the browser is open, you may encounter database lock errors.")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := browser.ExtractGeminiCookies(ctx, targetBrowser)
	if err != nil {
		return fmt.Errorf("failed to extract cookies: %w", err)
	}

	// Validate cookies
	if err := config.ValidateCookies(result.Cookies); err != nil {
		return fmt.Errorf("extracted cookies are invalid: %w", err)
	}

	// Save cookies
	if err := config.SaveCookies(result.Cookies); err != nil {
		return fmt.Errorf("failed to save cookies: %w", err)
	}

	cookiesPath, _ := config.GetCookiesPath()

	fmt.Printf("Successfully extracted cookies from %s\n", result.BrowserName)
	fmt.Printf("Cookies saved to: %s\n", cookiesPath)
	fmt.Println()
	fmt.Println("Extracted cookies:")
	fmt.Printf("  __Secure-1PSID:   %s...\n", truncateValue(result.Cookies.Secure1PSID, 20))
	if result.Cookies.Secure1PSIDTS != "" {
		fmt.Printf("  __Secure-1PSIDTS: %s...\n", truncateValue(result.Cookies.Secure1PSIDTS, 20))
	}
	fmt.Println()
	fmt.Println("You can now use geminiweb to chat with Gemini!")

	return nil
}

func runListBrowsers() error {
	browsers := browser.ListAvailableBrowsers()

	if len(browsers) == 0 {
		fmt.Println("No browsers with cookie stores found.")
		fmt.Println()
		fmt.Println("Supported browsers:")
		for _, b := range browser.AllSupportedBrowsers() {
			fmt.Printf("  - %s\n", b)
		}
		return nil
	}

	fmt.Println("Available browsers with cookie stores:")
	for _, b := range browsers {
		fmt.Printf("  - %s\n", b)
	}
	fmt.Println()
	fmt.Println("Use 'geminiweb auto-login -b <browser>' to extract cookies from a specific browser.")

	return nil
}

func truncateValue(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// GetAutoLoginCmd returns the auto-login command (for testing)
func GetAutoLoginCmd() *cobra.Command {
	return autoLoginCmd
}

// SupportedBrowsersHelp returns a help string listing supported browsers
func SupportedBrowsersHelp() string {
	browsers := browser.AllSupportedBrowsers()
	names := make([]string, len(browsers))
	for i, b := range browsers {
		names[i] = string(b)
	}
	return strings.Join(names, ", ")
}
</file>
<file path="internal/commands/chat.go">
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/tui"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with Gemini.

The chat maintains conversation context across messages.
Type 'exit', 'quit', or press Ctrl+C to end the session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChat()
	},
}

func runChat() error {
	modelName := getModel()
	model := models.ModelFromName(modelName)

	// Build client options
	clientOpts := []api.ClientOption{
		api.WithModel(model),
		api.WithAutoRefresh(true),
	}

	// Add browser refresh if enabled (also enables silent auto-login fallback)
	if browserType, enabled := getBrowserRefresh(); enabled {
		clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
	}

	// Create client with nil cookies - Init() will load from disk or browser
	client, err := api.NewClient(nil, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Initialize client with animation
	// Init() handles cookie loading from disk and browser fallback
	spin := newSpinner("Connecting to Gemini")
	spin.start()
	if err := client.Init(); err != nil {
		spin.stopWithError()
		return fmt.Errorf("failed to initialize: %w", err)
	}
	spin.stopWithSuccess("Connected")

	// Run chat TUI
	return tui.RunChat(client, modelName)
}
</file>
<file path="internal/commands/config.go">
package commands

import (
	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/tui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open configuration menu",
	Long:  `Interactive menu to configure geminiweb settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunConfig()
	},
}
</file>
<file path="internal/commands/gems.go">
package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/models"
)

// GemReaderInterface defines the interface for reading gem input
type GemReaderInterface interface {
	ReadString(delim byte) (string, error)
}

// GemStdInReader is the default implementation of GemReaderInterface
type GemStdInReader struct {
	reader *bufio.Reader
}

// NewGemStdInReader creates a new GemStdInReader
func NewGemStdInReader(reader io.Reader) GemReaderInterface {
	return &GemStdInReader{
		reader: bufio.NewReader(reader),
	}
}

// ReadString implements GemReaderInterface
func (r *GemStdInReader) ReadString(delim byte) (string, error) {
	return r.reader.ReadString(delim)
}

var gemsCmd = &cobra.Command{
	Use:   "gems",
	Short: "Manage Gemini Gems (server-side personas)",
	Long: `Gems are custom personas stored on Google's servers.
Unlike local personas, gems sync across devices with your Google account.

Use 'geminiweb gems list' to see available gems.
Use 'geminiweb gems create' to create a new gem.`,
}

var gemsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all gems",
	RunE:  runGemsList,
}

var gemsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new gem",
	Args:  cobra.ExactArgs(1),
	RunE:  runGemsCreate,
}

var gemsUpdateCmd = &cobra.Command{
	Use:   "update <id-or-name>",
	Short: "Update an existing gem",
	Args:  cobra.ExactArgs(1),
	RunE:  runGemsUpdate,
}

var gemsDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-name>",
	Short: "Delete a gem",
	Args:  cobra.ExactArgs(1),
	RunE:  runGemsDelete,
}

var gemsShowCmd = &cobra.Command{
	Use:   "show <id-or-name>",
	Short: "Show gem details",
	Args:  cobra.ExactArgs(1),
	RunE:  runGemsShow,
}

// Flags
var (
	gemsIncludeHidden bool
	gemPrompt         string
	gemDescription    string
	gemPromptFile     string
	gemName           string
)

func init() {
	gemsCmd.AddCommand(gemsListCmd)
	gemsCmd.AddCommand(gemsCreateCmd)
	gemsCmd.AddCommand(gemsUpdateCmd)
	gemsCmd.AddCommand(gemsDeleteCmd)
	gemsCmd.AddCommand(gemsShowCmd)

	// Flags
	gemsListCmd.Flags().BoolVar(&gemsIncludeHidden, "hidden", false, "Include hidden system gems")

	gemsCreateCmd.Flags().StringVarP(&gemPrompt, "prompt", "p", "", "System prompt for the gem")
	gemsCreateCmd.Flags().StringVarP(&gemDescription, "description", "d", "", "Description")
	gemsCreateCmd.Flags().StringVarP(&gemPromptFile, "file", "f", "", "Read prompt from file")

	gemsUpdateCmd.Flags().StringVarP(&gemPrompt, "prompt", "p", "", "New system prompt")
	gemsUpdateCmd.Flags().StringVarP(&gemDescription, "description", "d", "", "New description")
	gemsUpdateCmd.Flags().StringVarP(&gemPromptFile, "file", "f", "", "Read prompt from file")
	gemsUpdateCmd.Flags().StringVarP(&gemName, "name", "n", "", "New name for the gem")
}

func runGemsList(cmd *cobra.Command, args []string) error {
	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

	gems, err := client.FetchGems(gemsIncludeHidden)
	if err != nil {
		return fmt.Errorf("failed to fetch gems: %w", err)
	}

	if gems.Len() == 0 {
		fmt.Println("No gems found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tTYPE\tDESCRIPTION")
	_, _ = fmt.Fprintln(w, "--\t----\t----\t-----------")

	for _, gem := range gems.Values() {
		gemType := "custom"
		if gem.Predefined {
			gemType = "system"
		}
		desc := gem.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", gem.ID, gem.Name, gemType, desc)
	}
	return w.Flush()
}

func runGemsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	prompt := gemPrompt
	if gemPromptFile != "" {
		data, err := os.ReadFile(gemPromptFile)
		if err != nil {
			return fmt.Errorf("failed to read prompt file: %w", err)
		}
		prompt = string(data)
	}

	if prompt == "" {
		return fmt.Errorf("prompt is required (use -p or -f)")
	}

	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

	gem, err := client.CreateGem(name, prompt, gemDescription)
	if err != nil {
		return fmt.Errorf("failed to create gem: %w", err)
	}

	fmt.Printf("Created gem '%s' with ID: %s\n", gem.Name, gem.ID)
	return nil
}

func runGemsUpdate(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

	gems, err := client.FetchGems(false)
	if err != nil {
		return fmt.Errorf("failed to fetch gems: %w", err)
	}

	gem := gems.Get(idOrName, idOrName)
	if gem == nil {
		return fmt.Errorf("gem '%s' not found", idOrName)
	}

	if gem.Predefined {
		return fmt.Errorf("cannot update system gems")
	}

	// Use existing values if not provided
	newPrompt := gem.Prompt
	newDesc := gem.Description
	newName := gem.Name

	if gemPromptFile != "" {
		data, err := os.ReadFile(gemPromptFile)
		if err != nil {
			return fmt.Errorf("failed to read prompt file: %w", err)
		}
		newPrompt = string(data)
	} else if gemPrompt != "" {
		newPrompt = gemPrompt
	}

	if gemDescription != "" {
		newDesc = gemDescription
	}

	if gemName != "" {
		newName = gemName
	}

	updated, err := client.UpdateGem(gem.ID, newName, newPrompt, newDesc)
	if err != nil {
		return fmt.Errorf("failed to update gem: %w", err)
	}

	fmt.Printf("Updated gem '%s'\n", updated.Name)
	return nil
}

func runGemsDelete(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

	gems, err := client.FetchGems(false)
	if err != nil {
		return fmt.Errorf("failed to fetch gems: %w", err)
	}

	gem := gems.Get(idOrName, idOrName)
	if gem == nil {
		return fmt.Errorf("gem '%s' not found", idOrName)
	}

	if gem.Predefined {
		return fmt.Errorf("cannot delete system gems")
	}

	if err := client.DeleteGem(gem.ID); err != nil {
		return fmt.Errorf("failed to delete gem: %w", err)
	}

	fmt.Printf("Deleted gem '%s'\n", gem.Name)
	return nil
}

func runGemsShow(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	client, err := createGemsClient()
	if err != nil {
		return err
	}
	defer client.Close()

	gems, err := client.FetchGems(true)
	if err != nil {
		return fmt.Errorf("failed to fetch gems: %w", err)
	}

	gem := gems.Get(idOrName, idOrName)
	if gem == nil {
		return fmt.Errorf("gem '%s' not found", idOrName)
	}

	fmt.Printf("ID:          %s\n", gem.ID)
	fmt.Printf("Name:        %s\n", gem.Name)
	fmt.Printf("Description: %s\n", gem.Description)
	gemType := "custom"
	if gem.Predefined {
		gemType = "system"
	}
	fmt.Printf("Type:        %s\n", gemType)
	fmt.Printf("\nPrompt:\n%s\n", gem.Prompt)

	return nil
}

// createGemsClient creates a GeminiClient configured for gems operations
func createGemsClient() (*api.GeminiClient, error) {
	// Build client options
	clientOpts := []api.ClientOption{
		api.WithAutoRefresh(false),
	}

	// Add browser refresh if enabled
	if browserType, enabled := getBrowserRefresh(); enabled {
		clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
	}

	// Create client with nil cookies - Init() will load from disk or browser
	client, err := api.NewClient(nil, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Initialize client
	if err := client.Init(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return client, nil
}

// resolveGem resolves a gem by ID or name using the provided client
// Returns the gem ID if found, empty string otherwise
func resolveGem(client *api.GeminiClient, idOrName string) (*models.Gem, error) {
	gems, err := client.FetchGems(false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gems: %w", err)
	}

	gem := gems.Get(idOrName, idOrName)
	if gem == nil {
		return nil, fmt.Errorf("gem '%s' not found", idOrName)
	}

	return gem, nil
}

// parseGemPromptFromStdin reads a multi-line prompt from stdin
func parseGemPromptFromStdin(reader io.Reader) (string, error) {
	gemReader := NewGemStdInReader(reader)
	fmt.Println("Enter system prompt (end with an empty line):")
	var promptLines []string
	for {
		line, err := gemReader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			break
		}
		promptLines = append(promptLines, line)
	}
	return strings.Join(promptLines, "\n"), nil
}
</file>
<file path="internal/commands/history.go">
package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/history"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage conversation history",
	Long:  `View and manage your local conversation history.`,
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all conversations",
	RunE:  runHistoryList,
}

var historyShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a conversation",
	Args:  cobra.ExactArgs(1),
	RunE:  runHistoryShow,
}

var historyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a conversation",
	Args:  cobra.ExactArgs(1),
	RunE:  runHistoryDelete,
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all conversations",
	RunE:  runHistoryClear,
}

func init() {
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyShowCmd)
	historyCmd.AddCommand(historyDeleteCmd)
	historyCmd.AddCommand(historyClearCmd)
}

func runHistoryList(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	conversations, err := store.ListConversations()
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	if len(conversations) == 0 {
		fmt.Println("No conversations found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tMODEL\tMESSAGES\tUPDATED")
	_, _ = fmt.Fprintln(w, "--\t-----\t-----\t--------\t-------")

	for _, conv := range conversations {
		updated := conv.UpdatedAt.Format("2006-01-02 15:04")
		title := conv.Title
		if len(title) > 40 {
			title = title[:40] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			conv.ID, title, conv.Model, len(conv.Messages), updated)
	}

	return w.Flush()
}

func runHistoryShow(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	conv, err := store.GetConversation(args[0])
	if err != nil {
		return fmt.Errorf("conversation not found: %w", err)
	}

	fmt.Printf("ID: %s\n", conv.ID)
	fmt.Printf("Title: %s\n", conv.Title)
	fmt.Printf("Model: %s\n", conv.Model)
	fmt.Printf("Created: %s\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", conv.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Messages: %d\n", len(conv.Messages))
	fmt.Println()

	for i, msg := range conv.Messages {
		role := "You"
		if msg.Role == "assistant" {
			role = "Gemini"
		}
		fmt.Printf("[%d] %s (%s):\n", i+1, role, msg.Timestamp.Format("15:04"))

		if msg.Thoughts != "" {
			fmt.Printf("  💭 %s\n", msg.Thoughts)
		}

		content := msg.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		fmt.Printf("  %s\n\n", content)
	}

	return nil
}

func runHistoryDelete(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	if err := store.DeleteConversation(args[0]); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Printf("Deleted conversation: %s\n", args[0])
	return nil
}

func runHistoryClear(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	if err := store.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	fmt.Println("All conversations deleted.")
	return nil
}
</file>
<file path="internal/commands/import.go">
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/config"
)

var importCookiesCmd = &cobra.Command{
	Use:   "import-cookies <path>",
	Short: "Import cookies from a file",
	Long: `Import authentication cookies from a JSON file.

The cookies file should contain either:
1. A list of objects: [{"name": "__Secure-1PSID", "value": "..."}]
2. A simple dictionary: {"__Secure-1PSID": "..."}

Required cookie: __Secure-1PSID
Optional cookie: __Secure-1PSIDTS`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImportCookies(args[0])
	},
}

func runImportCookies(sourcePath string) error {
	if err := config.ImportCookies(sourcePath); err != nil {
		return fmt.Errorf("failed to import cookies: %w", err)
	}

	cookiesPath, _ := config.GetCookiesPath()
	fmt.Printf("Cookies imported successfully to %s\n", cookiesPath)
	return nil
}
</file>
<file path="internal/commands/persona.go">
package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/config"
)

// PersonaReaderInterface defines the interface for reading persona input
type PersonaReaderInterface interface {
	ReadString(delim byte) (string, error)
}

// StdInReader is the default implementation of PersonaReaderInterface
type StdInReader struct {
	reader *bufio.Reader
}

// NewStdInReader creates a new StdInReader
func NewStdInReader(reader io.Reader) PersonaReaderInterface {
	return &StdInReader{
		reader: bufio.NewReader(reader),
	}
}

// ReadString implements PersonaReaderInterface
func (r *StdInReader) ReadString(delim byte) (string, error) {
	return r.reader.ReadString(delim)
}

var personaCmd = &cobra.Command{
	Use:   "persona",
	Short: "Manage chat personas",
	Long:  `View and manage personas (system prompts) for chat sessions.`,
}

var personaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available personas",
	RunE:  runPersonaList,
}

var personaShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show persona details",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaShow,
}

var personaAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new persona",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaAdd,
}

var personaDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a persona",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaDelete,
}

var personaSetDefaultCmd = &cobra.Command{
	Use:   "default <name>",
	Short: "Set default persona",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaSetDefault,
}

func init() {
	personaCmd.AddCommand(personaListCmd)
	personaCmd.AddCommand(personaShowCmd)
	personaCmd.AddCommand(personaAddCmd)
	personaCmd.AddCommand(personaDeleteCmd)
	personaCmd.AddCommand(personaSetDefaultCmd)
}

func runPersonaList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadPersonas()
	if err != nil {
		return fmt.Errorf("failed to load personas: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDESCRIPTION\tDEFAULT")
	_, _ = fmt.Fprintln(w, "----\t-----------\t-------")

	for _, p := range cfg.Personas {
		isDefault := ""
		if p.Name == cfg.DefaultPersona {
			isDefault = "✓"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Description, isDefault)
	}

	return w.Flush()
}

func runPersonaShow(cmd *cobra.Command, args []string) error {
	persona, err := config.GetPersona(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Name: %s\n", persona.Name)
	fmt.Printf("Description: %s\n", persona.Description)
	if persona.Model != "" {
		fmt.Printf("Preferred Model: %s\n", persona.Model)
	}
	fmt.Printf("\nSystem Prompt:\n%s\n", persona.SystemPrompt)

	return nil
}

func runPersonaAdd(cmd *cobra.Command, args []string) error {
	return runPersonaAddWithReader(os.Stdin, args)
}

// runPersonaAddWithReader is the internal implementation that accepts a reader for testing
func runPersonaAddWithReader(reader io.Reader, args []string) error {
	name := args[0]

	// Check if already exists
	if _, err := config.GetPersona(name); err == nil {
		return fmt.Errorf("persona '%s' already exists", name)
	}

	personaReader := NewStdInReader(reader)

	fmt.Print("Enter description: ")
	desc, err := personaReader.ReadString('\n')
	if err != nil {
		return err
	}
	desc = strings.TrimSpace(desc)

	fmt.Println("Enter system prompt (end with an empty line):")
	var promptLines []string
	for {
		line, err := personaReader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			break
		}
		promptLines = append(promptLines, line)
	}
	prompt := strings.Join(promptLines, "\n")

	persona := config.Persona{
		Name:         name,
		Description:  desc,
		SystemPrompt: prompt,
	}

	if err := config.AddPersona(persona); err != nil {
		return err
	}

	fmt.Printf("Persona '%s' created.\n", name)
	return nil
}

func runPersonaDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := config.DeletePersona(name); err != nil {
		return err
	}

	fmt.Printf("Persona '%s' deleted.\n", name)
	return nil
}

func runPersonaSetDefault(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := config.SetDefaultPersona(name); err != nil {
		return err
	}

	fmt.Printf("Default persona set to '%s'.\n", name)
	return nil
}
</file>
<file path="internal/commands/query.go">
package commands

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/render"
)

// Gradient colors for animation
var gradientColors = []lipgloss.Color{
	lipgloss.Color("#ff6b6b"), // Red
	lipgloss.Color("#feca57"), // Yellow
	lipgloss.Color("#48dbfb"), // Cyan
	lipgloss.Color("#ff9ff3"), // Pink
	lipgloss.Color("#54a0ff"), // Blue
	lipgloss.Color("#5f27cd"), // Purple
	lipgloss.Color("#00d2d3"), // Teal
	lipgloss.Color("#1dd1a1"), // Green
}

var (
	colorText     = lipgloss.Color("#c0caf5")
	colorTextDim  = lipgloss.Color("#565f89")
	colorTextMute = lipgloss.Color("#3b4261")
	colorSuccess  = lipgloss.Color("#9ece6a")
	colorPrimary  = lipgloss.Color("#7aa2f7")
)

// Styles matching the chat TUI
var (
	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				MarginBottom(0)

	assistantBubbleStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Foreground(colorText).
				Padding(0, 1).
				MarginTop(1).
				MarginBottom(1)

	thoughtsStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorTextDim).
			BorderLeft(true).
			Foreground(colorTextDim).
			PaddingLeft(1).
			MarginLeft(1).
			Italic(true)
)

// spinner handles the animated loading indicator
type spinner struct {
	message string
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
	frame   int
}

// newSpinner creates a new animated spinner
func newSpinner(message string) *spinner {
	return &spinner{
		message: message,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// start begins the animation
func (s *spinner) start() {
	go func() {
		defer close(s.done)

		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		// Hide cursor
		fmt.Fprint(os.Stderr, "\033[?25l")

		for {
			select {
			case <-s.stop:
				// Clear line and show cursor
				fmt.Fprint(os.Stderr, "\r\033[K\033[?25h")
				return
			case <-ticker.C:
				s.mu.Lock()
				s.render()
				s.frame++
				s.mu.Unlock()
			}
		}
	}()
}

// render draws the current animation frame
func (s *spinner) render() {
	// Spinner characters
	chars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	barChars := []string{"█", "█", "█", "█", "█", "█", "▓", "▒", "░"}

	// Build spinner character with color
	spinIdx := s.frame % len(chars)
	spinColor := gradientColors[s.frame%len(gradientColors)]
	spinnerChar := lipgloss.NewStyle().Foreground(spinColor).Bold(true).Render(chars[spinIdx])

	// Build animated bar
	barWidth := 16
	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		colorIdx := (i + s.frame) % len(gradientColors)
		charIdx := (i + s.frame/2) % len(barChars)
		style := lipgloss.NewStyle().Foreground(gradientColors[colorIdx])
		bar.WriteString(style.Render(barChars[charIdx]))
	}

	// Build animated dots
	var dots strings.Builder
	numDots := (s.frame / 3) % 4
	for i := 0; i < 3; i++ {
		if i < numDots {
			dotColor := gradientColors[(s.frame+i)%len(gradientColors)]
			dots.WriteString(lipgloss.NewStyle().Foreground(dotColor).Render("●"))
		} else {
			dots.WriteString(lipgloss.NewStyle().Foreground(colorTextMute).Render("○"))
		}
	}

	// Message with color
	msg := lipgloss.NewStyle().Foreground(colorText).Render(s.message)

	// Print animation (clear line first)
	fmt.Fprintf(os.Stderr, "\r\033[K%s %s %s %s", spinnerChar, bar.String(), msg, dots.String())
}

// stopWithSuccess stops the spinner and shows success message
func (s *spinner) stopWithSuccess(message string) {
	close(s.stop)
	<-s.done

	checkmark := lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("✓")
	msg := lipgloss.NewStyle().Foreground(colorSuccess).Render(message)
	fmt.Fprintf(os.Stderr, "%s %s\n", checkmark, msg)
}

// stopWithError stops the spinner and shows error
func (s *spinner) stopWithError() {
	close(s.stop)
	<-s.done
}

// runQuery executes a single query and outputs the response
// If rawOutput is true, only the raw response text is printed without decoration
func runQuery(prompt string, rawOutput bool) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	modelName := getModel()
	model := models.ModelFromName(modelName)

	// Build client options
	clientOpts := []api.ClientOption{
		api.WithModel(model),
		api.WithAutoRefresh(false),
	}

	// Add browser refresh if enabled (also enables silent auto-login fallback)
	if browserType, enabled := getBrowserRefresh(); enabled {
		clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
	}

	// Create client with nil cookies - Init() will load from disk or browser
	client, err := api.NewClient(nil, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Initialize client
	// Init() handles cookie loading from disk and browser fallback
	var spin *spinner
	if !rawOutput {
		spin = newSpinner("Connecting to Gemini")
		spin.start()
	}

	if err := client.Init(); err != nil {
		if !rawOutput {
			spin.stopWithError()
			fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Failed to initialize"))
		}
		return fmt.Errorf("failed to initialize: %w", err)
	}
	if !rawOutput {
		spin.stopWithSuccess("Connected")
	}

	// Resolve gem if provided
	var gemID string
	if gemFlag != "" {
		if !rawOutput {
			spin = newSpinner("Loading gems")
			spin.start()
		}

		gem, err := resolveGem(client, gemFlag)
		if err != nil {
			if !rawOutput {
				spin.stopWithError()
				fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Gem resolution failed"))
			}
			return fmt.Errorf("gem resolution failed: %w", err)
		}
		gemID = gem.ID
		if !rawOutput {
			spin.stopWithSuccess(fmt.Sprintf("Using gem: %s", gem.Name))
		}
	}

	// Upload image if provided
	var images []*api.UploadedImage
	if imageFlag != "" {
		if !rawOutput {
			spin = newSpinner("Uploading image")
			spin.start()
		}

		img, err := client.UploadImage(imageFlag)
		if err != nil {
			if !rawOutput {
				spin.stopWithError()
				fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Failed to upload image"))
			}
			return fmt.Errorf("failed to upload image: %w", err)
		}
		images = append(images, img)
		if !rawOutput {
			spin.stopWithSuccess("Image uploaded")
		}
	}

	// For large prompts, upload as a file and use a reference prompt
	var actualPrompt string
	if len(prompt) > api.LargePromptThreshold {
		if !rawOutput {
			spin = newSpinner("Uploading large prompt as file")
			spin.start()
		}

		// Upload the prompt as a text file
		uploadedFile, err := client.UploadText(prompt, "prompt.md")
		if err != nil {
			if !rawOutput {
				spin.stopWithError()
				fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Failed to upload prompt"))
			}
			return fmt.Errorf("failed to upload prompt: %w", err)
		}

		// Add as an "image" (the API treats uploaded files similarly)
		images = append(images, uploadedFile)

		// Use a minimal prompt that references the uploaded file
		actualPrompt = "Please process and respond to the content in the uploaded file."
		if !rawOutput {
			spin.stopWithSuccess(fmt.Sprintf("Prompt uploaded (%d KB)", len(prompt)/1024))
		}
	} else {
		actualPrompt = prompt
	}

	// Generate content
	if !rawOutput {
		spin = newSpinner("Generating response")
		spin.start()
	}

	opts := &api.GenerateOptions{
		Images: images,
		GemID:  gemID,
	}

	output, err := client.GenerateContent(actualPrompt, opts)
	if err != nil {
		if !rawOutput {
			spin.stopWithError()
			fmt.Fprintln(os.Stderr, formatErrorMessage(err, "Generation failed"))
		}
		return fmt.Errorf("generation failed: %w", err)
	}
	if !rawOutput {
		spin.stopWithSuccess("Done")
	}

	text := output.Text()

	// Raw output mode: output only the raw text
	if rawOutput {
		// Output to file if specified
		if outputFlag != "" {
			if err := os.WriteFile(outputFlag, []byte(text), 0o644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			return nil
		}
		// Output raw text to stdout
		fmt.Print(text)
		return nil
	}

	// Decorated output mode (TTY)
	// Add spacing
	fmt.Fprintln(os.Stderr)

	// Copy to clipboard if enabled in config
	cfg, _ := config.LoadConfig()
	if cfg.CopyToClipboard {
		if err := clipboard.WriteAll(text); err != nil {
			// Log warning but don't fail
			warnMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")).Render(
				fmt.Sprintf("⚠ Failed to copy to clipboard: %v", err),
			)
			fmt.Fprintln(os.Stderr, warnMsg)
		} else {
			clipMsg := lipgloss.NewStyle().Foreground(colorSuccess).Render("✓ Copied to clipboard")
			fmt.Fprintln(os.Stderr, clipMsg)
		}
	}

	// Output to file if specified
	if outputFlag != "" {
		if err := os.WriteFile(outputFlag, []byte(text), 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		successMsg := lipgloss.NewStyle().Foreground(colorSuccess).Render(
			fmt.Sprintf("✓ Response saved to %s", outputFlag),
		)
		fmt.Fprintln(os.Stderr, successMsg)
		return nil
	}

	// Get terminal width for proper formatting
	termWidth := getTerminalWidth()
	bubbleWidth := termWidth - 4
	if bubbleWidth < 40 {
		bubbleWidth = 40
	}
	if bubbleWidth > 120 {
		bubbleWidth = 120
	}
	contentWidth := bubbleWidth - 4

	// Print assistant label (similar to chat TUI)
	label := assistantLabelStyle.Render("✦ Gemini")
	fmt.Println(label)

	// Print thoughts if present (with styled border)
	if thoughts := output.Thoughts(); thoughts != "" {
		thoughtsContent := thoughtsStyle.Width(contentWidth).Render("💭 " + thoughts)
		fmt.Println(thoughtsContent)
	}

	// Render markdown for terminal output
	rendered, err := render.MarkdownWithWidth(text, contentWidth)
	if err != nil {
		rendered = text
	}
	// Trim trailing newlines from glamour
	rendered = strings.TrimRight(rendered, "\n")

	// Wrap content in assistant bubble style
	bubble := assistantBubbleStyle.Width(bubbleWidth).Render(rendered)
	fmt.Println(bubble)

	return nil
}


// getTerminalWidth returns the terminal width or a default value
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80 // default width
	}
	return width
}

// isStdoutTTY returns true if stdout is connected to a terminal
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// formatErrorMessage formats an error with additional context from structured errors
func formatErrorMessage(err error, context string) string {
	if err == nil {
		return ""
	}

	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	dimStyle := lipgloss.NewStyle().Foreground(colorTextDim)

	var sb strings.Builder
	sb.WriteString(errorStyle.Render(fmt.Sprintf("✗ %s: %v", context, err)))

	// Extract additional context from structured errors
	if status := apierrors.GetHTTPStatus(err); status > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  HTTP Status: %d", status)))
	}

	if code := apierrors.GetErrorCode(err); code != apierrors.ErrCodeUnknown {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  Error Code: %d (%s)", code, code.String())))
	}

	if endpoint := apierrors.GetEndpoint(err); endpoint != "" {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n  Endpoint: %s", endpoint)))
	}

	// Provide helpful hints based on error type
	switch {
	case apierrors.IsAuthError(err):
		sb.WriteString(dimStyle.Render("\n  Hint: Try running 'geminiweb auto-login' to refresh your session"))
	case apierrors.IsRateLimitError(err):
		sb.WriteString(dimStyle.Render("\n  Hint: You've hit the usage limit. Try again later or use a different model"))
	case apierrors.IsNetworkError(err):
		sb.WriteString(dimStyle.Render("\n  Hint: Check your internet connection and try again"))
	case apierrors.IsTimeoutError(err):
		sb.WriteString(dimStyle.Render("\n  Hint: Request timed out. Try again or check your connection"))
	case apierrors.IsUploadError(err):
		sb.WriteString(dimStyle.Render("\n  Hint: File upload failed. Check the file exists and is accessible"))
	}

	return sb.String()
}
</file>
<file path="internal/commands/root.go">
// Package commands provides CLI commands for geminiweb.
package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/browser"
	"github.com/diogo/geminiweb/internal/config"
)

var (
	// Global flags
	modelFlag          string
	outputFlag         string
	fileFlag           string
	imageFlag          string
	browserRefreshFlag string
	gemFlag            string

	// Version info (set at build time)
	Version   = "0.1.0"
	BuildTime = "unknown"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "geminiweb [prompt]",
	Short: "CLI for Google Gemini Web API",
	Long: `geminiweb is a command-line interface for interacting with Google Gemini
via the web API. It uses cookie-based authentication and communicates
directly with Gemini's web interface.

Examples:
  geminiweb chat                        Start interactive chat
  geminiweb config                      Configure settings
  geminiweb import-cookies ~/cookies.json
  geminiweb "What is Go?"               Send a single query
  geminiweb -f prompt.md                Read prompt from file
  cat prompt.md | geminiweb             Read prompt from stdin
  geminiweb "Hello" -o response.md      Save response to file
  geminiweb --gem "Code Helper" "prompt" Use a gem (server-side persona)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for version flag
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Printf("geminiweb %s (built %s)\n", Version, BuildTime)
			return nil
		}

		// Check for stdin input
		stat, _ := os.Stdin.Stat()
		hasStdin := (stat.Mode() & os.ModeCharDevice) == 0

		// Determine raw output mode:
		// - If outputFlag is set (writing to file), use raw mode
		// - If stdin is piped AND stdout is not a TTY, use raw mode
		isTTY := isStdoutTTY()
		isFileOutput := outputFlag != ""
		isPipeOutput := hasStdin && !isTTY
		rawOutput := isFileOutput || isPipeOutput

		// Check for file input
		if fileFlag != "" {
			data, err := os.ReadFile(fileFlag)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			return runQuery(string(data), rawOutput)
		}

		// Check for stdin
		if hasStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			return runQuery(string(data), rawOutput)
		}

		// Check for positional argument
		if len(args) > 0 {
			return runQuery(args[0], rawOutput)
		}

		// No input - show help
		return cmd.Help()
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&modelFlag, "model", "m", "", "Model to use (e.g., gemini-2.5-flash)")
	rootCmd.PersistentFlags().StringVar(&browserRefreshFlag, "browser-refresh", "",
		"Auto-refresh cookies from browser on auth failure (auto, chrome, firefox, edge, chromium, opera)")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Save response to file")
	rootCmd.Flags().StringVarP(&fileFlag, "file", "f", "", "Read prompt from file")
	rootCmd.Flags().StringVarP(&imageFlag, "image", "i", "", "Path to image file to include")
	rootCmd.Flags().StringVar(&gemFlag, "gem", "", "Use a gem (by ID or name) - server-side persona")
	rootCmd.Flags().BoolP("version", "v", false, "Show version and exit")

	// Add subcommands
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(importCookiesCmd)
	rootCmd.AddCommand(autoLoginCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(personaCmd)
	rootCmd.AddCommand(gemsCmd)
}

// getModel returns the model to use (from flag or config)
func getModel() string {
	if modelFlag != "" {
		return modelFlag
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return "gemini-2.5-flash"
	}

	return cfg.DefaultModel
}

// getBrowserRefresh returns the browser type for auto-refresh, or empty if disabled
func getBrowserRefresh() (browser.SupportedBrowser, bool) {
	if browserRefreshFlag == "" {
		return "", false
	}

	browserType, err := browser.ParseBrowser(browserRefreshFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: invalid browser-refresh value '%s', disabling browser refresh\n", browserRefreshFlag)
		return "", false
	}

	return browserType, true
}
</file>
<file path="internal/config/config.go">
// Package config handles configuration and cookie management for geminiweb.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MarkdownConfig configures markdown rendering options
type MarkdownConfig struct {
	Style            string `json:"style"`              // "dark", "light", or path to JSON theme
	EnableEmoji      bool   `json:"enable_emoji"`       // Convert :emoji: to unicode
	PreserveNewLines bool   `json:"preserve_newlines"`  // Preserve original line breaks
	TableWrap        bool   `json:"table_wrap"`         // Enable word wrap in table cells
	InlineTableLinks bool   `json:"inline_table_links"` // Render links inline in tables
}

// Config represents the user configuration
type Config struct {
	DefaultModel    string         `json:"default_model"`
	AutoClose       bool           `json:"auto_close"`
	Verbose         bool           `json:"verbose"`
	CopyToClipboard bool           `json:"copy_to_clipboard"`
	Markdown        MarkdownConfig `json:"markdown,omitempty"`
}

// DefaultMarkdownConfig returns the default markdown configuration
func DefaultMarkdownConfig() MarkdownConfig {
	return MarkdownConfig{
		Style:            "dark",
		EnableEmoji:      true,
		PreserveNewLines: true,
		TableWrap:        true,
		InlineTableLinks: false,
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		DefaultModel:    "gemini-2.5-flash",
		AutoClose:       true,
		Verbose:         false,
		CopyToClipboard: false,
		Markdown:        DefaultMarkdownConfig(),
	}
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".geminiweb")
	return configDir, nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// GetCookiesPath returns the path to the cookies file
func GetCookiesPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "cookies.json"), nil
}

// LoadConfig loads the configuration from disk
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()

	configPath, err := GetConfigPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if config doesn't exist
		}
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(cfg Config) error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AvailableModels returns a list of available model names
func AvailableModels() []string {
	return []string{
		"gemini-2.5-flash",
		"gemini-2.5-pro",
		"gemini-3.0-pro",
		"unspecified",
	}
}
</file>
<file path="internal/config/cookies.go">
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Cookies represents the authentication cookies
type Cookies struct {
	Secure1PSID   string `json:"__Secure-1PSID"`
	Secure1PSIDTS string `json:"__Secure-1PSIDTS,omitempty"`
}

// CookieListItem represents a cookie in browser export format
type CookieListItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// LoadCookies loads cookies from the cookies file
func LoadCookies() (*Cookies, error) {
	cookiesPath, err := GetCookiesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cookiesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no cookies found. Please import cookies first:\n  geminiweb import-cookies <path-to-cookies.json>")
		}
		return nil, fmt.Errorf("failed to read cookies file: %w", err)
	}

	return parseCookies(data)
}

// parseCookies parses cookies from JSON data
// Supports both list format [{name, value}] and dict format {name: value}
func parseCookies(data []byte) (*Cookies, error) {
	// Try dict format first
	var dictFormat map[string]string
	if err := json.Unmarshal(data, &dictFormat); err == nil {
		psid, ok := dictFormat["__Secure-1PSID"]
		if !ok {
			return nil, fmt.Errorf("missing required cookie: __Secure-1PSID")
		}
		return &Cookies{
			Secure1PSID:   psid,
			Secure1PSIDTS: dictFormat["__Secure-1PSIDTS"],
		}, nil
	}

	// Try list format (browser export)
	var listFormat []CookieListItem
	if err := json.Unmarshal(data, &listFormat); err == nil {
		cookies := &Cookies{}
		for _, item := range listFormat {
			switch item.Name {
			case "__Secure-1PSID":
				cookies.Secure1PSID = item.Value
			case "__Secure-1PSIDTS":
				cookies.Secure1PSIDTS = item.Value
			}
		}

		if cookies.Secure1PSID == "" {
			return nil, fmt.Errorf("missing required cookie: __Secure-1PSID")
		}
		return cookies, nil
	}

	return nil, fmt.Errorf("invalid cookies format: expected list [{name, value}] or dict {name: value}")
}

// SaveCookies saves cookies to the cookies file
func SaveCookies(cookies *Cookies) error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	cookiesPath := configDir + "/cookies.json"

	// Save in list format for compatibility
	listFormat := []CookieListItem{
		{Name: "__Secure-1PSID", Value: cookies.Secure1PSID},
	}
	if cookies.Secure1PSIDTS != "" {
		listFormat = append(listFormat, CookieListItem{
			Name:  "__Secure-1PSIDTS",
			Value: cookies.Secure1PSIDTS,
		})
	}

	data, err := json.MarshalIndent(listFormat, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	// Save with restrictive permissions (owner read/write only)
	if err := os.WriteFile(cookiesPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write cookies file: %w", err)
	}

	return nil
}

// ImportCookies imports cookies from a source file
func ImportCookies(sourcePath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source file not found: %s", sourcePath)
		}
		return fmt.Errorf("could not read file: %w", err)
	}

	cookies, err := parseCookies(data)
	if err != nil {
		return err
	}

	return SaveCookies(cookies)
}

// ValidateCookies checks if cookies are valid
func ValidateCookies(cookies *Cookies) error {
	if cookies == nil {
		return fmt.Errorf("cookies are nil")
	}
	if cookies.Secure1PSID == "" {
		return fmt.Errorf("missing required cookie: __Secure-1PSID")
	}
	return nil
}

// ToMap converts cookies to a map for HTTP requests
func (c *Cookies) ToMap() map[string]string {
	m := map[string]string{
		"__Secure-1PSID": c.Secure1PSID,
	}
	if c.Secure1PSIDTS != "" {
		m["__Secure-1PSIDTS"] = c.Secure1PSIDTS
	}
	return m
}

// Update1PSIDTS updates the PSIDTS cookie value
func (c *Cookies) Update1PSIDTS(value string) {
	c.Secure1PSIDTS = value
}
</file>
<file path="internal/config/personas.go">
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Persona represents a system prompt configuration
type Persona struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	SystemPrompt string  `json:"system_prompt"`
	Model        string  `json:"model,omitempty"`       // Preferred model (optional)
	Temperature  float64 `json:"temperature,omitempty"` // For future use
}

// PersonaConfig stores all personas
type PersonaConfig struct {
	Personas       []Persona `json:"personas"`
	DefaultPersona string    `json:"default_persona,omitempty"`
}

// DefaultPersonas returns pre-configured personas
func DefaultPersonas() []Persona {
	return []Persona{
		{
			Name:         "default",
			Description:  "No system prompt",
			SystemPrompt: "",
		},
		{
			Name:        "coder",
			Description: "Expert programmer assistant",
			SystemPrompt: `You are an expert software engineer. When answering:
- Provide clean, well-structured code examples
- Explain your reasoning step by step
- Consider edge cases and error handling
- Suggest best practices and optimizations
- Use code comments only when necessary for clarity`,
		},
		{
			Name:        "writer",
			Description: "Creative writing assistant",
			SystemPrompt: `You are a creative writing assistant. Your goal is to:
- Help with creative writing, storytelling, and content creation
- Provide suggestions that enhance narrative flow
- Maintain consistent tone and style
- Offer multiple alternatives when asked
- Be concise but evocative in descriptions`,
		},
		{
			Name:        "analyst",
			Description: "Data and business analyst",
			SystemPrompt: `You are a data and business analyst. You should:
- Analyze information methodically
- Present findings in structured formats
- Use data to support conclusions
- Consider multiple perspectives
- Highlight key insights and actionable recommendations`,
		},
		{
			Name:        "teacher",
			Description: "Patient educational assistant",
			SystemPrompt: `You are a patient and thorough teacher. When explaining:
- Break down complex topics into simple parts
- Use analogies and examples
- Check understanding progressively
- Encourage questions
- Adapt explanations to the learner's level`,
		},
	}
}

// GetPersonasPath returns the path to the personas file
func GetPersonasPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "personas.json"), nil
}

// LoadPersonas loads the persona configuration
func LoadPersonas() (*PersonaConfig, error) {
	path, err := GetPersonasPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if file doesn't exist
			return &PersonaConfig{
				Personas:       DefaultPersonas(),
				DefaultPersona: "default",
			}, nil
		}
		return nil, fmt.Errorf("failed to read personas: %w", err)
	}

	var config PersonaConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse personas: %w", err)
	}

	// Merge with defaults (keep user customizations)
	config.Personas = mergePersonas(DefaultPersonas(), config.Personas)

	return &config, nil
}

// SavePersonas saves the persona configuration
func SavePersonas(config *PersonaConfig) error {
	path, err := GetPersonasPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal personas: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// GetPersona returns a persona by name
func GetPersona(name string) (*Persona, error) {
	config, err := LoadPersonas()
	if err != nil {
		return nil, err
	}

	for _, p := range config.Personas {
		if p.Name == name {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("persona '%s' not found", name)
}

// ListPersonaNames returns the names of all personas
func ListPersonaNames() ([]string, error) {
	config, err := LoadPersonas()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(config.Personas))
	for i, p := range config.Personas {
		names[i] = p.Name
	}
	return names, nil
}

// AddPersona adds a new persona
func AddPersona(persona Persona) error {
	config, err := LoadPersonas()
	if err != nil {
		return err
	}

	// Check if exists
	for _, p := range config.Personas {
		if p.Name == persona.Name {
			return fmt.Errorf("persona '%s' already exists", persona.Name)
		}
	}

	config.Personas = append(config.Personas, persona)
	return SavePersonas(config)
}

// UpdatePersona updates an existing persona
func UpdatePersona(persona Persona) error {
	config, err := LoadPersonas()
	if err != nil {
		return err
	}

	found := false
	for i, p := range config.Personas {
		if p.Name == persona.Name {
			config.Personas[i] = persona
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("persona '%s' not found", persona.Name)
	}

	return SavePersonas(config)
}

// DeletePersona removes a persona by name
func DeletePersona(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete the default persona")
	}

	config, err := LoadPersonas()
	if err != nil {
		return err
	}

	newPersonas := make([]Persona, 0, len(config.Personas))
	found := false
	for _, p := range config.Personas {
		if p.Name == name {
			found = true
			continue
		}
		newPersonas = append(newPersonas, p)
	}

	if !found {
		return fmt.Errorf("persona '%s' not found", name)
	}

	config.Personas = newPersonas

	// Reset default if deleted
	if config.DefaultPersona == name {
		config.DefaultPersona = "default"
	}

	return SavePersonas(config)
}

// SetDefaultPersona sets the default persona
func SetDefaultPersona(name string) error {
	// Verify persona exists
	_, err := GetPersona(name)
	if err != nil {
		return err
	}

	config, err := LoadPersonas()
	if err != nil {
		return err
	}

	config.DefaultPersona = name
	return SavePersonas(config)
}

// GetDefaultPersona returns the default persona
func GetDefaultPersona() (*Persona, error) {
	config, err := LoadPersonas()
	if err != nil {
		return nil, err
	}

	name := config.DefaultPersona
	if name == "" {
		name = "default"
	}

	return GetPersona(name)
}

func mergePersonas(defaults, custom []Persona) []Persona {
	result := make([]Persona, len(defaults))
	copy(result, defaults)

	// Add or replace with custom
	for _, cp := range custom {
		found := false
		for i, dp := range result {
			if dp.Name == cp.Name {
				result[i] = cp
				found = true
				break
			}
		}
		if !found {
			result = append(result, cp)
		}
	}

	return result
}

// FormatSystemPrompt formats the system prompt for inclusion in a message
func FormatSystemPrompt(persona *Persona, userMessage string) string {
	if persona == nil || persona.SystemPrompt == "" {
		return userMessage
	}

	return fmt.Sprintf(`[System Instructions]
%s

[User Message]
%s`, persona.SystemPrompt, userMessage)
}
</file>
<file path="internal/errors/errors.go">
// Package errors provides custom error types for the Gemini Web API client.
package errors

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// ErrorCode represents known API error codes from Gemini
type ErrorCode int

const (
	ErrCodeUnknown            ErrorCode = 0
	ErrCodePromptTooLong      ErrorCode = 3
	ErrCodeUsageLimitExceeded ErrorCode = 1037
	ErrCodeModelInconsistent  ErrorCode = 1050
	ErrCodeModelHeaderInvalid ErrorCode = 1052
	ErrCodeIPBlocked          ErrorCode = 1060
)

// String returns a human-readable description of the error code
func (c ErrorCode) String() string {
	switch c {
	case ErrCodePromptTooLong:
		return "prompt too long - reduce input size"
	case ErrCodeUsageLimitExceeded:
		return "usage limit exceeded"
	case ErrCodeModelInconsistent:
		return "model inconsistent with chat history"
	case ErrCodeModelHeaderInvalid:
		return "model header invalid or model unavailable"
	case ErrCodeIPBlocked:
		return "IP temporarily blocked"
	default:
		return "unknown error"
	}
}

// Sentinel errors for common cases
var (
	ErrAuthFailed      = errors.New("authentication failed")
	ErrCookiesExpired  = errors.New("cookies have expired")
	ErrNoCookies       = errors.New("no cookies found")
	ErrInvalidResponse = errors.New("invalid response format")
	ErrNoContent       = errors.New("no content in response")
	ErrNetworkFailure  = errors.New("network failure")
	ErrTimeout         = errors.New("request timed out")
	ErrRateLimited     = errors.New("rate limited")
)

// GeminiError is the base error type for all Gemini API errors.
// It implements the error interface and provides rich context for debugging.
type GeminiError struct {
	// Code is the internal error code from the Gemini API (e.g., 1037, 1050)
	Code ErrorCode

	// HTTPStatus is the HTTP status code (e.g., 401, 500)
	HTTPStatus int

	// Endpoint is the API endpoint that was called
	Endpoint string

	// Operation describes what was being attempted (e.g., "generate content", "get access token")
	Operation string

	// Message is a human-readable error message
	Message string

	// Body contains the response body (truncated) for diagnostic purposes
	Body string

	// Cause is the underlying error that caused this error
	Cause error
}

// Error implements the error interface
func (e *GeminiError) Error() string {
	var parts []string

	if e.Operation != "" {
		parts = append(parts, e.Operation+" failed")
	}

	if e.HTTPStatus > 0 {
		parts = append(parts, fmt.Sprintf("HTTP %d", e.HTTPStatus))
	}

	if e.Code != ErrCodeUnknown {
		parts = append(parts, fmt.Sprintf("code=%d (%s)", e.Code, e.Code.String()))
	}

	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	if e.Endpoint != "" {
		parts = append(parts, fmt.Sprintf("endpoint=%s", e.Endpoint))
	}

	if len(parts) == 0 {
		return "gemini error"
	}

	return strings.Join(parts, ": ")
}

// Unwrap returns the underlying cause
func (e *GeminiError) Unwrap() error {
	return e.Cause
}

// Is implements error matching for errors.Is()
func (e *GeminiError) Is(target error) bool {
	switch target {
	case ErrAuthFailed:
		return e.HTTPStatus == 401 || e.IsAuth()
	case ErrRateLimited:
		return e.Code == ErrCodeUsageLimitExceeded || e.HTTPStatus == 429
	case ErrNetworkFailure:
		return e.IsNetwork()
	case ErrTimeout:
		return e.IsTimeout()
	}

	if t, ok := target.(*GeminiError); ok {
		return e.Code == t.Code || (e.HTTPStatus != 0 && e.HTTPStatus == t.HTTPStatus)
	}

	return false
}

// IsAuth returns true if this is an authentication error
func (e *GeminiError) IsAuth() bool {
	return e.HTTPStatus == 401
}

// IsRateLimit returns true if this is a rate limit error
func (e *GeminiError) IsRateLimit() bool {
	return e.Code == ErrCodeUsageLimitExceeded || e.HTTPStatus == 429
}

// IsNetwork returns true if this is a network error
func (e *GeminiError) IsNetwork() bool {
	if e.Cause == nil {
		return false
	}

	var netErr net.Error
	return errors.As(e.Cause, &netErr)
}

// IsTimeout returns true if this is a timeout error
func (e *GeminiError) IsTimeout() bool {
	if e.Cause == nil {
		return false
	}

	var netErr net.Error
	if errors.As(e.Cause, &netErr) {
		return netErr.Timeout()
	}

	return false
}

// IsBlocked returns true if the IP is blocked
func (e *GeminiError) IsBlocked() bool {
	return e.Code == ErrCodeIPBlocked
}

// IsModelError returns true if this is a model-related error
func (e *GeminiError) IsModelError() bool {
	return e.Code == ErrCodeModelInconsistent || e.Code == ErrCodeModelHeaderInvalid
}

// WithBody adds the response body to the error (truncated for safety)
func (e *GeminiError) WithBody(body string) *GeminiError {
	const maxBodyLen = 500
	if len(body) > maxBodyLen {
		e.Body = body[:maxBodyLen] + "...(truncated)"
	} else {
		e.Body = body
	}
	return e
}

// NewGeminiError creates a new GeminiError with the given parameters
func NewGeminiError(operation string, message string) *GeminiError {
	return &GeminiError{
		Operation: operation,
		Message:   message,
	}
}

// NewGeminiErrorWithStatus creates a GeminiError with HTTP status
func NewGeminiErrorWithStatus(operation string, httpStatus int, endpoint string, message string) *GeminiError {
	return &GeminiError{
		Operation:  operation,
		HTTPStatus: httpStatus,
		Endpoint:   endpoint,
		Message:    message,
	}
}

// NewGeminiErrorWithCode creates a GeminiError with an API error code
func NewGeminiErrorWithCode(operation string, code ErrorCode, endpoint string) *GeminiError {
	return &GeminiError{
		Operation: operation,
		Code:      code,
		Endpoint:  endpoint,
		Message:   code.String(),
	}
}

// NewGeminiErrorWithCause creates a GeminiError wrapping another error
func NewGeminiErrorWithCause(operation string, cause error) *GeminiError {
	return &GeminiError{
		Operation: operation,
		Cause:     cause,
	}
}

// AuthError represents an authentication failure
type AuthError struct {
	*GeminiError
}

// NewAuthError creates a new AuthError
func NewAuthError(message string) *AuthError {
	return &AuthError{
		GeminiError: &GeminiError{
			HTTPStatus: 401,
			Operation:  "authentication",
			Message:    message,
		},
	}
}

// NewAuthErrorWithEndpoint creates a new AuthError with endpoint info
func NewAuthErrorWithEndpoint(message string, endpoint string) *AuthError {
	return &AuthError{
		GeminiError: &GeminiError{
			HTTPStatus: 401,
			Operation:  "authentication",
			Endpoint:   endpoint,
			Message:    message,
		},
	}
}

// Error implements the error interface
func (e *AuthError) Error() string {
	if e.GeminiError.Message == "" {
		return "authentication failed: cookies may have expired"
	}
	return fmt.Sprintf("authentication failed: %s", e.GeminiError.Message)
}

// Is allows comparison with sentinel errors
func (e *AuthError) Is(target error) bool {
	if target == ErrAuthFailed {
		return true
	}
	if _, ok := target.(*AuthError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying GeminiError
func (e *AuthError) Unwrap() error {
	return e.GeminiError.Cause
}

// APIError represents an API request failure
type APIError struct {
	*GeminiError
}

// NewAPIError creates a new APIError
func NewAPIError(statusCode int, endpoint, message string) *APIError {
	return &APIError{
		GeminiError: &GeminiError{
			HTTPStatus: statusCode,
			Endpoint:   endpoint,
			Operation:  "API request",
			Message:    message,
		},
	}
}

// NewAPIErrorWithCode creates an APIError with an internal error code
func NewAPIErrorWithCode(code ErrorCode, endpoint string) *APIError {
	return &APIError{
		GeminiError: &GeminiError{
			Code:      code,
			Endpoint:  endpoint,
			Operation: "API request",
			Message:   code.String(),
		},
	}
}

// NewAPIErrorWithBody creates an APIError with the response body
func NewAPIErrorWithBody(statusCode int, endpoint, message, body string) *APIError {
	e := NewAPIError(statusCode, endpoint, message)
	e.GeminiError.WithBody(body)
	return e
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.GeminiError.HTTPStatus > 0 {
		if e.GeminiError.Body != "" {
			return fmt.Sprintf("API error [%d] at %s: %s (body: %s)",
				e.GeminiError.HTTPStatus, e.GeminiError.Endpoint, e.GeminiError.Message, e.GeminiError.Body)
		}
		return fmt.Sprintf("API error [%d] at %s: %s",
			e.GeminiError.HTTPStatus, e.GeminiError.Endpoint, e.GeminiError.Message)
	}
	return fmt.Sprintf("API error at %s: %s", e.GeminiError.Endpoint, e.GeminiError.Message)
}

// StatusCode returns the HTTP status code (for backwards compatibility)
func (e *APIError) StatusCode() int {
	return e.GeminiError.HTTPStatus
}

// Is allows comparison with other errors
func (e *APIError) Is(target error) bool {
	if target == ErrAuthFailed && e.GeminiError.HTTPStatus == 401 {
		return true
	}
	if target == ErrRateLimited && (e.GeminiError.HTTPStatus == 429 || e.GeminiError.Code == ErrCodeUsageLimitExceeded) {
		return true
	}
	if t, ok := target.(*APIError); ok {
		return e.GeminiError.HTTPStatus == t.GeminiError.HTTPStatus
	}
	return false
}

// Unwrap returns the underlying error
func (e *APIError) Unwrap() error {
	return e.GeminiError.Cause
}

// NetworkError represents a network-level failure
type NetworkError struct {
	*GeminiError
}

// NewNetworkError creates a new NetworkError
func NewNetworkError(operation string, cause error) *NetworkError {
	return &NetworkError{
		GeminiError: &GeminiError{
			Operation: operation,
			Message:   "network request failed",
			Cause:     cause,
		},
	}
}

// NewNetworkErrorWithEndpoint creates a NetworkError with endpoint info
func NewNetworkErrorWithEndpoint(operation string, endpoint string, cause error) *NetworkError {
	return &NetworkError{
		GeminiError: &GeminiError{
			Operation: operation,
			Endpoint:  endpoint,
			Message:   "network request failed",
			Cause:     cause,
		},
	}
}

// Error implements the error interface
func (e *NetworkError) Error() string {
	if e.GeminiError.Cause != nil {
		return fmt.Sprintf("network error during %s: %v", e.GeminiError.Operation, e.GeminiError.Cause)
	}
	return fmt.Sprintf("network error during %s", e.GeminiError.Operation)
}

// Is allows comparison with sentinel errors
func (e *NetworkError) Is(target error) bool {
	if target == ErrNetworkFailure {
		return true
	}
	if target == ErrTimeout && e.GeminiError.IsTimeout() {
		return true
	}
	if _, ok := target.(*NetworkError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *NetworkError) Unwrap() error {
	return e.GeminiError.Cause
}

// TimeoutError represents a request timeout
type TimeoutError struct {
	*GeminiError
}

// NewTimeoutError creates a new TimeoutError
func NewTimeoutError(message string) *TimeoutError {
	return &TimeoutError{
		GeminiError: &GeminiError{
			Operation: "request",
			Message:   message,
		},
	}
}

// NewTimeoutErrorWithEndpoint creates a TimeoutError with endpoint info
func NewTimeoutErrorWithEndpoint(endpoint string, cause error) *TimeoutError {
	return &TimeoutError{
		GeminiError: &GeminiError{
			Operation: "request",
			Endpoint:  endpoint,
			Message:   "request timed out",
			Cause:     cause,
		},
	}
}

// Error implements the error interface
func (e *TimeoutError) Error() string {
	if e.GeminiError.Message == "" {
		return "request timed out"
	}
	return fmt.Sprintf("request timed out: %s", e.GeminiError.Message)
}

// Is allows comparison with sentinel errors
func (e *TimeoutError) Is(target error) bool {
	if target == ErrTimeout {
		return true
	}
	if _, ok := target.(*TimeoutError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *TimeoutError) Unwrap() error {
	return e.GeminiError.Cause
}

// UsageLimitError represents a usage limit exceeded error
type UsageLimitError struct {
	*GeminiError
}

// NewUsageLimitError creates a new UsageLimitError
func NewUsageLimitError(modelName string) *UsageLimitError {
	return &UsageLimitError{
		GeminiError: &GeminiError{
			Code:      ErrCodeUsageLimitExceeded,
			Operation: "generate content",
			Message:   fmt.Sprintf("usage limit exceeded for model %s", modelName),
		},
	}
}

// Error implements the error interface
func (e *UsageLimitError) Error() string {
	if e.GeminiError.Message == "" {
		return "usage limit exceeded"
	}
	return fmt.Sprintf("usage limit exceeded: %s", e.GeminiError.Message)
}

// Is allows comparison with sentinel errors
func (e *UsageLimitError) Is(target error) bool {
	if target == ErrRateLimited {
		return true
	}
	if _, ok := target.(*UsageLimitError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *UsageLimitError) Unwrap() error {
	return e.GeminiError.Cause
}

// ModelError represents a model-related error
type ModelError struct {
	*GeminiError
}

// NewModelError creates a new ModelError
func NewModelError(message string) *ModelError {
	return &ModelError{
		GeminiError: &GeminiError{
			Operation: "model selection",
			Message:   message,
		},
	}
}

// NewModelErrorWithCode creates a ModelError with an error code
func NewModelErrorWithCode(code ErrorCode) *ModelError {
	return &ModelError{
		GeminiError: &GeminiError{
			Code:      code,
			Operation: "model selection",
			Message:   code.String(),
		},
	}
}

// Error implements the error interface
func (e *ModelError) Error() string {
	return fmt.Sprintf("model error: %s", e.GeminiError.Message)
}

// Is allows comparison with other ModelErrors
func (e *ModelError) Is(target error) bool {
	if _, ok := target.(*ModelError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *ModelError) Unwrap() error {
	return e.GeminiError.Cause
}

// BlockedError represents an IP block error
type BlockedError struct {
	*GeminiError
}

// NewBlockedError creates a new BlockedError
func NewBlockedError(message string) *BlockedError {
	return &BlockedError{
		GeminiError: &GeminiError{
			Code:      ErrCodeIPBlocked,
			Operation: "API request",
			Message:   message,
		},
	}
}

// Error implements the error interface
func (e *BlockedError) Error() string {
	if e.GeminiError.Message == "" {
		return "content blocked"
	}
	return fmt.Sprintf("blocked: %s", e.GeminiError.Message)
}

// Is allows comparison with other BlockedErrors
func (e *BlockedError) Is(target error) bool {
	if _, ok := target.(*BlockedError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *BlockedError) Unwrap() error {
	return e.GeminiError.Cause
}

// ParseError represents a response parsing error
type ParseError struct {
	*GeminiError
	Path string // JSON path where parsing failed
}

// NewParseError creates a new ParseError
func NewParseError(message, path string) *ParseError {
	return &ParseError{
		GeminiError: &GeminiError{
			Operation: "parse response",
			Message:   message,
		},
		Path: path,
	}
}

// Error implements the error interface
func (e *ParseError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("parse error at %s: %s", e.Path, e.GeminiError.Message)
	}
	return fmt.Sprintf("parse error: %s", e.GeminiError.Message)
}

// Is allows comparison with sentinel errors
func (e *ParseError) Is(target error) bool {
	if target == ErrInvalidResponse {
		return true
	}
	if _, ok := target.(*ParseError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *ParseError) Unwrap() error {
	return e.GeminiError.Cause
}

// PromptTooLongError represents an error when the prompt exceeds the model's context limit
type PromptTooLongError struct {
	*GeminiError
}

// NewPromptTooLongError creates a new PromptTooLongError
func NewPromptTooLongError(modelName string) *PromptTooLongError {
	return &PromptTooLongError{
		GeminiError: &GeminiError{
			Code:      ErrCodePromptTooLong,
			Operation: "generate content",
			Message:   fmt.Sprintf("prompt too long for model %s - try reducing input size", modelName),
		},
	}
}

// Error implements the error interface
func (e *PromptTooLongError) Error() string {
	return fmt.Sprintf("prompt too long: %s", e.GeminiError.Message)
}

// Is allows comparison with other PromptTooLongErrors
func (e *PromptTooLongError) Is(target error) bool {
	if _, ok := target.(*PromptTooLongError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *PromptTooLongError) Unwrap() error {
	return e.GeminiError.Cause
}

// HandleErrorCode converts API error codes to appropriate errors
func HandleErrorCode(code ErrorCode, endpoint, modelName string) error {
	switch code {
	case ErrCodePromptTooLong:
		return NewPromptTooLongError(modelName)
	case ErrCodeUsageLimitExceeded:
		return NewUsageLimitError(modelName)
	case ErrCodeModelInconsistent:
		return NewModelErrorWithCode(code)
	case ErrCodeModelHeaderInvalid:
		return NewModelErrorWithCode(code)
	case ErrCodeIPBlocked:
		return NewBlockedError("IP temporarily blocked by Google")
	default:
		return NewAPIErrorWithCode(code, endpoint)
	}
}

// IsAuthError checks if an error is an authentication error
// using errors.Is and errors.As for proper error wrapping support
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	// Check using errors.Is for sentinel error
	if errors.Is(err, ErrAuthFailed) {
		return true
	}

	// Check for AuthError type
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return true
	}

	// Check for APIError with 401 status
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GeminiError.HTTPStatus == 401
	}

	// Check for GeminiError with 401 status
	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.HTTPStatus == 401
	}

	return false
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrNetworkFailure) {
		return true
	}

	var netErr *NetworkError
	return errors.As(err, &netErr)
}

// IsTimeoutError checks if an error is a timeout error
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrTimeout) {
		return true
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return true
	}

	// Check for net.Error timeout
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	return false
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrRateLimited) {
		return true
	}

	var usageErr *UsageLimitError
	return errors.As(err, &usageErr)
}

// GetHTTPStatus extracts the HTTP status code from an error, if available
func GetHTTPStatus(err error) int {
	if err == nil {
		return 0
	}

	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.HTTPStatus
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GeminiError.HTTPStatus
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.GeminiError.HTTPStatus
	}

	return 0
}

// GetErrorCode extracts the Gemini error code from an error, if available
func GetErrorCode(err error) ErrorCode {
	if err == nil {
		return ErrCodeUnknown
	}

	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.Code
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GeminiError.Code
	}

	return ErrCodeUnknown
}

// GetEndpoint extracts the endpoint from an error, if available
func GetEndpoint(err error) string {
	if err == nil {
		return ""
	}

	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.Endpoint
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GeminiError.Endpoint
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.GeminiError.Endpoint
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return netErr.GeminiError.Endpoint
	}

	var uploadErr *UploadError
	if errors.As(err, &uploadErr) {
		return uploadErr.GeminiError.Endpoint
	}

	return ""
}

// UploadError represents a file upload failure
type UploadError struct {
	*GeminiError
	FileName string
}

// NewUploadError creates a new UploadError
func NewUploadError(fileName, message string) *UploadError {
	return &UploadError{
		GeminiError: &GeminiError{
			Operation: "upload file",
			Endpoint:  "https://content-push.googleapis.com/upload",
			Message:   message,
		},
		FileName: fileName,
	}
}

// NewUploadErrorWithStatus creates an UploadError with HTTP status
func NewUploadErrorWithStatus(fileName string, statusCode int, body string) *UploadError {
	e := &UploadError{
		GeminiError: &GeminiError{
			HTTPStatus: statusCode,
			Operation:  "upload file",
			Endpoint:   "https://content-push.googleapis.com/upload",
			Message:    fmt.Sprintf("upload failed with status %d", statusCode),
		},
		FileName: fileName,
	}
	e.GeminiError.WithBody(body)
	return e
}

// NewUploadNetworkError creates an UploadError for network failures
func NewUploadNetworkError(fileName string, cause error) *UploadError {
	return &UploadError{
		GeminiError: &GeminiError{
			Operation: "upload file",
			Endpoint:  "https://content-push.googleapis.com/upload",
			Message:   "network request failed",
			Cause:     cause,
		},
		FileName: fileName,
	}
}

// Error implements the error interface
func (e *UploadError) Error() string {
	if e.FileName != "" {
		if e.GeminiError.HTTPStatus > 0 {
			return fmt.Sprintf("upload error for '%s': HTTP %d - %s",
				e.FileName, e.GeminiError.HTTPStatus, e.GeminiError.Message)
		}
		if e.GeminiError.Cause != nil {
			return fmt.Sprintf("upload error for '%s': %v", e.FileName, e.GeminiError.Cause)
		}
		return fmt.Sprintf("upload error for '%s': %s", e.FileName, e.GeminiError.Message)
	}
	return fmt.Sprintf("upload error: %s", e.GeminiError.Message)
}

// Is allows comparison with other errors
func (e *UploadError) Is(target error) bool {
	if _, ok := target.(*UploadError); ok {
		return true
	}
	if target == ErrNetworkFailure && e.GeminiError.Cause != nil {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *UploadError) Unwrap() error {
	return e.GeminiError.Cause
}

// IsUploadError checks if an error is an upload error
func IsUploadError(err error) bool {
	if err == nil {
		return false
	}

	var uploadErr *UploadError
	return errors.As(err, &uploadErr)
}

// GemError represents errors specific to gem operations
type GemError struct {
	*GeminiError
	GemID   string
	GemName string
}

// NewGemError creates a new generic gem error
func NewGemError(gemID, gemName, message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "gem operation",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
		GemID:   gemID,
		GemName: gemName,
	}
}

// NewGemNotFoundError creates an error for gem not found
func NewGemNotFoundError(idOrName string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "get gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   fmt.Sprintf("gem '%s' not found", idOrName),
		},
	}
}

// NewGemReadOnlyError creates an error for attempting to modify a system gem
func NewGemReadOnlyError(gemName string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "modify gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   fmt.Sprintf("cannot modify system gem '%s'", gemName),
		},
		GemName: gemName,
	}
}

// NewGemCreateError creates an error for gem creation failure
func NewGemCreateError(name, message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "create gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
		GemName: name,
	}
}

// NewGemUpdateError creates an error for gem update failure
func NewGemUpdateError(gemID, message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "update gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
		GemID: gemID,
	}
}

// NewGemDeleteError creates an error for gem deletion failure
func NewGemDeleteError(gemID, message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "delete gem",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
		GemID: gemID,
	}
}

// NewGemFetchError creates an error for gem fetch failure
func NewGemFetchError(message string) *GemError {
	return &GemError{
		GeminiError: &GeminiError{
			Operation: "fetch gems",
			Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
			Message:   message,
		},
	}
}

// Error implements the error interface
func (e *GemError) Error() string {
	if e.GemName != "" {
		return fmt.Sprintf("gem error (%s): %s", e.GemName, e.GeminiError.Message)
	}
	if e.GemID != "" {
		return fmt.Sprintf("gem error (ID: %s): %s", e.GemID, e.GeminiError.Message)
	}
	return fmt.Sprintf("gem error: %s", e.GeminiError.Message)
}

// Is allows comparison with other errors
func (e *GemError) Is(target error) bool {
	if _, ok := target.(*GemError); ok {
		return true
	}
	return false
}

// Unwrap returns the underlying cause
func (e *GemError) Unwrap() error {
	return e.GeminiError.Cause
}

// IsGemError checks if an error is a gem error
func IsGemError(err error) bool {
	if err == nil {
		return false
	}

	var gemErr *GemError
	return errors.As(err, &gemErr)
}
</file>
<file path="internal/history/store.go">
// Package history provides local conversation history storage.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Thoughts  string    `json:"thoughts,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Conversation represents a complete chat conversation
type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`

	// Gemini API metadata for resuming
	CID  string `json:"cid,omitempty"`
	RID  string `json:"rid,omitempty"`
	RCID string `json:"rcid,omitempty"`
}

// Store manages conversation history persistence
type Store struct {
	baseDir string
	mu      sync.RWMutex
}

// NewStore creates a new history store
func NewStore(baseDir string) (*Store, error) {
	historyDir := filepath.Join(baseDir, "history")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	return &Store{
		baseDir: historyDir,
	}, nil
}

// CreateConversation creates a new conversation
func (s *Store) CreateConversation(model string) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	conv := &Conversation{
		ID:        generateConvID(),
		Title:     fmt.Sprintf("Chat %s", now.Format("2006-01-02 15:04")),
		Model:     model,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []Message{},
	}

	if err := s.saveConversation(conv); err != nil {
		return nil, err
	}

	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *Store) GetConversation(id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadConversation(id)
}

// ListConversations returns all conversations, sorted by most recent
func (s *Store) ListConversations() ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}

	var conversations []*Conversation
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5] // Remove .json
		conv, err := s.loadConversation(id)
		if err != nil {
			continue // Skip corrupted files
		}
		conversations = append(conversations, conv)
	}

	// Sort by UpdatedAt descending
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
	})

	return conversations, nil
}

// AddMessage adds a message to a conversation
func (s *Store) AddMessage(id, role, content, thoughts string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, err := s.loadConversation(id)
	if err != nil {
		return err
	}

	msg := Message{
		Role:      role,
		Content:   content,
		Thoughts:  thoughts,
		Timestamp: time.Now(),
	}

	conv.Messages = append(conv.Messages, msg)
	conv.UpdatedAt = time.Now()

	// Update title from first user message if still default
	if role == "user" && len(conv.Messages) == 1 {
		title := content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		conv.Title = title
	}

	return s.saveConversation(conv)
}

// UpdateMetadata updates the Gemini API metadata for a conversation
func (s *Store) UpdateMetadata(id, cid, rid, rcid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, err := s.loadConversation(id)
	if err != nil {
		return err
	}

	conv.CID = cid
	conv.RID = rid
	conv.RCID = rcid
	conv.UpdatedAt = time.Now()

	return s.saveConversation(conv)
}

// DeleteConversation removes a conversation
func (s *Store) DeleteConversation(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.conversationPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("conversation not found: %s", id)
		}
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

// UpdateTitle updates the title of a conversation
func (s *Store) UpdateTitle(id, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, err := s.loadConversation(id)
	if err != nil {
		return err
	}

	conv.Title = title
	conv.UpdatedAt = time.Now()

	return s.saveConversation(conv)
}

// ClearAll deletes all conversations
func (s *Store) ClearAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read history directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.baseDir, entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to delete %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Internal methods

func (s *Store) conversationPath(id string) string {
	return filepath.Join(s.baseDir, id+".json")
}

func (s *Store) loadConversation(id string) (*Conversation, error) {
	path := s.conversationPath(id)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("conversation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read conversation: %w", err)
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("failed to parse conversation: %w", err)
	}

	return &conv, nil
}

func (s *Store) saveConversation(conv *Conversation) error {
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	path := s.conversationPath(conv.ID)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write conversation: %w", err)
	}

	return nil
}

func generateConvID() string {
	return fmt.Sprintf("conv-%d", time.Now().UnixNano())
}

// GetHistoryDir returns the default history directory path
func GetHistoryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".geminiweb"), nil
}

// DefaultStore creates a store using the default location
func DefaultStore() (*Store, error) {
	dir, err := GetHistoryDir()
	if err != nil {
		return nil, err
	}
	return NewStore(dir)
}
</file>
<file path="internal/models/constants.go">
// Package models contains data types and constants for the Gemini Web API.
package models

import apierrors "github.com/diogo/geminiweb/internal/errors"

// Endpoints for Gemini Web API
const (
	EndpointGoogle        = "https://www.google.com"
	EndpointInit          = "https://gemini.google.com/app"
	EndpointGenerate      = "https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate"
	EndpointRotateCookies = "https://accounts.google.com/RotateCookies"
	EndpointUpload        = "https://content-push.googleapis.com/upload"
	EndpointBatchExec     = "https://gemini.google.com/_/BardChatUi/data/batchexecute"
)

// RPC IDs para operações de Gems (batch execute)
const (
	RPCListGems   = "CNgdBe"
	RPCCreateGem  = "oMH3Zd"
	RPCUpdateGem  = "kHv0Vd"
	RPCDeleteGem  = "UXcSJb"
)

// Parâmetros para ListGems
const (
	ListGemsNormal        = 3 // Gems normais (visíveis na UI)
	ListGemsIncludeHidden = 4 // Incluir gems ocultos de sistema
	ListGemsCustom        = 2 // Gems customizados do usuário
)

// Model represents available Gemini models with their headers
type Model struct {
	Name   string
	Header map[string]string
}

// Available models
var (
	ModelUnspecified = Model{
		Name:   "unspecified",
		Header: nil,
	}

	Model25Flash = Model{
		Name: "gemini-2.5-flash",
		Header: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"9ec249fc9ad08861",null,null,0,[4]]`,
		},
	}

	Model25Pro = Model{
		Name: "gemini-2.5-pro",
		Header: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"4af6c7f5da75d65d",null,null,0,[4]]`,
		},
	}

	Model30Pro = Model{
		Name: "gemini-3.0-pro",
		Header: map[string]string{
			"x-goog-ext-525001261-jspb": `[1,null,null,null,"9d8ca3786ebdfbea",null,null,0,[4]]`,
		},
	}
)

// AllModels returns a list of all available models
func AllModels() []Model {
	return []Model{Model25Flash, Model25Pro, Model30Pro}
}

// ModelFromName returns a Model by its name
func ModelFromName(name string) Model {
	switch name {
	case "gemini-2.5-flash":
		return Model25Flash
	case "gemini-2.5-pro":
		return Model25Pro
	case "gemini-3.0-pro":
		return Model30Pro
	default:
		return ModelUnspecified
	}
}

// ErrorCode represents known API error codes
// Deprecated: Use errors.ErrorCode instead. These are kept for backward compatibility.
type ErrorCode = apierrors.ErrorCode

// Error code constants - aliased from errors package for backward compatibility
const (
	ErrUsageLimitExceeded = apierrors.ErrCodeUsageLimitExceeded
	ErrModelInconsistent  = apierrors.ErrCodeModelInconsistent
	ErrModelHeaderInvalid = apierrors.ErrCodeModelHeaderInvalid
	ErrIPBlocked          = apierrors.ErrCodeIPBlocked
)

// DefaultHeaders returns the default headers for Gemini requests
func DefaultHeaders() map[string]string {
	return map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded;charset=utf-8",
		"Host":          "gemini.google.com",
		"Origin":        "https://gemini.google.com",
		"Referer":       "https://gemini.google.com/",
		"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"X-Same-Domain": "1",
	}
}

// RotateCookiesHeaders returns headers for the cookie rotation endpoint
func RotateCookiesHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// UploadHeaders returns headers for the file upload endpoint
func UploadHeaders() map[string]string {
	return map[string]string{
		"Push-ID": "feeds/mcudyrk2a4khkz",
	}
}
</file>
<file path="internal/models/gems.go">
package models

import "strings"

// Gem representa uma persona customizada armazenada no servidor Google
type Gem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	Predefined  bool   `json:"predefined"` // true = gem de sistema, false = custom
}

// GemJar é uma coleção de Gems indexada por ID
type GemJar map[string]*Gem

// Get retorna um Gem por ID ou nome
func (j GemJar) Get(id, name string) *Gem {
	if id != "" {
		if gem, ok := j[id]; ok {
			return gem
		}
	}
	if name != "" {
		for _, gem := range j {
			if gem.Name == name {
				return gem
			}
		}
	}
	return nil
}

// Filter filtra gems por critérios
func (j GemJar) Filter(predefined *bool, nameContains string) GemJar {
	result := make(GemJar)
	for id, gem := range j {
		if predefined != nil && gem.Predefined != *predefined {
			continue
		}
		if nameContains != "" && !strings.Contains(strings.ToLower(gem.Name), strings.ToLower(nameContains)) {
			continue
		}
		result[id] = gem
	}
	return result
}

// Custom retorna apenas gems customizados (não predefinidos)
func (j GemJar) Custom() GemJar {
	predefined := false
	return j.Filter(&predefined, "")
}

// System retorna apenas gems de sistema (predefinidos)
func (j GemJar) System() GemJar {
	predefined := true
	return j.Filter(&predefined, "")
}

// Values retorna todos os gems como slice
func (j GemJar) Values() []*Gem {
	gems := make([]*Gem, 0, len(j))
	for _, gem := range j {
		gems = append(gems, gem)
	}
	return gems
}

// Len retorna o número de gems
func (j GemJar) Len() int {
	return len(j)
}
</file>
<file path="internal/models/message.go">
package models

// Message represents a chat message for TUI display
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}
</file>
<file path="internal/models/response.go">
package models

// Candidate represents a single response candidate from Gemini
type Candidate struct {
	RCID            string
	Text            string
	Thoughts        string // Only populated for thinking models
	WebImages       []WebImage
	GeneratedImages []GeneratedImage
}

// WebImage represents an image from web search results
type WebImage struct {
	URL   string
	Title string
	Alt   string
}

// GeneratedImage represents an AI-generated image
type GeneratedImage struct {
	URL   string
	Title string
	Alt   string
}

// ModelOutput represents the complete API response from Gemini
type ModelOutput struct {
	Metadata   []string // [cid, rid, rcid]
	Candidates []Candidate
	Chosen     int // Index of selected candidate
}

// Text returns the chosen candidate's text
func (m *ModelOutput) Text() string {
	if len(m.Candidates) == 0 {
		return ""
	}
	if m.Chosen >= len(m.Candidates) {
		return m.Candidates[0].Text
	}
	return m.Candidates[m.Chosen].Text
}

// Thoughts returns the chosen candidate's thoughts
func (m *ModelOutput) Thoughts() string {
	if len(m.Candidates) == 0 {
		return ""
	}
	if m.Chosen >= len(m.Candidates) {
		return m.Candidates[0].Thoughts
	}
	return m.Candidates[m.Chosen].Thoughts
}

// RCID returns the chosen candidate's RCID
func (m *ModelOutput) RCID() string {
	if len(m.Candidates) == 0 {
		return ""
	}
	if m.Chosen >= len(m.Candidates) {
		return m.Candidates[0].RCID
	}
	return m.Candidates[m.Chosen].RCID
}

// Images returns all images from the chosen candidate (web + generated)
func (m *ModelOutput) Images() []WebImage {
	if len(m.Candidates) == 0 {
		return nil
	}
	candidate := m.Candidates[m.Chosen]

	images := make([]WebImage, 0, len(candidate.WebImages)+len(candidate.GeneratedImages))
	images = append(images, candidate.WebImages...)

	// Convert generated images to WebImage format
	for _, img := range candidate.GeneratedImages {
		images = append(images, WebImage{
			URL:   img.URL,
			Title: img.Title,
			Alt:   img.Alt,
		})
	}

	return images
}

// CID returns the conversation ID from metadata
func (m *ModelOutput) CID() string {
	if len(m.Metadata) > 0 {
		return m.Metadata[0]
	}
	return ""
}

// RID returns the reply ID from metadata
func (m *ModelOutput) RID() string {
	if len(m.Metadata) > 1 {
		return m.Metadata[1]
	}
	return ""
}
</file>
<file path="internal/render/themes/catppuccin.json">
{
  "document": {
    "color": "#cdd6f4",
    "margin": 2
  },
  "block_quote": {
    "color": "#6c7086",
    "indent": 2,
    "indent_token": "| "
  },
  "paragraph": {},
  "list": {
    "level_indent": 2
  },
  "heading": {
    "block_suffix": "\n",
    "color": "#89b4fa",
    "bold": true
  },
  "h1": {
    "prefix": "# ",
    "color": "#89b4fa",
    "bold": true
  },
  "h2": {
    "prefix": "## ",
    "color": "#89b4fa"
  },
  "h3": {
    "prefix": "### ",
    "color": "#cba6f7"
  },
  "h4": {
    "prefix": "#### ",
    "color": "#cba6f7"
  },
  "h5": {
    "prefix": "##### ",
    "color": "#6c7086"
  },
  "h6": {
    "prefix": "###### ",
    "color": "#6c7086"
  },
  "text": {},
  "strikethrough": {
    "crossed_out": true
  },
  "emph": {
    "italic": true
  },
  "strong": {
    "bold": true,
    "color": "#cdd6f4"
  },
  "hr": {
    "color": "#45475a",
    "format": "\n--------\n"
  },
  "item": {
    "block_prefix": "* "
  },
  "enumeration": {
    "block_prefix": ". "
  },
  "task": {
    "ticked": "[x] ",
    "unticked": "[ ] "
  },
  "link": {
    "color": "#cba6f7",
    "underline": true
  },
  "link_text": {
    "color": "#89b4fa"
  },
  "image": {
    "color": "#f9e2af"
  },
  "image_text": {
    "color": "#f9e2af",
    "format": "Image: {{.text}}"
  },
  "code": {
    "color": "#a6e3a1",
    "background_color": "#313244"
  },
  "code_block": {
    "color": "#cdd6f4",
    "margin": 2,
    "block_prefix": "\n----------\n",
    "block_suffix": "----------\n",
    "chroma": {
      "text": {
        "color": "#cdd6f4"
      },
      "error": {
        "color": "#f38ba8"
      },
      "comment": {
        "color": "#6c7086"
      },
      "comment_preproc": {
        "color": "#6c7086"
      },
      "keyword": {
        "color": "#cba6f7"
      },
      "keyword_reserved": {
        "color": "#cba6f7"
      },
      "keyword_namespace": {
        "color": "#cba6f7"
      },
      "keyword_type": {
        "color": "#89b4fa"
      },
      "operator": {
        "color": "#89dceb"
      },
      "punctuation": {
        "color": "#cdd6f4"
      },
      "name": {
        "color": "#cdd6f4"
      },
      "name_builtin": {
        "color": "#89b4fa"
      },
      "name_tag": {
        "color": "#f38ba8"
      },
      "name_attribute": {
        "color": "#cba6f7"
      },
      "name_class": {
        "color": "#f9e2af"
      },
      "name_constant": {
        "color": "#fab387"
      },
      "name_decorator": {
        "color": "#f9e2af"
      },
      "name_exception": {
        "color": "#f38ba8"
      },
      "name_function": {
        "color": "#89b4fa"
      },
      "name_other": {
        "color": "#cdd6f4"
      },
      "literal": {
        "color": "#fab387"
      },
      "literal_number": {
        "color": "#fab387"
      },
      "literal_date": {
        "color": "#fab387"
      },
      "literal_string": {
        "color": "#a6e3a1"
      },
      "literal_string_escape": {
        "color": "#89dceb"
      },
      "generic_deleted": {
        "color": "#f38ba8"
      },
      "generic_emph": {
        "italic": true
      },
      "generic_inserted": {
        "color": "#a6e3a1"
      },
      "generic_strong": {
        "bold": true
      },
      "generic_subheading": {
        "color": "#89b4fa"
      },
      "background": {
        "background_color": "#1e1e2e"
      }
    }
  },
  "table": {
    "center_separator": "+",
    "column_separator": "|",
    "row_separator": "-"
  },
  "definition_list": {},
  "definition_term": {},
  "definition_description": {
    "block_prefix": "\n"
  },
  "html_block": {},
  "html_span": {}
}
</file>
<file path="internal/render/themes/dark.json">
{
  "document": {
    "block_prefix": "\n",
    "block_suffix": "\n",
    "color": "252",
    "margin": 2
  },
  "block_quote": {
    "indent": 1,
    "indent_token": "| "
  },
  "paragraph": {},
  "list": {
    "level_indent": 2
  },
  "heading": {
    "block_suffix": "\n",
    "color": "39",
    "bold": true
  },
  "h1": {
    "prefix": " ",
    "suffix": " ",
    "color": "228",
    "background_color": "63",
    "bold": true
  },
  "h2": {
    "prefix": "## "
  },
  "h3": {
    "prefix": "### "
  },
  "h4": {
    "prefix": "#### "
  },
  "h5": {
    "prefix": "##### "
  },
  "h6": {
    "prefix": "###### ",
    "color": "35",
    "bold": false
  },
  "text": {},
  "strikethrough": {
    "crossed_out": true
  },
  "emph": {
    "italic": true
  },
  "strong": {
    "bold": true
  },
  "hr": {
    "color": "240",
    "format": "\n--------\n"
  },
  "item": {
    "block_prefix": "* "
  },
  "enumeration": {
    "block_prefix": ". "
  },
  "task": {
    "ticked": "[x] ",
    "unticked": "[ ] "
  },
  "link": {
    "color": "30",
    "underline": true
  },
  "link_text": {
    "color": "35",
    "bold": true
  },
  "image": {
    "color": "212",
    "underline": true
  },
  "image_text": {
    "color": "243",
    "format": "Image: {{.text}}"
  },
  "code": {
    "prefix": " ",
    "suffix": " ",
    "color": "203",
    "background_color": "236"
  },
  "code_block": {
    "color": "244",
    "margin": 2,
    "block_prefix": "\n----------\n",
    "block_suffix": "----------\n",
    "chroma": {
      "text": {
        "color": "#C4C4C4"
      },
      "error": {
        "color": "#F1F1F1",
        "background_color": "#F05B5B"
      },
      "comment": {
        "color": "#676767"
      },
      "comment_preproc": {
        "color": "#FF875F"
      },
      "keyword": {
        "color": "#00AAFF"
      },
      "keyword_reserved": {
        "color": "#FF5FD2"
      },
      "keyword_namespace": {
        "color": "#FF5F87"
      },
      "keyword_type": {
        "color": "#6E6ED8"
      },
      "operator": {
        "color": "#EF8080"
      },
      "punctuation": {
        "color": "#E8E8A8"
      },
      "name": {
        "color": "#C4C4C4"
      },
      "name_builtin": {
        "color": "#FF8EC7"
      },
      "name_tag": {
        "color": "#B083EA"
      },
      "name_attribute": {
        "color": "#7A7AE6"
      },
      "name_class": {
        "color": "#F1F1F1",
        "underline": true,
        "bold": true
      },
      "name_constant": {},
      "name_decorator": {
        "color": "#FFFF87"
      },
      "name_exception": {},
      "name_function": {
        "color": "#00D787"
      },
      "name_other": {},
      "literal": {},
      "literal_number": {
        "color": "#6EEFC0"
      },
      "literal_date": {},
      "literal_string": {
        "color": "#C69669"
      },
      "literal_string_escape": {
        "color": "#AFFFD7"
      },
      "generic_deleted": {
        "color": "#FD5B5B"
      },
      "generic_emph": {
        "italic": true
      },
      "generic_inserted": {
        "color": "#00D787"
      },
      "generic_strong": {
        "bold": true
      },
      "generic_subheading": {
        "color": "#777777"
      },
      "background": {
        "background_color": "#373737"
      }
    }
  },
  "table": {
    "center_separator": "+",
    "column_separator": "|",
    "row_separator": "-"
  },
  "definition_list": {},
  "definition_term": {},
  "definition_description": {
    "block_prefix": "\n"
  },
  "html_block": {},
  "html_span": {}
}
</file>
<file path="internal/render/themes/tokyonight.json">
{
  "document": {
    "color": "#c0caf5",
    "margin": 2
  },
  "block_quote": {
    "color": "#565f89",
    "indent": 2,
    "indent_token": "| "
  },
  "paragraph": {},
  "list": {
    "level_indent": 2
  },
  "heading": {
    "block_suffix": "\n",
    "color": "#7aa2f7",
    "bold": true
  },
  "h1": {
    "prefix": "# ",
    "color": "#7aa2f7",
    "bold": true
  },
  "h2": {
    "prefix": "## ",
    "color": "#7aa2f7"
  },
  "h3": {
    "prefix": "### ",
    "color": "#bb9af7"
  },
  "h4": {
    "prefix": "#### ",
    "color": "#bb9af7"
  },
  "h5": {
    "prefix": "##### ",
    "color": "#565f89"
  },
  "h6": {
    "prefix": "###### ",
    "color": "#565f89"
  },
  "text": {},
  "strikethrough": {
    "crossed_out": true
  },
  "emph": {
    "italic": true
  },
  "strong": {
    "bold": true,
    "color": "#c0caf5"
  },
  "hr": {
    "color": "#414868",
    "format": "\n--------\n"
  },
  "item": {
    "block_prefix": "* "
  },
  "enumeration": {
    "block_prefix": ". "
  },
  "task": {
    "ticked": "[x] ",
    "unticked": "[ ] "
  },
  "link": {
    "color": "#bb9af7",
    "underline": true
  },
  "link_text": {
    "color": "#7aa2f7"
  },
  "image": {
    "color": "#e0af68"
  },
  "image_text": {
    "color": "#e0af68",
    "format": "Image: {{.text}}"
  },
  "code": {
    "color": "#9ece6a",
    "background_color": "#24283b"
  },
  "code_block": {
    "color": "#c0caf5",
    "margin": 2,
    "block_prefix": "\n----------\n",
    "block_suffix": "----------\n",
    "chroma": {
      "text": {
        "color": "#c0caf5"
      },
      "error": {
        "color": "#f7768e"
      },
      "comment": {
        "color": "#565f89"
      },
      "comment_preproc": {
        "color": "#565f89"
      },
      "keyword": {
        "color": "#bb9af7"
      },
      "keyword_reserved": {
        "color": "#bb9af7"
      },
      "keyword_namespace": {
        "color": "#bb9af7"
      },
      "keyword_type": {
        "color": "#7aa2f7"
      },
      "operator": {
        "color": "#89ddff"
      },
      "punctuation": {
        "color": "#c0caf5"
      },
      "name": {
        "color": "#c0caf5"
      },
      "name_builtin": {
        "color": "#7aa2f7"
      },
      "name_tag": {
        "color": "#f7768e"
      },
      "name_attribute": {
        "color": "#bb9af7"
      },
      "name_class": {
        "color": "#e0af68"
      },
      "name_constant": {
        "color": "#ff9e64"
      },
      "name_decorator": {
        "color": "#e0af68"
      },
      "name_exception": {
        "color": "#f7768e"
      },
      "name_function": {
        "color": "#7aa2f7"
      },
      "name_other": {
        "color": "#c0caf5"
      },
      "literal": {
        "color": "#ff9e64"
      },
      "literal_number": {
        "color": "#ff9e64"
      },
      "literal_date": {
        "color": "#ff9e64"
      },
      "literal_string": {
        "color": "#9ece6a"
      },
      "literal_string_escape": {
        "color": "#89ddff"
      },
      "generic_deleted": {
        "color": "#f7768e"
      },
      "generic_emph": {
        "italic": true
      },
      "generic_inserted": {
        "color": "#9ece6a"
      },
      "generic_strong": {
        "bold": true
      },
      "generic_subheading": {
        "color": "#7aa2f7"
      },
      "background": {
        "background_color": "#1a1b26"
      }
    }
  },
  "table": {
    "center_separator": "+",
    "column_separator": "|",
    "row_separator": "-"
  },
  "definition_list": {},
  "definition_term": {},
  "definition_description": {
    "block_prefix": "\n"
  },
  "html_block": {},
  "html_span": {}
}
</file>
<file path="internal/render/cache.go">
package render

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/glamour"
)

// rendererPool uses sync.Pool for thread-safe renderer reuse.
// Note: glamour.TermRenderer is NOT thread-safe for concurrent Render() calls,
// so we use sync.Pool to efficiently reuse renderers without sharing them.
type rendererPool struct {
	mu    sync.RWMutex
	pools map[string]*sync.Pool
}

var globalPool = &rendererPool{
	pools: make(map[string]*sync.Pool),
}

// cacheKey generates a unique key based on options.
func cacheKey(opts Options) string {
	return fmt.Sprintf("%s:%d:%t:%t:%t:%t",
		opts.Style,
		opts.Width,
		opts.EnableEmoji,
		opts.PreserveNewLines,
		opts.TableWrap,
		opts.InlineTableLinks,
	)
}

// getPool returns or creates a pool for the given options.
func (p *rendererPool) getPool(opts Options) *sync.Pool {
	key := cacheKey(opts)

	// Try fast read
	p.mu.RLock()
	if pool, ok := p.pools[key]; ok {
		p.mu.RUnlock()
		return pool
	}
	p.mu.RUnlock()

	// Create new pool
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if pool, ok := p.pools[key]; ok {
		return pool
	}

	pool := &sync.Pool{
		New: func() interface{} {
			renderer, err := createRenderer(opts)
			if err != nil {
				return nil
			}
			return renderer
		},
	}
	p.pools[key] = pool
	return pool
}

// get retrieves a renderer from the pool.
func (p *rendererPool) get(opts Options) (*glamour.TermRenderer, error) {
	pool := p.getPool(opts)
	renderer := pool.Get()
	if renderer == nil {
		// Pool's New function failed, try creating directly
		return createRenderer(opts)
	}
	return renderer.(*glamour.TermRenderer), nil
}

// put returns a renderer to the pool.
func (p *rendererPool) put(opts Options, renderer *glamour.TermRenderer) {
	if renderer == nil {
		return
	}
	pool := p.getPool(opts)
	pool.Put(renderer)
}

// createRenderer creates a new TermRenderer with the specified options.
func createRenderer(opts Options) (*glamour.TermRenderer, error) {
	style := opts.Style

	// Handle custom built-in themes (dark, tokyonight, catppuccin have custom versions with separators)
	if opts.Style == ThemeDark || opts.Style == ThemeTokyoNight || opts.Style == ThemeCatppuccin {
		tmpFile, err := WriteThemeToTempFile(opts.Style)
		if err != nil {
			return nil, err
		}
		if tmpFile != "" {
			style = tmpFile
		}
	}

	rendererOpts := []glamour.TermRendererOption{
		glamour.WithStylePath(style),
		glamour.WithWordWrap(opts.Width),
		glamour.WithTableWrap(opts.TableWrap),
		glamour.WithInlineTableLinks(opts.InlineTableLinks),
	}

	if opts.EnableEmoji {
		rendererOpts = append(rendererOpts, glamour.WithEmoji())
	}

	if opts.PreserveNewLines {
		rendererOpts = append(rendererOpts, glamour.WithPreservedNewLines())
	}

	return glamour.NewTermRenderer(rendererOpts...)
}

// ClearCache clears the renderer pools and theme cache (useful for testing).
func ClearCache() {
	globalPool.mu.Lock()
	globalPool.pools = make(map[string]*sync.Pool)
	globalPool.mu.Unlock()
	ClearThemeCache()
}

// CacheSize returns the number of unique pool configurations.
func CacheSize() int {
	globalPool.mu.RLock()
	defer globalPool.mu.RUnlock()
	return len(globalPool.pools)
}
</file>
<file path="internal/render/config.go">
package render

import (
	"os"

	"github.com/diogo/geminiweb/internal/config"
)

// LoadOptionsFromConfig loads render options from user configuration.
// Environment variables take precedence over config file values.
func LoadOptionsFromConfig() Options {
	opts := DefaultOptions()

	// Load from config file
	cfg, err := config.LoadConfig()
	if err == nil {
		md := cfg.Markdown
		// Only apply non-zero values from config
		if md.Style != "" {
			opts.Style = md.Style
		}
		// These booleans always overwrite defaults since they have explicit defaults in config
		opts.EnableEmoji = md.EnableEmoji
		opts.PreserveNewLines = md.PreserveNewLines
		opts.TableWrap = md.TableWrap
		opts.InlineTableLinks = md.InlineTableLinks
	}

	// Environment variable takes highest precedence for style
	if style := os.Getenv("GLAMOUR_STYLE"); style != "" {
		opts.Style = style
	}

	return opts
}

// LoadOptionsFromConfigWithWidth loads options from config with a specific width.
func LoadOptionsFromConfigWithWidth(width int) Options {
	opts := LoadOptionsFromConfig()
	opts.Width = width
	return opts
}
</file>
<file path="internal/render/options.go">
// Package render provides markdown rendering utilities for terminal output.
package render

// Options configures the markdown renderer behavior.
type Options struct {
	// Width defines the maximum output width (default: 80)
	Width int

	// Style defines the theme: "dark", "light", or path to JSON file
	Style string

	// EnableEmoji converts :emoji: to unicode characters
	EnableEmoji bool

	// PreserveNewLines preserves original line breaks
	PreserveNewLines bool

	// TableWrap enables word wrap in table cells (glamour v0.10.0+)
	TableWrap bool

	// InlineTableLinks renders links inline in tables (glamour v0.10.0+)
	InlineTableLinks bool
}

// DefaultOptions returns the default configuration.
func DefaultOptions() Options {
	return Options{
		Width:            80,
		Style:            "dark",
		EnableEmoji:      true,
		PreserveNewLines: true,
		TableWrap:        true,
		InlineTableLinks: false,
	}
}

// WithWidth returns Options with the specified width.
func (o Options) WithWidth(width int) Options {
	o.Width = width
	return o
}

// WithStyle returns Options with the specified style.
func (o Options) WithStyle(style string) Options {
	o.Style = style
	return o
}

// WithEmoji returns Options with emoji support enabled/disabled.
func (o Options) WithEmoji(enabled bool) Options {
	o.EnableEmoji = enabled
	return o
}

// WithPreserveNewLines returns Options with newline preservation enabled/disabled.
func (o Options) WithPreserveNewLines(enabled bool) Options {
	o.PreserveNewLines = enabled
	return o
}

// WithTableWrap returns Options with table wrap enabled/disabled.
func (o Options) WithTableWrap(enabled bool) Options {
	o.TableWrap = enabled
	return o
}

// WithInlineTableLinks returns Options with inline table links enabled/disabled.
func (o Options) WithInlineTableLinks(enabled bool) Options {
	o.InlineTableLinks = enabled
	return o
}
</file>
<file path="internal/render/render.go">
package render

// Markdown renders markdown content for terminal display.
// Uses a pooled renderer for better performance and thread safety.
func Markdown(content string, opts Options) (string, error) {
	renderer, err := globalPool.get(opts)
	if err != nil {
		return "", err
	}
	defer globalPool.put(opts, renderer)

	return renderer.Render(content)
}

// MarkdownWithWidth is a convenience function for rendering with specific width.
// Uses default options with the specified width.
func MarkdownWithWidth(content string, width int) (string, error) {
	opts := DefaultOptions().WithWidth(width)
	return Markdown(content, opts)
}
</file>
<file path="internal/render/themes.go">
package render

import (
	_ "embed"
	"os"
	"path/filepath"
	"sync"
)

//go:embed themes/dark.json
var darkTheme []byte

//go:embed themes/tokyonight.json
var tokyoNightTheme []byte

//go:embed themes/catppuccin.json
var catppuccinTheme []byte

// BuiltinTheme represents a built-in theme name
const (
	ThemeDark       = "dark"
	ThemeLight      = "light"
	ThemeTokyoNight = "tokyonight"
	ThemeCatppuccin = "catppuccin"
)

// themeFileCache stores paths to written theme files
var (
	themeFileMu    sync.RWMutex
	themeFileCache = make(map[string]string)
)

// GetBuiltinTheme returns the content of a built-in theme by name.
// Returns nil and false if the theme is not a built-in theme.
func GetBuiltinTheme(name string) ([]byte, bool) {
	switch name {
	case ThemeDark:
		return darkTheme, true
	case ThemeTokyoNight:
		return tokyoNightTheme, true
	case ThemeCatppuccin:
		return catppuccinTheme, true
	default:
		return nil, false
	}
}

// IsBuiltinStyle returns true if the style is a built-in style
// (either glamour built-in or our custom built-in themes).
func IsBuiltinStyle(style string) bool {
	switch style {
	case ThemeDark, ThemeLight, "dracula", "notty", "ascii":
		return true
	case ThemeTokyoNight, ThemeCatppuccin:
		return true
	default:
		return false
	}
}

// WriteThemeToTempFile writes a built-in theme to a temporary file
// and returns the file path. Thread-safe and caches file paths.
func WriteThemeToTempFile(name string) (string, error) {
	content, ok := GetBuiltinTheme(name)
	if !ok {
		return "", nil // Not a built-in theme, return empty string
	}

	// Check if already written and file still exists
	themeFileMu.RLock()
	if path, ok := themeFileCache[name]; ok {
		if _, err := os.Stat(path); err == nil {
			themeFileMu.RUnlock()
			return path, nil
		}
		// File was deleted, need to rewrite
	}
	themeFileMu.RUnlock()

	// Write the theme file
	themeFileMu.Lock()
	defer themeFileMu.Unlock()

	// Double-check after acquiring write lock
	if path, ok := themeFileCache[name]; ok {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "geminiweb-theme-"+name+".json")

	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		return "", err
	}

	themeFileCache[name] = tmpFile
	return tmpFile, nil
}

// ClearThemeCache clears the theme file cache (useful for testing).
func ClearThemeCache() {
	themeFileMu.Lock()
	defer themeFileMu.Unlock()
	themeFileCache = make(map[string]string)
}

// ThemeInfo contains information about a theme for display purposes.
type ThemeInfo struct {
	Name        string
	Description string
}

// AvailableThemes returns a list of all available themes (built-in and glamour styles).
func AvailableThemes() []ThemeInfo {
	return []ThemeInfo{
		{Name: ThemeDark, Description: "Dark theme (default)"},
		{Name: ThemeTokyoNight, Description: "Tokyo Night color scheme"},
		{Name: ThemeCatppuccin, Description: "Catppuccin Mocha color scheme"},
		{Name: ThemeLight, Description: "Light theme for bright terminals"},
		{Name: "dracula", Description: "Dracula color scheme"},
		{Name: "notty", Description: "Plain text (no styling)"},
		{Name: "ascii", Description: "ASCII-only output"},
	}
}

// ThemeNames returns just the theme names for selection.
func ThemeNames() []string {
	themes := AvailableThemes()
	names := make([]string, len(themes))
	for i, t := range themes {
		names[i] = t.Name
	}
	return names
}
</file>
<file path="internal/tui/config_model.go">
package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/render"
)

// configView represents the current view in the config menu
type configView int

const (
	viewMain configView = iota
	viewModelSelect
	viewThemeSelect
)

// Menu item indices for main view
const (
	menuDefaultModel = iota
	menuVerbose
	menuAutoClose
	menuCopyToClipboard
	menuTheme
	menuExit
	menuItemCount
)

// feedbackClearMsg is sent to clear feedback messages
type feedbackClearMsg struct{}

// ConfigModel represents the config TUI state
type ConfigModel struct {
	config       config.Config
	configDir    string
	cookiesPath  string
	cookiesExist bool

	// Navigation
	view        configView
	cursor      int
	modelCursor int
	themeCursor int

	// Feedback
	feedback        string
	feedbackTimeout time.Duration

	// Dimensions
	width  int
	height int
	ready  bool
}

// NewConfigModel creates a new config TUI model
func NewConfigModel() ConfigModel {
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	configDir, _ := config.GetConfigDir()
	cookiesPath, _ := config.GetCookiesPath()

	cookiesExist := false
	if _, err := os.Stat(cookiesPath); err == nil {
		cookiesExist = true
	}

	// Find current model index
	modelCursor := 0
	models := config.AvailableModels()
	for i, m := range models {
		if m == cfg.DefaultModel {
			modelCursor = i
			break
		}
	}

	// Find current theme index
	themeCursor := 0
	themes := render.ThemeNames()
	currentTheme := cfg.Markdown.Style
	if currentTheme == "" {
		currentTheme = render.ThemeDark
	}
	for i, t := range themes {
		if t == currentTheme {
			themeCursor = i
			break
		}
	}

	return ConfigModel{
		config:          cfg,
		configDir:       configDir,
		cookiesPath:     cookiesPath,
		cookiesExist:    cookiesExist,
		view:            viewMain,
		cursor:          0,
		modelCursor:     modelCursor,
		themeCursor:     themeCursor,
		feedbackTimeout: 2 * time.Second,
	}
}

// Init initializes the model
func (m ConfigModel) Init() tea.Cmd {
	return nil
}

// clearFeedback returns a command that clears the feedback message after a delay
func clearFeedback(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return feedbackClearMsg{}
	})
}

// Update handles messages and updates the model
func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case feedbackClearMsg:
		m.feedback = ""

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.view == viewModelSelect || m.view == viewThemeSelect {
				m.view = viewMain
			} else {
				return m, tea.Quit
			}

		case "up", "k":
			if m.view == viewMain {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = menuItemCount - 1
				}
			} else if m.view == viewModelSelect {
				m.modelCursor--
				if m.modelCursor < 0 {
					m.modelCursor = len(config.AvailableModels()) - 1
				}
			} else if m.view == viewThemeSelect {
				m.themeCursor--
				if m.themeCursor < 0 {
					m.themeCursor = len(render.ThemeNames()) - 1
				}
			}

		case "down", "j":
			if m.view == viewMain {
				m.cursor++
				if m.cursor >= menuItemCount {
					m.cursor = 0
				}
			} else if m.view == viewModelSelect {
				m.modelCursor++
				if m.modelCursor >= len(config.AvailableModels()) {
					m.modelCursor = 0
				}
			} else if m.view == viewThemeSelect {
				m.themeCursor++
				if m.themeCursor >= len(render.ThemeNames()) {
					m.themeCursor = 0
				}
			}

		case "enter", " ":
			return m.handleSelect()
		}
	}

	return m, nil
}

// handleSelect handles menu item selection
func (m ConfigModel) handleSelect() (tea.Model, tea.Cmd) {
	if m.view == viewMain {
		switch m.cursor {
		case menuDefaultModel:
			m.view = viewModelSelect
			return m, nil

		case menuVerbose:
			m.config.Verbose = !m.config.Verbose
			if err := config.SaveConfig(m.config); err != nil {
				m.feedback = fmt.Sprintf("Error: %v", err)
			} else {
				state := "disabled"
				if m.config.Verbose {
					state = "enabled"
				}
				m.feedback = fmt.Sprintf("Verbose logging %s", state)
			}
			return m, clearFeedback(m.feedbackTimeout)

		case menuAutoClose:
			m.config.AutoClose = !m.config.AutoClose
			if err := config.SaveConfig(m.config); err != nil {
				m.feedback = fmt.Sprintf("Error: %v", err)
			} else {
				state := "disabled"
				if m.config.AutoClose {
					state = "enabled"
				}
				m.feedback = fmt.Sprintf("Auto-close %s", state)
			}
			return m, clearFeedback(m.feedbackTimeout)

		case menuCopyToClipboard:
			m.config.CopyToClipboard = !m.config.CopyToClipboard
			if err := config.SaveConfig(m.config); err != nil {
				m.feedback = fmt.Sprintf("Error: %v", err)
			} else {
				state := "disabled"
				if m.config.CopyToClipboard {
					state = "enabled"
				}
				m.feedback = fmt.Sprintf("Copy to clipboard %s", state)
			}
			return m, clearFeedback(m.feedbackTimeout)

		case menuTheme:
			m.view = viewThemeSelect
			return m, nil

		case menuExit:
			return m, tea.Quit
		}
	} else if m.view == viewModelSelect {
		models := config.AvailableModels()
		m.config.DefaultModel = models[m.modelCursor]
		if err := config.SaveConfig(m.config); err != nil {
			m.feedback = fmt.Sprintf("Error: %v", err)
		} else {
			m.feedback = fmt.Sprintf("Model set to %s", m.config.DefaultModel)
		}
		m.view = viewMain
		return m, clearFeedback(m.feedbackTimeout)
	} else if m.view == viewThemeSelect {
		themes := render.ThemeNames()
		m.config.Markdown.Style = themes[m.themeCursor]
		if err := config.SaveConfig(m.config); err != nil {
			m.feedback = fmt.Sprintf("Error: %v", err)
		} else {
			m.feedback = fmt.Sprintf("Theme set to %s", m.config.Markdown.Style)
		}
		m.view = viewMain
		return m, clearFeedback(m.feedbackTimeout)
	}

	return m, nil
}

// View renders the TUI
func (m ConfigModel) View() string {
	if !m.ready {
		return loadingStyle.Render("  Initializing...")
	}

	var sections []string
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	// ═══════════════════════════════════════════════════════════════
	// HEADER
	// ═══════════════════════════════════════════════════════════════
	headerContent := configTitleStyle.Render("✦ Configuration")
	header := configHeaderStyle.Width(contentWidth).Render(headerContent)
	sections = append(sections, header)

	// ═══════════════════════════════════════════════════════════════
	// PATHS PANEL
	// ═══════════════════════════════════════════════════════════════
	pathsTitle := configSectionTitleStyle.Render("📁 Paths")

	configPath := configPathStyle.Render(m.configDir + "/config.json")
	cookiesPath := configPathStyle.Render(m.cookiesPath)

	var cookiesStatus string
	if m.cookiesExist {
		cookiesStatus = configStatusOkStyle.Render("✓ exists")
	} else {
		cookiesStatus = configStatusErrorStyle.Render("✗ not found")
	}

	pathsContent := lipgloss.JoinVertical(lipgloss.Left,
		pathsTitle,
		fmt.Sprintf("   Config:  %s", configPath),
		fmt.Sprintf("   Cookies: %s  %s", cookiesPath, cookiesStatus),
	)
	pathsPanel := configPanelStyle.Width(contentWidth).Render(pathsContent)
	sections = append(sections, pathsPanel)

	// ═══════════════════════════════════════════════════════════════
	// SETTINGS/MENU PANEL
	// ═══════════════════════════════════════════════════════════════
	var settingsContent string
	switch m.view {
	case viewMain:
		settingsContent = m.renderMainMenu(contentWidth)
	case viewModelSelect:
		settingsContent = m.renderModelSelect(contentWidth)
	case viewThemeSelect:
		settingsContent = m.renderThemeSelect(contentWidth)
	}

	settingsPanel := configPanelStyle.Width(contentWidth).Render(settingsContent)
	sections = append(sections, settingsPanel)

	// ═══════════════════════════════════════════════════════════════
	// FEEDBACK MESSAGE
	// ═══════════════════════════════════════════════════════════════
	if m.feedback != "" {
		feedbackMsg := configFeedbackStyle.Render("✓ " + m.feedback)
		sections = append(sections, feedbackMsg)
	}

	// ═══════════════════════════════════════════════════════════════
	// STATUS BAR
	// ═══════════════════════════════════════════════════════════════
	statusBar := m.renderStatusBar(contentWidth)
	sections = append(sections, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderMainMenu renders the main settings menu
func (m ConfigModel) renderMainMenu(width int) string {
	title := configSectionTitleStyle.Render("⚙ Settings")

	var items []string

	// Default Model
	cursor := "  "
	style := configMenuItemStyle
	if m.cursor == menuDefaultModel {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	modelValue := configValueStyle.Render(m.config.DefaultModel)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Default Model"),
		strings.Repeat(" ", 8),
		modelValue,
	))

	// Verbose
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuVerbose {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	verboseValue := m.renderBoolValue(m.config.Verbose)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Verbose Logging"),
		strings.Repeat(" ", 5),
		verboseValue,
	))

	// Auto Close
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuAutoClose {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	autoCloseValue := m.renderBoolValue(m.config.AutoClose)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Auto Close"),
		strings.Repeat(" ", 10),
		autoCloseValue,
	))

	// Copy to Clipboard
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuCopyToClipboard {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	clipboardValue := m.renderBoolValue(m.config.CopyToClipboard)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Copy to Clipboard"),
		strings.Repeat(" ", 3),
		clipboardValue,
	))

	// Theme
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuTheme {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	currentTheme := m.config.Markdown.Style
	if currentTheme == "" {
		currentTheme = render.ThemeDark
	}
	themeValue := configValueStyle.Render(currentTheme)
	items = append(items, fmt.Sprintf("%s%s%s%s",
		cursor,
		style.Render("Theme"),
		strings.Repeat(" ", 15),
		themeValue,
	))

	// Separator
	items = append(items, "")

	// Exit
	cursor = "  "
	style = configMenuItemStyle
	if m.cursor == menuExit {
		cursor = configCursorStyle.Render("▸ ")
		style = configMenuSelectedStyle
	}
	items = append(items, cursor+style.Render("Exit"))

	return lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, items...)...,
	)
}

// renderModelSelect renders the model selection sub-menu
func (m ConfigModel) renderModelSelect(width int) string {
	title := configSectionTitleStyle.Render("🤖 Select Model")

	models := config.AvailableModels()
	var items []string

	for i, model := range models {
		cursor := "  "
		style := configMenuItemStyle
		if m.modelCursor == i {
			cursor = configCursorStyle.Render("▸ ")
			style = configMenuSelectedStyle
		}

		current := ""
		if model == m.config.DefaultModel {
			current = configStatusOkStyle.Render(" (current)")
		}

		items = append(items, cursor+style.Render(model)+current)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, items...)...,
	)
}


// renderThemeSelect renders the theme selection sub-menu
func (m ConfigModel) renderThemeSelect(width int) string {
	title := configSectionTitleStyle.Render("🎨 Select Theme")

	themes := render.AvailableThemes()
	var items []string

	currentTheme := m.config.Markdown.Style
	if currentTheme == "" {
		currentTheme = render.ThemeDark
	}

	for i, theme := range themes {
		cursor := "  "
		style := configMenuItemStyle
		if m.themeCursor == i {
			cursor = configCursorStyle.Render("▸ ")
			style = configMenuSelectedStyle
		}

		current := ""
		if theme.Name == currentTheme {
			current = configStatusOkStyle.Render(" (current)")
		}

		// Format: "theme-name - description"
		themeText := fmt.Sprintf("%s - %s", theme.Name, theme.Description)
		items = append(items, cursor+style.Render(themeText)+current)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, items...)...,
	)
}

// renderBoolValue renders a boolean value with appropriate styling
func (m ConfigModel) renderBoolValue(value bool) string {
	if value {
		return configEnabledStyle.Render("enabled")
	}
	return configDisabledStyle.Render("disabled")
}

// renderStatusBar renders the bottom status bar
func (m ConfigModel) renderStatusBar(width int) string {
	var shortcuts []struct {
		key  string
		desc string
	}

	if m.view == viewMain {
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"↑↓", "Navigate"},
			{"Enter", "Select"},
			{"Esc", "Exit"},
		}
	} else {
		shortcuts = []struct {
			key  string
			desc string
		}{
			{"↑↓", "Navigate"},
			{"Enter", "Select"},
			{"Esc", "Back"},
		}
	}

	var items []string
	for _, s := range shortcuts {
		item := lipgloss.JoinHorizontal(
			lipgloss.Center,
			statusKeyStyle.Render(s.key),
			statusDescStyle.Render(" "+s.desc),
		)
		items = append(items, item)
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Center, strings.Join(items, "  │  "))
	return configStatusBarStyle.Width(width).Align(lipgloss.Center).Render(bar)
}

// RunConfig starts the config TUI
func RunConfig() error {
	m := NewConfigModel()

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
</file>
<file path="internal/tui/model.go">
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diogo/geminiweb/internal/api"
	apierrors "github.com/diogo/geminiweb/internal/errors"
	"github.com/diogo/geminiweb/internal/models"
	"github.com/diogo/geminiweb/internal/render"
)

// Animation tick message
type animationTickMsg time.Time

// Message types for the TUI
type (
	responseMsg struct {
		output *models.ModelOutput
	}
	errMsg struct {
		err error
	}
)

// ChatSessionInterface defines the interface for chat session operations needed by the TUI
type ChatSessionInterface interface {
	SendMessage(prompt string) (*models.ModelOutput, error)
	SetMetadata(cid, rid, rcid string)
	GetMetadata() []string
	CID() string
	RID() string
	RCID() string
	GetModel() models.Model
	SetModel(model models.Model)
	LastOutput() *models.ModelOutput
	ChooseCandidate(index int) error
}

// Model represents the TUI state
type Model struct {
	client    api.GeminiClientInterface
	session   ChatSessionInterface
	modelName string

	// UI components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// State
	messages       []chatMessage
	loading        bool
	ready          bool
	err            error
	animationFrame int // Frame counter for loading animation

	// Dimensions
	width  int
	height int
}

// chatMessage represents a message in the chat
type chatMessage struct {
	role     string // "user" or "assistant"
	content  string
	thoughts string
}

// NewChatModel creates a new chat TUI model
func NewChatModel(client api.GeminiClientInterface, modelName string) Model {
	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.CharLimit = 4000
	ta.ShowLineNumbers = false
	ta.SetHeight(2)
	ta.Focus()

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(colorText)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(colorTextDim)
	ta.BlurredStyle = ta.FocusedStyle

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = loadingStyle

	return Model{
		client:    client,
		session:   client.StartChat(),
		modelName: modelName,
		textarea:  ta,
		spinner:   s,
		messages:  []chatMessage{},
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// animationTick returns a command that sends animation tick messages
func animationTick() tea.Cmd {
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return animationTickMsg(t)
	})
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate component heights
		headerHeight := 4 // Header panel with border
		inputHeight := 6  // Input panel with border
		statusHeight := 1 // Status bar
		padding := 2      // Extra spacing

		vpHeight := m.height - headerHeight - inputHeight - statusHeight - padding
		if vpHeight < 5 {
			vpHeight = 5
		}

		contentWidth := m.width - 4

		// Initialize viewport on first size message
		if !m.ready {
			m.viewport = viewport.New(contentWidth, vpHeight)
			m.textarea.SetWidth(contentWidth - 4)
			m.ready = true
		} else {
			m.viewport.Width = contentWidth
			m.viewport.Height = vpHeight
			m.textarea.SetWidth(contentWidth - 4)
		}
		m.updateViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.loading {
				m.loading = false
			} else {
				return m, tea.Quit
			}

		case "enter":
			if !m.loading && strings.TrimSpace(m.textarea.Value()) != "" {
				// Check for exit commands
				input := strings.TrimSpace(m.textarea.Value())
				if input == "exit" || input == "quit" || input == "/exit" || input == "/quit" {
					return m, tea.Quit
				}

				// Add user message
				m.messages = append(m.messages, chatMessage{
					role:    "user",
					content: input,
				})
				m.updateViewport()
				m.viewport.GotoBottom()

				// Start loading
				m.loading = true
				m.err = nil
				m.animationFrame = 0
				userMsg := m.textarea.Value()
				m.textarea.Reset()

				cmd = m.sendMessage(userMsg)

				return m, tea.Batch(
					cmd,
					m.spinner.Tick,
					animationTick(),
				)
			}
		}

	case responseMsg:
		m.loading = false
		m.messages = append(m.messages, chatMessage{
			role:     "assistant",
			content:  msg.output.Text(),
			thoughts: msg.output.Thoughts(),
		})
		m.updateViewport()
		m.viewport.GotoBottom()

	case errMsg:
		m.loading = false
		m.err = msg.err

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case animationTickMsg:
		if m.loading {
			m.animationFrame++
			cmds = append(cmds, animationTick())
		}
	}

	// Update child components - only pass KeyMsg to textarea to prevent escape sequence leaks
	if !m.loading {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return loadingStyle.Render("  Initializing...")
	}

	var sections []string
	contentWidth := m.width - 4

	// ═══════════════════════════════════════════════════════════════
	// HEADER
	// ═══════════════════════════════════════════════════════════════
	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Render("✦ Gemini Chat"),
		hintStyle.Render("  •  "),
		subtitleStyle.Render(m.modelName),
	)
	header := headerStyle.Width(contentWidth).Render(headerContent)
	sections = append(sections, header)

	// ═══════════════════════════════════════════════════════════════
	// MESSAGES AREA
	// ═══════════════════════════════════════════════════════════════
	var messagesContent string
	if len(m.messages) == 0 {
		// Welcome message when empty
		messagesContent = m.renderWelcome()
	} else {
		messagesContent = m.viewport.View()
	}

	messagesPanel := messagesAreaStyle.
		Width(contentWidth).
		Height(m.viewport.Height).
		Render(messagesContent)
	sections = append(sections, messagesPanel)

	// ═══════════════════════════════════════════════════════════════
	// INPUT AREA
	// ═══════════════════════════════════════════════════════════════
	var inputContent string
	if m.loading {
		// Use colorful animated loading indicator
		inputContent = m.renderLoadingAnimation()
	} else {
		inputContent = lipgloss.JoinVertical(
			lipgloss.Left,
			inputLabelStyle.Render("You"),
			m.textarea.View(),
		)
	}

	inputPanel := inputPanelStyle.Width(contentWidth).Render(inputContent)
	sections = append(sections, inputPanel)

	// ═══════════════════════════════════════════════════════════════
	// STATUS BAR
	// ═══════════════════════════════════════════════════════════════
	statusBar := m.renderStatusBar(contentWidth)
	sections = append(sections, statusBar)

	// ═══════════════════════════════════════════════════════════════
	// ERROR DISPLAY
	// ═══════════════════════════════════════════════════════════════
	if m.err != nil {
		errorDisplay := m.formatError(m.err)
		sections = append(sections, errorDisplay)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderWelcome renders the welcome screen when no messages exist
func (m Model) renderWelcome() string {
	width := m.viewport.Width - 4
	height := m.viewport.Height

	icon := welcomeIconStyle.Width(width).Render("✦")
	title := welcomeTitleStyle.Width(width).Render("Welcome to Gemini Chat")
	subtitle := welcomeStyle.Width(width).Render("Start a conversation by typing a message below")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		icon,
		"",
		title,
		"",
		subtitle,
		"",
	)

	// Center vertically
	contentHeight := lipgloss.Height(content)
	topPadding := (height - contentHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	return strings.Repeat("\n", topPadding) + content
}

// renderLoadingAnimation renders a colorful animated loading indicator
func (m Model) renderLoadingAnimation() string {
	// Animation characters
	chars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	barChars := []string{"█", "█", "█", "█", "█", "█", "█", "█", "▓", "▒", "░"}

	// Get current animation frame
	frame := m.animationFrame

	// Render spinning character with color
	spinIdx := frame % len(chars)
	spinColor := gradientColors[frame%len(gradientColors)]
	spinner := lipgloss.NewStyle().Foreground(spinColor).Bold(true).Render(chars[spinIdx])

	// Render animated bar with gradient
	barWidth := 20
	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		// Calculate which color to use based on position and frame
		colorIdx := (i + frame) % len(gradientColors)
		charIdx := (i + frame/2) % len(barChars)

		style := lipgloss.NewStyle().Foreground(gradientColors[colorIdx])
		bar.WriteString(style.Render(barChars[charIdx]))
	}

	// Animated dots
	dots := ""
	numDots := (frame / 3) % 4
	for i := 0; i < numDots; i++ {
		dotColor := gradientColors[(frame+i)%len(gradientColors)]
		dots += lipgloss.NewStyle().Foreground(dotColor).Render("●")
	}
	for i := numDots; i < 3; i++ {
		dots += lipgloss.NewStyle().Foreground(colorTextMute).Render("○")
	}

	// Combine elements
	text := lipgloss.NewStyle().Foreground(colorText).Render(" Gemini is thinking ")

	return fmt.Sprintf("%s %s %s %s", spinner, bar.String(), text, dots)
}

// renderStatusBar renders the bottom status bar with shortcuts
func (m Model) renderStatusBar(width int) string {
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"Enter", "Send"},
		{"Esc", "Quit"},
		{"↑↓", "Scroll"},
	}

	var items []string
	for _, s := range shortcuts {
		item := lipgloss.JoinHorizontal(
			lipgloss.Center,
			statusKeyStyle.Render(s.key),
			statusDescStyle.Render(" "+s.desc),
		)
		items = append(items, item)
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Center, strings.Join(items, "  │  "))
	return statusBarStyle.Width(width).Align(lipgloss.Center).Render(bar)
}

// sendMessage creates a command to send a message to the API
func (m Model) sendMessage(prompt string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.session.SendMessage(prompt)
		if err != nil {
			return errMsg{err: err}
		}
		return responseMsg{output: output}
	}
}

// updateViewport refreshes the viewport content with styled messages
func (m *Model) updateViewport() {
	var content strings.Builder
	bubbleWidth := m.viewport.Width - 6

	for i, msg := range m.messages {
		if i > 0 {
			content.WriteString("\n")
		}

		if msg.role == "user" {
			// User message
			label := userLabelStyle.Render("⬤ You")
			bubble := userBubbleStyle.Width(bubbleWidth).Render(msg.content)
			content.WriteString(label + "\n" + bubble)
		} else {
			// Assistant message
			label := assistantLabelStyle.Render("✦ Gemini")

			// Render thoughts if present
			if msg.thoughts != "" {
				thoughtsContent := thoughtsStyle.Width(bubbleWidth - 4).Render(
					"💭 " + msg.thoughts,
				)
				content.WriteString(label + "\n" + thoughtsContent + "\n")
			} else {
				content.WriteString(label + "\n")
			}

			// Render markdown content
			rendered, err := render.MarkdownWithWidth(msg.content, bubbleWidth-4)
			if err != nil {
				rendered = msg.content
			}
			// Trim trailing newlines from glamour
			rendered = strings.TrimRight(rendered, "\n")

			bubble := assistantBubbleStyle.Width(bubbleWidth).Render(rendered)
			content.WriteString(bubble)
		}
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
}


// formatError formats an error with structured error details for display
func (m Model) formatError(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	// Main error message
	sb.WriteString(errorStyle.Render(fmt.Sprintf("⚠ Error: %v", err)))

	// Add structured error details
	detailStyle := lipgloss.NewStyle().Foreground(colorTextDim).PaddingLeft(2)

	if status := apierrors.GetHTTPStatus(err); status > 0 {
		sb.WriteString("\n")
		sb.WriteString(detailStyle.Render(fmt.Sprintf("HTTP Status: %d", status)))
	}

	if code := apierrors.GetErrorCode(err); code != apierrors.ErrCodeUnknown {
		sb.WriteString("\n")
		sb.WriteString(detailStyle.Render(fmt.Sprintf("Error Code: %d (%s)", code, code.String())))
	}

	// Add helpful hints
	hintStyle := lipgloss.NewStyle().Foreground(colorPrimary).PaddingLeft(2)
	switch {
	case apierrors.IsAuthError(err):
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("💡 Try 'geminiweb auto-login' to refresh your session"))
	case apierrors.IsRateLimitError(err):
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("💡 Usage limit reached. Try again later or use a different model"))
	case apierrors.IsNetworkError(err):
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("💡 Check your internet connection"))
	case apierrors.IsTimeoutError(err):
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("💡 Request timed out. Try again"))
	}

	return sb.String()
}

// RunChat starts the chat TUI
func RunChat(client api.GeminiClientInterface, modelName string) error {
	m := NewChatModel(client, modelName)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
</file>
<file path="internal/tui/styles.go">
// Package tui provides the terminal user interface for geminiweb.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Modern color palette
var (
	// Base colors
	colorBackground = lipgloss.Color("#1a1b26") // Dark background
	colorSurface    = lipgloss.Color("#24283b") // Slightly lighter surface
	colorBorder     = lipgloss.Color("#414868") // Subtle border

	// Accent colors
	colorPrimary   = lipgloss.Color("#7aa2f7") // Soft blue
	colorSecondary = lipgloss.Color("#9ece6a") // Soft green
	colorAccent    = lipgloss.Color("#bb9af7") // Purple accent
	colorWarning   = lipgloss.Color("#e0af68") // Warm yellow
	colorError     = lipgloss.Color("#f7768e") // Soft red

	// Text colors
	colorText     = lipgloss.Color("#c0caf5") // Main text
	colorTextDim  = lipgloss.Color("#565f89") // Dimmed text
	colorTextMute = lipgloss.Color("#3b4261") // Very dim text
)

// Header panel style
var headerStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(0, 2).
	MarginBottom(1)

// Title style for header
var titleStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Subtitle/model name style
var subtitleStyle = lipgloss.NewStyle().
	Foreground(colorAccent)

// Hint text style
var hintStyle = lipgloss.NewStyle().
	Foreground(colorTextDim).
	Italic(true)

// Messages area panel
var messagesAreaStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(1, 1)

// User message bubble
var userBubbleStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorSecondary).
	Foreground(colorText).
	Padding(0, 1).
	MarginTop(1).
	MarginBottom(1)

// User label style
var userLabelStyle = lipgloss.NewStyle().
	Foreground(colorSecondary).
	Bold(true).
	MarginBottom(0)

// Assistant message bubble
var assistantBubbleStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorPrimary).
	Foreground(colorText).
	Padding(0, 1).
	MarginTop(1).
	MarginBottom(1)

// Assistant label style
var assistantLabelStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true).
	MarginBottom(0)

// Thoughts panel style
var thoughtsStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(colorTextDim).
	BorderLeft(true).
	Foreground(colorTextDim).
	PaddingLeft(1).
	MarginLeft(1).
	Italic(true)

// Input area panel
var inputPanelStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorSecondary).
	Padding(0, 1).
	MarginTop(1)

// Input label style
var inputLabelStyle = lipgloss.NewStyle().
	Foreground(colorSecondary).
	Bold(true)

// Loading/spinner style
var loadingStyle = lipgloss.NewStyle().
	Foreground(colorAccent).
	Bold(true)

// Gradient colors for animated spinner
var gradientColors = []lipgloss.Color{
	lipgloss.Color("#ff6b6b"), // Red
	lipgloss.Color("#feca57"), // Yellow
	lipgloss.Color("#48dbfb"), // Cyan
	lipgloss.Color("#ff9ff3"), // Pink
	lipgloss.Color("#54a0ff"), // Blue
	lipgloss.Color("#5f27cd"), // Purple
	lipgloss.Color("#00d2d3"), // Teal
	lipgloss.Color("#1dd1a1"), // Green
}

// Status bar style
var statusBarStyle = lipgloss.NewStyle().
	Foreground(colorTextDim).
	MarginTop(1)

// Status bar key style
var statusKeyStyle = lipgloss.NewStyle().
	Foreground(colorText).
	Background(colorSurface).
	Padding(0, 1)

// Status bar description style
var statusDescStyle = lipgloss.NewStyle().
	Foreground(colorTextDim)

// Error style
var errorStyle = lipgloss.NewStyle().
	Foreground(colorError).
	Bold(true)

// Welcome message style
var welcomeStyle = lipgloss.NewStyle().
	Foreground(colorTextDim).
	Align(lipgloss.Center)

// Welcome title style
var welcomeTitleStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true).
	Align(lipgloss.Center)

// Welcome icon style
var welcomeIconStyle = lipgloss.NewStyle().
	Foreground(colorAccent).
	Align(lipgloss.Center)

// ═══════════════════════════════════════════════════════════════════════════════
// CONFIG MENU STYLES
// ═══════════════════════════════════════════════════════════════════════════════

// Config header style
var configHeaderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(0, 2).
	MarginBottom(1)

// Config title style
var configTitleStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Config panel style (for paths and settings)
var configPanelStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(1, 2).
	MarginBottom(1)

// Config section title style
var configSectionTitleStyle = lipgloss.NewStyle().
	Foreground(colorAccent).
	Bold(true).
	MarginBottom(1)

// Config menu item style (not selected)
var configMenuItemStyle = lipgloss.NewStyle().
	Foreground(colorText).
	PaddingLeft(2)

// Config menu item style (selected/highlighted)
var configMenuSelectedStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Config cursor style
var configCursorStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Config value style (for settings values)
var configValueStyle = lipgloss.NewStyle().
	Foreground(colorAccent)

// Config enabled value style
var configEnabledStyle = lipgloss.NewStyle().
	Foreground(colorSecondary)

// Config disabled value style
var configDisabledStyle = lipgloss.NewStyle().
	Foreground(colorTextDim)

// Config path style
var configPathStyle = lipgloss.NewStyle().
	Foreground(colorTextDim)

// Config status ok style
var configStatusOkStyle = lipgloss.NewStyle().
	Foreground(colorSecondary)

// Config status error style
var configStatusErrorStyle = lipgloss.NewStyle().
	Foreground(colorError)

// Config feedback message style
var configFeedbackStyle = lipgloss.NewStyle().
	Foreground(colorSecondary).
	Bold(true).
	MarginTop(1)

// Config status bar style
var configStatusBarStyle = lipgloss.NewStyle().
	Foreground(colorTextDim).
	MarginTop(1)
</file>
<file path="AGENTS.md">
# AGENTS.md

## Build & Test Commands
```bash
make build          # Build binary to build/geminiweb (CGO_ENABLED=1 required)
make build-dev      # Fast dev build without optimizations
make test           # Run all tests: go test -v ./...
go test -v ./internal/api -run TestClientInit   # Run single test
make lint           # golangci-lint run ./...
make fmt            # go fmt + gofumpt
make check          # Verify build compiles
```

## Architecture
- `cmd/geminiweb/` - CLI entrypoint using Cobra
- `internal/api/` - GeminiClient: TLS client, token fetch, cookie rotation, content generation
- `internal/commands/` - Cobra commands: chat, query, config, import-cookies
- `internal/config/` - Cookie storage and config management (~/.geminiweb/)
- `internal/models/` - Types: Model, Response, Message, API constants/endpoints
- `internal/tui/` - Bubbletea interactive UI with Glamour markdown rendering
- `internal/errors/` - Custom error types

## Code Style
- Go 1.23+, use functional options pattern (see `ClientOption` in api/client.go)
- Errors: wrap with context using `fmt.Errorf("...: %w", err)`
- Imports: stdlib first, blank line, external deps, blank line, internal packages
- Use `tidwall/gjson` for JSON parsing, `bogdanfinn/tls-client` for HTTP
</file>
<file path="CLAUDE.md">
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**geminiweb-go** is a CLI for interacting with Google Gemini via the web API. It uses cookie-based authentication (not API keys) and browser-like TLS fingerprinting (Chrome 120 profile) to communicate with Gemini's web interface.

## Build & Development Commands

```bash
# Build (requires CGO_ENABLED=1 for TLS client)
make build                # Production build with version info
make build-dev            # Fast development build
make install              # Install to GOPATH/bin

# Run
make run ARGS="chat"      # Build and run with arguments
./build/geminiweb         # Direct execution

# Testing
make test                 # Run all tests: go test -v ./...
go test -v ./internal/api -run TestClientInit   # Single test
make test-coverage        # Tests with HTML coverage report

# Quality
make lint                 # golangci-lint
make fmt                  # go fmt + gofumpt
```

## Architecture

### Package Structure

- **`cmd/geminiweb/`** - CLI entrypoint
- **`internal/api/`** - GeminiClient: TLS client, token extraction (SNlM0e), cookie rotation, content generation, browser refresh
- **`internal/browser/`** - Browser cookie extraction using `browserutils/kooky` (Chrome, Firefox, Edge, Chromium, Opera)
- **`internal/commands/`** - Cobra commands: root (query), chat, config, import-cookies, auto-login, history, persona
- **`internal/config/`** - Settings and cookie storage in `~/.geminiweb/`
- **`internal/models/`** - Types (ModelOutput, Candidate, WebImage), model definitions, API constants/endpoints
- **`internal/tui/`** - Bubble Tea TUI with Glamour markdown rendering and Lipgloss styling
- **`internal/render/`** - Markdown rendering with Glamour, pooled renderers, configurable themes and caching
- **`internal/history/`** - JSON-based conversation history persistence
- **`internal/errors/`** - Custom error types

### Key Dependencies

- **TLS/HTTP**: `bogdanfinn/tls-client` (Chrome fingerprinting), `bogdanfinn/fhttp`
- **CLI**: `spf13/cobra`
- **TUI**: `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `charmbracelet/glamour`
- **JSON**: `tidwall/gjson`
- **Browser Cookies**: `browserutils/kooky` (cross-platform cookie extraction with decryption)

### Client Lifecycle

```go
client, err := api.NewClient(cookies,
    api.WithModel(models.Model25Flash),
    api.WithBrowserRefresh(browser.BrowserAuto), // Optional: auto-refresh from browser on auth failure
)
err := client.Init()              // Fetches access token (SNlM0e)
response, err := client.GenerateContent("prompt", opts)  // Auto-retries with fresh cookies on 401
chat := client.StartChat()
response, err := chat.SendMessage("hello")
client.Close()
```

### Key Patterns

1. **Functional Options** - `ClientOption` functions configure GeminiClient (WithModel, WithAutoRefresh, WithRefreshInterval, WithBrowserRefresh)
2. **TLS Fingerprinting** - Chrome 120 profile via `bogdanfinn/tls-client` to appear as real browser
3. **Auto Cookie Rotation** - Background goroutine refreshes tokens at `/accounts.google.com/RotateCookies` (default 9 min interval)
4. **Browser Cookie Refresh** - On auth failure (401), automatically extracts fresh cookies from browser and retries (rate-limited to 30s)
5. **Bubble Tea Architecture** - TUI uses Model/Update/View pattern; messages flow through Update, never mutate state directly
6. **Dependency Injection** - Key components use interfaces (`GeminiClientInterface`, `ChatSessionInterface`, `BrowserCookieExtractor`) and option functions (`WithRefreshFunc`, `WithCookieLoader`) for testability

### TUI Notes

- **Glamour markdown**: Use `glamour.WithStylePath("dark")` instead of `glamour.WithAutoStyle()` to avoid OSC 11 terminal query escape sequence leaks into stdin
- **Textarea input filtering**: Only pass `tea.KeyMsg` to textarea.Update() to prevent escape sequences from appearing as garbage characters
- **Viewport**: Always updated with all messages for scrolling support

## Code Style

- Go 1.23+, functional options pattern
- Errors: wrap with context using `fmt.Errorf("...: %w", err)`
- Imports: stdlib → blank line → external deps → blank line → internal packages
- Use `tidwall/gjson` for JSON parsing (not encoding/json for reads)
- Use `bogdanfinn/fhttp` for HTTP requests (not net/http)

## Models

Default model is `models.Model30Pro` (gemini-3.0-pro).

- `models.Model25Flash` - Fast model (gemini-2.5-flash)
- `models.Model25Pro` - Balanced model (gemini-2.5-pro)
- `models.Model30Pro` - Advanced model (gemini-3.0-pro)
</file>
<file path="go.mod">
module github.com/diogo/geminiweb

go 1.23.10

require (
	github.com/atotto/clipboard v0.1.4
	github.com/bogdanfinn/fhttp v0.5.34
	github.com/bogdanfinn/tls-client v1.9.1
	github.com/browserutils/kooky v0.2.4
	github.com/charmbracelet/bubbles v0.21.0
	github.com/charmbracelet/bubbletea v1.3.4
	github.com/charmbracelet/glamour v0.10.0
	github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834
	github.com/spf13/cobra v1.8.1
	github.com/tidwall/gjson v1.18.0
	golang.org/x/term v0.33.0
)

require (
	github.com/Velocidex/json v0.0.0-20220224052537-92f3c0326e5a // indirect
	github.com/Velocidex/ordereddict v0.0.0-20250626035939-2f7f022fc719 // indirect
	github.com/Velocidex/yaml/v2 v2.2.8 // indirect
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/bogdanfinn/utls v1.6.5 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/exp/slice v0.0.0-20250327172914-2fdc97757edf // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/cloudflare/circl v1.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-sqlite/sqlite3 v0.0.0-20180313105335-53dd8e640ee7 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gonuts/binary v0.2.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/keybase/go-keychain v0.0.1 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/quic-go v0.48.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/tam7t/hpkp v0.0.0-20160821193359-2b70b4024ed5 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yuin/goldmark v1.7.8 // indirect
	github.com/yuin/goldmark-emoji v1.0.5 // indirect
	github.com/zalando/go-keyring v0.2.6 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	www.velocidex.com/golang/go-ese v0.2.0 // indirect
)
</file>
<file path="go.sum">
github.com/MakeNowJust/heredoc v1.0.0 h1:cXCdzVdstXyiTqTvfqk9SDHpKNjxuom+DOlyEeQ4pzQ=
github.com/MakeNowJust/heredoc v1.0.0/go.mod h1:mG5amYoWBHf8vpLOuehzbGGw0EHxpZZ6lCpQ4fNJ8LE=
github.com/Velocidex/json v0.0.0-20220224052537-92f3c0326e5a h1:AeXPUzhU0yhID/v5JJEIkjaE85ASe+Vh4Kuv1RSLL+4=
github.com/Velocidex/json v0.0.0-20220224052537-92f3c0326e5a/go.mod h1:ukJBuruT9b24pdgZwWDvOaCYHeS03B7oQPCUWh25bwM=
github.com/Velocidex/ordereddict v0.0.0-20220107075049-3dbe58412844/go.mod h1:Y5Tfx5SKGOzkulpqfonrdILSPIuNg+GqKE/DhVJgnpg=
github.com/Velocidex/ordereddict v0.0.0-20250626035939-2f7f022fc719 h1:7wx3n0HY8WkEQYehirMb2bhf1zTsw4Di4mjpVysl2Sc=
github.com/Velocidex/ordereddict v0.0.0-20250626035939-2f7f022fc719/go.mod h1:+MqO5UMBemyFSm+yRXslbpFTwPUDhFHUf7HPV92twg4=
github.com/Velocidex/yaml/v2 v2.2.8 h1:GUrSy4SBJ6RjGt43k6MeBKtw2z/27gh4A3hfFmFY3No=
github.com/Velocidex/yaml/v2 v2.2.8/go.mod h1:PlXIg/Pxmoja48C1vMHo7C5pauAZvLq/UEPOQ3DsjS4=
github.com/alecthomas/assert v1.0.0 h1:3XmGh/PSuLzDbK3W2gUbRXwgW5lqPkuqvRgeQ30FI5o=
github.com/alecthomas/assert v1.0.0/go.mod h1:va/d2JC+M7F6s+80kl/R3G7FUiW6JzUO+hPhLyJ36ZY=
github.com/alecthomas/assert/v2 v2.7.0 h1:QtqSACNS3tF7oasA8CU6A6sXZSBDqnm7RfpLl9bZqbE=
github.com/alecthomas/assert/v2 v2.7.0/go.mod h1:Bze95FyfUr7x34QZrjL+XP+0qgp/zg8yS+TtBj1WA3k=
github.com/alecthomas/chroma/v2 v2.14.0 h1:R3+wzpnUArGcQz7fCETQBzO5n9IMNi13iIs46aU4V9E=
github.com/alecthomas/chroma/v2 v2.14.0/go.mod h1:QolEbTfmUHIMVpBqxeDnNBj2uoeI4EbYP4i6n68SG4I=
github.com/alecthomas/colour v0.1.0/go.mod h1:QO9JBoKquHd+jz9nshCh40fOfO+JzsoXy8qTHF68zU0=
github.com/alecthomas/repr v0.0.0-20210801044451-80ca428c5142/go.mod h1:2kn6fqh/zIyPLmm3ugklbEi5hg5wS435eygvNfaDQL8=
github.com/alecthomas/repr v0.1.1/go.mod h1:Fr0507jx4eOXV7AlPV6AVZLYrLIuIeSOWtW57eE/O/4=
github.com/alecthomas/repr v0.4.0 h1:GhI2A8MACjfegCPVq9f1FLvIBS+DrQ2KQBFZP1iFzXc=
github.com/alecthomas/repr v0.4.0/go.mod h1:Fr0507jx4eOXV7AlPV6AVZLYrLIuIeSOWtW57eE/O/4=
github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751/go.mod h1:LOuyumcjzFXgccqObfd/Ljyb9UuFJ6TxHnclSeseNhc=
github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137/go.mod h1:OMCwj8VM1Kc9e19TLln2VL61YJF0x1XFtfdL4JdbSyE=
github.com/andybalholm/brotli v1.1.1 h1:PR2pgnyFznKEugtsUo0xLdDop5SKXd5Qf5ysW+7XdTA=
github.com/andybalholm/brotli v1.1.1/go.mod h1:05ib4cKhjx3OQYUY22hTVd34Bc8upXjOLL2rKwwZBoA=
github.com/atotto/clipboard v0.1.4 h1:EH0zSVneZPSuFR11BlR9YppQTVDbh5+16AmcJi4g1z4=
github.com/atotto/clipboard v0.1.4/go.mod h1:ZY9tmq7sm5xIbd9bOK4onWV4S6X0u6GY7Vn0Yu86PYI=
github.com/aymanbagabas/go-osc52/v2 v2.0.1 h1:HwpRHbFMcZLEVr42D4p7XBqjyuxQH5SMiErDT4WkJ2k=
github.com/aymanbagabas/go-osc52/v2 v2.0.1/go.mod h1:uYgXzlJ7ZpABp8OJ+exZzJJhRNQ2ASbcXHWsFqH8hp8=
github.com/aymanbagabas/go-udiff v0.2.0 h1:TK0fH4MteXUDspT88n8CKzvK0X9O2xu9yQjWpi6yML8=
github.com/aymanbagabas/go-udiff v0.2.0/go.mod h1:RE4Ex0qsGkTAJoQdQQCA0uG+nAzJO/pI/QwceO5fgrA=
github.com/aymerick/douceur v0.2.0 h1:Mv+mAeH1Q+n9Fr+oyamOlAkUNPWPlA8PPGR0QAaYuPk=
github.com/aymerick/douceur v0.2.0/go.mod h1:wlT5vV2O3h55X9m7iVYN0TBM0NH/MmbLnd30/FjWUq4=
github.com/bogdanfinn/fhttp v0.5.34 h1:avRD2JNYqj6I6DqjSrI9tl8mP8Nk7T4CCmUsPz7afhg=
github.com/bogdanfinn/fhttp v0.5.34/go.mod h1:BlcawVfXJ4uhk5yyNGOOY2bwo8UmMi6ccMszP1KGLkU=
github.com/bogdanfinn/tls-client v1.9.1 h1:Br0WkKL+/7Q9FSNM1zBMdlYXW8bm+XXGMn9iyb9a/7Y=
github.com/bogdanfinn/tls-client v1.9.1/go.mod h1:ehNITC7JBFeh6S7QNWtfD+PBKm0RsqvizAyyij2d/6g=
github.com/bogdanfinn/utls v1.6.5 h1:rVMQvhyN3zodLxKFWMRLt19INGBCZ/OM2/vBWPNIt1w=
github.com/bogdanfinn/utls v1.6.5/go.mod h1:czcHxHGsc1q9NjgWSeSinQZzn6MR76zUmGVIGanSXO0=
github.com/browserutils/kooky v0.2.4 h1:szrKufBIaZRc6AXs8MF7+4rgcoSZNckQE2q0sJw49kw=
github.com/browserutils/kooky v0.2.4/go.mod h1:Ez5Gw643UabvRkvEnWIgb8Q6qPzxanMuHCTTqlwBHuw=
github.com/charmbracelet/bubbles v0.21.0 h1:9TdC97SdRVg/1aaXNVWfFH3nnLAwOXr8Fn6u6mfQdFs=
github.com/charmbracelet/bubbles v0.21.0/go.mod h1:HF+v6QUR4HkEpz62dx7ym2xc71/KBHg+zKwJtMw+qtg=
github.com/charmbracelet/bubbletea v1.3.4 h1:kCg7B+jSCFPLYRA52SDZjr51kG/fMUEoPoZrkaDHyoI=
github.com/charmbracelet/bubbletea v1.3.4/go.mod h1:dtcUCyCGEX3g9tosuYiut3MXgY/Jsv9nKVdibKKRRXo=
github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc h1:4pZI35227imm7yK2bGPcfpFEmuY1gc2YSTShr4iJBfs=
github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc/go.mod h1:X4/0JoqgTIPSFcRA/P6INZzIuyqdFY5rm8tb41s9okk=
github.com/charmbracelet/glamour v0.10.0 h1:MtZvfwsYCx8jEPFJm3rIBFIMZUfUJ765oX8V6kXldcY=
github.com/charmbracelet/glamour v0.10.0/go.mod h1:f+uf+I/ChNmqo087elLnVdCiVgjSKWuXa/l6NU2ndYk=
github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834 h1:ZR7e0ro+SZZiIZD7msJyA+NjkCNNavuiPBLgerbOziE=
github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834/go.mod h1:aKC/t2arECF6rNOnaKaVU6y4t4ZeHQzqfxedE/VkVhA=
github.com/charmbracelet/x/ansi v0.8.0 h1:9GTq3xq9caJW8ZrBTe0LIe2fvfLR/bYXKTx2llXn7xE=
github.com/charmbracelet/x/ansi v0.8.0/go.mod h1:wdYl/ONOLHLIVmQaxbIYEC/cRKOQyjTkowiI4blgS9Q=
github.com/charmbracelet/x/cellbuf v0.0.13 h1:/KBBKHuVRbq1lYx5BzEHBAFBP8VcQzJejZ/IA3iR28k=
github.com/charmbracelet/x/cellbuf v0.0.13/go.mod h1:xe0nKWGd3eJgtqZRaN9RjMtK7xUYchjzPr7q6kcvCCs=
github.com/charmbracelet/x/exp/golden v0.0.0-20241011142426-46044092ad91 h1:payRxjMjKgx2PaCWLZ4p3ro9y97+TVLZNaRZgJwSVDQ=
github.com/charmbracelet/x/exp/golden v0.0.0-20241011142426-46044092ad91/go.mod h1:wDlXFlCrmJ8J+swcL/MnGUuYnqgQdW9rhSD61oNMb6U=
github.com/charmbracelet/x/exp/slice v0.0.0-20250327172914-2fdc97757edf h1:rLG0Yb6MQSDKdB52aGX55JT1oi0P0Kuaj7wi1bLUpnI=
github.com/charmbracelet/x/exp/slice v0.0.0-20250327172914-2fdc97757edf/go.mod h1:B3UgsnsBZS/eX42BlaNiJkD1pPOUa+oF1IYC6Yd2CEU=
github.com/charmbracelet/x/term v0.2.1 h1:AQeHeLZ1OqSXhrAWpYUtZyX1T3zVxfpZuEQMIQaGIAQ=
github.com/charmbracelet/x/term v0.2.1/go.mod h1:oQ4enTYFV7QN4m0i9mzHrViD7TQKvNEEkHUMCmsxdUg=
github.com/cloudflare/circl v1.5.0 h1:hxIWksrX6XN5a1L2TI/h53AGPhNHoUBo+TD1ms9+pys=
github.com/cloudflare/circl v1.5.0/go.mod h1:uddAzsPgqdMAYatqJ0lsjX1oECcQLIlRpzZh3pJrofs=
github.com/cpuguy83/go-md2man/v2 v2.0.4/go.mod h1:tgQtvFlXSQOSOSIRvRPT7W67SCa46tRHOmNcaadrF8o=
github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc h1:U9qPSI2PIWSS1VwoXQT9A3Wy9MM3WgvqSxFWenqJduM=
github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/dlclark/regexp2 v1.11.0 h1:G/nrcoOa7ZXlpoa/91N3X7mM3r8eIlMBBJZvsz/mxKI=
github.com/dlclark/regexp2 v1.11.0/go.mod h1:DHkYz0B9wPfa6wondMfaivmHpzrQ3v9q8cnmRbL6yW8=
github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f h1:Y/CXytFA4m6baUTXGLOoWe4PQhGxaX0KpnayAqC48p4=
github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f/go.mod h1:vw97MGsxSvLiUE2X8qFplwetxpGLQrlU1Q9AUEIzCaM=
github.com/go-ini/ini v1.67.0 h1:z6ZrTEZqSWOTyH2FlglNbNgARyHG8oLW9gMELqKr06A=
github.com/go-ini/ini v1.67.0/go.mod h1:ByCAeIL28uOIIG0E3PJtZPDL8WnHpFKFOtgjp+3Ies8=
github.com/go-sqlite/sqlite3 v0.0.0-20180313105335-53dd8e640ee7 h1:ow5vK9Q/DSKkxbEIJHBST6g+buBDwdaDIyk1dGGwpQo=
github.com/go-sqlite/sqlite3 v0.0.0-20180313105335-53dd8e640ee7/go.mod h1:JxSQ+SvsjFb+p8Y+bn+GhTkiMfKVGBD0fq43ms2xw04=
github.com/godbus/dbus/v5 v5.1.0 h1:4KLkAxT3aOY8Li4FRJe/KvhoNFFxo0m6fNuFUO8QJUk=
github.com/godbus/dbus/v5 v5.1.0/go.mod h1:xhWf0FNVPg57R7Z0UbKHbJfkEywrmjJnf7w5xrFpKfA=
github.com/gonuts/binary v0.2.0 h1:caITwMWAoQWlL0RNvv2lTU/AHqAJlVuu6nZmNgfbKW4=
github.com/gonuts/binary v0.2.0/go.mod h1:kM+CtBrCGDSKdv8WXTuCUsw+loiy8f/QEI8YCCC0M/E=
github.com/gorilla/css v1.0.1 h1:ntNaBIghp6JmvWnxbZKANoLyuXTPZ4cAMlo6RyhlbO8=
github.com/gorilla/css v1.0.1/go.mod h1:BvnYkspnSzMmwRK+b8/xgNPLiIuNZr6vbZBTPQ2A3b0=
github.com/hexops/gotextdiff v1.0.3 h1:gitA9+qJrrTCsiCl7+kh75nPqQt1cx4ZkudSTLoUqJM=
github.com/hexops/gotextdiff v1.0.3/go.mod h1:pSWU5MAI3yDq+fZBTazCSJysOMbxWL1BSow5/V2vxeg=
github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
github.com/keybase/go-keychain v0.0.1 h1:way+bWYa6lDppZoZcgMbYsvC7GxljxrskdNInRtuthU=
github.com/keybase/go-keychain v0.0.1/go.mod h1:PdEILRW3i9D8JcdM+FmY6RwkHGnhHxXwkPPMeUgOK1k=
github.com/klauspost/compress v1.17.11 h1:In6xLpyWOi1+C7tXUUWv2ot1QvBjxevKAaI6IXrJmUc=
github.com/klauspost/compress v1.17.11/go.mod h1:pMDklpSncoRMuLFrf1W9Ss9KT+0rH90U12bZKk7uwG0=
github.com/kr/pretty v0.1.0 h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0 h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/lucasb-eyer/go-colorful v1.2.0 h1:1nnpGOrhyZZuNyfu1QjKiUICQ74+3FNCN69Aj6K7nkY=
github.com/lucasb-eyer/go-colorful v1.2.0/go.mod h1:R4dSotOR9KMtayYi1e77YzuveK+i7ruzyGqttikkLy0=
github.com/mattn/go-isatty v0.0.14/go.mod h1:7GGIvUiUoEMVVmxf/4nioHXj79iQHKdU27kJ6hsGG94=
github.com/mattn/go-isatty v0.0.20 h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY=
github.com/mattn/go-isatty v0.0.20/go.mod h1:W+V8PltTTMOvKvAeJH7IuucS94S2C6jfK/D7dTCTo3Y=
github.com/mattn/go-localereader v0.0.1 h1:ygSAOl7ZXTx4RdPYinUpg6W99U8jWvWi9Ye2JC/oIi4=
github.com/mattn/go-localereader v0.0.1/go.mod h1:8fBrzywKY7BI3czFoHkuzRoWE9C+EiG4R1k4Cjx5p88=
github.com/mattn/go-runewidth v0.0.12/go.mod h1:RAqKPSqVFrSLVXbA8x7dzmKdmGzieGRCM46jaSJTDAk=
github.com/mattn/go-runewidth v0.0.16 h1:E5ScNMtiwvlvB5paMFdw9p4kSQzbXFikJ5SQO6TULQc=
github.com/mattn/go-runewidth v0.0.16/go.mod h1:Jdepj2loyihRzMpdS35Xk/zdY8IAYHsh153qUoGf23w=
github.com/microcosm-cc/bluemonday v1.0.27 h1:MpEUotklkwCSLeH+Qdx1VJgNqLlpY2KXwXFM08ygZfk=
github.com/microcosm-cc/bluemonday v1.0.27/go.mod h1:jFi9vgW+H7c3V0lb6nR74Ib/DIB5OBs92Dimizgw2cA=
github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 h1:ZK8zHtRHOkbHy6Mmr5D264iyp3TiX5OmNcI5cIARiQI=
github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6/go.mod h1:CJlz5H+gyd6CUWT45Oy4q24RdLyn7Md9Vj2/ldJBSIo=
github.com/muesli/cancelreader v0.2.2 h1:3I4Kt4BQjOR54NavqnDogx/MIoWBFa0StPA8ELUXHmA=
github.com/muesli/cancelreader v0.2.2/go.mod h1:3XuTXfFS2VjM+HTLZY9Ak0l6eUKfijIfMUZ4EgX0QYo=
github.com/muesli/reflow v0.3.0 h1:IFsN6K9NfGtjeggFP+68I4chLZV2yIKsXJFNZ+eWh6s=
github.com/muesli/reflow v0.3.0/go.mod h1:pbwTDkVPibjO2kyvBQRBxTWEEGDGq0FlB1BIKtnHY/8=
github.com/muesli/termenv v0.16.0 h1:S5AlUN9dENB57rsbnkPyfdGuWIlkmzJjbFf0Tf5FWUc=
github.com/muesli/termenv v0.16.0/go.mod h1:ZRfOIKPFDYQoDFF4Olj7/QJbW60Ol/kL1pU3VfY/Cnk=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 h1:Jamvg5psRIccs7FGNTlIRMkT8wgtp5eCXdBlqhYGL6U=
github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/quic-go/quic-go v0.48.1 h1:y/8xmfWI9qmGTc+lBr4jKRUWLGSlSigv847ULJ4hYXA=
github.com/quic-go/quic-go v0.48.1/go.mod h1:yBgs3rWBOADpga7F+jJsb6Ybg1LSYiQvwWlLX+/6HMs=
github.com/rivo/uniseg v0.1.0/go.mod h1:J6wj4VEh+S6ZtnVlnTBMWIodfgj8LQOQFoIToxlJtxc=
github.com/rivo/uniseg v0.2.0/go.mod h1:J6wj4VEh+S6ZtnVlnTBMWIodfgj8LQOQFoIToxlJtxc=
github.com/rivo/uniseg v0.4.7 h1:WUdvkW8uEhrYfLC4ZzdpI2ztxP1I582+49Oc5Mq64VQ=
github.com/rivo/uniseg v0.4.7/go.mod h1:FN3SvrM+Zdj16jyLfmOkMNblXMcoc8DfTHruCPUcx88=
github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
github.com/sebdah/goldie v1.0.0/go.mod h1:jXP4hmWywNEwZzhMuv2ccnqTSFpuq8iyQhtQdkkZBH4=
github.com/sergi/go-diff v1.2.0/go.mod h1:STckp+ISIX8hZLjrqAeVduY0gWCT9IjLuqbuNXdaHfM=
github.com/spf13/cobra v1.8.1 h1:e5/vxKd/rZsfSJMUX1agtjeTDf+qv1/JdBF8gg5k9ZM=
github.com/spf13/cobra v1.8.1/go.mod h1:wHxEcudfqmLYa8iTfL+OuZPbBZkmvliBWKIezN3kD9Y=
github.com/spf13/pflag v1.0.5/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
github.com/spf13/pflag v1.0.6 h1:jFzHGLGAlb3ruxLB8MhbI6A8+AQX/2eW4qeyNZXNp2o=
github.com/spf13/pflag v1.0.6/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/objx v0.4.0/go.mod h1:YvHI0jy2hoMjB+UWwv71VJQ9isScKT/TqJzVSSt89Yw=
github.com/stretchr/objx v0.5.0/go.mod h1:Yh+to48EsGEfYuaHDzXPcE3xhTkx73EhmCGUpEOglKo=
github.com/stretchr/testify v1.2.2/go.mod h1:a8OnRcib4nhh0OaRAV+Yts87kKdq0PP7pXfy6kDkUVs=
github.com/stretchr/testify v1.3.0/go.mod h1:M5WIy9Dh21IEIfnGCwXGc5bZfKNJtfHm1UVUgZn+9EI=
github.com/stretchr/testify v1.4.0/go.mod h1:j7eGeouHqKxXV5pUuKE4zz7dFj8WfuZ+81PSLYec5m4=
github.com/stretchr/testify v1.7.0/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.7.1/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.8.0/go.mod h1:yNjHg4UonilssWZ8iaSj1OCr/vHnekPRkoO+kdMU+MU=
github.com/stretchr/testify v1.8.1/go.mod h1:w2LPCIKwWwSfY2zedu0+kehJoqGctiVI29o6fzry7u4=
github.com/stretchr/testify v1.10.0 h1:Xv5erBjTwe/5IxqUQTdXv5kgmIvbHo3QQyRwhJsOfJA=
github.com/stretchr/testify v1.10.0/go.mod h1:r2ic/lqez/lEtzL7wO/rwa5dbSLXVDPFyf8C91i36aY=
github.com/tam7t/hpkp v0.0.0-20160821193359-2b70b4024ed5 h1:YqAladjX7xpA6BM04leXMWAEjS0mTZ5kUU9KRBriQJc=
github.com/tam7t/hpkp v0.0.0-20160821193359-2b70b4024ed5/go.mod h1:2JjD2zLQYH5HO74y5+aE3remJQvl6q4Sn6aWA2wD1Ng=
github.com/tidwall/gjson v1.18.0 h1:FIDeeyB800efLX89e5a8Y0BNH+LOngJyGrIWxG2FKQY=
github.com/tidwall/gjson v1.18.0/go.mod h1:/wbyibRr2FHMks5tjHJ5F8dMZh3AcwJEMf5vlfC0lxk=
github.com/tidwall/match v1.1.1 h1:+Ho715JplO36QYgwN9PGYNhgZvoUSc9X2c80KVTi+GA=
github.com/tidwall/match v1.1.1/go.mod h1:eRSPERbgtNPcGhD8UCthc6PmLEQXEWd3PRB5JTxsfmM=
github.com/tidwall/pretty v1.2.0 h1:RWIZEg2iJ8/g6fDDYzMpobmaoGh5OLl4AXtGUGPcqCs=
github.com/tidwall/pretty v1.2.0/go.mod h1:ITEVvHYasfjBbM0u2Pg8T2nJnzm8xPwvNhhsoaGGjNU=
github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e h1:JVG44RsyaB9T2KIHavMF/ppJZNG9ZpyihvCd0w101no=
github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e/go.mod h1:RbqR21r5mrJuqunuUZ/Dhy/avygyECGrLceyNeo4LiM=
github.com/xyproto/randomstring v1.0.5 h1:YtlWPoRdgMu3NZtP45drfy1GKoojuR7hmRcnhZqKjWU=
github.com/xyproto/randomstring v1.0.5/go.mod h1:rgmS5DeNXLivK7YprL0pY+lTuhNQW3iGxZ18UQApw/E=
github.com/yuin/goldmark v1.7.1/go.mod h1:uzxRWxtg69N339t3louHJ7+O03ezfj6PlliRlaOzY1E=
github.com/yuin/goldmark v1.7.8 h1:iERMLn0/QJeHFhxSt3p6PeN9mGnvIKSpG9YYorDMnic=
github.com/yuin/goldmark v1.7.8/go.mod h1:uzxRWxtg69N339t3louHJ7+O03ezfj6PlliRlaOzY1E=
github.com/yuin/goldmark-emoji v1.0.5 h1:EMVWyCGPlXJfUXBXpuMu+ii3TIaxbVBnEX9uaDC4cIk=
github.com/yuin/goldmark-emoji v1.0.5/go.mod h1:tTkZEbwu5wkPmgTcitqddVxY9osFZiavD+r4AzQrh1U=
github.com/zalando/go-keyring v0.2.6 h1:r7Yc3+H+Ux0+M72zacZoItR3UDxeWfKTcabvkI8ua9s=
github.com/zalando/go-keyring v0.2.6/go.mod h1:2TCrxYrbUNYfNS/Kgy/LSrkSQzZ5UPVH85RwfczwvcI=
golang.org/x/crypto v0.40.0 h1:r4x+VvoG5Fm+eJcxMaY8CQM7Lb0l1lsmjGBQ6s8BfKM=
golang.org/x/crypto v0.40.0/go.mod h1:Qr1vMER5WyS2dfPHAlsOj01wgLbsyWtFn/aY+5+ZdxY=
golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 h1:vr/HnozRka3pE4EsMEg1lgkXJkTFJCVUX+S/ZT6wYzM=
golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842/go.mod h1:XtvwrStGgqGPLc4cjQfWqZHG1YFdYs6swckp8vpsjnc=
golang.org/x/net v0.42.0 h1:jzkYrhi3YQWD6MLBJcsklgQsoAcw89EcZbJw8Z614hs=
golang.org/x/net v0.42.0/go.mod h1:FF1RA5d3u7nAYA4z2TkclSCKh68eSXtiFwcWQpPXdt8=
golang.org/x/sync v0.16.0 h1:ycBJEhp9p4vXvUZNszeOq0kGTPghopOL8q0fq3vstxw=
golang.org/x/sync v0.16.0/go.mod h1:1dzgHSNfp02xaA81J2MS99Qcpr2w7fw1gpm99rleRqA=
golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.1.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.6.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.34.0 h1:H5Y5sJ2L2JRdyv7ROF1he/lPdvFsd0mJHFw2ThKHxLA=
golang.org/x/sys v0.34.0/go.mod h1:BJP2sWEmIv4KK5OTEluFJCKSidICx8ciO85XgH3Ak8k=
golang.org/x/term v0.33.0 h1:NuFncQrRcaRvVmgRkvM3j/F00gWIAlcmlB8ACEKmGIg=
golang.org/x/term v0.33.0/go.mod h1:s18+ql9tYWp1IfpV9DmCtQDDSRBUjKaw9M1eAv5UeF0=
golang.org/x/text v0.27.0 h1:4fGWRpyh641NLlecmyl4LOe6yDdfaYNrGb2zdfo4JV4=
golang.org/x/text v0.27.0/go.mod h1:1D28KMCvyooCX9hBiosv5Tz/+YLxj0j7XhWjpSUF7CU=
gopkg.in/alecthomas/kingpin.v2 v2.2.6/go.mod h1:FMv+mEhP44yOT+4EoQTLFTRgOQ1FBLkstjWtayDeSgw=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 h1:YR8cESwS4TdDjEe65xsg0ogRM/Nc3DYOhEAlW+xobZo=
gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v2 v2.2.2/go.mod h1:hI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuI=
gopkg.in/yaml.v2 v2.2.4/go.mod h1:hI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuI=
gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
www.velocidex.com/golang/go-ese v0.2.0 h1:8/hzEMupfqEF0oMi1/EzsMN1xLN0GBFcB3GqxqRnb9s=
www.velocidex.com/golang/go-ese v0.2.0/go.mod h1:6fC9T6UGLbM7icuA0ugomU5HbFC5XA5I30zlWtZT8YE=
</file>
<file path="Makefile">
# Makefile for geminiweb Go CLI

BINARY_NAME=geminiweb
BUILD_DIR=build
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/diogo/geminiweb/internal/commands.Version=$(VERSION) -X github.com/diogo/geminiweb/internal/commands.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test lint fmt deps install run

all: build

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build the binary
build: deps
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/geminiweb

# Build for development (faster, no optimization)
build-dev: deps
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/geminiweb

# Install to GOPATH/bin
install: deps
	CGO_ENABLED=1 go install $(LDFLAGS) ./cmd/geminiweb

# Run the CLI
run: build-dev
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Remove temporary/useless files
clean-repo: clean
	rm -f plan-tests.md coverage-plan.md test-coverage-improvement-report.md

# Run linter
lint:
	golangci-lint run ./...

# Show coverage breakdown by function
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

# Format code
fmt:
	go fmt ./...
	gofumpt -w .

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Build for all platforms (requires goreleaser)
release-snapshot:
	goreleaser release --snapshot --clean

# Check if build would succeed
check:
	go build -o /dev/null ./cmd/geminiweb

# Verify go.mod is tidy
verify-mod:
	go mod tidy
	git diff --exit-code go.mod go.sum

# Help
help:
	@echo "Available targets:"
	@echo "  deps            Download dependencies"
	@echo "  build           Build the binary"
	@echo "  build-dev       Build for development (faster)"
	@echo "  install         Install to GOPATH/bin"
	@echo "  run ARGS=...    Build and run with arguments"
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  lint            Run linter"
	@echo "  clean-repo      Remove temporary/useless files           "
	@echo "(coverage reports, plans, build dir)"
	@echo "  fmt             Format code"
	@echo "  clean           Remove build artifacts"
	@echo "  release-snapshot Build for all platforms"
	@echo "  check           Verify build would succeed"
	@echo "  verify-mod      Verify go.mod is tidy"
</file>
<file path="README.md">
# geminiweb (Go)

A high-performance CLI for interacting with Google Gemini via the web API. Built in Go with browser-like TLS fingerprinting for reliable authentication.

## Features

- **Interactive Chat** - Full-featured TUI with markdown rendering
- **Single Queries** - Quick questions from command line, files, or stdin
- **Multiple Models** - Support for Gemini 2.5 Flash, 2.5 Pro, and 3.0 Pro
- **Cookie Authentication** - Uses browser cookies for authentication
- **Auto Cookie Extraction** - Extract cookies directly from browsers (Chrome, Firefox, Edge, etc.)
- **Auto Cookie Refresh** - Background token rotation and automatic browser refresh on auth failure
- **TLS Fingerprinting** - Chrome-like TLS profile to avoid detection

## Installation

### Build from source

```bash
# Clone the repository
cd geminiweb-go

# Build
make build

# Install to GOPATH/bin
make install
```

### Requirements

- Go 1.23+
- CGO enabled (for TLS client)

## Usage

### Setup

**Option 1: Auto-extract from browser (recommended)**

```bash
# Auto-detect browser and extract cookies
geminiweb auto-login

# Or specify browser
geminiweb auto-login -b firefox
geminiweb auto-login -b chrome
```

**Option 2: Manual import**

1. Export cookies from your browser after logging into [gemini.google.com](https://gemini.google.com)
2. Import cookies:

```bash
geminiweb import-cookies ~/cookies.json
```

> **Note:** For auto-login, close the browser first to avoid database lock errors.

### Interactive Chat

```bash
geminiweb chat
```

### Single Query

```bash
# Direct prompt
geminiweb "What is Go?"

# From file
geminiweb -f prompt.md

# From stdin
cat prompt.md | geminiweb

# Save to file
geminiweb "Hello" -o response.md
```

### Configuration

```bash
geminiweb config
```

Available settings:
- **default_model**: gemini-2.5-flash, gemini-2.5-pro, gemini-3.0-pro
- **auto_close**: Auto-close connections after inactivity
- **verbose**: Enable debug logging

### Model Selection

```bash
# Use specific model for a query
geminiweb -m gemini-3.0-pro "Explain quantum computing"

# In chat mode
geminiweb chat -m gemini-2.5-pro
```

### Auto Cookie Refresh

Enable automatic cookie refresh from browser when authentication fails:

```bash
# Auto-detect browser for refresh
geminiweb --browser-refresh=auto "Hello"
geminiweb --browser-refresh=auto chat

# Use specific browser
geminiweb --browser-refresh=firefox "Hello"
geminiweb --browser-refresh=chrome chat
```

Supported browsers: `chrome`, `chromium`, `firefox`, `edge`, `opera`, `auto`

## Cookie Format

The cookies file supports two formats:

**Browser export format (list):**
```json
[
  {"name": "__Secure-1PSID", "value": "..."},
  {"name": "__Secure-1PSIDTS", "value": "..."}
]
```

**Simple format (dict):**
```json
{
  "__Secure-1PSID": "...",
  "__Secure-1PSIDTS": "..."
}
```

Required: `__Secure-1PSID`
Optional: `__Secure-1PSIDTS`

## Project Structure

```
geminiweb-go/
├── cmd/geminiweb/       # Entry point
├── internal/
│   ├── api/             # API client (TLS, token, generation, browser refresh)
│   ├── browser/         # Browser cookie extraction (Chrome, Firefox, Edge, etc.)
│   ├── commands/        # CLI commands (Cobra)
│   ├── config/          # Configuration and cookies
│   ├── errors/          # Custom error types
│   ├── history/         # Conversation history persistence
│   ├── models/          # Data types and constants
│   └── tui/             # Terminal UI (Bubble Tea)
├── Makefile
└── go.mod
```

## Development

```bash
# Download dependencies
make deps

# Build for development (faster)
make build-dev

# Run with arguments
make run ARGS="chat"

# Run tests
make test

# Format code
make fmt
```

## License

MIT
</file>


---

## Instructions

Please analyze the provided information and:

1. Understand the task requirements
2. Review the project structure
3. Consider the specified rules and constraints
4. Provide a detailed solution
