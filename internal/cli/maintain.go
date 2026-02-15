package cli

import (
	"fmt"

	"github.com/lu-zhengda/macbroom/internal/maintain"
	"github.com/spf13/cobra"
)

var maintainYes bool

var maintainCmd = &cobra.Command{
	Use:   "maintain",
	Short: "Run system maintenance tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		tasks := maintain.AvailableTasks()
		fmt.Printf("Available maintenance tasks (%d):\n\n", len(tasks))
		for i, task := range tasks {
			sudo := ""
			if task.NeedsSudo {
				sudo = " (requires sudo)"
			}
			fmt.Printf("  %d. %s%s\n     %s\n\n", i+1, task.Name, sudo, task.Description)
		}

		printYoloWarning()

		if !shouldSkipConfirm(maintainYes) {
			if !confirmAction("Run all maintenance tasks?") {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		fmt.Println()
		results := maintain.RunAll()
		for _, r := range results {
			status := "OK"
			if !r.Success {
				status = fmt.Sprintf("FAILED: %v", r.Error)
			}
			fmt.Printf("  [%s] %s\n", status, r.Task.Name)
		}

		return nil
	},
}

func init() {
	maintainCmd.Flags().BoolVarP(&maintainYes, "yes", "y", false, "Skip confirmation prompt")
}
