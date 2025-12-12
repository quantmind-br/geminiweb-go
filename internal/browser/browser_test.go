package browser

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/browserutils/kooky"
)

func TestParseBrowser(t *testing.T) {
	tests := []struct {
		input    string
		expected SupportedBrowser
		wantErr  bool
	}{
		{"auto", BrowserAuto, false},
		{"", BrowserAuto, false},
		{"chrome", BrowserChrome, false},
		{"Chrome", BrowserChrome, false},
		{"CHROME", BrowserChrome, false},
		{"google-chrome", BrowserChrome, false},
		{"chromium", BrowserChromium, false},
		{"firefox", BrowserFirefox, false},
		{"Firefox", BrowserFirefox, false},
		{"mozilla", BrowserFirefox, false},
		{"mozilla-firefox", BrowserFirefox, false},
		{"edge", BrowserEdge, false},
		{"microsoft-edge", BrowserEdge, false},
		{"msedge", BrowserEdge, false},
		{"opera", BrowserOpera, false},
		{"invalid", "", true},
		{"safari", "", true}, // Not supported
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseBrowser(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseBrowser(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParseBrowser(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParseBrowser(%q) = %v, want %v", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestSupportedBrowserString(t *testing.T) {
	tests := []struct {
		browser  SupportedBrowser
		expected string
	}{
		{BrowserAuto, "auto"},
		{BrowserChrome, "chrome"},
		{BrowserChromium, "chromium"},
		{BrowserFirefox, "firefox"},
		{BrowserEdge, "edge"},
		{BrowserOpera, "opera"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if result := tt.browser.String(); result != tt.expected {
				t.Errorf("SupportedBrowser.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAllSupportedBrowsers(t *testing.T) {
	browsers := AllSupportedBrowsers()

	if len(browsers) == 0 {
		t.Error("AllSupportedBrowsers() returned empty slice")
	}

	// Check that all expected browsers are present
	expected := map[SupportedBrowser]bool{
		BrowserChrome:   true,
		BrowserChromium: true,
		BrowserFirefox:  true,
		BrowserEdge:     true,
		BrowserOpera:    true,
	}

	for _, browser := range browsers {
		if !expected[browser] {
			t.Errorf("Unexpected browser in AllSupportedBrowsers(): %v", browser)
		}
		delete(expected, browser)
	}

	if len(expected) > 0 {
		t.Errorf("Missing browsers in AllSupportedBrowsers(): %v", expected)
	}
}

func TestMatchesBrowser(t *testing.T) {
	tests := []struct {
		browserName string
		target      SupportedBrowser
		expected    bool
	}{
		{"chrome", BrowserChrome, true},
		{"Google Chrome", BrowserChrome, true},
		{"chromium", BrowserChrome, false}, // chromium should not match chrome
		{"chromium", BrowserChromium, true},
		{"Chromium", BrowserChromium, true},
		{"firefox", BrowserFirefox, true},
		{"Firefox", BrowserFirefox, true},
		{"Mozilla Firefox", BrowserFirefox, true},
		{"edge", BrowserEdge, true},
		{"Microsoft Edge", BrowserEdge, true},
		{"opera", BrowserOpera, true},
		{"Opera", BrowserOpera, true},
		{"safari", BrowserChrome, false},
		{"", BrowserChrome, false},
	}

	for _, tt := range tests {
		t.Run(tt.browserName+"_"+tt.target.String(), func(t *testing.T) {
			result := matchesBrowser(tt.browserName, tt.target)
			if result != tt.expected {
				t.Errorf("matchesBrowser(%q, %v) = %v, want %v", tt.browserName, tt.target, result, tt.expected)
			}
		})
	}
}

func TestListAvailableBrowsers(t *testing.T) {
	// This test just ensures the function doesn't panic
	// The actual result depends on the system's installed browsers
	browsers := ListAvailableBrowsers()
	t.Logf("Found %d browsers: %v", len(browsers), browsers)
}

func TestExtractGeminiCookies_InvalidBrowser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with a browser that likely doesn't exist
	_, err := ExtractGeminiCookies(ctx, "nonexistent")
	if err == nil {
		t.Error("ExtractGeminiCookies with nonexistent browser should return error")
	}
}

func TestExtractGeminiCookies_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := ExtractGeminiCookies(ctx, BrowserChrome)
	// The function should handle the cancelled context gracefully
	// It may or may not return an error depending on timing
	t.Logf("Result with cancelled context: %v", err)
}

func TestExtractFromAllBrowsers(t *testing.T) {
	ctx := context.Background()

	// Test extractFromAllBrowsers - may succeed or fail depending on environment
	result, err := extractFromAllBrowsers(ctx)
	if err != nil {
		// Expected on systems without Gemini cookies
		t.Logf("extractFromAllBrowsers returned error (expected on systems without Gemini login): %v", err)
	} else if result != nil {
		// Found cookies - verify result structure
		if result.Cookies == nil {
			t.Error("extractFromAllBrowsers returned non-nil result with nil Cookies")
		}
		if result.Cookies.Secure1PSID == "" {
			t.Error("extractFromAllBrowsers returned empty Secure1PSID")
		}
		t.Logf("Found cookies from: %s", result.BrowserName)
	}
}

func TestExtractCookiesFromStore(t *testing.T) {
	ctx := context.Background()

	// Create a mock cookie store for testing
	// Since we can't easily create a real store, we'll test the error paths
	stores := kooky.FindAllCookieStores(ctx)
	if len(stores) == 0 {
		t.Skip("No cookie stores available for testing")
	}

	defer func() {
		for _, store := range stores {
			_ = store.Close()
		}
	}()

	// Test with first available store only
	if len(stores) > 0 {
		store := stores[0]
		result, err := extractCookiesFromStore(ctx, store, store.Browser(), store.Profile())
		// We expect either success (if Gemini cookies exist) or specific error
		if err != nil {
			// Check that error message is descriptive
			if !strings.Contains(err.Error(), "__Secure-1PSID") {
				t.Errorf("Expected error to mention __Secure-1PSID cookie, got: %v", err)
			}
		} else if result != nil {
			// Verify result structure if successful
			if result.Cookies == nil {
				t.Error("extractCookiesFromStore returned nil Cookies in result")
			}
			if result.BrowserName == "" {
				t.Error("extractCookiesFromStore returned empty BrowserName in result")
			}
		}
	}
}
