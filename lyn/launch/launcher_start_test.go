package launch

import (
	"errors"
	"testing"
	"time"
)

func TestLaunchRoutesCodeRequestToStarter(t *testing.T) {
	original := startCommand
	t.Cleanup(func() { startCommand = original })
	var gotPath string
	var gotAction string
	var gotCommand launchCommand
	startCommand = func(path string, cmd launchCommand, action string) error {
		gotPath = path
		gotAction = action
		gotCommand = cmd
		return nil
	}
	started := time.Now()
	result := Launch(Request{Path: `C:\src\lyn-tools`, Action: "code"})
	if time.Since(started) > 100*time.Millisecond {
		t.Fatal("launch request should hand off without waiting for the launched application")
	}
	if result.Error != "" {
		t.Fatalf("unexpected error %q", result.Error)
	}
	if gotPath != `C:\src\lyn-tools` || gotAction != "code" {
		t.Fatalf("unexpected route path=%q action=%q", gotPath, gotAction)
	}
	if gotCommand.Name == "" || len(gotCommand.Args) == 0 || gotCommand.Args[len(gotCommand.Args)-1] != gotPath {
		t.Fatalf("unexpected command %#v", gotCommand)
	}
}

func TestLaunchReturnsStarterError(t *testing.T) {
	original := startCommand
	t.Cleanup(func() { startCommand = original })
	startCommand = func(path string, cmd launchCommand, action string) error {
		return errors.New("start failed")
	}
	result := Launch(Request{Path: `C:\src\lyn-tools`, Action: "code"})
	if result.Error != "start failed" {
		t.Fatalf("expected starter error, got %#v", result)
	}
}
