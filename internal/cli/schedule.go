package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/lu-zhengda/macbroom/internal/schedule"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage scheduled automatic cleaning",
	Long:  "Enable, disable, or check the status of scheduled automatic cleaning via macOS LaunchAgent.",
}

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable scheduled cleaning",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := schedule.DefaultPath()
		timeStr := appConfig.Schedule.Time
		interval := appConfig.Schedule.Interval

		fmt.Printf("Installing LaunchAgent for %s cleanup at %s...\n", interval, timeStr)

		if err := schedule.Install(path, timeStr, interval); err != nil {
			return fmt.Errorf("failed to install schedule: %w", err)
		}

		// Load the LaunchAgent via launchctl bootstrap.
		uid := currentUID()
		loadCmd := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%s", uid), path)
		if out, err := loadCmd.CombinedOutput(); err != nil {
			// If already loaded, try bootout first then bootstrap again.
			if strings.Contains(string(out), "already loaded") || strings.Contains(string(out), "service already loaded") {
				unloadCmd := exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%s", uid), path)
				_ = unloadCmd.Run()
				loadCmd = exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%s", uid), path)
				if _, err := loadCmd.CombinedOutput(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: plist written but launchctl bootstrap failed: %v\n", err)
					fmt.Println("You may need to run: launchctl bootstrap gui/" + uid + " " + path)
					return nil
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: plist written but launchctl bootstrap failed: %v\n", err)
				fmt.Println("You may need to run: launchctl bootstrap gui/" + uid + " " + path)
				return nil
			}
		}

		fmt.Println("Scheduled cleaning enabled.")
		fmt.Printf("  Schedule: %s at %s\n", interval, timeStr)
		fmt.Printf("  Plist:    %s\n", path)
		return nil
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable scheduled cleaning",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := schedule.DefaultPath()

		if !schedule.Status(path) {
			fmt.Println("Scheduled cleaning is not currently enabled.")
			return nil
		}

		// Unload the LaunchAgent via launchctl bootout.
		uid := currentUID()
		unloadCmd := exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%s", uid), path)
		if out, err := unloadCmd.CombinedOutput(); err != nil {
			// Not fatal — the plist may not be loaded.
			if !strings.Contains(string(out), "not find") && !strings.Contains(string(out), "No such") {
				fmt.Fprintf(os.Stderr, "Warning: launchctl bootout failed: %v\n", err)
			}
		}

		if err := schedule.Uninstall(path); err != nil {
			return fmt.Errorf("failed to remove schedule: %w", err)
		}

		fmt.Println("Scheduled cleaning disabled.")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show scheduled cleaning status",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := schedule.DefaultPath()

		if schedule.Status(path) {
			fmt.Println("Scheduled cleaning: enabled")
			fmt.Printf("  Schedule: %s at %s\n", appConfig.Schedule.Interval, appConfig.Schedule.Time)
			fmt.Printf("  Notify:   %v\n", appConfig.Schedule.Notify)
			fmt.Printf("  Plist:    %s\n", path)
		} else {
			fmt.Println("Scheduled cleaning: disabled")
			fmt.Println("  Run 'macbroom schedule enable' to set up automatic cleaning.")
		}
		return nil
	},
}

// currentUID returns the current user's UID as a string.
func currentUID() string {
	return strconv.Itoa(os.Getuid())
}

func init() {
	scheduleCmd.AddCommand(enableCmd)
	scheduleCmd.AddCommand(disableCmd)
	scheduleCmd.AddCommand(statusCmd)
}
