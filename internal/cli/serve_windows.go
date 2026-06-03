//go:build windows

package cli

import (
	"os/exec"
	"strconv"
	"syscall"
)

// setProcessGroup starts the child in a new process group so it and its
// descendants can be terminated together on restart.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// killProcessGroup terminates the child process tree. Windows has no
// process-group signalling like Unix, so taskkill force-kills the `go run`
// process and every process it spawned (/T = tree, /F = force).
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
