package lyn

import (
	"strings"

	"lyn.tools/launcher/lyn/hotkey"
	"lyn.tools/launcher/lyn/launch"
)

const (
	vkReturn = 0x0D
	vkC      = 0x43
	vkE      = 0x45
	vkU      = 0x55
)

func (a *App) registerLaunchKeyInterceptor() {
	cleanup := hotkey.SetKeyInterceptor(func(vkCode uint32) bool {
		action, handled := nativeShortcutAction(vkCode)
		if !handled || !isWindowForeground() {
			return false
		}
		if !a.projects.hasCachedLaunchSelection() {
			a.debugLog("launch.native.missing")
			return false
		}
		go a.runNativeShortcut(action)
		return true
	})
	a.stateMu.Lock()
	a.keyInterceptorCleanup = cleanup
	a.stateMu.Unlock()
}

func launchablePath(path string) bool {
	return strings.TrimSpace(path) != "" && !isSystemCommandPath(path)
}

func (a *App) runNativeShortcut(action string) {
	request := a.projects.currentLaunchSelection()
	if !launchablePath(request.Path) {
		a.debugLog("launch.native.missing")
		return
	}
	if action != "" {
		request.Action = action
	}
	a.debugLog("launch.native.shortcut", "action", request.Action, "path", request.Path)
	if launch.NormalizedAction(request.Action) != "reveal" {
		a.Hide()
	}
	a.Launch(request)
}

func nativeShortcutAction(vkCode uint32) (string, bool) {
	if isAltDown() {
		return "", false
	}
	ctrl := isControlDown()
	shift := isShiftDown()
	if vkCode == vkReturn {
		switch {
		case ctrl && shift:
			return "run-admin", true
		case ctrl:
			return "open", true
		case shift:
			return "terminal", true
		default:
			return "", true
		}
	}
	if !ctrl || !shift {
		return "", false
	}
	switch vkCode {
	case vkC:
		return "terminal", true
	case vkE:
		return "reveal", true
	case vkU:
		return "run-user", true
	default:
		return "", false
	}
}
