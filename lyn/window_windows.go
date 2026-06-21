//go:build windows

package lyn

import (
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	swHide          = 0
	swShownormal    = 1
	vkLbutton       = 0x01
	vkRbutton       = 0x02
	vkMbutton       = 0x04
	vkShift         = 0x10
	vkControl       = 0x11
	vkMenu          = 0x12
	wsExToolwindow  = 0x00000080
	wsExAppwindow   = 0x00040000
	wsSysmenu       = 0x00080000
	wsMinimizebox   = 0x00020000
	wsMaximizebox   = 0x00010000
	wmClose         = 0x0010
	wmSyscommand    = 0x0112
	scClose         = 0xF060
	scMask          = 0xFFF0
	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZorder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
)

var gwlStyle = ^uintptr(15)

var gwlExstyle = ^uintptr(19)

var gwlpWndproc = ^uintptr(3)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procSendInput                = user32.NewProc("SendInput")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procEnumChildWindows         = user32.NewProc("EnumChildWindows")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procGetWindowLongPtr         = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr         = user32.NewProc("SetWindowLongPtrW")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procSetActiveWindow          = user32.NewProc("SetActiveWindow")
	procSetFocus                 = user32.NewProc("SetFocus")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	procCallWindowProc           = user32.NewProc("CallWindowProcW")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
)

var (
	closeToTrayOriginalProc uintptr
	closeToTrayCallback     uintptr
	closeToTrayHide         func()
	closeToTrayQuitting     func() bool
	closeToTrayInstalled    bool
)

func init() {
	configureNativeWindowAppearance = configureWindowsWindowAppearance
	prepareNativeWindowActivation = prepareWindowsWindowActivation
	activateNativeWindow = activateWindowsWindow
	isNativeWindowMinimized = isWindowsWindowMinimized
	isNativeWindowForeground = isWindowsWindowForeground
	didNativePointerActivateOutsideWindow = didWindowsPointerActivateOutsideWindow
	isNativeControlDown = func() bool { return asyncKeyDown(vkControl) }
	isNativeShiftDown = func() bool { return asyncKeyDown(vkShift) }
	isNativeAltDown = func() bool { return asyncKeyDown(vkMenu) }
	installNativeCloseToTray = installWindowsCloseToTray
}

func installWindowsCloseToTray(isQuitting func() bool, hide func()) {
	if closeToTrayInstalled || isQuitting == nil || hide == nil {
		return
	}
	hwnd := currentProcessWindow()
	if hwnd == 0 {
		return
	}
	closeToTrayQuitting = isQuitting
	closeToTrayHide = hide
	closeToTrayCallback = windows.NewCallback(closeToTrayWndProc)
	original, _, _ := procSetWindowLongPtr.Call(uintptr(hwnd), gwlpWndproc, closeToTrayCallback)
	if original == 0 {
		closeToTrayQuitting = nil
		closeToTrayHide = nil
		return
	}
	closeToTrayOriginalProc = original
	closeToTrayInstalled = true
}

func closeToTrayWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	if closeToTrayShouldHide(msg, wparam) {
		go closeToTrayHide()
		return 0
	}
	ret, _, _ := procCallWindowProc.Call(closeToTrayOriginalProc, hwnd, msg, wparam, lparam)
	return ret
}

func closeToTrayShouldHide(msg, wparam uintptr) bool {
	if closeToTrayQuitting == nil || closeToTrayHide == nil || closeToTrayQuitting() {
		return false
	}
	switch msg {
	case wmClose:
		return true
	case wmSyscommand:
		return wparam&scMask == scClose
	default:
		return false
	}
}

type windowsMouseInput struct {
	Dx        int32
	Dy        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type windowsInput struct {
	Type  uint32
	Pad   uint32
	Mouse windowsMouseInput
}

type windowsPoint struct {
	X int32
	Y int32
}

type windowsRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func configureWindowsWindowAppearance() {
	hwnd := currentProcessWindow()
	if hwnd == 0 {
		return
	}
	hideWindowsWindowFromTaskbar(hwnd)
	removeWindowsCaptionButtons(hwnd)
}

func removeWindowsCaptionButtons(hwnd windows.Handle) {
	style, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlStyle)
	updated := style &^ (wsSysmenu | wsMinimizebox | wsMaximizebox)
	if updated == style {
		return
	}
	procSetWindowLongPtr.Call(uintptr(hwnd), gwlStyle, updated)
	procSetWindowPos.Call(uintptr(hwnd), 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZorder|swpNoActivate|swpFrameChanged)
}

func prepareWindowsWindowActivation() {
	input := windowsInput{}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&input)), unsafe.Sizeof(input))
}

func activateWindowsWindow() {
	hwnd := currentProcessWindow()
	if hwnd == 0 {
		return
	}
	focusHwnd := webviewHostWindow(hwnd)
	foreground, _, _ := procGetForegroundWindow.Call()
	var foregroundPID uint32
	foregroundThread, _, _ := procGetWindowThreadProcessID.Call(foreground, uintptr(unsafe.Pointer(&foregroundPID)))
	var targetPID uint32
	targetThread, _, _ := procGetWindowThreadProcessID.Call(uintptr(focusHwnd), uintptr(unsafe.Pointer(&targetPID)))
	currentThread := uintptr(windows.GetCurrentThreadId())
	attached := attachInputThreads(currentThread, foregroundThread, targetThread)
	defer detachInputThreads(currentThread, attached)
	procShowWindow.Call(uintptr(hwnd), swShownormal)
	procBringWindowToTop.Call(uintptr(hwnd))
	procSetForegroundWindow.Call(uintptr(hwnd))
	procSetActiveWindow.Call(uintptr(hwnd))
	procSetFocus.Call(uintptr(hwnd))
	if focusHwnd != hwnd {
		procSetFocus.Call(uintptr(focusHwnd))
	}
}

func isWindowsWindowMinimized() bool {
	hwnd := currentProcessWindow()
	if hwnd == 0 {
		return false
	}
	minimized, _, _ := procIsIconic.Call(uintptr(hwnd))
	return minimized != 0
}

func isWindowsWindowForeground() bool {
	hwnd := currentProcessWindow()
	if hwnd == 0 {
		return false
	}
	foreground, _, _ := procGetForegroundWindow.Call()
	return foreground == uintptr(hwnd)
}

func hideWindowsWindowFromTaskbar(hwnd windows.Handle) {
	style, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlExstyle)
	updated := taskbarHiddenExStyle(style)
	if updated == style {
		return
	}
	visible := isWindowHandleVisible(hwnd)
	if visible {
		procShowWindow.Call(uintptr(hwnd), swHide)
	}
	procSetWindowLongPtr.Call(uintptr(hwnd), gwlExstyle, updated)
	if visible {
		procShowWindow.Call(uintptr(hwnd), swShownormal)
	}
}

func taskbarHiddenExStyle(style uintptr) uintptr {
	return style&^wsExAppwindow | wsExToolwindow
}

func isWindowHandleVisible(hwnd windows.Handle) bool {
	visible, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return visible != 0
}

func didWindowsPointerActivateOutsideWindow() bool {
	if !isMouseButtonDown() {
		return false
	}
	hwnd := currentProcessWindow()
	if hwnd == 0 {
		return false
	}
	var point windowsPoint
	ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	if ok == 0 {
		return false
	}
	var rect windowsRect
	ok, _, _ = procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ok == 0 {
		return false
	}
	return !pointInWindowRect(point, rect)
}

func isMouseButtonDown() bool {
	return asyncKeyDown(vkLbutton) || asyncKeyDown(vkRbutton) || asyncKeyDown(vkMbutton)
}

func asyncKeyDown(virtualKey uintptr) bool {
	state, _, _ := procGetAsyncKeyState.Call(virtualKey)
	return state&0x8000 != 0
}

func pointInWindowRect(point windowsPoint, rect windowsRect) bool {
	return point.X >= rect.Left && point.X < rect.Right && point.Y >= rect.Top && point.Y < rect.Bottom
}

func attachInputThreads(currentThread uintptr, threads ...uintptr) []uintptr {
	attached := make([]uintptr, 0, len(threads))
	seen := make(map[uintptr]bool, len(threads))
	for _, thread := range threads {
		if thread == 0 || thread == currentThread || seen[thread] {
			continue
		}
		procAttachThreadInput.Call(currentThread, thread, 1)
		attached = append(attached, thread)
		seen[thread] = true
	}
	return attached
}

func detachInputThreads(currentThread uintptr, threads []uintptr) {
	for i := len(threads) - 1; i >= 0; i-- {
		procAttachThreadInput.Call(currentThread, threads[i], 0)
	}
}

var (
	webviewHostMu       sync.Mutex
	webviewHostFound    windows.Handle
	webviewHostCallback = windows.NewCallback(webviewHostEnumProc)
)

func webviewHostEnumProc(child windows.Handle, lparam uintptr) uintptr {
	visible, _, _ := procIsWindowVisible.Call(uintptr(child))
	if visible != 0 && windowClass(child) == "Chrome_WidgetWin_1" {
		webviewHostFound = child
		return 0
	}
	return 1
}

func webviewHostWindow(hwnd windows.Handle) windows.Handle {
	webviewHostMu.Lock()
	defer webviewHostMu.Unlock()
	webviewHostFound = 0
	procEnumChildWindows.Call(uintptr(hwnd), webviewHostCallback, 0)
	if webviewHostFound != 0 {
		return webviewHostFound
	}
	return hwnd
}

func windowClass(hwnd windows.Handle) string {
	var buffer [256]uint16
	length, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if length == 0 {
		return ""
	}
	return windows.UTF16ToString(buffer[:length])
}

var (
	currentWindowMu       sync.Mutex
	currentWindowPID      uint32
	currentWindowClass    windows.Handle
	currentWindowVisible  windows.Handle
	currentWindowProcess  windows.Handle
	currentWindowCallback = windows.NewCallback(currentProcessWindowEnumProc)
)

func currentProcessWindowEnumProc(hwnd windows.Handle, lparam uintptr) uintptr {
	var windowPID uint32
	procGetWindowThreadProcessID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowPID)))
	if windowPID != currentWindowPID {
		return 1
	}
	if currentWindowProcess == 0 {
		currentWindowProcess = hwnd
	}
	if isWindowHandleVisible(hwnd) && currentWindowVisible == 0 {
		currentWindowVisible = hwnd
	}
	if windowClass(hwnd) == NativeWindowClassName {
		currentWindowClass = hwnd
		return 0
	}
	return 1
}

func currentProcessWindow() windows.Handle {
	currentWindowMu.Lock()
	defer currentWindowMu.Unlock()
	currentWindowPID = uint32(os.Getpid())
	currentWindowClass = 0
	currentWindowVisible = 0
	currentWindowProcess = 0
	procEnumWindows.Call(currentWindowCallback, 0)
	if currentWindowClass != 0 {
		return currentWindowClass
	}
	if currentWindowVisible != 0 {
		return currentWindowVisible
	}
	return currentWindowProcess
}
