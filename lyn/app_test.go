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

func TestRestartArgsStripsStaleAwaitFlag(t *testing.T) {
	args := restartArgs([]string{awaitExitFlag, "999", "--debug", "--start-hidden"})
	if len(args) != 2 || args[0] != "--debug" || args[1] != "--start-hidden" {
		t.Fatalf("unexpected args %#v", args)
	}
}

func TestAwaitExitPID(t *testing.T) {
	pid, ok := awaitExitPID([]string{"--debug", awaitExitFlag, "1234", "--start-hidden"})
	if !ok || pid != 1234 {
		t.Fatalf("unexpected pid %d ok %v", pid, ok)
	}
	if _, ok := awaitExitPID([]string{"--debug"}); ok {
		t.Fatal("did not expect pid")
	}
	if _, ok := awaitExitPID([]string{awaitExitFlag}); ok {
		t.Fatal("did not expect pid without value")
	}
	if _, ok := awaitExitPID([]string{awaitExitFlag, "0"}); ok {
		t.Fatal("did not expect non-positive pid")
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
	app.window.setContext(app.ctx)
	app.Restart()
	if !started || !quit {
		t.Fatalf("expected started and quit, got started=%v quit=%v", started, quit)
	}
}

func TestBeforeCloseMinimizesToTray(t *testing.T) {
	originalHide := windowHide
	defer func() { windowHide = originalHide }()
	hidden := false
	windowHide = func(context.Context) { hidden = true }
	app := NewApp()
	app.window.setContext(context.Background())
	if prevent := app.BeforeClose(context.Background()); !prevent {
		t.Fatal("expected BeforeClose to prevent the close")
	}
	if !hidden {
		t.Fatal("expected window to be hidden")
	}
	if app.window.shown {
		t.Fatal("expected shown to be false after minimize")
	}
}

func TestBeforeCloseAllowsQuit(t *testing.T) {
	originalHide := windowHide
	originalQuit := quitRuntime
	defer func() {
		windowHide = originalHide
		quitRuntime = originalQuit
	}()
	hidden := false
	windowHide = func(context.Context) { hidden = true }
	quitRuntime = func(context.Context) {}
	app := NewApp()
	app.ctx = context.Background()
	app.window.setContext(app.ctx)
	app.Quit()
	if prevent := app.BeforeClose(context.Background()); prevent {
		t.Fatal("expected BeforeClose to allow the close after Quit")
	}
	if hidden {
		t.Fatal("expected no minimize when quitting")
	}
}

func TestBeforeCloseAllowsSettingsWindow(t *testing.T) {
	originalHide := windowHide
	defer func() { windowHide = originalHide }()
	windowHide = func(context.Context) { t.Fatal("settings window must not minimize to tray") }
	app := NewApp()
	app.SetWindowMode(SettingsWindowMode)
	app.ctx = context.Background()
	if prevent := app.BeforeClose(context.Background()); prevent {
		t.Fatal("expected settings window close to be allowed")
	}
}

func TestHidePathsReleaseStateMuBeforeWindowHide(t *testing.T) {
	originalHide := windowHide
	defer func() { windowHide = originalHide }()

	check := func(t *testing.T, app *App, action func()) {
		t.Helper()
		called := false
		held := false
		windowHide = func(context.Context) {
			called = true
			if app.window.mu.TryLock() {
				app.window.mu.Unlock()
				return
			}
			held = true
		}
		action()
		if !called {
			t.Fatal("expected windowHide to be called")
		}
		if held {
			t.Fatal("stateMu held during windowHide; GUI-thread close can deadlock")
		}
	}

	t.Run("Hide", func(t *testing.T) {
		app := NewApp()
		app.window.setContext(context.Background())
		check(t, app, app.Hide)
	})
	t.Run("Toggle", func(t *testing.T) {
		app := NewApp()
		app.window.setContext(context.Background())
		app.window.shown = true
		check(t, app, app.Toggle)
	})
	t.Run("hideAfterFocusLoss", func(t *testing.T) {
		app := NewApp()
		ctx := context.Background()
		app.window.setContext(ctx)
		app.window.shown = true
		check(t, app, func() { app.window.hideAfterFocusLoss(ctx, app.window.showSequence) })
	})
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
	app.projects.configure(t.Context(), DefaultConfig())
	app.projects.setStore(store)
	indexed, err := app.projects.projects()
	if err != nil {
		t.Fatal(err)
	}
	indexed = withoutSystemCommands(indexed)
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
	config := DefaultConfig()
	config.Scanner.Roots = []string{root}
	app.UseConfig(config)
	app.projects.configure(ctx, config)
	app.projects.setStore(store)
	items, err := app.Projects()
	if err != nil {
		t.Fatal(err)
	}
	items = withoutSystemCommands(items)
	if len(items) != 1 || items[0].Path != cached.Path {
		t.Fatalf("unexpected projects %#v", items)
	}
}

func withoutSystemCommands(projects []Project) []Project {
	filtered := make([]Project, 0, len(projects))
	for _, project := range projects {
		if project.Kind == projectKindSystemCommand {
			continue
		}
		filtered = append(filtered, project)
	}
	return filtered
}
