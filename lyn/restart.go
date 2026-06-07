package lyn

import (
	"os"
	"os/exec"
)

var startRestartProcess = startCurrentExecutable

func restartArgs(args []string) []string {
	result := make([]string, 0, len(args)+1)
	hasStartHidden := false
	for _, arg := range args {
		if arg == "--start-hidden" {
			hasStartHidden = true
		}
		result = append(result, arg)
	}
	if !hasStartHidden {
		result = append(result, "--start-hidden")
	}
	return result
}

func startCurrentExecutable(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	process := exec.Command(exe, restartArgs(args)...)
	if err := process.Start(); err != nil {
		return err
	}
	return process.Process.Release()
}
