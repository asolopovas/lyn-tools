package lyn

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"lyn.tools/launcher/lyn/hotkey"
	"lyn.tools/launcher/lyn/launch"
	"lyn.tools/launcher/lyn/startup"
	"lyn.tools/launcher/lyn/tray"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx                   context.Context
	config                Config
	store                 *Store
	hotkey                hotkey.Registration
	watcher               *Watcher
	stateMu               sync.Mutex
	launchSelectionMu     sync.Mutex
	launchSelection       launch.Request
	searchIndex           []searchProject
	keyInterceptorCleanup func()
	scanMutex             sync.Mutex
	adminSessionMu        sync.Mutex
	adminSession          *adminSession
	shown                 bool
	showSequence          uint64
	autoHideSuppressed    bool
	quitting              bool
	debug                 *DebugLogger
	mode                  WindowMode
}

type ScanResult struct {
	Count int    `json:"count"`
	Error string `json:"error,omitempty"`
}

func NewApp(debug ...*DebugLogger) *App {
	config := DefaultConfig()
	app := &App{config: config, shown: true, mode: LauncherWindowMode}
	if len(debug) > 0 {
		app.debug = debug[0]
	}
	return app
}

func (a *App) UseConfig(config Config) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	a.config = config
}

func (a *App) StartHidden() bool {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.mode != SettingsWindowMode && a.config.Startup.Enabled && a.config.Startup.StartHidden
}

func (a *App) Startup(ctx context.Context) {
	a.startup(ctx)
}

func (a *App) Shutdown(ctx context.Context) {
	a.shutdown(ctx)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.debugLog("startup.begin")
	config, err := LoadConfig("")
	if err == nil {
		a.config = config
		a.debugLog("config.loaded", "path", config.Path, "hotkey", config.Hotkey.Binding, "roots", len(config.Scanner.Roots), "watch", config.Scanner.Watch)
	} else {
		a.debugLog("config.error", "error", err)
	}
	store, err := OpenStore(filepath.Join(a.config.Cache.Dir, "lyn.db"))
	if err == nil {
		a.store = store
		a.debugLog("store.opened", "dir", a.config.Cache.Dir)
	} else {
		a.debugLog("store.error", "error", err)
	}
	if a.windowMode() == SettingsWindowMode {
		a.markSettingsWindowActive()
		placeWindow(a.ctx)
		a.debugLog("startup.settings.shown")
		return
	}
	a.registerHotkey()
	a.registerLaunchKeyInterceptor()
	a.startWatcher()
	configureWindowAppearance()
	installCloseToTray(a.isQuitting, a.minimizeToTray)
	tray.Start(a)
	a.startFastRefresh()
	if a.config.Startup.Enabled && a.config.Startup.StartHidden {
		runtime.WindowHide(a.ctx)
		a.shown = false
		a.debugLog("startup.hidden")
		return
	}
	placeWindow(a.ctx)
	prepareWindowActivation()
	a.focusAndActivate(a.ctx)
	a.startWindowAutoHideMonitor(a.ctx, a.showSequence)
	a.debugLog("startup.shown")
}

func (a *App) startFastRefresh() {
	ctx, _, store := a.snapshot()
	if ctx == nil || store == nil {
		return
	}
	go func() {
		a.debugLog("fast-refresh.begin")
		_, sourceError, err := a.rescan()
		if err != nil {
			a.debugLog("fast-refresh.error", "error", err)
			logRuntimeError(ctx, err)
			return
		}
		logRuntimeError(ctx, sourceError)
		a.debugLog("fast-refresh.end")
		runtime.EventsEmit(ctx, "projects-updated")
	}()
}

func (a *App) shutdown(context.Context) {
	a.debugLog("shutdown.begin")
	if a.windowMode() == SettingsWindowMode {
		a.clearSettingsWindowActive()
	}
	a.stateMu.Lock()
	ctx := a.ctx
	hotkey := a.hotkey
	watcher := a.watcher
	store := a.store
	keyInterceptorCleanup := a.keyInterceptorCleanup
	a.hotkey = nil
	a.watcher = nil
	a.store = nil
	a.keyInterceptorCleanup = nil
	a.stateMu.Unlock()
	if hotkey != nil {
		logRuntimeError(ctx, hotkey.Unregister())
	}
	if watcher != nil {
		logRuntimeError(ctx, watcher.Close())
	}
	if keyInterceptorCleanup != nil {
		keyInterceptorCleanup()
	}
	a.stopAdminSession()
	tray.Stop()
	if store != nil {
		logRuntimeError(ctx, store.Close())
	}
	a.debugLog("shutdown.end")
	if a.debug != nil {
		a.debug.Close()
	}
}

func (a *App) Config() Config {
	if err := a.reloadConfigFromDisk(); err != nil {
		a.debugLog("config.reload.error", "error", err)
	}
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.config
}

func (a *App) reloadConfigFromDisk() error {
	a.stateMu.Lock()
	path := a.config.Path
	a.stateMu.Unlock()
	loaded, err := LoadConfig(path)
	if err != nil {
		return err
	}
	a.reactToConfigChange(loaded)
	return nil
}

func (a *App) reactToConfigChange(loaded Config) {
	a.stateMu.Lock()
	old := a.config
	if reflect.DeepEqual(old, loaded) {
		a.stateMu.Unlock()
		return
	}
	ctx := a.ctx
	oldHotkey := a.hotkey
	launcherMode := a.mode != SettingsWindowMode
	hotkeyChanged := old.Hotkey.Binding != loaded.Hotkey.Binding
	scannerChanged := !reflect.DeepEqual(old.Scanner, loaded.Scanner)
	a.config = loaded
	if launcherMode && hotkeyChanged {
		a.hotkey = nil
	}
	a.stateMu.Unlock()
	a.debugLog("config.reloaded", "path", loaded.Path, "theme", loaded.UI.Theme)
	if !launcherMode || ctx == nil {
		return
	}
	if hotkeyChanged {
		if oldHotkey != nil {
			logRuntimeError(ctx, oldHotkey.Unregister())
		}
		a.registerHotkey()
	}
	if scannerChanged {
		a.restartWatcher()
		go func() {
			_, sourceError, err := a.rescan()
			logRuntimeError(ctx, firstError(err, sourceError))
			runtime.EventsEmit(ctx, "projects-updated")
		}()
	}
}

func (a *App) snapshot() (context.Context, Config, *Store) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.ctx, a.config, a.store
}

func (a *App) currentSearchIndex() []searchProject {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.searchIndex
}

func (a *App) setSearchIndex(projects []Project) {
	index := newSearchIndex(projects)
	a.stateMu.Lock()
	a.searchIndex = index
	a.stateMu.Unlock()
}

func (a *App) indexProjects(cacheable, recent []Project) []Project {
	items := mergeProjects(cacheable, recent)
	a.setSearchIndex(items)
	return items
}

func scanAppSources(ctx context.Context) (apps, recent []Project, appErr, recentErr error) {
	apps, appErr = ScanApplications(ctx)
	recent, recentErr = ScanVSCodeRecentProjects(ctx)
	return apps, recent, appErr, recentErr
}

func (a *App) updateSearchIndexLaunch(path string) {
	index := a.currentSearchIndex()
	if len(index) == 0 {
		return
	}
	now := time.Now().UTC()
	projects := make([]Project, 0, len(index))
	for _, item := range index {
		project := item.project
		if project.Path == path {
			project.UsageCount++
			project.LastLaunchedAt = now
		}
		projects = append(projects, project)
	}
	a.setSearchIndex(projects)
}

func (a *App) SaveConfig(config Config) (Config, error) {
	a.debugLog("config.save.begin", "hotkey", config.Hotkey.Binding, "roots", len(config.Scanner.Roots), "watch", config.Scanner.Watch)
	saved, err := SaveConfig(config)
	if err != nil {
		a.debugLog("config.save.error", "error", err)
		return saved, err
	}
	if err := startup.Configure(saved.Startup.Enabled); err != nil {
		return saved, err
	}
	a.reactToConfigChange(saved)
	a.debugLog("config.save.end", "path", saved.Path)
	return saved, nil
}

func (a *App) Projects() ([]Project, error) {
	return a.projects()
}

func (a *App) SearchProjects(query string) ([]Project, error) {
	index := a.currentSearchIndex()
	if len(index) == 0 {
		projects, err := a.projects()
		if err != nil {
			return projects, err
		}
		index = a.currentSearchIndex()
	}
	_, config, _ := a.snapshot()
	return searchProjectIndex(index, query, config.UI.WorkspaceQueryShortcut), nil
}

func (a *App) projects() ([]Project, error) {
	ctx, _, store := a.snapshot()
	if store != nil {
		projects, err := store.ListProjects(ctx)
		if err != nil {
			return nil, err
		}
		if len(projects) > 0 {
			return a.indexProjects(projects, a.liveRecents(ctx)), nil
		}
	}
	items, sourceError, err := a.rescan()
	if err != nil {
		return items, err
	}
	return items, sourceError
}

func (a *App) RefreshProjects() ([]Project, error) {
	ctx, _, store := a.snapshot()
	apps, recent, applicationError, recentProjectError := scanAppSources(ctx)
	sourceError := firstError(applicationError, recentProjectError)
	if store == nil {
		return a.indexProjects(apps, recent), sourceError
	}
	if err := updateProjectKind(ctx, store, apps, projectKindApp, applicationError); err != nil {
		return nil, err
	}
	projects, err := store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	return a.indexProjects(mergeProjects(projects, apps), recent), sourceError
}

func updateProjectKind(ctx context.Context, store *Store, projects []Project, kind string, sourceError error) error {
	if sourceError == nil {
		return store.ReplaceProjectKinds(ctx, projects, kind)
	}
	if len(projects) > 0 {
		return store.UpsertProjects(ctx, projects)
	}
	return nil
}

func (a *App) Scan() ScanResult {
	items, sourceError, err := a.rescan()
	if err != nil {
		return ScanResult{Count: len(items), Error: err.Error()}
	}
	if sourceError != nil {
		return ScanResult{Count: len(items), Error: sourceError.Error()}
	}
	return ScanResult{Count: len(items)}
}

var launchRequest = launch.Launch

var launchAsync = func(fn func()) { go fn() }

func (a *App) Launch(request launch.Request) launch.Result {
	a.debugLog("launch.begin", "action", launch.NormalizedAction(request.Action), "path", request.Path)
	result := a.dispatchLaunch(request)
	a.debugLog("launch.end", "command", result.Command, "args", strings.Join(result.Args, " "), "error", result.Error)
	if result.Error == "" && request.Action != "reveal" {
		launchAsync(func() { a.recordLaunch(request.Path) })
	}
	return result
}

func (a *App) dispatchLaunch(request launch.Request) launch.Result {
	if session := a.adminLaunchSession(request); session != nil {
		return session.run(request)
	}
	return launchRequest(request)
}

func (a *App) adminLaunchSession(request launch.Request) *adminSession {
	if !shouldUseAdminHelper(request) {
		return nil
	}
	a.adminSessionMu.Lock()
	defer a.adminSessionMu.Unlock()
	if a.adminSession == nil {
		a.adminSession = newAdminLaunchSession(launchRequest, a.debugLog)
	}
	return a.adminSession
}

func (a *App) stopAdminSession() {
	a.adminSessionMu.Lock()
	session := a.adminSession
	a.adminSession = nil
	a.adminSessionMu.Unlock()
	if session != nil {
		session.stop()
	}
}

func (a *App) recordLaunch(path string) {
	ctx, _, store := a.snapshot()
	if store == nil {
		return
	}
	logRuntimeError(ctx, store.RecordLaunch(ctx, path))
	a.updateSearchIndexLaunch(path)
}

func (a *App) SetLaunchSelection(request launch.Request) {
	request.Path = strings.TrimSpace(request.Path)
	request.Action = launch.NormalizedAction(request.Action)
	a.launchSelectionMu.Lock()
	a.launchSelection = request
	a.launchSelectionMu.Unlock()
	a.debugLog("launch.selection", "action", request.Action, "path", request.Path)
}

func (a *App) Debug(stage string, detail string) {
	a.debugLog("frontend."+stage, "detail", detail)
}

func (a *App) ChooseFolder() (string, error) {
	ctx, _, _ := a.snapshot()
	if ctx == nil {
		return "", nil
	}
	a.setWindowAutoHideSuppressed(true)
	defer a.setWindowAutoHideSuppressed(false)
	return runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{Title: "Add indexed folder"})
}

func (a *App) ChooseWSLFolder() (WSLRoot, error) {
	ctx, _, _ := a.snapshot()
	if ctx == nil {
		return WSLRoot{}, nil
	}
	a.setWindowAutoHideSuppressed(true)
	defer a.setWindowAutoHideSuppressed(false)
	selected, err := runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{Title: "Add WSL folder", DefaultDirectory: wslDialogStartFolder()})
	if err != nil || strings.TrimSpace(selected) == "" {
		return WSLRoot{}, err
	}
	distro, unix, ok := wslUnixFromUNC(selected)
	if !ok {
		a.debugLog("wsl.folder.invalid", "path", selected)
		return WSLRoot{}, fmt.Errorf("%q is not a WSL folder under %s", selected, wslLocalhostPrefix)
	}
	a.debugLog("wsl.folder.added", "distro", distro, "path", unix)
	return WSLRoot{Distro: distro, Path: unix}, nil
}

func (a *App) WSLDistros() []string {
	return listWSLDistros()
}

func (a *App) Show() {
	a.debugLog("window.show.request")
	a.stateMu.Lock()
	if a.ctx == nil {
		a.stateMu.Unlock()
		return
	}
	a.showSequence++
	sequence := a.showSequence
	ctx := a.ctx
	a.shown = true
	a.stateMu.Unlock()
	a.showLauncher(ctx, sequence)
}

func (a *App) ShowSettings() {
	a.Show()
	ctx, _, _ := a.snapshot()
	if ctx != nil {
		runtime.EventsEmit(ctx, "open-settings")
	}
}

func (a *App) Hide() {
	a.debugLog("window.hide.request")
	a.stateMu.Lock()
	if a.ctx == nil {
		a.stateMu.Unlock()
		return
	}
	ctx := a.ctx
	a.shown = false
	a.showSequence++
	a.stateMu.Unlock()
	windowHide(ctx)
}

var (
	quitRuntime = runtime.Quit
	windowHide  = runtime.WindowHide
)

func (a *App) BeforeClose(context.Context) bool {
	if a.isQuitting() || a.windowMode() == SettingsWindowMode {
		a.debugLog("window.close.allow")
		return false
	}
	a.minimizeToTray()
	return true
}

func (a *App) minimizeToTray() {
	a.debugLog("window.close.minimize")
	a.Hide()
}

func (a *App) isQuitting() bool {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.quitting
}

func (a *App) requestQuit(ctx context.Context) {
	a.stateMu.Lock()
	a.quitting = true
	a.stateMu.Unlock()
	quitRuntime(ctx)
}

func (a *App) Quit() {
	ctx, _, _ := a.snapshot()
	if ctx != nil {
		a.requestQuit(ctx)
	}
}

func (a *App) Restart() {
	a.debugLog("restart.request")
	ctx, _, _ := a.snapshot()
	if ctx == nil {
		return
	}
	if err := startRestartProcess(os.Args[1:]); err != nil {
		a.debugLog("restart.error", "error", err)
		logRuntimeError(ctx, err)
		return
	}
	a.debugLog("restart.started")
	a.requestQuit(ctx)
}

func (a *App) Toggle() {
	a.debugLog("window.toggle.request")
	a.stateMu.Lock()
	if a.ctx == nil {
		a.stateMu.Unlock()
		return
	}
	if a.shouldHideOnToggleLocked() {
		ctx := a.ctx
		a.shown = false
		a.showSequence++
		a.stateMu.Unlock()
		windowHide(ctx)
		a.debugLog("window.toggle.hidden")
		return
	}
	a.showSequence++
	sequence := a.showSequence
	ctx := a.ctx
	a.shown = true
	a.stateMu.Unlock()
	a.showLauncher(ctx, sequence)
}

func (a *App) shouldHideOnToggleLocked() bool {
	return a.shown && !isWindowMinimized()
}

func (a *App) shouldHideOnFocusLossLocked(sequence uint64) bool {
	return a.ctx != nil && a.shown && a.showSequence == sequence && !a.autoHideSuppressed
}

func (a *App) setWindowAutoHideSuppressed(suppressed bool) {
	a.stateMu.Lock()
	a.autoHideSuppressed = suppressed
	a.stateMu.Unlock()
}

func (a *App) registerHotkey() {
	_, config, _ := a.snapshot()
	a.debugLog("hotkey.register.begin", "binding", config.Hotkey.Binding)
	registration, err := hotkey.Register(config.Hotkey.Binding, func() {
		a.debugLog("hotkey.pressed", "binding", config.Hotkey.Binding)
		if a.settingsWindowActive() {
			a.debugLog("hotkey.suppressed", "reason", "settings-active")
			return
		}
		a.Toggle()
	})
	if err != nil {
		a.debugLog("hotkey.register.error", "binding", config.Hotkey.Binding, "error", err)
		logRuntimeError(a.ctx, err)
		return
	}
	a.stateMu.Lock()
	a.hotkey = registration
	a.stateMu.Unlock()
	a.debugLog("hotkey.register.end", "binding", config.Hotkey.Binding)
}

func (a *App) startWatcher() {
	ctx, config, _ := a.snapshot()
	watcher := a.newWatcher(ctx, config.Scanner)
	if watcher == nil {
		return
	}
	a.stateMu.Lock()
	oldWatcher := a.watcher
	a.watcher = watcher
	a.stateMu.Unlock()
	if oldWatcher != nil {
		logRuntimeError(ctx, oldWatcher.Close())
	}
}

func (a *App) restartWatcher() {
	a.stateMu.Lock()
	ctx := a.ctx
	oldWatcher := a.watcher
	a.watcher = nil
	a.stateMu.Unlock()
	if oldWatcher != nil {
		logRuntimeError(ctx, oldWatcher.Close())
	}
	a.startWatcher()
}

func (a *App) newWatcher(ctx context.Context, scanner ScannerConfig) *Watcher {
	if ctx == nil || !scanner.Watch {
		return nil
	}
	watcher, err := StartWatcher(ctx, scanner, func() {
		_, sourceError, err := a.rescan()
		logRuntimeError(ctx, firstError(err, sourceError))
		if ctx != nil {
			runtime.EventsEmit(ctx, "projects-updated")
		}
	})
	if err != nil {
		logRuntimeError(ctx, err)
		return nil
	}
	return watcher
}

func (a *App) liveRecents(ctx context.Context) []Project {
	recent, err := ScanVSCodeRecentProjects(ctx)
	if err != nil {
		a.debugLog("vscode-recent.live.error", "error", err)
		return nil
	}
	return recent
}

func (a *App) rescan() ([]Project, error, error) {
	a.scanMutex.Lock()
	defer a.scanMutex.Unlock()
	return a.rescanUnlocked()
}

func (a *App) rescanUnlocked() ([]Project, error, error) {
	ctx, config, store := a.snapshot()
	a.debugLog("scan.begin", "roots", len(config.Scanner.Roots), "maxDepth", config.Scanner.MaxDepth)
	items, skipped, err := ScanProjects(ctx, config.Scanner)
	for _, root := range skipped {
		a.debugLog("scan.root.skip", "root", root)
	}
	apps, recent, applicationError, recentProjectError := scanAppSources(ctx)
	cacheItems := mergeProjects(items, apps)
	if store != nil {
		var storeError error
		if shouldReplaceCache(err, applicationError, skipped) {
			storeError = store.ReplaceProjects(ctx, cacheItems)
		} else {
			storeError = store.UpsertProjects(ctx, cacheItems)
		}
		if storeError != nil && err == nil {
			err = storeError
		}
	}
	sourceError := firstError(applicationError, recentProjectError)
	items = a.indexProjects(cacheItems, recent)
	a.debugLog("scan.end", "items", len(items), "skipped", len(skipped), "error", err, "sourceError", sourceError)
	return items, sourceError, err
}

func shouldReplaceCache(scanError, applicationError error, skippedRoots []string) bool {
	return scanError == nil && applicationError == nil && len(skippedRoots) == 0
}

func firstError(errorsToCheck ...error) error {
	for _, err := range errorsToCheck {
		if err != nil {
			return err
		}
	}
	return nil
}

func logRuntimeError(ctx context.Context, err error) {
	if err == nil || ctx == nil {
		return
	}
	runtime.LogError(ctx, err.Error())
}

func (a *App) debugLog(stage string, values ...any) {
	if a.debug != nil {
		a.debug.Log(stage, values...)
	}
}

func (a *App) showLauncher(ctx context.Context, sequence uint64) {
	a.debugLog("window.show.begin", "sequence", sequence)
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
	a.debugLog("window.show.frontend", "sequence", sequence, "ready", readyState)
	a.stateMu.Lock()
	current := a.showSequence == sequence && a.shown
	a.stateMu.Unlock()
	if !current {
		a.debugLog("window.show.cancelled", "sequence", sequence)
		return
	}
	runtime.WindowUnminimise(ctx)
	prepareWindowActivation()
	runtime.WindowShow(ctx)
	a.debugLog("window.show.end", "sequence", sequence)
	a.focusAndActivate(ctx)
	a.startWindowAutoHideMonitor(ctx, sequence)
	a.startFocusRetries(ctx, sequence)
}

func (a *App) focusAndActivate(ctx context.Context) {
	activateWindow()
	runtime.WindowExecJS(ctx, focusQueryScript())
}

func (a *App) startFocusRetries(ctx context.Context, sequence uint64) {
	go func() {
		for _, delay := range []time.Duration{60, 140, 280, 520} {
			time.Sleep(delay * time.Millisecond)
			a.stateMu.Lock()
			current := a.showSequence == sequence && a.shown
			a.stateMu.Unlock()
			if !current {
				return
			}
			a.focusAndActivate(ctx)
		}
	}()
}

func (a *App) startWindowAutoHideMonitor(ctx context.Context, sequence uint64) {
	go func() {
		timer := time.NewTimer(350 * time.Millisecond)
		defer timer.Stop()
		<-timer.C
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		misses := 0
		for range ticker.C {
			a.stateMu.Lock()
			current := a.shouldHideOnFocusLossLocked(sequence)
			suppressed := a.autoHideSuppressed
			a.stateMu.Unlock()
			if !current {
				return
			}
			if suppressed {
				misses = 0
				continue
			}
			if didPointerActivateOutsideWindow() {
				if a.hideAfterFocusLoss(ctx, sequence) {
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
			if a.hideAfterFocusLoss(ctx, sequence) {
				return
			}
		}
	}()
}

func (a *App) hideAfterFocusLoss(ctx context.Context, sequence uint64) bool {
	a.stateMu.Lock()
	if !a.shouldHideOnFocusLossLocked(sequence) {
		a.stateMu.Unlock()
		return false
	}
	a.shown = false
	a.showSequence++
	a.stateMu.Unlock()
	windowHide(ctx)
	a.debugLog("window.hide.focus-loss", "sequence", sequence)
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
