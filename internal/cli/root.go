package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tidymac",
	Short: "A lightweight macOS cleanup tool",
	Long:  "tidymac scans and cleans system junk, browser caches, Xcode artifacts, and more.\nLaunch without subcommands for interactive TUI mode.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(maintainCmd)
	rootCmd.AddCommand(spacelensCmd)
}
