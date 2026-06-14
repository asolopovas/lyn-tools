package lyn

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldReplaceCache(t *testing.T) {
	cases := []struct {
		name        string
		scanError   error
		appError    error
		skipped     []string
		wantReplace bool
	}{
		{name: "clean scan replaces", wantReplace: true},
		{name: "scan error preserves", scanError: errors.New("timeout")},
		{name: "application error preserves", appError: errors.New("apps")},
		{name: "skipped root preserves", skipped: []string{"/missing"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldReplaceCache(tc.scanError, tc.appError, tc.skipped); got != tc.wantReplace {
				t.Fatalf("shouldReplaceCache = %v, want %v", got, tc.wantReplace)
			}
		})
	}
}

func TestRestartArgsAddsStartHidden(t *testing.T) {
	args := restartArgs([]string{"--debug"})
	if len(args) != 2 || args[0] != "--debug" || args[1] != "--start-hidden" {
		t.Fatalf("unexpected restart args %#v", args)
	}
}

func TestRestartArgsDoesNotDuplicateStartHidden(t *testing.T) {
	args := restartArgs([]string{"--start-hidden", "--debug"})
	if len(args) != 2 || args[0] != "--start-hidden" || args[1] != "--debug" {
		t.Fatalf("unexpected restart args %#v", args)
	}
}

func TestRestartStartsProcessThenQuits(t *testing.T) {
	originalStart := startRestartProcess
	originalQuit := quitRuntime
	defer func() {
		startRestartProcess = originalStart
		quitRuntime = originalQuit
	}()
	started := false
	quit := false
	startRestartProcess = func([]string) error {
		started = true
		return nil
	}
	quitRuntime = func(context.Context) {
		quit = true
	}
	app := NewApp()
	app.ctx = context.Background()
	app.Restart()
	if !started || !quit {
		t.Fatalf("expected started and quit, got started=%v quit=%v", started, quit)
	}
}

func TestConfigReloadsChangesSavedBySettingsProcess(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("AppData", configDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	initial := DefaultConfig()
	savedInitial, err := SaveConfig(initial)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.UseConfig(savedInitial)
	external := savedInitial
	external.UI.Theme = "tron-legacy"
	external, err = SaveConfig(external)
	if err != nil {
		t.Fatal(err)
	}
	if external.UI.Theme != "tron-legacy" {
		t.Fatalf("expected saved theme, got %q", external.UI.Theme)
	}
	reloaded := app.Config()
	if reloaded.UI.Theme != "tron-legacy" {
		t.Fatalf("expected reloaded theme, got %q", reloaded.UI.Theme)
	}
}

func TestProjectsReadsVSCodeRecentsLiveWithoutCopyingToStore(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("AppData", appData)
	t.Setenv("XDG_CONFIG_HOME", appData)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	storage := filepath.Join(appData, "Code", "User", "globalStorage")
	dbPath := filepath.Join(storage, "state.vscdb")
	if err := os.MkdirAll(storage, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE ItemTable(key TEXT PRIMARY KEY, value TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO ItemTable(key, value) VALUES(?, ?)", "history.recentlyOpenedPathsList", `{"entries":[{"label":"live","folderUri":"vscode-remote://ssh-remote%2Blivehost/home/deploy/live"}]}`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cached := newProject("cached", filepath.Join(t.TempDir(), "cached"), projectKindGo)
	if err := store.UpsertProjects(t.Context(), []Project{cached}); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.ctx = t.Context()
	app.store = store
	indexed, err := app.projects()
	if err != nil {
		t.Fatal(err)
	}
	if len(indexed) != 2 {
		t.Fatalf("expected cached and live recent projects in search index, got %#v", indexed)
	}
	items, err := store.ListProjects(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Path != cached.Path {
		t.Fatalf("expected VS Code recent project to stay out of local store, got %#v", items)
	}
}

func TestProjectsReturnsCachedItemsBeforeScanning(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cached := Project{Name: "Cached", Path: "/tmp/cached", Kind: "go", DetectedAt: time.Now().UTC()}
	if err := store.UpsertProjects(ctx, []Project{cached}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/live\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.ctx = ctx
	app.store = store
	app.config = DefaultConfig()
	app.config.Scanner.Roots = []string{root}
	items, err := app.Projects()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Path != cached.Path {
		t.Fatalf("unexpected projects %#v", items)
	}
}
