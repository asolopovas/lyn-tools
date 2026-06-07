package lyn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type DebugLogger struct {
	mu      sync.Mutex
	enabled bool
	path    string
	file    *os.File
}

func NewDebugLogger(args []string) *DebugLogger {
	enabled := debugEnvEnabled()
	path := strings.TrimSpace(os.Getenv("LYN_DEBUG_LOG"))
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "--debug":
			enabled = true
		case arg == "--debug-log" && i+1 < len(args):
			enabled = true
			path = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--debug-log="):
			enabled = true
			path = strings.TrimSpace(strings.TrimPrefix(arg, "--debug-log="))
		}
	}
	logger := &DebugLogger{enabled: enabled}
	if !enabled {
		return logger
	}
	if path == "" {
		path = filepath.Join(userCacheDir(), "lyn", "debug.log")
	}
	logger.path = path
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return logger
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return logger
	}
	logger.file = file
	logger.Log("debug.start", "path", path)
	return logger
}

func debugEnvEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LYN_DEBUG"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (d *DebugLogger) Enabled() bool {
	return d != nil && d.enabled && d.file != nil
}

func (d *DebugLogger) Path() string {
	if d == nil {
		return ""
	}
	return d.path
}

func (d *DebugLogger) Log(stage string, values ...any) {
	if !d.Enabled() {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	var line strings.Builder
	line.WriteString(time.Now().Format(time.RFC3339Nano))
	line.WriteByte(' ')
	line.WriteString(stage)
	for i := 0; i+1 < len(values); i += 2 {
		line.WriteByte(' ')
		fmt.Fprint(&line, values[i])
		line.WriteByte('=')
		fmt.Fprint(&line, formatDebugValue(values[i+1]))
	}
	line.WriteByte('\n')
	_, _ = d.file.WriteString(line.String())
}

func (d *DebugLogger) Close() {
	if d == nil || d.file == nil {
		return
	}
	d.Log("debug.stop")
	d.mu.Lock()
	defer d.mu.Unlock()
	_ = d.file.Close()
	d.file = nil
}

func formatDebugValue(value any) string {
	text := fmt.Sprint(value)
	text = strings.ReplaceAll(text, "\r", "\\r")
	text = strings.ReplaceAll(text, "\n", "\\n")
	if strings.ContainsAny(text, " \t=") {
		return fmt.Sprintf("%q", text)
	}
	return text
}
