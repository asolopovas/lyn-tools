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

func TestWindowsDesktopHotkeyHookSuppressesDesktopAndRunsOnceAfterWinRelease(t *testing.T) {
	suppressions, releases := stubDesktopHotkeySideEffects(t)
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 2)
	onPress := func() { pressed <- struct{}{} }
	if hook.handleEvent(wmKeydown, vkLWin, onPress) {
		t.Fatal("win down should pass through")
	}
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress) {
		t.Fatal("win+d down should be suppressed")
	}
	select {
	case <-pressed:
		t.Fatal("launch callback should wait for win release")
	default:
	}
	if !hook.handleEvent(wmKeyup, uint32(keys["d"]), onPress) {
		t.Fatal("win+d up should be suppressed")
	}
	if hook.handleEvent(wmKeyup, vkLWin, onPress) {
		t.Fatal("win up should pass through so Windows releases the key")
	}
	waitForSignal(t, suppressions, "start menu suppression")
	waitForSignal(t, releases, "win+d key release")
	waitForSignal(t, pressed, "launch callback")
	select {
	case <-pressed:
		t.Fatal("expected one launch callback")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestWindowsDesktopHotkeyFinishesWhenAsyncWinStateIsStale(t *testing.T) {
	_, releases := stubDesktopHotkeySideEffects(t)
	originalAsyncKeyDown := asyncKeyDown
	asyncKeyDown = func(vkCode uint32) bool { return vkCode == vkLWin }
	t.Cleanup(func() { asyncKeyDown = originalAsyncKeyDown })
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 1)
	onPress := func() { pressed <- struct{}{} }
	hook.handleEvent(wmKeydown, vkLWin, onPress)
	hook.handleEvent(wmKeydown, uint32(keys["d"]), onPress)
	hook.handleEvent(wmKeyup, uint32(keys["d"]), onPress)
	if hook.handleEvent(wmKeyup, vkLWin, onPress) {
		t.Fatal("win up should pass through")
	}
	waitForSignal(t, releases, "win+d key release")
	waitForSignal(t, pressed, "launch callback")
}

func TestWindowsDesktopHotkeyUsesAsyncWinState(t *testing.T) {
	_, releases := stubDesktopHotkeySideEffects(t)
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
	select {
	case <-pressed:
		t.Fatal("launch callback should wait for win release")
	default:
	}
	asyncWinDown = false
	if hook.handleEvent(wmKeyup, vkLWin, func() { pressed <- struct{}{} }) {
		t.Fatal("win up should pass through")
	}
	waitForSignal(t, releases, "win+d key release")
	waitForSignal(t, pressed, "launch callback")
}

func TestWindowsDesktopHotkeySuppressesDKeyupAfterWinReleasesFirst(t *testing.T) {
	stubDesktopHotkeySideEffects(t)
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 1)
	if hook.handleEvent(wmKeydown, vkLWin, func() { pressed <- struct{}{} }) {
		t.Fatal("win down should pass through")
	}
	if !hook.handleEvent(wmKeydown, uint32(keys["d"]), func() { pressed <- struct{}{} }) {
		t.Fatal("win+d down should be suppressed")
	}
	if hook.handleEvent(wmKeyup, vkLWin, func() { pressed <- struct{}{} }) {
		t.Fatal("win up should pass through")
	}
	waitForSignal(t, pressed, "launch callback")
	if !hook.handleEvent(wmKeyup, uint32(keys["d"]), func() { pressed <- struct{}{} }) {
		t.Fatal("orphan d keyup should be suppressed")
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
	hook := &keyboardHookHotkey{}
	pressed := make(chan struct{}, 2)
	onPress := func() { pressed <- struct{}{} }
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
	hook.handleEvent(wmKeyup, vkLWin, onPress)
	waitForSignal(t, pressed, "launch callback")
	select {
	case <-pressed:
		t.Fatal("expected one launch callback")
	case <-time.After(150 * time.Millisecond):
	}
}

func stubDesktopHotkeySideEffects(t *testing.T) (chan struct{}, chan struct{}) {
	t.Helper()
	suppressions := make(chan struct{}, 4)
	releases := make(chan struct{}, 4)
	originalSuppressStartMenu := suppressStartMenuAfterWinChord
	originalFinish := finishWindowsDesktopHotkey
	suppressStartMenuAfterWinChord = func() { suppressions <- struct{}{} }
	finishWindowsDesktopHotkey = func(onPress func()) {
		releases <- struct{}{}
		onPress()
	}
	t.Cleanup(func() {
		suppressStartMenuAfterWinChord = originalSuppressStartMenu
		finishWindowsDesktopHotkey = originalFinish
	})
	return suppressions, releases
}

func waitForSignal(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected %s", name)
	}
}
