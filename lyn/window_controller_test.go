package lyn

import (
	"context"
	"testing"
)

func newTestWindowController() *windowController {
	return newWindowController(func(string, ...any) {})
}

func TestWindowControllerShowHideSequencing(t *testing.T) {
	controller := newTestWindowController()
	ctx := t.Context()
	controller.setContext(ctx)
	showContext, sequence, ok := controller.beginShow()
	if !ok || showContext != ctx || sequence != 1 || !controller.shown {
		t.Fatalf("unexpected show state: ok=%v sequence=%d shown=%v", ok, sequence, controller.shown)
	}
	hideContext, ok := controller.beginHide()
	if !ok || hideContext != ctx || controller.shown || controller.showSequence != 2 {
		t.Fatalf("unexpected hide state: ok=%v sequence=%d shown=%v", ok, controller.showSequence, controller.shown)
	}
}

func TestWindowControllerFocusLossCancellationAndSuppression(t *testing.T) {
	originalHide := windowHide
	t.Cleanup(func() { windowHide = originalHide })
	hides := 0
	windowHide = func(context.Context) { hides++ }
	controller := newTestWindowController()
	ctx := t.Context()
	controller.setContext(ctx)
	_, sequence, _ := controller.beginShow()
	controller.setAutoHideSuppressed(true)
	if controller.hideAfterFocusLoss(ctx, sequence) {
		t.Fatal("suppressed focus loss hid the window")
	}
	controller.setAutoHideSuppressed(false)
	controller.beginShow()
	if controller.hideAfterFocusLoss(ctx, sequence) {
		t.Fatal("stale focus sequence hid the window")
	}
	current := controller.showSequence
	if !controller.hideAfterFocusLoss(ctx, current) || hides != 1 {
		t.Fatalf("current focus loss did not hide once: %d", hides)
	}
}

func TestWindowControllerSettingsModeAndQuitState(t *testing.T) {
	originalQuit := quitRuntime
	t.Cleanup(func() { quitRuntime = originalQuit })
	quitCalled := false
	quitRuntime = func(context.Context) { quitCalled = true }
	controller := newTestWindowController()
	controller.setMode(SettingsWindowMode)
	if controller.windowMode() != SettingsWindowMode {
		t.Fatal("settings mode was not retained")
	}
	controller.requestQuit(t.Context())
	if !controller.isQuitting() || !quitCalled {
		t.Fatal("quit state was not set before the native quit call")
	}
}

func TestWindowControllerReleasesLockBeforeNativeHide(t *testing.T) {
	originalHide := windowHide
	t.Cleanup(func() { windowHide = originalHide })
	controller := newTestWindowController()
	controller.setContext(t.Context())
	lockHeld := false
	windowHide = func(context.Context) {
		if controller.mu.TryLock() {
			controller.mu.Unlock()
			return
		}
		lockHeld = true
	}
	controller.hide()
	if lockHeld {
		t.Fatal("window mutex held during native hide")
	}
}
