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
  geminiweb "Hello" -o response.md      Save response to file`,
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

		// Check for file input
		if fileFlag != "" {
			data, err := os.ReadFile(fileFlag)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			return runQuery(string(data))
		}

		// Check for stdin
		if hasStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			return runQuery(string(data))
		}

		// Check for positional argument
		if len(args) > 0 {
			return runQuery(args[0])
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
	rootCmd.Flags().BoolP("version", "v", false, "Show version and exit")

	// Add subcommands
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(importCookiesCmd)
	rootCmd.AddCommand(autoLoginCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(personaCmd)
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
