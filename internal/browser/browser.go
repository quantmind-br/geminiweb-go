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
