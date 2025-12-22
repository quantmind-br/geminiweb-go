package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestExecuteWrapperSuccess(t *testing.T) {
	old := rootCmd
	rootCmd = &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	defer func() { rootCmd = old }()

	// Should not call os.Exit for successful execution
	Execute()
}
