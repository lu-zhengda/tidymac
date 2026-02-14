package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var spacelensCmd = &cobra.Command{
	Use:   "spacelens [path]",
	Short: "Visualize disk space usage",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "/"
		if len(args) > 0 {
			path = args[0]
		}
		fmt.Printf("Analyzing %s... (not yet implemented)\n", path)
		return nil
	},
}
