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
