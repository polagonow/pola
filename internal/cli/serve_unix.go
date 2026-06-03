//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts the child in its own process group so the whole group
// (the `go run` process and any child it spawns) can be signalled together on
// restart.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup sends SIGTERM to the entire process group. The negative PID
// targets the group rather than just the `go run` process.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}
