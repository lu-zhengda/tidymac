package cli

import (
	"fmt"
	"strings"

	"github.com/zhengda-lu/macbroom/internal/scanner"
	"github.com/zhengda-lu/macbroom/internal/utils"
)

func printScanResults(targets []scanner.Target) {
	if len(targets) == 0 {
		fmt.Println("No junk files found.")
		return
	}

	grouped := make(map[string][]scanner.Target)
	for _, t := range targets {
		grouped[t.Category] = append(grouped[t.Category], t)
	}

	var totalSize int64
	for category, items := range grouped {
		var catSize int64
		for _, item := range items {
			catSize += item.Size
		}
		totalSize += catSize

		fmt.Printf("\n%s (%s, %d items)\n", category, utils.FormatSize(catSize), len(items))
		fmt.Println(strings.Repeat("-", 60))

		for _, item := range items {
			risk := ""
			if item.Risk >= scanner.Moderate {
				risk = fmt.Sprintf(" [%s]", item.Risk)
			}
			fmt.Printf("  %-40s %10s%s\n", truncatePath(item.Path, 40), utils.FormatSize(item.Size), risk)
		}
	}

	fmt.Printf("\nTotal reclaimable: %s\n", utils.FormatSize(totalSize))
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

func confirmAction(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}
