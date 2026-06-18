package launch

import (
	"strings"
	"testing"
)

func TestSystemCommandLinuxLogoutUsesUsername(t *testing.T) {
	original := lookupUsername
	t.Cleanup(func() { lookupUsername = original })
	lookupUsername = func() string { return "example" }

	cmd, err := BuildLaunchCommand(Request{Path: "lyn:system:logout", Action: "open"}, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "loginctl" || strings.Join(cmd.Args, " ") != "terminate-user example" {
		t.Fatalf("unexpected logout command %+v", cmd)
	}
}

func TestSystemCommandLinuxLogoutWithoutUserFails(t *testing.T) {
	original := lookupUsername
	t.Cleanup(func() { lookupUsername = original })
	lookupUsername = func() string { return "" }

	if _, err := BuildLaunchCommand(Request{Path: "lyn:system:logout", Action: "open"}, "linux"); err == nil {
		t.Fatal("expected error when no active user session")
	}
}

func TestSystemCommandDarwinUsesOsascript(t *testing.T) {
	cmd, err := BuildLaunchCommand(Request{Path: "lyn:system:shutdown", Action: "open"}, "darwin")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "osascript" || !strings.Contains(strings.Join(cmd.Args, " "), "shut down") {
		t.Fatalf("unexpected darwin command %+v", cmd)
	}
}
