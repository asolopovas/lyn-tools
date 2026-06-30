package lyn

import (
	"os"
	"os/exec"
	"slices"
	"strconv"
	"time"
)

const awaitExitFlag = "--await-exit"

const (
	priorInstanceWait = 10 * time.Second
	priorInstancePoll = 50 * time.Millisecond
)

var startRestartProcess = startCurrentExecutable

func restartArgs(args []string) []string {
	result := stripAwaitExit(args)
	if !slices.Contains(result, "--start-hidden") {
		result = append(result, "--start-hidden")
	}
	return result
}

func startCurrentExecutable(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	spawnArgs := append([]string{awaitExitFlag, strconv.Itoa(os.Getpid())}, restartArgs(args)...)
	process := exec.Command(exe, spawnArgs...)
	if err := process.Start(); err != nil {
		return err
	}
	return process.Process.Release()
}

func stripAwaitExit(args []string) []string {
	result := make([]string, 0, len(args))
	skip := false
	for i, arg := range args {
		if skip {
			skip = false
			continue
		}
		if arg == awaitExitFlag {
			skip = i+1 < len(args)
			continue
		}
		result = append(result, arg)
	}
	return result
}

func awaitExitPID(args []string) (int, bool) {
	for i, arg := range args {
		if arg != awaitExitFlag {
			continue
		}
		if i+1 >= len(args) {
			return 0, false
		}
		pid, err := strconv.Atoi(args[i+1])
		if err != nil || pid <= 0 {
			return 0, false
		}
		return pid, true
	}
	return 0, false
}

func WaitForPriorInstance(args []string) {
	pid, ok := awaitExitPID(args)
	if !ok {
		return
	}
	deadline := time.Now().Add(priorInstanceWait)
	for time.Now().Before(deadline) {
		if !processRunning(pid) {
			return
		}
		time.Sleep(priorInstancePoll)
	}
}
