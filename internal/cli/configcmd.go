package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config management",
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := configPath
		if cfgPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to determine home directory: %w", err)
			}
			cfgPath = filepath.Join(home, ".config", "macbroom", "config.yaml")
		}

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to read config file %q: %w", cfgPath, err)
		}

		_, warnings := config.LoadAndValidate(data)

		if len(warnings) == 0 {
			fmt.Printf("Config OK (%s)\n", cfgPath)
			return nil
		}

		fmt.Printf("Found %d warning(s) in %s:\n", len(warnings), cfgPath)
		for _, w := range warnings {
			if w.Field != "" {
				fmt.Printf("  [%s] %s\n", w.Field, w.Message)
			} else {
				fmt.Printf("  %s\n", w.Message)
			}
			if w.Suggestion != "" {
				fmt.Printf("    suggestion: %s\n", w.Suggestion)
			}
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)
}
