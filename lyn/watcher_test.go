package lyn

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestAddWatchPathAddsNestedDirectories(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "project", "src")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	addWatchPath(w, root, 3, 0)
	watches := map[string]bool{}
	for _, path := range w.WatchList() {
		watches[path] = true
	}
	for _, path := range []string{root, filepath.Join(root, "project"), nested} {
		if !watches[path] {
			t.Fatalf("expected watch for %q in %#v", path, w.WatchList())
		}
	}
}

func TestAddWatchPathSkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	ignored := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(ignored, 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	addWatchPath(w, root, 3, 0)
	for _, path := range w.WatchList() {
		if path == ignored {
			t.Fatalf("unexpected watch for %q", ignored)
		}
	}
}

func TestWatchRootsIncludesApplicationAndVSCodeRecentDirs(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	cfg := DefaultConfig().Scanner
	cfg.Roots = []string{"~/src"}
	roots := watchRoots(cfg, "linux")
	foundApplications := false
	foundVSCodeStorage := false
	foundVSCodeSharedStorage := false
	for _, root := range roots {
		if filepath.ToSlash(root) == "/usr/share/applications" {
			foundApplications = true
		}
		if root == filepath.Join(configHome, "Code", "User", "globalStorage") {
			foundVSCodeStorage = true
		}
		if filepath.Base(filepath.Dir(root)) == ".vscode-shared" && filepath.Base(root) == "sharedStorage" {
			foundVSCodeSharedStorage = true
		}
	}
	if !foundApplications || !foundVSCodeStorage || !foundVSCodeSharedStorage {
		t.Fatalf("expected application and VS Code recent roots in %#v", roots)
	}
}

func TestStartWatcherEmitsDebouncedChange(t *testing.T) {
	root := t.TempDir()
	cfg := DefaultConfig().Scanner
	cfg.Roots = []string{root}
	cfg.MaxDepth = 2
	cfg.Watch = true
	changes := make(chan struct{}, 1)
	w, err := StartWatcher(t.Context(), cfg, func() {
		select {
		case changes <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.test/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-changes:
	case <-time.After(4 * time.Second):
		t.Fatal("expected watcher change")
	}
}

func TestStartWatcherThrottlesContinuousChurn(t *testing.T) {
	original := watcherMinScanInterval
	watcherMinScanInterval = 500 * time.Millisecond
	defer func() { watcherMinScanInterval = original }()

	root := t.TempDir()
	cfg := DefaultConfig().Scanner
	cfg.Roots = []string{root}
	cfg.MaxDepth = 1
	cfg.Watch = true
	var scans atomic.Int32
	w, err := StartWatcher(t.Context(), cfg, func() { scans.Add(1) })
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	deadline := time.After(1600 * time.Millisecond)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	i := 0
loop:
	for {
		select {
		case <-deadline:
			break loop
		case <-ticker.C:
			i++
			_ = os.WriteFile(filepath.Join(root, "churn.txt"), []byte{byte(i)}, 0o644)
		}
	}
	if got := scans.Load(); got < 1 {
		t.Fatal("expected at least one scan")
	}
	if got := scans.Load(); got > 5 {
		t.Fatalf("expected throttle to cap scans over ~1.6s of churn, got %d", got)
	}
}

func TestStartWatcherDetectsVSCodeRecentStorage(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("AppData", appData)
	t.Setenv("XDG_CONFIG_HOME", appData)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	storage := filepath.Join(appData, "Code", "User", "globalStorage")
	if err := os.MkdirAll(storage, 0o755); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	cfg := DefaultConfig().Scanner
	cfg.Roots = []string{root}
	cfg.MaxDepth = 1
	cfg.Watch = true
	changes := make(chan struct{}, 1)
	w, err := StartWatcher(t.Context(), cfg, func() {
		select {
		case changes <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(filepath.Join(storage, "state.vscdb-wal"), []byte("change"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-changes:
	case <-time.After(4 * time.Second):
		t.Fatal("expected watcher change after VS Code recent storage changed")
	}
}
