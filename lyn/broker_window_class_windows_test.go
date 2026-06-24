//go:build windows

package lyn

import (
	"testing"

	"lyn.tools/launcher/lyn/hotkey"
)

func TestBrokerLauncherWindowClassMatches(t *testing.T) {
	if hotkey.LauncherWindowClass != NativeWindowClassName {
		t.Fatalf("broker foreground class %q must match NativeWindowClassName %q", hotkey.LauncherWindowClass, NativeWindowClassName)
	}
}
