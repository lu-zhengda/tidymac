package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [app-name]",
	Short: "Completely uninstall an application",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Uninstalling %s... (not yet implemented)\n", args[0])
		return nil
	},
}
