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
	if options.Width != 720 || options.Height != 650 {
		t.Fatalf("unexpected settings size %dx%d", options.Width, options.Height)
	}
}
