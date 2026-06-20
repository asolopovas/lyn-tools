package lyn

import (
	"runtime"
	"strings"

	"lyn.tools/launcher/lyn/launch"
	"lyn.tools/launcher/lyn/sysadmin"
)

const (
	systemCommandRestart  = "lyn:system:restart"
	systemCommandShutdown = "lyn:system:shutdown"
	systemCommandLogout   = "lyn:system:logout"
)

func systemCommands() []Project {
	return systemCommandsFor(runtime.GOOS)
}

func systemCommandsFor(goos string) []Project {
	defs := []struct {
		name string
		path string
	}{
		{"Restart", systemCommandRestart},
		{"Shut Down", systemCommandShutdown},
		{"Log Out", systemCommandLogout},
	}
	commands := make([]Project, 0, len(defs))
	for _, def := range defs {
		commands = append(commands, newProject(def.name, def.path, projectKindSystemCommand))
	}
	if goos == "linux" {
		for _, tool := range sysadmin.Tools() {
			commands = append(commands, newProject(tool.Name, launch.AdminToolPrefix+tool.Key, projectKindSystemCommand))
		}
	}
	return commands
}

func isSystemCommandPath(path string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(path)), "lyn:system:")
}

func (a *App) installSystemToolsScript() {
	if runtime.GOOS != "linux" {
		return
	}
	path, err := sysadmin.EnsureScript(a.config.Cache.Dir)
	if err != nil {
		a.debugLog("sysadmin.script.error", "error", err)
		return
	}
	launch.SetAdminScriptPath(path)
	a.debugLog("sysadmin.script.ready", "path", path)
}
