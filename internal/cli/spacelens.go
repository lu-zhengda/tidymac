package cli

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/tui"
	"github.com/lu-zhengda/macbroom/internal/utils"
	"github.com/spf13/cobra"
)

var (
	spacelensDepth       int
	spacelensInteractive bool
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

		if spacelensInteractive {
			p := tea.NewProgram(tui.NewSpaceLensModel(path), tea.WithAltScreen())
			_, err := p.Run()
			return err
		}

		fmt.Printf("Analyzing %s...\n\n", path)
		sl := scanner.NewSpaceLens(path, spacelensDepth)
		nodes, err := sl.Analyze(context.Background())
		if err != nil {
			return fmt.Errorf("failed to analyze: %w", err)
		}

		printSpaceLensNodes(nodes, 0)
		return nil
	},
}

func init() {
	spacelensCmd.Flags().IntVar(&spacelensDepth, "depth", 2, "Maximum directory depth to analyze")
	spacelensCmd.Flags().BoolVarP(&spacelensInteractive, "interactive", "i", false, "Launch interactive TUI mode")
}

func printSpaceLensNodes(nodes []scanner.SpaceLensNode, indent int) {
	if len(nodes) == 0 {
		return
	}
	for _, node := range nodes {
		prefix := strings.Repeat("  ", indent)
		icon := "  "
		if node.IsDir {
			icon = "D "
		}
		bar := sizeBar(node.Size, nodes[0].Size)
		fmt.Printf("%s%s %-40s %10s %s\n", prefix, icon, node.Name, utils.FormatSize(node.Size), bar)

		if len(node.Children) > 0 {
			printSpaceLensNodes(node.Children, indent+1)
		}
	}
}

func sizeBar(size, maxSize int64) string {
	if maxSize == 0 {
		return ""
	}
	const maxBarLen = 30
	ratio := float64(size) / float64(maxSize)
	barLen := int(ratio * maxBarLen)
	if barLen == 0 && size > 0 {
		barLen = 1
	}
	return "[" + strings.Repeat("#", barLen) + strings.Repeat(".", maxBarLen-barLen) + "]"
}
