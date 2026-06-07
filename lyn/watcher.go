package lyn

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	inner  *fsnotify.Watcher
	cancel context.CancelFunc
	done   chan struct{}
}

func StartWatcher(parent context.Context, cfg ScannerConfig, onChange func()) (*Watcher, error) {
	if parent == nil || !cfg.Watch {
		return nil, nil
	}
	inner, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	w := &Watcher{inner: inner, cancel: cancel, done: make(chan struct{})}
	if err := addWatchRoots(inner, cfg); err != nil {
		cancel()
		inner.Close()
		return nil, err
	}
	go w.run(ctx, cfg, onChange)
	return w, nil
}

func (w *Watcher) Close() error {
	if w == nil {
		return nil
	}
	w.cancel()
	err := w.inner.Close()
	<-w.done
	return err
}

func (w *Watcher) run(ctx context.Context, cfg ScannerConfig, onChange func()) {
	defer close(w.done)
	const debounceDelay = 200 * time.Millisecond
	var timer *time.Timer
	var timerC <-chan time.Time
	schedule := func() {
		if timer == nil {
			timer = time.NewTimer(debounceDelay)
			timerC = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(debounceDelay)
		timerC = timer.C
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.inner.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				addWatchPath(w.inner, event.Name, cfg.MaxDepth, 0)
			}
			schedule()
		case _, ok := <-w.inner.Errors:
			if !ok {
				return
			}
		case <-timerC:
			timerC = nil
			if onChange != nil {
				onChange()
			}
		}
	}
}

func addWatchRoots(w *fsnotify.Watcher, cfg ScannerConfig) error {
	var firstErr error
	added := false
	for _, root := range watchRoots(cfg, runtime.GOOS) {
		if err := addWatchPath(w, root, cfg.MaxDepth, 0); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		added = true
	}
	if !added {
		return firstErr
	}
	return nil
}

func watchRoots(cfg ScannerConfig, goos string) []string {
	roots := expandRoots(cfg.Roots)
	roots = append(roots, applicationDirs(goos)...)
	roots = append(roots, vscodeStorageDirs(goos)...)
	roots = append(roots, vscodeSharedStorageDirs(goos)...)
	return cleanUniquePaths(roots)
}

func addWatchPath(w *fsnotify.Watcher, path string, maxDepth int, depth int) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return w.Add(filepath.Dir(path))
	}
	if shouldSkip(filepath.Base(path)) {
		return nil
	}
	if err := w.Add(path); err != nil {
		return err
	}
	if depth >= maxDepth {
		return nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || shouldSkip(entry.Name()) {
			continue
		}
		if err := addWatchPath(w, filepath.Join(path, entry.Name()), maxDepth, depth+1); err != nil {
			return err
		}
	}
	return nil
}
