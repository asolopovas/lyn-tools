package lyn

import "testing"

func TestSystemCommandsProvideThreeVirtualEntries(t *testing.T) {
	commands := systemCommands()
	wantPaths := map[string]string{
		systemCommandRestart:  "Restart",
		systemCommandShutdown: "Shut Down",
		systemCommandLogout:   "Log Out",
	}
	if len(commands) != len(wantPaths) {
		t.Fatalf("expected %d commands, got %d", len(wantPaths), len(commands))
	}
	for _, command := range commands {
		if command.Kind != projectKindSystemCommand {
			t.Fatalf("expected system-command kind, got %q", command.Kind)
		}
		name, ok := wantPaths[command.Path]
		if !ok {
			t.Fatalf("unexpected system command path %q", command.Path)
		}
		if command.Name != name {
			t.Fatalf("expected name %q for %q, got %q", name, command.Path, command.Name)
		}
		if !isSystemCommandPath(command.Path) {
			t.Fatalf("path %q not recognized as system command", command.Path)
		}
	}
}
