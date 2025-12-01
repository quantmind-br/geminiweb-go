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
