//go:build windows

package hotkey

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	nativehotkey "golang.design/x/hotkey"
	"golang.org/x/sys/windows"
)

const (
	whKeyboardLL   = 13
	wmKeydown      = 0x0100
	wmKeyup        = 0x0101
	wmSyskeydown   = 0x0104
	wmSyskeyup     = 0x0105
	wmQuit         = 0x0012
	vkControl      = 0x11
	vkLWin         = 0x5B
	vkRWin         = 0x5C
	keyeventfKeyup = 0x0002
	llkhfInjected  = 0x10
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procKeybdEvent          = user32.NewProc("keybd_event")
	keyInterceptorMu        sync.Mutex
	keyInterceptor          func(uint32) bool
	keyInterceptorID        uint64
)

type keyboardHookHotkey struct {
	hook           windows.Handle
	threadID       uint32
	callback       uintptr
	done           chan struct{}
	closeOnce      sync.Once
	pressed        atomic.Bool
	suppressDKeyup atomic.Bool
	lWinDown       atomic.Bool
	rWinDown       atomic.Bool
}

type keyboardHookEvent struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type windowsPoint struct {
	X int32
	Y int32
}

type windowsMessage struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      windowsPoint
}

func registerHotkeyBinding(binding Binding, onPress func()) (Registration, error) {
	if isWindowsDesktopHotkey(binding) {
		return registerKeyboardHookHotkey(onPress)
	}
	hk := nativehotkey.New(binding.Modifiers, binding.Key)
	if err := hk.Register(); err != nil {
		return nil, err
	}
	go func() {
		for range hk.Keydown() {
			onPress()
		}
	}()
	return hk, nil
}

func isWindowsDesktopHotkey(binding Binding) bool {
	return len(binding.Modifiers) == 1 && binding.Modifiers[0] == nativehotkey.ModWin && binding.Key == nativehotkey.KeyD
}

func registerKeyboardHookHotkey(onPress func()) (*keyboardHookHotkey, error) {
	hook := &keyboardHookHotkey{done: make(chan struct{})}
	ready := make(chan error, 1)
	hook.callback = windows.NewCallback(func(nCode int, wParam uintptr, lParam unsafe.Pointer) uintptr {
		event := (*keyboardHookEvent)(lParam)
		if nCode == 0 && event.Flags&llkhfInjected == 0 && hook.handleEvent(uint32(wParam), event.VkCode, onPress) {
			return 1
		}
		result, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, uintptr(lParam))
		return result
	})
	go hook.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return hook, nil
}

func SetKeyInterceptor(interceptor func(uint32) bool) func() {
	keyInterceptorMu.Lock()
	keyInterceptorID++
	id := keyInterceptorID
	previous := keyInterceptor
	keyInterceptor = interceptor
	keyInterceptorMu.Unlock()
	return func() {
		keyInterceptorMu.Lock()
		if keyInterceptorID == id {
			keyInterceptor = previous
		}
		keyInterceptorMu.Unlock()
	}
}

func interceptKey(vkCode uint32) bool {
	keyInterceptorMu.Lock()
	interceptor := keyInterceptor
	keyInterceptorMu.Unlock()
	return interceptor != nil && interceptor(vkCode)
}

func (h *keyboardHookHotkey) handleEvent(message uint32, vkCode uint32, onPress func()) bool {
	if (message == wmKeydown || message == wmSyskeydown) && interceptKey(vkCode) {
		return true
	}
	if vkCode == vkLWin || vkCode == vkRWin {
		switch message {
		case wmKeydown, wmSyskeydown:
			h.setWinDown(vkCode, true)
		case wmKeyup, wmSyskeyup:
			h.setWinDown(vkCode, false)
			if !h.trackedWinDown() {
				h.pressed.Store(false)
			}
		}
		return false
	}
	if vkCode != uint32(nativehotkey.KeyD) {
		return false
	}
	h.clearStaleWinState()
	suppressKeyup := h.suppressDKeyup.Load()
	switch message {
	case wmKeydown, wmSyskeydown:
		if suppressKeyup && !h.pressed.Load() && !h.trackedWinDown() {
			h.suppressDKeyup.Store(false)
			return false
		}
		if h.isWinDown() || h.pressed.Load() {
			if !h.pressed.Swap(true) {
				h.suppressDKeyup.Store(true)
				suppressStartMenuAfterWinChord()
				go finishWindowsDesktopHotkey(onPress)
			}
			return true
		}
	case wmKeyup, wmSyskeyup:
		if h.pressed.Load() || suppressKeyup {
			h.pressed.Store(false)
			h.suppressDKeyup.Store(false)
			return true
		}
	}
	return false
}

func (h *keyboardHookHotkey) setWinDown(vkCode uint32, down bool) {
	if vkCode == vkLWin {
		h.lWinDown.Store(down)
		return
	}
	h.rWinDown.Store(down)
}

func (h *keyboardHookHotkey) trackedWinDown() bool {
	return h.lWinDown.Load() || h.rWinDown.Load()
}

func (h *keyboardHookHotkey) isWinDown() bool {
	return h.trackedWinDown() || asyncKeyDown(vkLWin) || asyncKeyDown(vkRWin)
}

func (h *keyboardHookHotkey) clearStaleWinState() {
	if !h.trackedWinDown() {
		return
	}
	stale := false
	if h.lWinDown.Load() && !asyncKeyDown(vkLWin) {
		h.lWinDown.Store(false)
		stale = true
	}
	if h.rWinDown.Load() && !asyncKeyDown(vkRWin) {
		h.rWinDown.Store(false)
		stale = true
	}
	if stale && !h.isWinDown() {
		h.pressed.Store(false)
		h.suppressDKeyup.Store(false)
	}
}

var asyncKeyDown = func(vkCode uint32) bool {
	state, _, _ := procGetAsyncKeyState.Call(uintptr(vkCode))
	return state&0x8000 != 0
}

var suppressStartMenuAfterWinChord = func() {
	procKeybdEvent.Call(uintptr(vkControl), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vkControl), 0, keyeventfKeyup, 0)
}

var finishWindowsDesktopHotkey = func(onPress func()) {
	onPress()
}

func (h *keyboardHookHotkey) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h.threadID = windows.GetCurrentThreadId()
	hook, _, err := procSetWindowsHookExW.Call(whKeyboardLL, h.callback, 0, 0)
	if hook == 0 {
		ready <- err
		close(h.done)
		return
	}
	h.hook = windows.Handle(hook)
	ready <- nil
	var msg windowsMessage
	for {
		result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if result == 0 || int32(result) == -1 {
			break
		}
	}
	procUnhookWindowsHookEx.Call(uintptr(h.hook))
	close(h.done)
}

func (h *keyboardHookHotkey) Unregister() error {
	var err error
	h.closeOnce.Do(func() {
		if h.threadID != 0 {
			result, _, callErr := procPostThreadMessageW.Call(uintptr(h.threadID), wmQuit, 0, 0)
			if result == 0 {
				err = callErr
			}
		}
		<-h.done
	})
	return err
}
