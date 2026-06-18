package lyn

import "testing"

func TestSettingsWindowArgs(t *testing.T) {
	args := settingsWindowArgs([]string{"--debug", "--start-hidden", "--settings-window"})
	want := []string{"--debug", "--settings-window"}
	if len(args) != len(want) {
		t.Fatalf("unexpected args %#v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("unexpected args %#v", args)
		}
	}
}

func TestSettingsWindowActivityMarkerTracksRunningProcess(t *testing.T) {
	app := NewApp()
	config := DefaultConfig()
	config.Cache.Dir = t.TempDir()
	app.UseConfig(config)
	if app.settingsWindowActive() {
		t.Fatal("did not expect settings window active before marker")
	}
	app.markSettingsWindowActive()
	if !app.settingsWindowActive() {
		t.Fatal("expected settings window active after marker")
	}
	app.clearSettingsWindowActive()
	if app.settingsWindowActive() {
		t.Fatal("did not expect settings window active after clear")
	}
}

func TestOpenSettingsWindowStartsSettingsProcess(t *testing.T) {
	originalExecutable := settingsWindowExecutable
	originalStart := startSettingsWindowProcess
	defer func() {
		settingsWindowExecutable = originalExecutable
		startSettingsWindowProcess = originalStart
	}()
	settingsWindowExecutable = func() (string, error) {
		return "lyn-test", nil
	}
	var executable string
	var args []string
	startSettingsWindowProcess = func(path string, received []string) error {
		executable = path
		args = append([]string(nil), received...)
		return nil
	}
	app := NewApp()
	config := DefaultConfig()
	config.Cache.Dir = t.TempDir()
	app.UseConfig(config)
	if err := app.OpenSettingsWindow(); err != nil {
		t.Fatal(err)
	}
	if executable != "lyn-test" {
		t.Fatalf("unexpected executable %q", executable)
	}
	if len(args) == 0 || args[len(args)-1] != "--settings-window" {
		t.Fatalf("expected settings argument, got %#v", args)
	}
}
