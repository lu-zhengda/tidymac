package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/lu-zhengda/tidymac/internal/engine"
	"github.com/lu-zhengda/tidymac/internal/scanner"
	"github.com/lu-zhengda/tidymac/internal/tui"
	"github.com/lu-zhengda/tidymac/internal/utils"
)

var (
	yoloMode bool

	// Set via ldflags at build time.
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "tidymac",
	Short:   "A lightweight macOS cleanup tool",
	Long:    "tidymac scans and cleans system junk, browser caches, Xcode artifacts, and more.\nLaunch without subcommands for interactive TUI mode.",
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
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
	rootCmd.SetVersionTemplate(fmt.Sprintf("tidymac %s\n", version))
	rootCmd.PersistentFlags().BoolVar(&yoloMode, "yolo", false, "Skip ALL confirmation prompts (dangerous!)")
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(maintainCmd)
	rootCmd.AddCommand(spacelensCmd)
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
	e := engine.New()
	e.Register(scanner.NewSystemScanner(""))
	e.Register(scanner.NewBrowserScanner("", ""))
	e.Register(scanner.NewXcodeScanner(""))

	home := utils.HomeDir()
	e.Register(scanner.NewLargeFileScanner(
		[]string{
			filepath.Join(home, "Downloads"),
			filepath.Join(home, "Desktop"),
		},
		100*1024*1024,
		90*24*time.Hour,
	))

	return e
}

func selectedCategories(system, browser, xcode, large bool) []string {
	if !system && !browser && !xcode && !large {
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
