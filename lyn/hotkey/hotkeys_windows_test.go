//go:build windows

package hotkey

import (
	"testing"
	"time"
)

func TestWindowsDesktopHotkeyDetected(t *testing.T) {
	binding, err := ParseHotkey("Win+D")
	if err != nil {
		t.Fatal(err)
	}
	if !isWindowsDesktopHotkey(binding) {
		t.Fatal("expected Win+D to use the desktop hotkey fallback")
	}
}

func TestWindowsDesktopHotkeyFiresOnKeydown(t *testing.T) {
	suppressions, _ := stubDesktopHotkeySideEffects(t)
	winDown := stubAsyncWinState(t)
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 2)
	onPress := func() { pressed <- struct{}{} }
	*winDown = true
	if hook.handleEvent(wmKeydown, vkLWin, onPress) {
		t.Fatal("win down should pass through")
	}
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("win+d down should be suppressed")
	}
	waitForSignal(t, pressed, "launch callback on keydown")
	waitForSignal(t, suppressions, "start menu suppression")
	if !hook.handleEvent(wmKeyup, uint32(keys["d"]), onPress) {
		t.Fatal("win+d up should be suppressed")
	}
	*winDown = false
	if hook.handleEvent(wmKeyup, vkLWin, onPress) {
		t.Fatal("win up should pass through so Windows releases the key")
	}
	select {
	case <-pressed:
		t.Fatal("expected one launch callback")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestWindowsDesktopHotkeyUsesAsyncWinState(t *testing.T) {
	stubDesktopHotkeySideEffects(t)
	originalAsyncKeyDown := asyncKeyDown
	asyncWinDown := true
	asyncKeyDown = func(vkCode uint32) bool {
		return asyncWinDown && vkCode == vkLWin
	}
	t.Cleanup(func() { asyncKeyDown = originalAsyncKeyDown })
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 1)
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), func() { pressed <- struct{}{} }) {
		t.Fatal("win+d down should be suppressed from async win state")
	}
	waitForSignal(t, pressed, "launch callback")
}

func TestWindowsDesktopHotkeySuppressesDKeyupAfterWinReleasesFirst(t *testing.T) {
	stubDesktopHotkeySideEffects(t)
	winDown := stubAsyncWinState(t)
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 2)
	onPress := func() { pressed <- struct{}{} }
	*winDown = true
	if hook.handleEvent(wmKeydown, vkLWin, onPress) {
		t.Fatal("win down should pass through")
	}
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("win+d down should be suppressed")
	}
	waitForSignal(t, pressed, "launch callback")
	*winDown = false
	if hook.handleEvent(wmKeyup, vkLWin, onPress) {
		t.Fatal("win up should pass through")
	}
	if !hook.handleEvent(wmKeyup, uint32(keys["d"]), onPress) {
		t.Fatal("orphan d keyup should be suppressed")
	}
	if hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("normal d keydown after chord should pass through")
	}
	select {
	case <-pressed:
		t.Fatal("expected one launch callback")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestWindowsDesktopHotkeyStaleSuppressionDoesNotBlockNormalDKeydown(t *testing.T) {
	stubDesktopHotkeySideEffects(t)
	originalAsyncKeyDown := asyncKeyDown
	asyncKeyDown = func(vkCode uint32) bool { return vkCode == vkLWin }
	t.Cleanup(func() { asyncKeyDown = originalAsyncKeyDown })
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 1)
	hook.suppressDKeyup.Store(true)
	if hook.handleEvent(wmKeydown, uint32(keys["d"]), func() { pressed <- struct{}{} }) {
		t.Fatal("normal d keydown after shortcut should pass through")
	}
	if hook.suppressDKeyup.Load() {
		t.Fatal("expected stale d keyup suppression to be cleared")
	}
	select {
	case <-pressed:
		t.Fatal("normal d keydown should not trigger shortcut")
	default:
	}
}

func TestWindowsDesktopHotkeySuppressesRepeatsAndRunsOnce(t *testing.T) {
	stubDesktopHotkeySideEffects(t)
	winDown := stubAsyncWinState(t)
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 2)
	onPress := func() { pressed <- struct{}{} }
	*winDown = true
	hook.handleEvent(wmKeydown, vkLWin, onPress)
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("first win+d down should be suppressed")
	}
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("repeated win+d down should be suppressed")
	}
	if !hook.handleEvent(wmKeyup, uint32(keys["d"]), onPress) {
		t.Fatal("win+d up should be suppressed")
	}
	*winDown = false
	hook.handleEvent(wmKeyup, vkLWin, onPress)
	waitForSignal(t, pressed, "launch callback")
	select {
	case <-pressed:
		t.Fatal("expected one launch callback")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestWindowsDesktopHotkeyTogglesOnSecondTapWhileWinHeld(t *testing.T) {
	stubDesktopHotkeySideEffects(t)
	winDown := stubAsyncWinState(t)
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 2)
	onPress := func() { pressed <- struct{}{} }
	*winDown = true
	hook.handleEvent(wmKeydown, vkLWin, onPress)
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("first win+d down should be suppressed")
	}
	waitForSignal(t, pressed, "first launch callback")
	if !hook.handleEvent(wmKeyup, uint32(keys["d"]), onPress) {
		t.Fatal("win+d up should be suppressed")
	}
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("second win+d down should be suppressed")
	}
	waitForSignal(t, pressed, "second launch callback")
}

func TestWindowsDesktopHotkeyRecoversWhenWinKeyupWasMissed(t *testing.T) {
	stubDesktopHotkeySideEffects(t)
	winDown := stubAsyncWinState(t)
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 1)
	onPress := func() { pressed <- struct{}{} }
	*winDown = true
	hook.handleEvent(wmKeydown, vkLWin, onPress)
	*winDown = false
	if hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("d keydown after missed win keyup should pass through")
	}
	if hook.handleEvent(wmKeyup, uint32(keys["d"]), onPress) {
		t.Fatal("d keyup after missed win keyup should pass through")
	}
	select {
	case <-pressed:
		t.Fatal("missed win keyup should not trigger shortcut")
	default:
	}
}

func stubAsyncWinState(t *testing.T) *bool {
	t.Helper()
	down := false
	original := asyncKeyDown
	asyncKeyDown = func(vkCode uint32) bool {
		return down && (vkCode == vkLWin || vkCode == vkRWin)
	}
	t.Cleanup(func() { asyncKeyDown = original })
	return &down
}

func stubDesktopHotkeySideEffects(t *testing.T) (chan struct{}, chan struct{}) {
	t.Helper()
	suppressions := make(chan struct{}, 4)
	finishes := make(chan struct{}, 4)
	originalSuppressStartMenu := suppressStartMenuAfterWinChord
	originalFinish := finishWindowsDesktopHotkey
	suppressStartMenuAfterWinChord = func() { suppressions <- struct{}{} }
	finishWindowsDesktopHotkey = func(onPress func()) {
		finishes <- struct{}{}
		onPress()
	}
	t.Cleanup(func() {
		suppressStartMenuAfterWinChord = originalSuppressStartMenu
		finishWindowsDesktopHotkey = originalFinish
	})
	return suppressions, finishes
}

func waitForSignal(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected %s", name)
	}
}
