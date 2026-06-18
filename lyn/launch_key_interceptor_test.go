package lyn

import (
	"context"
	"path/filepath"
	"testing"

	"lyn.tools/launcher/lyn/launch"
)

func TestCurrentLaunchSelectionUsesExplicitFrontendSelection(t *testing.T) {
	app := NewApp()
	app.SetLaunchSelection(launch.Request{Path: `C:\src\selected`, Action: "code"})
	request := app.currentLaunchSelection()
	if request.Path != `C:\src\selected` || request.Action != "code" {
		t.Fatalf("unexpected request %#v", request)
	}
}

func TestCurrentLaunchSelectionRejectsSystemCommandSelection(t *testing.T) {
	app := NewApp()
	app.SetLaunchSelection(launch.Request{Path: "lyn:system:logout", Action: "open"})
	request := app.currentLaunchSelection()
	if request.Path != "" {
		t.Fatalf("unexpected request %#v", request)
	}
}

func TestHasCachedLaunchSelectionAvoidsStoreInHookPath(t *testing.T) {
	app := NewApp()
	if app.hasCachedLaunchSelection() {
		t.Fatal("expected no cached selection by default")
	}
	app.SetLaunchSelection(launch.Request{Path: `C:\src\selected`, Action: "code"})
	if !app.hasCachedLaunchSelection() {
		t.Fatal("expected cached selection to be reported")
	}
	app.SetLaunchSelection(launch.Request{Path: "lyn:system:logout", Action: "open"})
	if app.hasCachedLaunchSelection() {
		t.Fatal("expected disabled system selection to be rejected")
	}
}

func TestNativeShortcutAction(t *testing.T) {
	originalControl := isNativeControlDown
	originalShift := isNativeShiftDown
	originalAlt := isNativeAltDown
	t.Cleanup(func() {
		isNativeControlDown = originalControl
		isNativeShiftDown = originalShift
		isNativeAltDown = originalAlt
	})
	setModifiers := func(ctrl bool, shift bool, alt bool) {
		isNativeControlDown = func() bool { return ctrl }
		isNativeShiftDown = func() bool { return shift }
		isNativeAltDown = func() bool { return alt }
	}
	tests := []struct {
		name    string
		vkCode  uint32
		ctrl    bool
		shift   bool
		alt     bool
		action  string
		handled bool
	}{
		{name: "enter", vkCode: vkReturn, handled: true},
		{name: "ctrl enter", vkCode: vkReturn, ctrl: true, action: "open", handled: true},
		{name: "shift enter", vkCode: vkReturn, shift: true, action: "terminal", handled: true},
		{name: "ctrl shift enter", vkCode: vkReturn, ctrl: true, shift: true, action: "run-admin", handled: true},
		{name: "ctrl shift c", vkCode: vkC, ctrl: true, shift: true, action: "terminal", handled: true},
		{name: "ctrl shift e", vkCode: vkE, ctrl: true, shift: true, action: "reveal", handled: true},
		{name: "ctrl shift u", vkCode: vkU, ctrl: true, shift: true, action: "run-user", handled: true},
		{name: "alt ignored", vkCode: vkReturn, alt: true},
		{name: "plain c ignored", vkCode: vkC},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setModifiers(test.ctrl, test.shift, test.alt)
			action, handled := nativeShortcutAction(test.vkCode)
			if action != test.action || handled != test.handled {
				t.Fatalf("unexpected action=%q handled=%v", action, handled)
			}
		})
	}
}

func TestCurrentLaunchSelectionFallsBackToTopStoredProject(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.UpsertProjects(ctx, []Project{
		{Name: "Log Out", Path: "lyn:system:logout", Kind: projectKindSystemCommand, UsageCount: 100},
		{Name: "lyn-tools", Path: `C:\src\lyn-tools`, Kind: "go"},
	}); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.ctx = ctx
	app.store = store
	request := app.currentLaunchSelection()
	if request.Path != `C:\src\lyn-tools` || request.Action != "code" {
		t.Fatalf("unexpected fallback request %#v", request)
	}
}
