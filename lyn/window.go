package lyn

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	NativeWindowClassName         = "LynLauncherWindow"
	NativeSettingsWindowClassName = "LynSettingsWindow"
)

func focusQueryScript() string {
	return `(() => { for (let i = 0; i < 12; i++) setTimeout(() => { const el = document.querySelector('[data-lyn-query]'); if (el) { window.focus(); el.focus({ preventScroll: true }); } }, i * 35); })()`
}

func placeWindow(ctx context.Context) {
	if ctx == nil {
		return
	}
	runtime.WindowCenter(ctx)
}

var (
	configureNativeWindowAppearance       func()
	prepareNativeWindowActivation         func()
	activateNativeWindow                  func()
	isNativeWindowMinimized               func() bool
	isNativeWindowForeground              func() bool
	didNativePointerActivateOutsideWindow func() bool
	isNativeControlDown                   func() bool
	isNativeShiftDown                     func() bool
	isNativeAltDown                       func() bool
)

func configureWindowAppearance() {
	if configureNativeWindowAppearance != nil {
		configureNativeWindowAppearance()
	}
}

func prepareWindowActivation() {
	if prepareNativeWindowActivation != nil {
		prepareNativeWindowActivation()
	}
}

func activateWindow() {
	if activateNativeWindow != nil {
		activateNativeWindow()
	}
}

func isWindowMinimized() bool {
	return isNativeWindowMinimized != nil && isNativeWindowMinimized()
}

func isWindowForeground() bool {
	return isNativeWindowForeground == nil || isNativeWindowForeground()
}

func didPointerActivateOutsideWindow() bool {
	return didNativePointerActivateOutsideWindow != nil && didNativePointerActivateOutsideWindow()
}

func isControlDown() bool {
	return isNativeControlDown != nil && isNativeControlDown()
}

func isShiftDown() bool {
	return isNativeShiftDown != nil && isNativeShiftDown()
}

func isAltDown() bool {
	return isNativeAltDown != nil && isNativeAltDown()
}
