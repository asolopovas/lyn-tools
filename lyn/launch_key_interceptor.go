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
		if !handled {
			return false
		}
		if !isWindowForeground() {
			return false
		}
		request := a.currentLaunchSelection()
		if strings.TrimSpace(request.Path) == "" || isDisabledSystemPath(request.Path) {
			a.debugLog("launch.native.missing")
			return false
		}
		if action != "" {
			request.Action = action
		}
		a.debugLog("launch.native.shortcut", "action", request.Action, "path", request.Path)
		go func() {
			result := a.Launch(request)
			if result.Error == "" && launch.NormalizedAction(request.Action) != "reveal" {
				a.Hide()
			}
		}()
		return true
	})
	a.stateMu.Lock()
	a.keyInterceptorCleanup = cleanup
	a.stateMu.Unlock()
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

func (a *App) currentLaunchSelection() launch.Request {
	a.launchSelectionMu.Lock()
	request := a.launchSelection
	a.launchSelectionMu.Unlock()
	request.Action = launch.NormalizedAction(request.Action)
	if strings.TrimSpace(request.Path) != "" {
		if isDisabledSystemPath(request.Path) {
			return launch.Request{}
		}
		return request
	}
	ctx, _, store := a.snapshot()
	if store == nil {
		return request
	}
	projects, err := store.ListProjects(ctx)
	if err != nil || len(projects) == 0 {
		return request
	}
	project := Project{}
	for _, candidate := range projects {
		if candidate.Kind != projectKindSystemCommand && !isDisabledSystemPath(candidate.Path) {
			project = candidate
			break
		}
	}
	if project.Path == "" {
		return request
	}
	action := "code"
	if project.Kind == projectKindApp {
		action = "open"
	}
	a.debugLog("launch.native.fallback", "action", action, "path", project.Path, "error", err)
	return launch.Request{Path: project.Path, Action: action}
}

func isDisabledSystemPath(path string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(path)), "lyn:system:")
}
