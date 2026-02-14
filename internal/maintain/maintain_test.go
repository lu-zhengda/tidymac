package maintain

import "testing"

func TestTaskList(t *testing.T) {
	tasks := AvailableTasks()
	if len(tasks) == 0 {
		t.Fatal("expected at least one maintenance task")
	}
	for _, task := range tasks {
		if task.Name == "" {
			t.Error("task name should not be empty")
		}
		if task.Command == "" {
			t.Error("task command should not be empty")
		}
	}
}
