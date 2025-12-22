package commands

import (
	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/tui"
)

// NewConfigCmd creates a new config command
func NewConfigCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Open configuration menu",
		Long:  `Interactive menu to configure geminiweb settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunConfig()
		},
	}
}

// Backward compatibility global
var configCmd = NewConfigCmd(nil)