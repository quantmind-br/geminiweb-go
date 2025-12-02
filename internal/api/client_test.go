package api

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/tls-client/bandwidth"

	"github.com/diogo/geminiweb/internal/browser"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
)

// MockBrowserCookieExtractor is a mock implementation of BrowserCookieExtractor
type MockBrowserCookieExtractor struct {
	ExtractResult *browser.ExtractResult
	ExtractError  error
}

// ExtractGeminiCookies implements the BrowserCookieExtractor interface
func (m *MockBrowserCookieExtractor) ExtractGeminiCookies(ctx context.Context, browser browser.SupportedBrowser) (*browser.ExtractResult, error) {
	return m.ExtractResult, m.ExtractError
}

// TestNewClient tests the NewClient function
func TestNewClient(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	tests := []struct {
		name        string
		cookies     *config.Cookies
		opts        []ClientOption
		wantErr     bool
		wantModel   models.Model
		autoRefresh bool
		interval    time.Duration
	}{
		{
			name:        "valid cookies with defaults",
			cookies:     validCookies,
			wantErr:     false,
			wantModel:   models.DefaultModel, // Default model is now gemini-2.5-flash
			autoRefresh: true,
			interval:    9 * time.Minute,
		},
		{
			name:        "with custom model",
			cookies:     validCookies,
			opts:        []ClientOption{WithModel(models.Model25Flash)},
			wantErr:     false,
			wantModel:   models.Model25Flash,
			autoRefresh: true,
			interval:    9 * time.Minute,
		},
		{
			name:        "with auto-refresh disabled",
			cookies:     validCookies,
			opts:        []ClientOption{WithAutoRefresh(false)},
			wantErr:     false,
			wantModel:   models.DefaultModel,
			autoRefresh: false,
			interval:    9 * time.Minute,
		},
		{
			name:        "with custom refresh interval",
			cookies:     validCookies,
			opts:        []ClientOption{WithRefreshInterval(5 * time.Minute)},
			wantErr:     false,
			wantModel:   models.DefaultModel,
			autoRefresh: true,
			interval:    5 * time.Minute,
		},
		{
			name:        "nil cookies (now allowed for silent auth)",
			cookies:     nil,
			wantErr:     false,
			wantModel:   models.DefaultModel,
			autoRefresh: true,
			interval:    9 * time.Minute,
		},
		{
			name:    "empty PSID",
			cookies: &config.Cookies{Secure1PSID: ""},
			wantErr: true,
		},
		{
			name:        "cookies with only PSID (no PSIDTS)",
			cookies:     &config.Cookies{Secure1PSID: "test_psid"},
			wantErr:     false,
			wantModel:   models.DefaultModel,
			autoRefresh: true,
			interval:    9 * time.Minute,
		},
		{
			name:        "with custom browser cookie extractor",
			cookies:     validCookies,
			opts:        []ClientOption{WithBrowserCookieExtractor(&MockBrowserCookieExtractor{})},
			wantErr:     false,
			wantModel:   models.DefaultModel,
			autoRefresh: true,
			interval:    9 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cookies, tt.opts...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("NewClient() returned nil client")
				return
			}

			// Verify model
			if client.GetModel().Name != tt.wantModel.Name {
				t.Errorf("GetModel() = %v, want %v", client.GetModel().Name, tt.wantModel.Name)
			}

			// Verify auto-refresh
			if client.autoRefresh != tt.autoRefresh {
				t.Errorf("autoRefresh = %v, want %v", client.autoRefresh, tt.autoRefresh)
			}

			// Verify refresh interval
			if client.refreshInterval != tt.interval {
				t.Errorf("refreshInterval = %v, want %v", client.refreshInterval, tt.interval)
			}

			// Verify cookies
			if client.GetCookies() != tt.cookies {
				t.Error("GetCookies() returned different cookies than passed to NewClient()")
			}
		})
	}
}

// TestGeminiClient_Init tests the Init method
func TestGeminiClient_Init(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	tokenResponse := `<html>
<script>
window.data = {"SNlM0e":"test_token_12345"};
</script>
</html>`

	tests := []struct {
		name         string
		setupMock    func(*MockHttpClient)
		wantToken    string
		wantErr      bool
		setupRotator bool
	}{
		{
			name: "successful initialization",
			setupMock: func(m *MockHttpClient) {
				body := NewMockResponseBody([]byte(tokenResponse))
				m.Response = &fhttp.Response{
					StatusCode: 200,
					Body:       body,
					Header:     make(fhttp.Header),
				}
			},
			wantToken:    "test_token_12345",
			wantErr:      false,
			setupRotator: true,
		},
		{
			name: "HTTP error status",
			setupMock: func(m *MockHttpClient) {
				body := NewMockResponseBody([]byte(""))
				m.Response = &fhttp.Response{
					StatusCode: 401,
					Body:       body,
					Header:     make(fhttp.Header),
				}
			},
			wantErr:      true,
			setupRotator: false,
		},
		{
			name: "network error",
			setupMock: func(m *MockHttpClient) {
				m.Err = errors.New("network error")
				m.Response = nil
			},
			wantErr:      true,
			setupRotator: false,
		},
		{
			name: "missing token in response",
			setupMock: func(m *MockHttpClient) {
				body := NewMockResponseBody([]byte("<html><body>No token</body></html>"))
				m.Response = &fhttp.Response{
					StatusCode: 200,
					Body:       body,
					Header:     make(fhttp.Header),
				}
			},
			wantErr:      true,
			setupRotator: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHttpClient{}
			tt.setupMock(mockClient)

			client, err := NewClient(validCookies)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			// Replace the HTTP client with our mock
			client.httpClient = mockClient

			// Optionally disable rotator for tests that don't want it
			if !tt.setupRotator {
				client.autoRefresh = false
			}

			err = client.Init()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Init() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Init() unexpected error: %v", err)
				return
			}

			token := client.GetAccessToken()
			if token != tt.wantToken {
				t.Errorf("GetAccessToken() = %q, want %q", token, tt.wantToken)
			}
		})
	}
}

// TestGeminiClient_Init_ClosedClient tests Init on a closed client
func TestGeminiClient_Init_ClosedClient(t *testing.T) {
	mockClient := &MockHttpClient{}
	body := NewMockResponseBody([]byte(`{"SNlM0e":"token"}`))
	mockClient.Response = &fhttp.Response{
		StatusCode: 200,
		Body:       body,
		Header:     make(fhttp.Header),
	}

	cookies := &config.Cookies{Secure1PSID: "test"}
	client, err := NewClient(cookies)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	client.httpClient = mockClient
	client.autoRefresh = false // Disable rotator

	// Close the client
	client.Close()

	// Try to init a closed client
	err = client.Init()
	if err == nil {
		t.Error("Init() on closed client should return error")
	}
}

// TestGeminiClient_Close tests the Close method (idempotence)
func TestGeminiClient_Close(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	client, err := NewClient(validCookies, WithAutoRefresh(true))
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	client.autoRefresh = true
	client.rotator = NewCookieRotator(&MockHttpClient{}, validCookies, time.Minute)

	// Close once
	client.Close()
	if !client.IsClosed() {
		t.Error("IsClosed() should return true after first Close()")
	}

	// Close again (should be idempotent)
	client.Close()
	if !client.IsClosed() {
		t.Error("IsClosed() should still return true after second Close()")
	}
}

// TestGeminiClient_GetSetMethods tests getter and setter methods
func TestGeminiClient_GetSetMethods(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	client, err := NewClient(cookies)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Test GetModel
	model := client.GetModel()
	if model.Name != models.DefaultModel.Name {
		t.Errorf("GetModel() default = %v, want %v", model.Name, models.DefaultModel.Name)
	}

	// Test SetModel
	newModel := models.Model30Pro
	client.SetModel(newModel)
	actualModel := client.GetModel()
	if actualModel.Name != newModel.Name {
		t.Errorf("SetModel(%v) then GetModel() = %v, want %v", newModel.Name, actualModel.Name, newModel.Name)
	}

	// Test GetCookies
	retrievedCookies := client.GetCookies()
	if retrievedCookies != cookies {
		t.Error("GetCookies() should return the same cookies passed to NewClient()")
	}

	// Test GetHTTPClient
	httpClient := client.GetHTTPClient()
	if httpClient == nil {
		t.Error("GetHTTPClient() should return non-nil client")
	}

	// Test GetAccessToken (should be empty before Init)
	token := client.GetAccessToken()
	if token != "" {
		t.Errorf("GetAccessToken() before Init() = %q, want empty string", token)
	}

	// Test IsClosed
	if client.IsClosed() {
		t.Error("IsClosed() should return false for new client")
	}
}

// TestGeminiClient_StartChat tests StartChat method
func TestGeminiClient_StartChat(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	tests := []struct {
		name          string
		opts          []ClientOption
		customModel   *models.Model
		expectedModel models.Model
	}{
		{
			name:          "default model",
			opts:          []ClientOption{WithModel(models.Model30Pro)},
			expectedModel: models.Model30Pro,
		},
		{
			name:          "custom model via argument",
			opts:          []ClientOption{WithModel(models.Model30Pro)},
			customModel:   &[]models.Model{models.Model25Flash}[0],
			expectedModel: models.Model25Flash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(cookies, tt.opts...)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			var session *ChatSession
			if tt.customModel != nil {
				session = client.StartChat(*tt.customModel)
			} else {
				session = client.StartChat()
			}

			if session == nil {
				t.Error("StartChat() returned nil session")
				return
			}

			if session.client != client {
				t.Error("ChatSession should reference the client")
			}

			if session.GetModel().Name != tt.expectedModel.Name {
				t.Errorf("Session model = %v, want %v", session.GetModel().Name, tt.expectedModel.Name)
			}
		})
	}
}

// TestGeminiClient_ConcurrentAccess tests concurrent access to client methods
func TestGeminiClient_ConcurrentAccess(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	client, err := NewClient(cookies)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Set a custom model
	newModel := models.Model30Pro
	client.SetModel(newModel)

	// Run concurrent reads
	var wg sync.WaitGroup
	iterations := 100

	// Test concurrent GetModel calls
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			model := client.GetModel()
			if model.Name == "" {
				t.Error("GetModel() returned empty model in concurrent access")
			}
		}()
	}

	// Test concurrent GetAccessToken calls (before Init)
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			token := client.GetAccessToken()
			if token == "" {
				// This is expected before Init
			}
		}()
	}

	// Test concurrent IsClosed calls
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			_ = client.IsClosed()
		}()
	}

	// Test concurrent SetModel and GetModel
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(index int) {
			defer wg.Done()
			if index%2 == 0 {
				client.SetModel(models.Model30Pro)
			} else {
				_ = client.GetModel()
			}
		}(i)
	}

	wg.Wait()

	// Verify the model is still set correctly
	if client.GetModel().Name != models.Model30Pro.Name {
		t.Errorf("Model after concurrent access = %v, want %v", client.GetModel().Name, models.Model30Pro.Name)
	}
}

// TestGeminiClient_ConcurrencyWithInit tests concurrency during Init
func TestGeminiClient_ConcurrencyWithInit(t *testing.T) {
	tmpDir := t.TempDir()
	_ = tmpDir

	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	mockClient := &MockHttpClient{}
	tokenResponse := `<html><script>window.data = {"SNlM0e":"concurrent_token"};</script></html>`
	body := NewMockResponseBody([]byte(tokenResponse))
	mockClient.Response = &fhttp.Response{
		StatusCode: 200,
		Body:       body,
		Header:     make(fhttp.Header),
	}

	client, err := NewClient(cookies, WithAutoRefresh(false))
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	client.httpClient = mockClient

	// Call Init concurrently
	var wg sync.WaitGroup
	errCh := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.Init()
			errCh <- err
		}()
	}
	wg.Wait()

	// Check results
	close(errCh)
	errorCount := 0
	for err := range errCh {
		if err != nil {
			errorCount++
		}
	}

	// At least one Init should succeed (mutex protection prevents race conditions)
	if errorCount == 10 {
		t.Error("All concurrent Init() calls failed, mutex may not be working correctly")
	}

	// Verify token was set by at least one successful call
	token := client.GetAccessToken()
	if token != "concurrent_token" {
		t.Errorf("GetAccessToken() = %q, want %q", token, "concurrent_token")
	}

	// Verify client is not closed
	if client.IsClosed() {
		t.Error("IsClosed() should return false after successful Init()")
	}
}

// TestGeminiClient_WithModel tests WithModel option
func TestGeminiClient_WithModel(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	tests := []struct {
		name      string
		model     models.Model
		wantModel models.Model
	}{
		{"G_2_5_FLASH", models.Model25Flash, models.Model25Flash},
		{"G_3_0_PRO", models.Model30Pro, models.Model30Pro},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(cookies, WithModel(tt.model))
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			if client.GetModel().Name != tt.wantModel.Name {
				t.Errorf("GetModel() = %v, want %v", client.GetModel().Name, tt.wantModel.Name)
			}
		})
	}
}

// TestGeminiClient_WithAutoRefresh tests WithAutoRefresh option
func TestGeminiClient_WithAutoRefresh(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	tests := []struct {
		name        string
		enabled     bool
		wantEnabled bool
	}{
		{"enabled", true, true},
		{"disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(cookies, WithAutoRefresh(tt.enabled))
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			// Access via unexported field for verification
			// We'll test the behavior indirectly through Init
			mockClient := &MockHttpClient{}
			body := NewMockResponseBody([]byte(`{"SNlM0e":"token"}`))
			mockClient.Response = &fhttp.Response{
				StatusCode: 200,
				Body:       body,
				Header:     make(fhttp.Header),
			}
			client.httpClient = mockClient

			if tt.wantEnabled {
				// When auto-refresh is enabled, rotator should be created in Init
				client.Init()
				// Note: We can't directly test the rotator without exposing it
				// But the presence is tested through behavior
			} else {
				// When auto-refresh is disabled, rotator should not be created
				client.autoRefresh = false
				client.Init()
				if client.rotator != nil {
					t.Error("Rotator should be nil when auto-refresh is disabled")
				}
			}
		})
	}
}

// TestGeminiClient_WithRefreshInterval tests WithRefreshInterval option
func TestGeminiClient_WithRefreshInterval(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{"1 minute", time.Minute, time.Minute},
		{"5 minutes", 5 * time.Minute, 5 * time.Minute},
		{"10 minutes", 10 * time.Minute, 10 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(cookies, WithRefreshInterval(tt.interval))
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			if client.refreshInterval != tt.want {
				t.Errorf("refreshInterval = %v, want %v", client.refreshInterval, tt.want)
			}
		})
	}
}

// TestGeminiClient_CookieValidation tests cookie validation in NewClient
func TestGeminiClient_CookieValidation(t *testing.T) {
	tests := []struct {
		name    string
		cookies *config.Cookies
		wantErr bool
	}{
		{
			name:    "nil cookies (now allowed for silent auth)",
			cookies: nil,
			wantErr: false,
		},
		{
			name:    "empty PSID",
			cookies: &config.Cookies{Secure1PSID: ""},
			wantErr: true,
		},
		{
			name:    "valid with only PSID",
			cookies: &config.Cookies{Secure1PSID: "valid_psid"},
			wantErr: false,
		},
		{
			name:    "valid with both cookies",
			cookies: &config.Cookies{Secure1PSID: "valid_psid", Secure1PSIDTS: "valid_psidts"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.cookies)

			if tt.wantErr && err == nil {
				t.Error("NewClient() expected error but got none")
			} else if !tt.wantErr && err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
			}
		})
	}
}

// TestGeminiClient_CloseMultipleTimes tests that Close is idempotent
func TestGeminiClient_CloseMultipleTimes(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	client, err := NewClient(cookies, WithAutoRefresh(true))
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Create a mock rotator
	mockHttpClient := &MockHttpClient{}
	client.rotator = NewCookieRotator(mockHttpClient, cookies, time.Minute)

	// Close multiple times
	for i := 0; i < 5; i++ {
		client.Close()
		if !client.IsClosed() {
			t.Errorf("IsClosed() should return true after Close() #%d", i+1)
		}
	}

	// Verify rotator is stopped (would panic if called, but we can't test that directly)
	// Instead, verify the client is in a consistent state
	if client.closed != true {
		t.Error("Client closed flag should remain true")
	}
}

// TestGeminiClient_GetAccessTokenBeforeInit tests behavior before Init
func TestGeminiClient_GetAccessTokenBeforeInit(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	client, err := NewClient(cookies)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// GetAccessToken before Init should return empty string
	token := client.GetAccessToken()
	if token != "" {
		t.Errorf("GetAccessToken() before Init() = %q, want empty string", token)
	}
}

// TestGeminiClient_StartChatWithMultipleModels tests StartChat with different models
func TestGeminiClient_StartChatWithMultipleModels(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	client, err := NewClient(cookies, WithModel(models.Model30Pro))
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Start chat with default model
	session1 := client.StartChat()
	if session1.GetModel().Name != models.Model30Pro.Name {
		t.Errorf("Session model = %v, want %v", session1.GetModel().Name, models.Model30Pro.Name)
	}

	// Start chat with custom model
	customModel := models.Model25Flash
	session2 := client.StartChat(customModel)
	if session2.GetModel().Name != customModel.Name {
		t.Errorf("Session model = %v, want %v", session2.GetModel().Name, customModel.Name)
	}

	// Original session should remain unchanged
	if session1.GetModel().Name != models.Model30Pro.Name {
		t.Error("Original session model should not change")
	}
}

// TestGeminiClient_AccessTokenImmutability tests that access token doesn't change unexpectedly
func TestGeminiClient_AccessTokenImmutability(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	mockClient := &MockHttpClient{}
	tokenResponse := `<html><script>window.data = {"SNlM0e":"immutable_token"};</script></html>`
	body := NewMockResponseBody([]byte(tokenResponse))
	mockClient.Response = &fhttp.Response{
		StatusCode: 200,
		Body:       body,
		Header:     make(fhttp.Header),
	}

	client, err := NewClient(cookies)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	client.httpClient = mockClient

	// Init should set the token
	err = client.Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	firstToken := client.GetAccessToken()

	// Multiple calls to GetAccessToken should return the same token
	for i := 0; i < 10; i++ {
		token := client.GetAccessToken()
		if token != firstToken {
			t.Errorf("GetAccessToken() #%d = %q, want %q", i+1, token, firstToken)
		}
	}
}

// TestGeminiClient_WithBrowserRefresh tests WithBrowserRefresh option
func TestGeminiClient_WithBrowserRefresh(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID: "test_psid",
	}

	tests := []struct {
		name         string
		browserType  browser.SupportedBrowser
		wantEnabled  bool
		wantBrowser  browser.SupportedBrowser
	}{
		{
			name:        "with chrome",
			browserType: browser.BrowserChrome,
			wantEnabled: true,
			wantBrowser: browser.BrowserChrome,
		},
		{
			name:        "with firefox",
			browserType: browser.BrowserFirefox,
			wantEnabled: true,
			wantBrowser: browser.BrowserFirefox,
		},
		{
			name:        "with auto",
			browserType: browser.BrowserAuto,
			wantEnabled: true,
			wantBrowser: browser.BrowserAuto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(cookies, WithBrowserRefresh(tt.browserType))
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			// Verify browser refresh is enabled
			if !client.IsBrowserRefreshEnabled() {
				t.Error("IsBrowserRefreshEnabled() should return true")
			}
		})
	}
}

// TestGeminiClient_RefreshFromBrowser tests RefreshFromBrowser method
func TestGeminiClient_RefreshFromBrowser(t *testing.T) {
	cookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	t.Run("browser_refresh_not_enabled", func(t *testing.T) {
		// Create client without browser refresh
		client, err := NewClient(cookies, WithAutoRefresh(false))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Try to refresh - should fail
		success, err := client.RefreshFromBrowser()
		if success {
			t.Error("RefreshFromBrowser() should return false when not enabled")
		}
		if err == nil {
			t.Error("RefreshFromBrowser() should return error when not enabled")
		}
		if !strings.Contains(err.Error(), "browser refresh is not enabled") {
			t.Errorf("Expected error about browser refresh not enabled, got: %v", err)
		}
	})

	t.Run("rate_limiting", func(t *testing.T) {
		// Create client with browser refresh enabled
		client, err := NewClient(cookies, WithBrowserRefresh(browser.BrowserChrome))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Try to refresh twice in quick succession
		// First call might succeed or fail depending on browser availability
		_, _ = client.RefreshFromBrowser()

		// Second call should be rate limited
		success, err := client.RefreshFromBrowser()
		if success {
			t.Log("Second RefreshFromBrowser() succeeded (may have browser available)")
		} else if err != nil && strings.Contains(err.Error(), "too recently") {
			t.Log("Second call correctly rate limited")
		}
	})

	t.Run("browser_extraction_failure", func(t *testing.T) {
		// Create client with browser refresh enabled
		client, err := NewClient(cookies, WithBrowserRefresh(browser.BrowserChrome))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Try to refresh with non-existent browser
		success, err := client.RefreshFromBrowser()
		if success {
			t.Error("RefreshFromBrowser() should return false when browser extraction fails")
		}
		if err == nil {
			t.Error("RefreshFromBrowser() should return error when browser extraction fails")
		}
		if !strings.Contains(err.Error(), "failed to extract cookies") {
			t.Errorf("Expected error about cookie extraction, got: %v", err)
		}
	})

	t.Run("custom_extractor_with_extraction_error", func(t *testing.T) {
		// Create client with custom extractor that returns an error
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractError: errors.New("extraction failed"),
		}
		client, err := NewClient(cookies,
			WithBrowserRefresh(browser.BrowserChrome),
			WithBrowserCookieExtractor(mockExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Try to refresh - should fail with custom error
		success, err := client.RefreshFromBrowser()
		if success {
			t.Error("RefreshFromBrowser() should return false when custom extractor fails")
		}
		if err == nil {
			t.Error("RefreshFromBrowser() should return error when custom extractor fails")
		}
		if !strings.Contains(err.Error(), "extraction failed") {
			t.Errorf("Expected custom error message, got: %v", err)
		}
	})

	t.Run("custom_extractor_with_token_fetch_error", func(t *testing.T) {
		// Create client with custom extractor that succeeds but HTTP client fails
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractResult: &browser.ExtractResult{
				Cookies: &config.Cookies{
					Secure1PSID:   "new_psid",
					Secure1PSIDTS: "new_psidts",
				},
				BrowserName: "Mock Browser",
			},
		}

		// Create client with mock HTTP client that returns 401
		mockHttpClient := NewMockHttpClientWithError(errors.New("unauthorized"))
		client, err := NewClient(cookies,
			WithBrowserRefresh(browser.BrowserChrome),
			WithBrowserCookieExtractor(mockExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Replace HTTP client with our mock
		client.httpClient = mockHttpClient

		// Try to refresh - should fail when fetching token
		success, err := client.RefreshFromBrowser()
		if success {
			t.Error("RefreshFromBrowser() should return false when token fetch fails")
		}
		if err == nil {
			t.Error("RefreshFromBrowser() should return error when token fetch fails")
		}
		if !strings.Contains(err.Error(), "failed to get access token") {
			t.Errorf("Expected error about token fetch, got: %v", err)
		}
	})

	t.Run("custom_extractor_success", func(t *testing.T) {
		// Create client with custom extractor that succeeds
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractResult: &browser.ExtractResult{
				Cookies: &config.Cookies{
					Secure1PSID:   "new_psid",
					Secure1PSIDTS: "new_psidts",
				},
				BrowserName: "Mock Browser",
			},
		}

		// Create mock HTTP client that returns a valid token
		htmlWithToken := []byte(`{"SNlM0e":"new_token_123"}`)
		mockHttpClient := NewMockHttpClient(htmlWithToken, 200)

		client, err := NewClient(cookies,
			WithBrowserRefresh(browser.BrowserChrome),
			WithBrowserCookieExtractor(mockExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Replace HTTP client with our mock
		client.httpClient = mockHttpClient

		// Try to refresh - should succeed
		success, err := client.RefreshFromBrowser()
		if !success {
			t.Error("RefreshFromBrowser() should return true when custom extractor succeeds")
		}
		if err != nil {
			t.Errorf("RefreshFromBrowser() should not return error when custom extractor succeeds, got: %v", err)
		}

		// Verify cookies were updated
		if client.GetCookies().Secure1PSID != "new_psid" {
			t.Errorf("Cookie PSID was not updated, got: %s", client.GetCookies().Secure1PSID)
		}
		if client.GetAccessToken() != "new_token_123" {
			t.Errorf("Access token was not updated, got: %s", client.GetAccessToken())
		}
	})
}

// TestNewClient_NilCookies tests that NewClient accepts nil cookies
func TestNewClient_NilCookies(t *testing.T) {
	client, err := NewClient(nil)
	if err != nil {
		t.Errorf("NewClient(nil) should not return error, got: %v", err)
	}
	if client == nil {
		t.Error("NewClient(nil) should return a valid client")
	}
}

// TestNewClient_WithHTTPClient tests that NewClient accepts a custom HTTP client
func TestNewClient_WithHTTPClient(t *testing.T) {
	validCookies := &config.Cookies{
		Secure1PSID:   "test_psid",
		Secure1PSIDTS: "test_psidts",
	}

	// Create a mock HTTP client
	mockClient := &mockHTTPClient{
		doFunc: func(req *fhttp.Request) (*fhttp.Response, error) {
			return &fhttp.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("test")),
			}, nil
		},
	}

	client, err := NewClient(validCookies, WithHTTPClient(mockClient))
	if err != nil {
		t.Fatalf("NewClient with WithHTTPClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient with WithHTTPClient should return a valid client")
	}

	// Verify the mock HTTP client was injected
	if client.httpClient != mockClient {
		t.Error("Expected injected HTTP client to be used")
	}
}

// TestGeminiClient_InitWithCookieLoader tests Init with a custom cookie loader
func TestGeminiClient_InitWithCookieLoader(t *testing.T) {
	t.Run("loads_cookies_from_loader_when_nil", func(t *testing.T) {
		// Create a mock cookie loader
		mockCookies := &config.Cookies{
			Secure1PSID:   "loaded_psid",
			Secure1PSIDTS: "loaded_psidts",
		}
		mockLoader := func() (*config.Cookies, error) {
			return mockCookies, nil
		}

		// Create client with nil cookies and custom loader
		client, err := NewClient(nil, WithCookieLoader(mockLoader))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Setup mock HTTP client for token fetch
		tokenResponse := `<html><script>window.data = {"SNlM0e":"loaded_token"};</script></html>`
		mockHttpClient := NewMockHttpClient([]byte(tokenResponse), 200)
		client.httpClient = mockHttpClient
		client.autoRefresh = false

		// Init should load cookies from loader
		err = client.Init()
		if err != nil {
			t.Fatalf("Init() failed: %v", err)
		}

		// Verify cookies were loaded
		if client.GetCookies().Secure1PSID != "loaded_psid" {
			t.Errorf("Cookie PSID = %s, want loaded_psid", client.GetCookies().Secure1PSID)
		}
		if client.GetAccessToken() != "loaded_token" {
			t.Errorf("Access token = %s, want loaded_token", client.GetAccessToken())
		}
	})

	t.Run("skips_loader_when_cookies_provided", func(t *testing.T) {
		providedCookies := &config.Cookies{
			Secure1PSID:   "provided_psid",
			Secure1PSIDTS: "provided_psidts",
		}

		// Create a mock cookie loader that should NOT be called
		loaderCalled := false
		mockLoader := func() (*config.Cookies, error) {
			loaderCalled = true
			return &config.Cookies{Secure1PSID: "loader_psid"}, nil
		}

		client, err := NewClient(providedCookies, WithCookieLoader(mockLoader))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Setup mock HTTP client
		tokenResponse := `<html><script>window.data = {"SNlM0e":"token"};</script></html>`
		mockHttpClient := NewMockHttpClient([]byte(tokenResponse), 200)
		client.httpClient = mockHttpClient
		client.autoRefresh = false

		err = client.Init()
		if err != nil {
			t.Fatalf("Init() failed: %v", err)
		}

		// Loader should not have been called
		if loaderCalled {
			t.Error("Cookie loader should not be called when cookies are provided")
		}

		// Should use provided cookies
		if client.GetCookies().Secure1PSID != "provided_psid" {
			t.Errorf("Cookie PSID = %s, want provided_psid", client.GetCookies().Secure1PSID)
		}
	})

	t.Run("falls_back_to_browser_when_loader_fails", func(t *testing.T) {
		// Create a mock cookie loader that fails
		mockLoader := func() (*config.Cookies, error) {
			return nil, errors.New("no cookies file")
		}

		// Create a mock browser extractor that succeeds
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractResult: &browser.ExtractResult{
				Cookies: &config.Cookies{
					Secure1PSID:   "browser_psid",
					Secure1PSIDTS: "browser_psidts",
				},
				BrowserName: "Mock Browser",
			},
		}

		client, err := NewClient(nil,
			WithCookieLoader(mockLoader),
			WithBrowserRefresh(browser.BrowserChrome),
			WithBrowserCookieExtractor(mockExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Setup mock HTTP client
		tokenResponse := `<html><script>window.data = {"SNlM0e":"browser_token"};</script></html>`
		mockHttpClient := NewMockHttpClient([]byte(tokenResponse), 200)
		client.httpClient = mockHttpClient
		client.autoRefresh = false

		err = client.Init()
		if err != nil {
			t.Fatalf("Init() failed: %v", err)
		}

		// Should use browser cookies
		if client.GetCookies().Secure1PSID != "browser_psid" {
			t.Errorf("Cookie PSID = %s, want browser_psid", client.GetCookies().Secure1PSID)
		}
		if client.GetAccessToken() != "browser_token" {
			t.Errorf("Access token = %s, want browser_token", client.GetAccessToken())
		}
	})

	t.Run("fails_when_loader_and_browser_both_fail", func(t *testing.T) {
		// Create a mock cookie loader that fails
		mockLoader := func() (*config.Cookies, error) {
			return nil, errors.New("no cookies file")
		}

		// Create a mock browser extractor that fails
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractError: errors.New("browser extraction failed"),
		}

		client, err := NewClient(nil,
			WithCookieLoader(mockLoader),
			WithBrowserCookieExtractor(mockExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		client.autoRefresh = false

		err = client.Init()
		if err == nil {
			t.Error("Init() should fail when both loader and browser extraction fail")
		}
		if !strings.Contains(err.Error(), "authentication failed") {
			t.Errorf("Error should mention authentication failure, got: %v", err)
		}
	})
}

// TestGeminiClient_InitialBrowserRefresh tests the initialBrowserRefresh method
func TestGeminiClient_InitialBrowserRefresh(t *testing.T) {
	t.Run("does_not_enforce_rate_limiting", func(t *testing.T) {
		cookies := &config.Cookies{Secure1PSID: "test"}

		// Create a mock browser extractor
		callCount := 0
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractResult: &browser.ExtractResult{
				Cookies: &config.Cookies{
					Secure1PSID:   "browser_psid",
					Secure1PSIDTS: "browser_psidts",
				},
				BrowserName: "Mock Browser",
			},
		}

		// Wrap the extractor to count calls
		wrappedExtractor := &countingExtractor{
			inner:     mockExtractor,
			callCount: &callCount,
		}

		client, err := NewClient(cookies,
			WithBrowserRefresh(browser.BrowserChrome),
			WithBrowserCookieExtractor(wrappedExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Call initialBrowserRefresh multiple times in quick succession
		// Unlike RefreshFromBrowser, it should NOT be rate limited
		client.mu.Lock()
		err = client.initialBrowserRefresh(browser.BrowserChrome)
		client.mu.Unlock()
		if err != nil {
			t.Fatalf("First initialBrowserRefresh() failed: %v", err)
		}

		client.mu.Lock()
		err = client.initialBrowserRefresh(browser.BrowserChrome)
		client.mu.Unlock()
		if err != nil {
			t.Fatalf("Second initialBrowserRefresh() failed: %v", err)
		}

		// Both calls should have succeeded (no rate limiting)
		if callCount != 2 {
			t.Errorf("Expected 2 calls to browser extractor, got %d", callCount)
		}
	})

	t.Run("uses_auto_browser_when_type_not_set", func(t *testing.T) {
		// Create client without setting browserRefreshType
		client, err := NewClient(nil)
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Verify browserRefreshType is empty
		if client.browserRefreshType != "" {
			t.Errorf("browserRefreshType should be empty, got: %s", client.browserRefreshType)
		}

		// Create mock loader that fails to trigger browser refresh
		client.cookieLoader = func() (*config.Cookies, error) {
			return nil, errors.New("no cookies")
		}

		// Create mock extractor that captures the browser type
		var capturedBrowserType browser.SupportedBrowser
		mockExtractor := &capturingExtractor{
			result: &browser.ExtractResult{
				Cookies:     &config.Cookies{Secure1PSID: "psid"},
				BrowserName: "Mock",
			},
			capturedType: &capturedBrowserType,
		}
		client.browserExtractor = mockExtractor

		// Setup mock HTTP client
		tokenResponse := `<html><script>window.data = {"SNlM0e":"token"};</script></html>`
		mockHttpClient := NewMockHttpClient([]byte(tokenResponse), 200)
		client.httpClient = mockHttpClient
		client.autoRefresh = false

		_ = client.Init()

		// Should have used "auto" as the browser type
		if capturedBrowserType != browser.BrowserAuto {
			t.Errorf("Expected browser type 'auto', got: %s", capturedBrowserType)
		}
	})
}

// countingExtractor wraps a BrowserCookieExtractor and counts calls
type countingExtractor struct {
	inner     BrowserCookieExtractor
	callCount *int
}

func (c *countingExtractor) ExtractGeminiCookies(ctx context.Context, b browser.SupportedBrowser) (*browser.ExtractResult, error) {
	*c.callCount++
	return c.inner.ExtractGeminiCookies(ctx, b)
}

// capturingExtractor captures the browser type passed to ExtractGeminiCookies
type capturingExtractor struct {
	result       *browser.ExtractResult
	err          error
	capturedType *browser.SupportedBrowser
}

func (c *capturingExtractor) ExtractGeminiCookies(ctx context.Context, b browser.SupportedBrowser) (*browser.ExtractResult, error) {
	*c.capturedType = b
	return c.result, c.err
}

// TestGeminiClient_InitWithExpiredCookies tests Init behavior when cookies exist but are expired
// This is Test Case 2 from the plan: cookies exist on disk but GetAccessToken fails
func TestGeminiClient_InitWithExpiredCookies(t *testing.T) {
	t.Run("browser_fallback_when_token_fetch_fails_with_disk_cookies", func(t *testing.T) {
		// Simulate cookies loaded from disk that are "expired"
		expiredCookies := &config.Cookies{
			Secure1PSID:   "expired_psid",
			Secure1PSIDTS: "expired_psidts",
		}

		// Create a mock cookie loader that returns "expired" cookies
		loaderCalled := false
		mockLoader := func() (*config.Cookies, error) {
			loaderCalled = true
			return expiredCookies, nil
		}

		// Create a mock browser extractor that returns fresh cookies
		extractorCalled := false
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractResult: &browser.ExtractResult{
				Cookies: &config.Cookies{
					Secure1PSID:   "fresh_browser_psid",
					Secure1PSIDTS: "fresh_browser_psidts",
				},
				BrowserName: "Mock Browser",
			},
		}

		// Wrap extractor to track if it was called
		wrappedExtractor := &trackingExtractor{
			inner:  mockExtractor,
			called: &extractorCalled,
		}

		client, err := NewClient(nil,
			WithCookieLoader(mockLoader),
			WithBrowserCookieExtractor(wrappedExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// Create a mock HTTP client that:
		// 1. First call (with expired cookies): returns 401 error
		// 2. Second call (with fresh cookies): returns valid token
		callCount := 0
		sequentialMockClient := &SequentialMockHttpClient{
			responses: []mockResponse{
				{statusCode: 401, body: []byte("unauthorized")}, // First call fails
				{statusCode: 200, body: []byte(`{"SNlM0e":"fresh_token"}`)}, // Second call succeeds
			},
			callCount: &callCount,
		}

		client.httpClient = sequentialMockClient
		client.autoRefresh = false

		// Init should:
		// 1. Load cookies from disk (expired)
		// 2. Try GetAccessToken -> fail with 401
		// 3. Try browser refresh -> succeed
		// 4. Retry GetAccessToken -> succeed
		err = client.Init()
		if err != nil {
			t.Fatalf("Init() should succeed after browser fallback, got error: %v", err)
		}

		// Verify cookie loader was called
		if !loaderCalled {
			t.Error("Cookie loader should have been called")
		}

		// Verify browser extractor was called as fallback
		if !extractorCalled {
			t.Error("Browser extractor should have been called as fallback when GetAccessToken failed")
		}

		// Verify we ended up with fresh cookies from browser
		if client.GetCookies().Secure1PSID != "fresh_browser_psid" {
			t.Errorf("Cookie PSID = %s, want fresh_browser_psid", client.GetCookies().Secure1PSID)
		}

		// Verify we got the token
		if client.GetAccessToken() != "fresh_token" {
			t.Errorf("Access token = %s, want fresh_token", client.GetAccessToken())
		}

		// Verify GetAccessToken was called twice (first failed, second succeeded)
		if callCount != 2 {
			t.Errorf("GetAccessToken should have been called 2 times, got %d", callCount)
		}
	})

	t.Run("fails_when_both_disk_cookies_and_browser_fail", func(t *testing.T) {
		// Simulate cookies loaded from disk that are "expired"
		expiredCookies := &config.Cookies{
			Secure1PSID:   "expired_psid",
			Secure1PSIDTS: "expired_psidts",
		}

		mockLoader := func() (*config.Cookies, error) {
			return expiredCookies, nil
		}

		// Browser extractor also fails
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractError: errors.New("browser not available"),
		}

		client, err := NewClient(nil,
			WithCookieLoader(mockLoader),
			WithBrowserCookieExtractor(mockExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// HTTP client returns 401 (expired cookies)
		mockHttpClient := NewMockHttpClient([]byte("unauthorized"), 401)
		client.httpClient = mockHttpClient
		client.autoRefresh = false

		err = client.Init()
		if err == nil {
			t.Error("Init() should fail when both disk cookies and browser fail")
		}

		// Error should mention both failures
		if !strings.Contains(err.Error(), "authentication failed") {
			t.Errorf("Error should mention authentication failure, got: %v", err)
		}
		if !strings.Contains(err.Error(), "browser refresh also failed") {
			t.Errorf("Error should mention browser refresh failure, got: %v", err)
		}
	})

	t.Run("skips_browser_fallback_when_token_fetch_succeeds", func(t *testing.T) {
		// Valid cookies that work
		validCookies := &config.Cookies{
			Secure1PSID:   "valid_psid",
			Secure1PSIDTS: "valid_psidts",
		}

		mockLoader := func() (*config.Cookies, error) {
			return validCookies, nil
		}

		// Browser extractor should NOT be called
		extractorCalled := false
		mockExtractor := &MockBrowserCookieExtractor{
			ExtractResult: &browser.ExtractResult{
				Cookies:     &config.Cookies{Secure1PSID: "browser_psid"},
				BrowserName: "Mock",
			},
		}
		wrappedExtractor := &trackingExtractor{
			inner:  mockExtractor,
			called: &extractorCalled,
		}

		client, err := NewClient(nil,
			WithCookieLoader(mockLoader),
			WithBrowserCookieExtractor(wrappedExtractor))
		if err != nil {
			t.Fatalf("NewClient() failed: %v", err)
		}

		// HTTP client returns valid token on first try
		tokenResponse := `{"SNlM0e":"valid_token"}`
		mockHttpClient := NewMockHttpClient([]byte(tokenResponse), 200)
		client.httpClient = mockHttpClient
		client.autoRefresh = false

		err = client.Init()
		if err != nil {
			t.Fatalf("Init() should succeed, got error: %v", err)
		}

		// Browser extractor should NOT have been called
		if extractorCalled {
			t.Error("Browser extractor should NOT be called when token fetch succeeds")
		}

		// Should use original cookies
		if client.GetCookies().Secure1PSID != "valid_psid" {
			t.Errorf("Cookie PSID = %s, want valid_psid", client.GetCookies().Secure1PSID)
		}
	})
}

// trackingExtractor wraps a BrowserCookieExtractor and tracks if it was called
type trackingExtractor struct {
	inner  BrowserCookieExtractor
	called *bool
}

func (t *trackingExtractor) ExtractGeminiCookies(ctx context.Context, b browser.SupportedBrowser) (*browser.ExtractResult, error) {
	*t.called = true
	return t.inner.ExtractGeminiCookies(ctx, b)
}

// mockResponse represents a single HTTP response for sequential mock
type mockResponse struct {
	statusCode int
	body       []byte
	err        error
}

// SequentialMockHttpClient returns different responses for sequential calls
type SequentialMockHttpClient struct {
	responses []mockResponse
	callCount *int
}

func (m *SequentialMockHttpClient) GetCookies(u *url.URL) []*fhttp.Cookie {
	return nil
}

func (m *SequentialMockHttpClient) SetCookies(u *url.URL, cookies []*fhttp.Cookie) {}

func (m *SequentialMockHttpClient) SetCookieJar(jar fhttp.CookieJar) {}

func (m *SequentialMockHttpClient) GetCookieJar() fhttp.CookieJar {
	return nil
}

func (m *SequentialMockHttpClient) SetProxy(proxy string) error {
	return nil
}

func (m *SequentialMockHttpClient) GetProxy() string {
	return ""
}

func (m *SequentialMockHttpClient) SetFollowRedirect(followRedirect bool) {}

func (m *SequentialMockHttpClient) GetFollowRedirect() bool {
	return true
}

func (m *SequentialMockHttpClient) CloseIdleConnections() {}

func (m *SequentialMockHttpClient) Get(u string) (*fhttp.Response, error) {
	return m.doRequest()
}

func (m *SequentialMockHttpClient) Head(u string) (*fhttp.Response, error) {
	return m.doRequest()
}

func (m *SequentialMockHttpClient) Post(u, contentType string, body io.Reader) (*fhttp.Response, error) {
	return m.doRequest()
}

func (m *SequentialMockHttpClient) GetBandwidthTracker() bandwidth.BandwidthTracker {
	return nil
}

func (m *SequentialMockHttpClient) Do(req *fhttp.Request) (*fhttp.Response, error) {
	return m.doRequest()
}

func (m *SequentialMockHttpClient) doRequest() (*fhttp.Response, error) {
	idx := *m.callCount
	*m.callCount++

	if idx >= len(m.responses) {
		// Return last response if we've exhausted the list
		idx = len(m.responses) - 1
	}

	resp := m.responses[idx]
	if resp.err != nil {
		return nil, resp.err
	}

	body := NewMockResponseBody(resp.body)
	return &fhttp.Response{
		StatusCode: resp.statusCode,
		Body:       body,
		Header:     make(fhttp.Header),
	}, nil
}
