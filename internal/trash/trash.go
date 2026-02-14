package trash

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func MoveToTrash(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %w", path, err)
	}

	script := fmt.Sprintf(
		`tell application "Finder" to delete POSIX file %q`,
		absPath,
	)

	cmd := exec.Command("osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to trash %s: %w (%s)", path, err, string(out))
	}
	return nil
}

func PermanentDelete(path string) error {
	return os.RemoveAll(path)
}
