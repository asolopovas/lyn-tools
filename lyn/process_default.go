//go:build !windows

package lyn

import (
	"os"
	"os/exec"
	"syscall"
)

func hideConsoleWindow(*exec.Cmd) {}

func processRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
