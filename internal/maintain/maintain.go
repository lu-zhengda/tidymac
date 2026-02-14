package maintain

import (
	"os/exec"
)

type Task struct {
	Name        string
	Description string
	Command     string
	Args        []string
	NeedsSudo   bool
}

type Result struct {
	Task    Task
	Output  string
	Success bool
	Error   error
}

func AvailableTasks() []Task {
	return []Task{
		{Name: "Flush DNS Cache", Description: "Clear the DNS resolver cache", Command: "dscacheutil", Args: []string{"-flushcache"}, NeedsSudo: false},
		{Name: "Kill DNS Responder", Description: "Restart mDNSResponder to apply DNS flush", Command: "sudo", Args: []string{"killall", "-HUP", "mDNSResponder"}, NeedsSudo: true},
		{Name: "Rebuild Spotlight Index", Description: "Re-index Spotlight for faster search", Command: "sudo", Args: []string{"mdutil", "-E", "/"}, NeedsSudo: true},
		{Name: "Purge Inactive Memory", Description: "Free up inactive memory", Command: "sudo", Args: []string{"purge"}, NeedsSudo: true},
		{Name: "Rebuild Launch Services", Description: "Fix duplicate entries in Open With menus", Command: "/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister", Args: []string{"-kill", "-r", "-domain", "local", "-domain", "system", "-domain", "user"}, NeedsSudo: false},
	}
}

func Run(task Task) Result {
	cmd := exec.Command(task.Command, task.Args...)
	output, err := cmd.CombinedOutput()
	return Result{Task: task, Output: string(output), Success: err == nil, Error: err}
}

func RunAll() []Result {
	tasks := AvailableTasks()
	results := make([]Result, 0, len(tasks))
	for _, task := range tasks {
		results = append(results, Run(task))
	}
	return results
}
