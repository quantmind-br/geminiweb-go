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
