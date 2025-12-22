package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/diogo/geminiweb/internal/browser"
	"github.com/diogo/geminiweb/internal/config"
)

type mockBrowserExtractor struct {
	result *browser.ExtractResult
	err    error
}

func (m *mockBrowserExtractor) ExtractGeminiCookies(ctx context.Context, b browser.SupportedBrowser) (*browser.ExtractResult, error) {
	return m.result, m.err
}

func TestRunAutoLogin_Table(t *testing.T) {
	tests := []struct {
		name        string
		browserName string
		mockRes     *browser.ExtractResult
		mockErr     error
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "success chrome",
			browserName: "chrome",
			mockRes: &browser.ExtractResult{
				BrowserName: "chrome",
				Cookies: &config.Cookies{
					Secure1PSID:   "psid-12345678901234567890",
					Secure1PSIDTS: "psidts-12345678901234567890",
				},
			},
			wantErr: false,
		},
		{
			name:        "invalid browser",
			browserName: "netscape",
			wantErr:     true,
			errMsg:      "unsupported browser",
		},
		{
			name:        "extraction error",
			browserName: "firefox",
			mockErr:     fmt.Errorf("db locked"),
			wantErr:     true,
			errMsg:      "failed to extract cookies: db locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := &Dependencies{
				BrowserExtractor: &mockBrowserExtractor{
					result: tt.mockRes,
					err:    tt.mockErr,
				},
			}

			// Mock HOME for config saving
			tmpDir := t.TempDir()
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", tmpDir)
			defer os.Setenv("HOME", oldHome)

			err := runAutoLogin(deps, tt.browserName)

			if (err != nil) != tt.wantErr {
				t.Errorf("runAutoLogin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("runAutoLogin() error = %v, errMsg %v", err, tt.errMsg)
			}
		})
	}
}
