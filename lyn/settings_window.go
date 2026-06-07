package lyn

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type WindowMode string

const (
	LauncherWindowMode WindowMode = "launcher"
	SettingsWindowMode WindowMode = "settings"
)

func (a *App) SetWindowMode(mode WindowMode) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if mode == SettingsWindowMode {
		a.mode = mode
		return
	}
	a.mode = LauncherWindowMode
}

func (a *App) WindowMode() string {
	return string(a.windowMode())
}

func (a *App) windowMode() WindowMode {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if a.mode == SettingsWindowMode {
		return SettingsWindowMode
	}
	return LauncherWindowMode
}

func (a *App) OpenSettingsWindow() error {
	if a.settingsWindowActive() {
		return nil
	}
	executable, err := settingsWindowExecutable()
	if err != nil {
		return err
	}
	return startSettingsWindowProcess(executable, settingsWindowArgs(os.Args[1:]))
}

func (a *App) CloseSettingsWindow() {
	ctx, _, _ := a.snapshot()
	if ctx != nil {
		runtime.Quit(ctx)
	}
}

var settingsWindowExecutable = os.Executable

var startSettingsWindowProcess = func(executable string, args []string) error {
	command := exec.Command(executable, args...)
	return command.Start()
}

func (a *App) markSettingsWindowActive() {
	path := a.settingsActivityPath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		a.debugLog("settings.activity.error", "error", err)
		return
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		a.debugLog("settings.activity.error", "error", err)
	}
}

func (a *App) clearSettingsWindowActive() {
	path := a.settingsActivityPath()
	if path != "" {
		_ = os.Remove(path)
	}
}

func (a *App) settingsWindowActive() bool {
	path := a.settingsActivityPath()
	if path == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || !settingsProcessRunning(pid) {
		_ = os.Remove(path)
		return false
	}
	return true
}

func (a *App) settingsActivityPath() string {
	_, config, _ := a.snapshot()
	if config.Cache.Dir == "" {
		return ""
	}
	return filepath.Join(config.Cache.Dir, "settings.active")
}

func settingsWindowArgs(args []string) []string {
	filtered := make([]string, 0, len(args)+1)
	for _, arg := range args {
		if arg == "--start-hidden" || arg == "--settings-window" {
			continue
		}
		filtered = append(filtered, arg)
	}
	filtered = append(filtered, "--settings-window")
	return filtered
}
