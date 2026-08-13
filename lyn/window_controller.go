package lyn

import (
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type windowController struct {
	mu                 sync.Mutex
	ctx                context.Context
	shown              bool
	showSequence       uint64
	autoHideSuppressed bool
	quitting           bool
	mode               WindowMode
	debugLog           func(string, ...any)
}

func newWindowController(debugLog func(string, ...any)) *windowController {
	return &windowController{shown: true, mode: LauncherWindowMode, debugLog: debugLog}
}

func (w *windowController) setContext(ctx context.Context) {
	w.mu.Lock()
	w.ctx = ctx
	w.mu.Unlock()
}

func (w *windowController) clearContext() {
	w.mu.Lock()
	w.ctx = nil
	w.mu.Unlock()
}

func (w *windowController) setMode(mode WindowMode) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if mode == SettingsWindowMode {
		w.mode = mode
		return
	}
	w.mode = LauncherWindowMode
}

func (w *windowController) windowMode() WindowMode {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.mode == SettingsWindowMode {
		return SettingsWindowMode
	}
	return LauncherWindowMode
}

func (w *windowController) show() {
	w.debugLog("window.show.request")
	ctx, sequence, ok := w.beginShow()
	if ok {
		w.showLauncher(ctx, sequence)
	}
}

func (w *windowController) beginShow() (context.Context, uint64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx == nil {
		return nil, 0, false
	}
	w.showSequence++
	w.shown = true
	return w.ctx, w.showSequence, true
}

func (w *windowController) hide() {
	w.debugLog("window.hide.request")
	ctx, ok := w.beginHide()
	if ok {
		windowHide(ctx)
	}
}

func (w *windowController) beginHide() (context.Context, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx == nil {
		return nil, false
	}
	w.shown = false
	w.showSequence++
	return w.ctx, true
}

func (w *windowController) toggle() {
	w.debugLog("window.toggle.request")
	w.mu.Lock()
	if w.ctx == nil {
		w.mu.Unlock()
		return
	}
	if w.shouldHideOnToggleLocked() {
		ctx := w.ctx
		w.shown = false
		w.showSequence++
		w.mu.Unlock()
		windowHide(ctx)
		w.debugLog("window.toggle.hidden")
		return
	}
	w.showSequence++
	sequence := w.showSequence
	ctx := w.ctx
	w.shown = true
	w.mu.Unlock()
	w.showLauncher(ctx, sequence)
}

func (w *windowController) shouldHideOnToggleLocked() bool {
	return w.shown && !isWindowMinimized()
}

func (w *windowController) shouldHideOnFocusLossLocked(sequence uint64) bool {
	return w.ctx != nil && w.shown && w.showSequence == sequence && !w.autoHideSuppressed
}

func (w *windowController) setAutoHideSuppressed(suppressed bool) {
	w.mu.Lock()
	w.autoHideSuppressed = suppressed
	w.mu.Unlock()
}

func (w *windowController) markStartupHidden(ctx context.Context) {
	runtime.WindowHide(ctx)
	w.mu.Lock()
	w.shown = false
	w.mu.Unlock()
}

func (w *windowController) activateInitial(ctx context.Context) {
	w.mu.Lock()
	sequence := w.showSequence
	w.mu.Unlock()
	w.focusAndActivate(ctx)
	w.startAutoHideMonitor(ctx, sequence)
}

func (w *windowController) isQuitting() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.quitting
}

func (w *windowController) requestQuit(ctx context.Context) {
	w.mu.Lock()
	w.quitting = true
	w.mu.Unlock()
	quitRuntime(ctx)
}

func (w *windowController) currentContext() context.Context {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ctx
}

func (w *windowController) showLauncher(ctx context.Context, sequence uint64) {
	w.debugLog("window.show.begin", "sequence", sequence)
	ready := make(chan struct{}, 1)
	cancelReady := runtime.EventsOnce(ctx, "launcher-ready", func(optionalData ...any) {
		if len(optionalData) == 0 || sequenceValue(optionalData[0]) != sequence {
			return
		}
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	defer cancelReady()
	runtime.EventsEmit(ctx, "launcher-shown", sequence)
	readyState := "ack"
	select {
	case <-ready:
	case <-time.After(250 * time.Millisecond):
		readyState = "timeout"
	}
	w.debugLog("window.show.frontend", "sequence", sequence, "ready", readyState)
	w.mu.Lock()
	current := w.showSequence == sequence && w.shown
	w.mu.Unlock()
	if !current {
		w.debugLog("window.show.cancelled", "sequence", sequence)
		return
	}
	runtime.WindowUnminimise(ctx)
	prepareWindowActivation()
	runtime.WindowShow(ctx)
	w.debugLog("window.show.end", "sequence", sequence)
	w.focusAndActivate(ctx)
	w.startAutoHideMonitor(ctx, sequence)
	w.startFocusRetries(ctx, sequence)
}

func (w *windowController) focusAndActivate(ctx context.Context) {
	activateWindow()
	runtime.WindowExecJS(ctx, focusQueryScript())
}

func (w *windowController) startFocusRetries(ctx context.Context, sequence uint64) {
	go func() {
		for _, delay := range []time.Duration{60, 140, 280, 520} {
			time.Sleep(delay * time.Millisecond)
			w.mu.Lock()
			current := w.showSequence == sequence && w.shown
			w.mu.Unlock()
			if !current {
				return
			}
			w.focusAndActivate(ctx)
		}
	}()
}

func (w *windowController) startAutoHideMonitor(ctx context.Context, sequence uint64) {
	go func() {
		timer := time.NewTimer(350 * time.Millisecond)
		defer timer.Stop()
		<-timer.C
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		misses := 0
		for range ticker.C {
			w.mu.Lock()
			current := w.shouldHideOnFocusLossLocked(sequence)
			suppressed := w.autoHideSuppressed
			w.mu.Unlock()
			if !current {
				return
			}
			if suppressed {
				misses = 0
				continue
			}
			if didPointerActivateOutsideWindow() {
				if w.hideAfterFocusLoss(ctx, sequence) {
					return
				}
				continue
			}
			if isWindowForeground() {
				misses = 0
				continue
			}
			misses++
			if misses < 2 {
				continue
			}
			if w.hideAfterFocusLoss(ctx, sequence) {
				return
			}
		}
	}()
}

func (w *windowController) hideAfterFocusLoss(ctx context.Context, sequence uint64) bool {
	w.mu.Lock()
	if !w.shouldHideOnFocusLossLocked(sequence) {
		w.mu.Unlock()
		return false
	}
	w.shown = false
	w.showSequence++
	w.mu.Unlock()
	windowHide(ctx)
	w.debugLog("window.hide.focus-loss", "sequence", sequence)
	return true
}

func sequenceValue(value any) uint64 {
	switch typed := value.(type) {
	case float64:
		return uint64(typed)
	case int:
		return uint64(typed)
	case uint64:
		return typed
	default:
		return 0
	}
}
