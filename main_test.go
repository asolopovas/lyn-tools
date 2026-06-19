package main

import (
	"testing"

	"lyn.tools/launcher/lyn"
)

func TestStartHiddenArg(t *testing.T) {
	if !startHiddenArg([]string{"--debug", "--start-hidden"}) {
		t.Fatal("expected start hidden argument")
	}
	if startHiddenArg([]string{"--debug", "--start-hidden=false"}) {
		t.Fatal("did not expect start hidden argument")
	}
}

func TestWailsOptionsDisableResize(t *testing.T) {
	options := newWailsOptions(lyn.NewApp())
	if !options.DisableResize {
		t.Fatal("expected launcher window resize to be disabled")
	}
	if options.Width != 640 || options.Height != 306 {
		t.Fatalf("unexpected launcher size %dx%d", options.Width, options.Height)
	}
}

func TestElevatedHelperArg(t *testing.T) {
	name, ok := elevatedHelperArg([]string{"--start-hidden", `--elevated-helper=\\.\pipe\lyn-elevated-abc`})
	if !ok || name != `\\.\pipe\lyn-elevated-abc` {
		t.Fatalf("unexpected helper arg %q ok=%v", name, ok)
	}
	if _, ok := elevatedHelperArg([]string{"--debug", "--settings-window"}); ok {
		t.Fatal("did not expect helper arg")
	}
}

func TestWindowModeFromArgs(t *testing.T) {
	if windowModeFromArgs([]string{"--debug", "--settings-window"}) != lyn.SettingsWindowMode {
		t.Fatal("expected settings mode")
	}
	if windowModeFromArgs([]string{"--debug"}) != lyn.LauncherWindowMode {
		t.Fatal("expected launcher mode")
	}
}

func TestWailsOptionsSettingsWindow(t *testing.T) {
	app := lyn.NewApp()
	app.SetWindowMode(lyn.SettingsWindowMode)
	options := newWailsOptions(app)
	if options.Title != "Lyn Settings" {
		t.Fatalf("unexpected title %q", options.Title)
	}
	if options.DisableResize {
		t.Fatal("expected settings window resize to be enabled")
	}
	if options.AlwaysOnTop {
		t.Fatal("expected settings window to not be always on top")
	}
	if options.Width != 760 || options.Height != 660 {
		t.Fatalf("unexpected settings size %dx%d", options.Width, options.Height)
	}
}
