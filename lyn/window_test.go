package lyn

import (
	"strings"
	"testing"
)

func TestFocusQueryScriptDoesNotSelectTypedText(t *testing.T) {
	script := focusQueryScript()
	if !strings.Contains(script, ".focus()") {
		t.Fatalf("expected focus call in %q", script)
	}
	if strings.Contains(script, ".select()") {
		t.Fatalf("unexpected select call in %q", script)
	}
}

func TestToggleShowsWhenTrackedWindowWasMinimizedExternally(t *testing.T) {
	original := isNativeWindowMinimized
	isNativeWindowMinimized = func() bool { return true }
	t.Cleanup(func() { isNativeWindowMinimized = original })
	app := NewApp()
	app.shown = true
	if app.shouldHideOnToggleLocked() {
		t.Fatal("expected externally minimized window to be shown instead of hidden")
	}
}

func TestToggleHidesWhenTrackedWindowIsShown(t *testing.T) {
	original := isNativeWindowMinimized
	isNativeWindowMinimized = func() bool { return false }
	t.Cleanup(func() { isNativeWindowMinimized = original })
	app := NewApp()
	app.shown = true
	if !app.shouldHideOnToggleLocked() {
		t.Fatal("expected shown window to hide")
	}
}
