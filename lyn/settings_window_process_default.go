//go:build !windows

package lyn

import (
	"os"
	"syscall"
)

func settingsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
