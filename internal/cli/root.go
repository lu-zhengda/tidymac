package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lu-zhengda/macbroom/internal/config"
	"github.com/lu-zhengda/macbroom/internal/engine"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/tui"
	"github.com/lu-zhengda/macbroom/internal/utils"
	"github.com/spf13/cobra"
)

var (
	yoloMode   bool
	configPath string
	appConfig  *config.Config

	// Set via ldflags at build time.
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "macbroom",
	Short:   "A lightweight macOS cleanup tool",
	Long:    "macbroom scans and cleans system junk, browser caches, Xcode artifacts, and more.\nLaunch without subcommands for interactive TUI mode.",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "help" || cmd.Flags().Changed("version") {
			appConfig = config.Default()
			return nil
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		appConfig = cfg
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if shell, _ := cmd.Flags().GetString("generate-completion"); shell != "" {
			switch shell {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			default:
				return fmt.Errorf("unsupported shell: %s (use bash, zsh, or fish)", shell)
			}
		}
		e := buildEngine()
		p := tea.NewProgram(tui.New(e), tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("macbroom %s\n", version))
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolVar(&yoloMode, "yolo", false, "Skip ALL confirmation prompts (dangerous!)")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default ~/.config/macbroom/config.yaml)")
	rootCmd.Flags().String("generate-completion", "", "Generate shell completion (bash, zsh, fish)")
	rootCmd.Flags().MarkHidden("generate-completion")
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(maintainCmd)
	rootCmd.AddCommand(spacelensCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(dupesCmd)
	rootCmd.AddCommand(scheduleCmd)
}

// shouldSkipConfirm returns true if the user wants to skip confirmation,
// either via command-specific --yes or global --yolo.
func shouldSkipConfirm(cmdYes bool) bool {
	return cmdYes || yoloMode
}

// printYoloWarning prints a warning banner when --yolo mode is active.
func printYoloWarning() {
	if yoloMode {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  WARNING: --yolo mode is active. All confirmations will be skipped!")
		fmt.Fprintln(os.Stderr, "  Files will be deleted without asking. Press Ctrl+C NOW to abort.")
		fmt.Fprintln(os.Stderr, "")
	}
}

func buildEngine() *engine.Engine {
	if appConfig == nil {
		appConfig = config.Default()
	}

	e := engine.New()

	if appConfig.Scanners.System {
		e.Register(scanner.NewSystemScanner(""))
	}
	if appConfig.Scanners.Browser {
		e.Register(scanner.NewBrowserScanner("", ""))
	}
	if appConfig.Scanners.Xcode {
		e.Register(scanner.NewXcodeScanner(""))
	}
	if appConfig.Scanners.LargeFiles {
		paths := expandPaths(appConfig.LargeFiles.Paths)
		minAge := config.ParseDuration(appConfig.LargeFiles.MinAge)
		e.Register(scanner.NewLargeFileScanner(paths, appConfig.LargeFiles.MinSize, minAge))
	}
	if appConfig.Scanners.Docker {
		e.Register(scanner.NewDockerScanner())
	}
	if appConfig.Scanners.Node {
		home := utils.HomeDir()
		paths := expandPaths(appConfig.DevTools.SearchPaths)
		minAge := config.ParseDuration(appConfig.DevTools.MinAge)
		e.Register(scanner.NewNodeScanner(home, paths, minAge))
	}
	if appConfig.Scanners.Homebrew {
		e.Register(scanner.NewHomebrewScanner())
	}
	if appConfig.Scanners.IOSSimulators {
		e.Register(scanner.NewSimulatorScanner(""))
	}
	if appConfig.Scanners.Python {
		home := utils.HomeDir()
		paths := expandPaths(appConfig.DevTools.SearchPaths)
		minAge := config.ParseDuration(appConfig.DevTools.MinAge)
		e.Register(scanner.NewPythonScanner(home, paths, minAge))
	}
	if appConfig.Scanners.Rust {
		home := utils.HomeDir()
		paths := expandPaths(appConfig.DevTools.SearchPaths)
		minAge := config.ParseDuration(appConfig.DevTools.MinAge)
		e.Register(scanner.NewRustScanner(home, paths, minAge))
	}
	if appConfig.Scanners.Go {
		home := utils.HomeDir()
		e.Register(scanner.NewGoScanner(home))
	}
	if appConfig.Scanners.JetBrains {
		home := utils.HomeDir()
		e.Register(scanner.NewJetBrainsScanner(home))
	}

	return e
}

func selectedCategories(system, browser, xcode, large, docker, node, homebrew, simulator, python, rust, golang, jetbrains bool) []string {
	if !system && !browser && !xcode && !large && !docker && !node && !homebrew && !simulator && !python && !rust && !golang && !jetbrains {
		return nil
	}
	var cats []string
	if system {
		cats = append(cats, "System Junk")
	}
	if browser {
		cats = append(cats, "Browser Cache")
	}
	if xcode {
		cats = append(cats, "Xcode Junk")
	}
	if large {
		cats = append(cats, "Large & Old Files")
	}
	if docker {
		cats = append(cats, "Docker")
	}
	if node {
		cats = append(cats, "Node.js")
	}
	if homebrew {
		cats = append(cats, "Homebrew")
	}
	if simulator {
		cats = append(cats, "iOS Simulators")
	}
	if python {
		cats = append(cats, "Python")
	}
	if rust {
		cats = append(cats, "Rust")
	}
	if golang {
		cats = append(cats, "Go")
	}
	if jetbrains {
		cats = append(cats, "JetBrains")
	}
	return cats
}

func scanWithCategories(e *engine.Engine, cats []string) ([]scanner.Target, error) {
	ctx := context.Background()

	if cats == nil {
		return e.ScanAll(ctx)
	}

	var all []scanner.Target
	for _, cat := range cats {
		targets, err := e.ScanByCategory(ctx, cat)
		if err != nil {
			continue
		}
		all = append(all, targets...)
	}
	return all, nil
}

// RootCmd returns the root cobra command for documentation generation.
func RootCmd() *cobra.Command {
	return rootCmd
}

// expandPaths expands ~ to the user's home directory in each path.
func expandPaths(paths []string) []string {
	home := utils.HomeDir()
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if strings.HasPrefix(p, "~/") {
			p = filepath.Join(home, p[2:])
		} else if p == "~" {
			p = home
		}
		result = append(result, p)
	}
	return result
}
