package api

import (
	"context"

	"github.com/diogo/geminiweb/internal/browser"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
)

// MockGeminiClient is a mock implementation of GeminiClientInterface for testing
type MockGeminiClient struct {
	// Mock return values
	InitErr              error
	AccessToken          string
	Cookies              *config.Cookies
	Model                models.Model
	IsClosedVal          bool
	IsAutoCloseEnabledVal bool
	ChatSession          *ChatSession
	GenerateContentVal   *models.ModelOutput
	GenerateContentErr   error
	UploadImageVal       *UploadedImage
	UploadImageErr       error
	UploadFileVal        *UploadedFile
	UploadFileErr        error
	DownloadImageVal     string
	DownloadImageErr     error
	DownloadAllImagesVal []string
	DownloadAllImagesErr error
	RefreshFromBrowserVal bool
	RefreshFromBrowserErr error
	BrowserRefreshEnabled bool
	GemsJar              *models.GemJar
	GemsErr              error
	GemVal               *models.Gem
	GemErr               error
	BatchResponseVal     []BatchResponse
	BatchResponseErr     error

	// Call counters/recorders
	InitCalled           bool
	CloseCalled          bool
	GenerateContentCalled bool
	LastPrompt           string
}

// Ensure MockGeminiClient implements GeminiClientInterface
var _ GeminiClientInterface = (*MockGeminiClient)(nil)

func (m *MockGeminiClient) Init() error {
	m.InitCalled = true
	return m.InitErr
}

func (m *MockGeminiClient) Close() {
	m.CloseCalled = true
}

func (m *MockGeminiClient) GetAccessToken() string {
	return m.AccessToken
}

func (m *MockGeminiClient) GetCookies() *config.Cookies {
	return m.Cookies
}

func (m *MockGeminiClient) GetModel() models.Model {
	return m.Model
}

func (m *MockGeminiClient) SetModel(model models.Model) {
	m.Model = model
}

func (m *MockGeminiClient) IsClosed() bool {
	return m.IsClosedVal
}

func (m *MockGeminiClient) StartChat(model ...models.Model) *ChatSession {
	if m.ChatSession != nil {
		return m.ChatSession
	}
	return &ChatSession{client: m, model: m.Model}
}

func (m *MockGeminiClient) StartChatWithOptions(opts ...ChatOption) *ChatSession {
	if m.ChatSession != nil {
		return m.ChatSession
	}
	s := &ChatSession{client: m, model: m.Model}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (m *MockGeminiClient) GenerateContent(prompt string, opts *GenerateOptions) (*models.ModelOutput, error) {
	m.GenerateContentCalled = true
	m.LastPrompt = prompt
	return m.GenerateContentVal, m.GenerateContentErr
}

func (m *MockGeminiClient) UploadImage(filePath string) (*UploadedImage, error) {
	return m.UploadImageVal, m.UploadImageErr
}

func (m *MockGeminiClient) UploadFile(filePath string) (*UploadedFile, error) {

        return m.UploadFileVal, m.UploadFileErr

}



func (m *MockGeminiClient) UploadText(content string, fileName string) (*UploadedFile, error) {

        return m.UploadFileVal, m.UploadFileErr

}



func (m *MockGeminiClient) DownloadImage(img models.WebImage, opts ImageDownloadOptions) (string, error) {


	return m.DownloadImageVal, m.DownloadImageErr
}

func (m *MockGeminiClient) DownloadGeneratedImage(img models.GeneratedImage, opts ImageDownloadOptions) (string, error) {
	return m.DownloadImageVal, m.DownloadImageErr
}

func (m *MockGeminiClient) DownloadAllImages(output *models.ModelOutput, opts ImageDownloadOptions) ([]string, error) {
	return m.DownloadAllImagesVal, m.DownloadAllImagesErr
}

func (m *MockGeminiClient) DownloadSelectedImages(output *models.ModelOutput, indices []int, opts ImageDownloadOptions) ([]string, error) {
	return m.DownloadAllImagesVal, m.DownloadAllImagesErr
}

func (m *MockGeminiClient) RefreshFromBrowser() (bool, error) {
	return m.RefreshFromBrowserVal, m.RefreshFromBrowserErr
}

func (m *MockGeminiClient) IsBrowserRefreshEnabled() bool {
	return m.BrowserRefreshEnabled
}

func (m *MockGeminiClient) IsAutoCloseEnabled() bool {
	return m.IsAutoCloseEnabledVal
}

func (m *MockGeminiClient) FetchGems(includeHidden bool) (*models.GemJar, error) {
	return m.GemsJar, m.GemsErr
}

func (m *MockGeminiClient) CreateGem(name, prompt, description string) (*models.Gem, error) {
	return m.GemVal, m.GemErr
}

func (m *MockGeminiClient) UpdateGem(gemID, name, prompt, description string) (*models.Gem, error) {
	return m.GemVal, m.GemErr
}

func (m *MockGeminiClient) DeleteGem(gemID string) error {
	return m.GemErr
}

func (m *MockGeminiClient) Gems() *models.GemJar {
	return m.GemsJar
}

func (m *MockGeminiClient) GetGem(id, name string) *models.Gem {
	return m.GemVal
}

func (m *MockGeminiClient) BatchExecute(requests []RPCData) ([]BatchResponse, error) {
	return m.BatchResponseVal, m.BatchResponseErr
}

// MockBrowserExtractor is a mock for browser cookie extraction
type MockBrowserExtractor struct {
	Result *browser.ExtractResult
	Err    error
}

func (m *MockBrowserExtractor) ExtractGeminiCookies(ctx context.Context, browserType browser.SupportedBrowser) (*browser.ExtractResult, error) {
	return m.Result, m.Err
}
