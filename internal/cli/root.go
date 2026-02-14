package cli

import (
	"context"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/zhengda-lu/macbroom/internal/engine"
	"github.com/zhengda-lu/macbroom/internal/scanner"
	"github.com/zhengda-lu/macbroom/internal/tui"
	"github.com/zhengda-lu/macbroom/internal/utils"
)

var rootCmd = &cobra.Command{
	Use:   "macbroom",
	Short: "A lightweight macOS cleanup tool",
	Long:  "macbroom scans and cleans system junk, browser caches, Xcode artifacts, and more.\nLaunch without subcommands for interactive TUI mode.",
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
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(maintainCmd)
	rootCmd.AddCommand(spacelensCmd)
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
