package lyn

import "strings"

const (
	systemCommandRestart  = "lyn:system:restart"
	systemCommandShutdown = "lyn:system:shutdown"
	systemCommandLogout   = "lyn:system:logout"
)

func systemCommands() []Project {
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
	return commands
}

func isSystemCommandPath(path string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(path)), "lyn:system:")
}
