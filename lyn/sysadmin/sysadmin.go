package sysadmin

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
)

//go:embed lyn-sysadmin.sh
var scriptFS embed.FS

const (
	scriptSource = "lyn-sysadmin.sh"
	scriptName   = "lyn-sysadmin"
)

type Tool struct {
	Key  string
	Name string
}

func Tools() []Tool {
	return []Tool{
		{Key: "monitor", Name: "System Monitor"},
		{Key: "logs", Name: "System Logs"},
		{Key: "services", Name: "Services"},
		{Key: "network", Name: "Network"},
		{Key: "firewall", Name: "Firewall"},
		{Key: "disk", Name: "Disk Usage"},
	}
}

func EnsureScript(dir string) (string, error) {
	data, err := scriptFS.ReadFile(scriptSource)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, scriptName)
	if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, data) {
		return path, nil
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		return "", err
	}
	return path, nil
}
