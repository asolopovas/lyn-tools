package lyn

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestNormalizeElevationMode(t *testing.T) {
	for _, mode := range []string{elevationModeStandard, elevationModeAdmin} {
		if got, err := normalizeElevationMode(mode); err != nil || got != mode {
			t.Fatalf("normalizeElevationMode(%q) = %q, %v", mode, got, err)
		}
	}
	if _, err := normalizeElevationMode("root"); err == nil {
		t.Fatal("expected unsupported mode error")
	}
}

func TestSwitchElevationStartsTargetModeThenQuits(t *testing.T) {
	originalDetect := detectElevationStatus
	originalStart := startElevationProcess
	originalQuit := quitRuntime
	defer func() {
		detectElevationStatus = originalDetect
		startElevationProcess = originalStart
		quitRuntime = originalQuit
	}()
	detectElevationStatus = func() ElevationStatus {
		return ElevationStatus{Mode: elevationModeStandard, CanSwitch: true}
	}
	var startedMode string
	var startedArgs []string
	startElevationProcess = func(mode string, args []string) error {
		startedMode = mode
		startedArgs = append([]string(nil), args...)
		return nil
	}
	quit := false
	quitRuntime = func(context.Context) { quit = true }
	app := NewApp()
	app.ctx = context.Background()
	status, err := app.SwitchElevation(elevationModeAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if status.Mode != elevationModeAdmin || startedMode != elevationModeAdmin || !quit {
		t.Fatalf("unexpected status=%#v startedMode=%q quit=%v", status, startedMode, quit)
	}
	if !reflect.DeepEqual(startedArgs, os.Args[1:]) {
		t.Fatalf("unexpected args %#v", startedArgs)
	}
}

func TestSwitchElevationNoopsForCurrentMode(t *testing.T) {
	originalDetect := detectElevationStatus
	originalStart := startElevationProcess
	originalQuit := quitRuntime
	defer func() {
		detectElevationStatus = originalDetect
		startElevationProcess = originalStart
		quitRuntime = originalQuit
	}()
	detectElevationStatus = func() ElevationStatus {
		return ElevationStatus{Mode: elevationModeAdmin, CanSwitch: true}
	}
	startElevationProcess = func(string, []string) error {
		t.Fatal("unexpected elevation restart")
		return nil
	}
	quitRuntime = func(context.Context) { t.Fatal("unexpected quit") }
	app := NewApp()
	app.ctx = context.Background()
	status, err := app.SwitchElevation(elevationModeAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if status.Mode != elevationModeAdmin {
		t.Fatalf("unexpected status %#v", status)
	}
}

func TestSwitchElevationReturnsStartErrorWithoutQuit(t *testing.T) {
	originalDetect := detectElevationStatus
	originalStart := startElevationProcess
	originalQuit := quitRuntime
	defer func() {
		detectElevationStatus = originalDetect
		startElevationProcess = originalStart
		quitRuntime = originalQuit
	}()
	startErr := errors.New("uac canceled")
	detectElevationStatus = func() ElevationStatus {
		return ElevationStatus{Mode: elevationModeStandard, CanSwitch: true}
	}
	startElevationProcess = func(string, []string) error { return startErr }
	quitRuntime = func(context.Context) { t.Fatal("unexpected quit") }
	app := NewApp()
	app.ctx = context.Background()
	_, err := app.SwitchElevation(elevationModeAdmin)
	if !errors.Is(err, startErr) {
		t.Fatalf("expected %v, got %v", startErr, err)
	}
}
