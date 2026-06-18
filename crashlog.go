package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

var crashLogFile *os.File

func setupCrashLog(dir string) {
	if dir == "" || crashLogFile != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(dir, "crash.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	crashLogFile = file
	fmt.Fprintf(file, "=== start %s pid=%d ===\n", time.Now().Format(time.RFC3339), os.Getpid())
	debug.SetCrashOutput(file, debug.CrashOptions{})
}
