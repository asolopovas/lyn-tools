//go:build windows

package hotkey

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	brokerExeName       = "lyn-hook.exe"
	BrokerArg           = "--hook-broker"
	BrokerParentArg     = "--parent"
	toggleEventName     = `Local\LynHotkeyToggle`
	brokerStopName      = `Local\LynHotkeyBrokerStop`
	brokerReadyName     = `Local\LynHotkeyBrokerReady`
	synchronize         = 0x00100000
	waitObject0         = 0
	infinite            = 0xFFFFFFFF
	swShowNormal        = 1
	LauncherWindowClass = "LynLauncherWindow"
	allowForegroundAny  = 0xFFFFFFFF
	brokerReadyTimeout  = 3000
)

var (
	shell32             = windows.NewLazySystemDLL("shell32.dll")
	procShellExecute    = shell32.NewProc("ShellExecuteW")
	procFindWindow      = user32.NewProc("FindWindowW")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
	procSetForeground   = user32.NewProc("SetForegroundWindow")
	procBringToTop      = user32.NewProc("BringWindowToTop")
	procAllowForeground = user32.NewProc("AllowSetForegroundWindow")
)

func foregroundLauncherWindow() {
	classPtr, err := windows.UTF16PtrFromString(LauncherWindowClass)
	if err != nil {
		return
	}
	for i := 0; i < 15; i++ {
		hwnd, _, _ := procFindWindow.Call(uintptr(unsafe.Pointer(classPtr)), 0)
		if hwnd != 0 {
			if visible, _, _ := procIsWindowVisible.Call(hwnd); visible != 0 {
				procAllowForeground.Call(allowForegroundAny)
				procBringToTop.Call(hwnd)
				procSetForeground.Call(hwnd)
				return
			}
		}
		time.Sleep(40 * time.Millisecond)
	}
}

func brokerExePath() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	path := filepath.Join(filepath.Dir(exe), brokerExeName)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

type brokerRegistration struct {
	toggle     windows.Handle
	stop       windows.Handle
	localStop  windows.Handle
	closeOnce  sync.Once
	waiterDone chan struct{}
}

func tryBrokerHotkey(onPress func()) (Registration, bool) {
	path, ok := brokerExePath()
	if !ok {
		return nil, false
	}
	handles := newBrokerHandles()
	if handles == nil {
		return nil, false
	}
	if err := spawnBroker(path); err != nil {
		handles.closeAll()
		return nil, false
	}
	if res, _ := windows.WaitForSingleObject(handles.ready, brokerReadyTimeout); res != waitObject0 {
		windows.SetEvent(handles.stop)
		handles.closeAll()
		return nil, false
	}
	windows.CloseHandle(handles.ready)
	reg := &brokerRegistration{toggle: handles.toggle, stop: handles.stop, localStop: handles.localStop, waiterDone: make(chan struct{})}
	go reg.wait(onPress)
	return reg, true
}

type brokerHandles struct {
	toggle    windows.Handle
	stop      windows.Handle
	ready     windows.Handle
	localStop windows.Handle
}

func newBrokerHandles() *brokerHandles {
	toggle, err := createNamedEvent(toggleEventName, false)
	if err != nil {
		return nil
	}
	stop, err := createNamedEvent(brokerStopName, true)
	if err != nil {
		windows.CloseHandle(toggle)
		return nil
	}
	ready, err := createNamedEvent(brokerReadyName, true)
	if err != nil {
		windows.CloseHandle(toggle)
		windows.CloseHandle(stop)
		return nil
	}
	localStop, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		windows.CloseHandle(toggle)
		windows.CloseHandle(stop)
		windows.CloseHandle(ready)
		return nil
	}
	return &brokerHandles{toggle: toggle, stop: stop, ready: ready, localStop: localStop}
}

func (h *brokerHandles) closeAll() {
	windows.CloseHandle(h.toggle)
	windows.CloseHandle(h.stop)
	windows.CloseHandle(h.ready)
	windows.CloseHandle(h.localStop)
}

func (r *brokerRegistration) wait(onPress func()) {
	defer close(r.waiterDone)
	for {
		res, err := windows.WaitForMultipleObjects([]windows.Handle{r.toggle, r.localStop}, false, infinite)
		if err != nil || res != waitObject0 {
			return
		}
		onPress()
	}
}

func (r *brokerRegistration) Unregister() error {
	r.closeOnce.Do(func() {
		windows.SetEvent(r.stop)
		windows.SetEvent(r.localStop)
		<-r.waiterDone
		windows.CloseHandle(r.toggle)
		windows.CloseHandle(r.stop)
		windows.CloseHandle(r.localStop)
	})
	return nil
}

func createNamedEvent(name string, manualReset bool) (windows.Handle, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	var mr uint32
	if manualReset {
		mr = 1
	}
	handle, err := windows.CreateEvent(nil, mr, 0, namePtr)
	if handle == 0 {
		return 0, err
	}
	windows.ResetEvent(handle)
	return handle, nil
}

func openNamedEvent(name string) (windows.Handle, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	return windows.OpenEvent(windows.EVENT_MODIFY_STATE|synchronize, false, namePtr)
}

func openNamedEventRetry(name string) (windows.Handle, error) {
	var lastErr error
	for i := 0; i < 50; i++ {
		handle, err := openNamedEvent(name)
		if err == nil {
			return handle, nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	return 0, lastErr
}

func spawnBroker(path string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	params := BrokerArg + " " + BrokerParentArg + " " + strconv.Itoa(os.Getpid())
	paramsPtr, err := windows.UTF16PtrFromString(params)
	if err != nil {
		return err
	}
	dir, err := windows.UTF16PtrFromString(filepath.Dir(path))
	if err != nil {
		return err
	}
	ret, _, callErr := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(paramsPtr)),
		uintptr(unsafe.Pointer(dir)),
		swShowNormal,
	)
	if ret <= 32 {
		if callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.Errno(ret)
	}
	return nil
}

func RunBroker(parentPID uint32) error {
	if parentPID == 0 {
		return syscall.EINVAL
	}
	parent, err := windows.OpenProcess(synchronize, false, parentPID)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(parent)

	toggle, err := openNamedEventRetry(toggleEventName)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(toggle)
	stop, err := openNamedEventRetry(brokerStopName)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(stop)
	ready, err := openNamedEventRetry(brokerReadyName)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(ready)

	registration, err := registerKeyboardHookHotkey(func() {
		windows.SetEvent(toggle)
		go foregroundLauncherWindow()
	})
	if err != nil {
		return err
	}
	defer registration.Unregister()

	windows.SetEvent(ready)
	_, err = windows.WaitForMultipleObjects([]windows.Handle{stop, parent}, false, infinite)
	return err
}

func ParseBrokerArgs(args []string) (bool, uint32) {
	isBroker := false
	var parent uint32
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case BrokerArg:
			isBroker = true
		case BrokerParentArg:
			if i+1 < len(args) {
				if value, err := strconv.ParseUint(args[i+1], 10, 32); err == nil {
					parent = uint32(value)
				}
				i++
			}
		}
	}
	return isBroker, parent
}
