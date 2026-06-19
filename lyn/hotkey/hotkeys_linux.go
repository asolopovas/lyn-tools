//go:build linux

package hotkey

/*
#cgo LDFLAGS: -lX11
#include <stdint.h>

void lynRunHotkey(uintptr_t handle, int keysym, unsigned int baseMod, unsigned int *grabMods, int nMods, int stopFD);
*/
import "C"

import (
	"runtime"
	"sync"
	"syscall"

	nativehotkey "golang.design/x/hotkey"
)

const capsLockMask = nativehotkey.Modifier(1 << 1)

func hotkeyLockVariants(base []nativehotkey.Modifier) [][]nativehotkey.Modifier {
	extras := [][]nativehotkey.Modifier{
		nil,
		{nativehotkey.Mod2},
		{capsLockMask},
		{nativehotkey.Mod2, capsLockMask},
	}
	variants := make([][]nativehotkey.Modifier, 0, len(extras))
	for _, extra := range extras {
		mods := append(append([]nativehotkey.Modifier{}, base...), extra...)
		variants = append(variants, mods)
	}
	return variants
}

var (
	fireMu       sync.Mutex
	fireSeq      uintptr
	fireHandlers = map[uintptr]func(){}
)

func addFireHandler(onPress func()) uintptr {
	fireMu.Lock()
	defer fireMu.Unlock()
	fireSeq++
	fireHandlers[fireSeq] = onPress
	return fireSeq
}

func removeFireHandler(handle uintptr) {
	fireMu.Lock()
	defer fireMu.Unlock()
	delete(fireHandlers, handle)
}

//export goHotkeyFire
func goHotkeyFire(handle C.uintptr_t) {
	fireMu.Lock()
	onPress := fireHandlers[uintptr(handle)]
	fireMu.Unlock()
	if onPress != nil {
		go onPress()
	}
}

type linuxRegistration struct {
	once    sync.Once
	writeFD int
	done    chan struct{}
}

func (r *linuxRegistration) Unregister() error {
	r.once.Do(func() {
		_, _ = syscall.Write(r.writeFD, []byte{0})
		<-r.done
		_ = syscall.Close(r.writeFD)
	})
	return nil
}

func registerHotkeyBinding(binding Binding, onPress func()) (Registration, error) {
	var baseMod C.uint
	for _, m := range binding.Modifiers {
		baseMod |= C.uint(m)
	}
	variants := hotkeyLockVariants(binding.Modifiers)
	grabMods := make([]C.uint, 0, len(variants))
	for _, mods := range variants {
		var mask C.uint
		for _, m := range mods {
			mask |= C.uint(m)
		}
		grabMods = append(grabMods, mask)
	}

	var fds [2]int
	if err := syscall.Pipe(fds[:]); err != nil {
		return nil, err
	}

	handle := addFireHandler(onPress)
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		C.lynRunHotkey(
			C.uintptr_t(handle),
			C.int(binding.Key),
			baseMod,
			&grabMods[0],
			C.int(len(grabMods)),
			C.int(fds[0]),
		)
		runtime.KeepAlive(grabMods)
		_ = syscall.Close(fds[0])
		removeFireHandler(handle)
		close(done)
	}()

	return &linuxRegistration{writeFD: fds[1], done: done}, nil
}
