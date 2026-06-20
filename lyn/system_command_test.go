package lyn

import (
	"strings"
	"testing"

	"lyn.tools/launcher/lyn/launch"
)

func TestSystemCommandsProvideThreeVirtualEntries(t *testing.T) {
	commands := systemCommandsFor("windows")
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

func TestSystemCommandsLinuxIncludesAdminTools(t *testing.T) {
	var admin int
	for _, command := range systemCommandsFor("linux") {
		if command.Kind != projectKindSystemCommand {
			t.Fatalf("expected system-command kind, got %q", command.Kind)
		}
		if strings.HasPrefix(command.Path, launch.AdminToolPrefix) {
			admin++
			if !isSystemCommandPath(command.Path) {
				t.Fatalf("admin path %q not recognized as system command", command.Path)
			}
		}
	}
	if admin == 0 {
		t.Fatal("expected admin system tools on linux")
	}
}

func TestSystemCommandsNonLinuxOmitsAdminTools(t *testing.T) {
	for _, command := range systemCommandsFor("windows") {
		if strings.HasPrefix(command.Path, launch.AdminToolPrefix) {
			t.Fatalf("admin tool %q should not appear on windows", command.Path)
		}
	}
}
