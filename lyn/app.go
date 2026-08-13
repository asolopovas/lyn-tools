package lyn

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"lyn.tools/launcher/lyn/hotkey"
	"lyn.tools/launcher/lyn/launch"
	"lyn.tools/launcher/lyn/startup"
	"lyn.tools/launcher/lyn/tray"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	stateMu               sync.Mutex
	ctx                   context.Context
	config                Config
	hotkey                hotkey.Registration
	watcher               *Watcher
	keyInterceptorCleanup func()
	debug                 *DebugLogger
	projects              *projectService
	window                *windowController
}

type ScanResult struct {
	Count int    `json:"count"`
	Error string `json:"error,omitempty"`
}

func NewApp(debug ...*DebugLogger) *App {
	config := DefaultConfig()
	app := &App{config: config}
	if len(debug) > 0 {
		app.debug = debug[0]
	}
	app.projects = newProjectService(config, app.debugLog)
	app.window = newWindowController(app.debugLog)
	return app
}

func (a *App) UseConfig(config Config) {
	a.stateMu.Lock()
	a.config = config
	a.stateMu.Unlock()
	a.projects.useConfig(config)
}

func (a *App) StartHidden() bool {
	_, config, _ := a.snapshot()
	return a.window.windowMode() != SettingsWindowMode && config.Startup.Enabled && config.Startup.StartHidden
}

func (a *App) Startup(ctx context.Context) {
	a.startup(ctx)
}

func (a *App) Shutdown(ctx context.Context) {
	a.shutdown(ctx)
}

func (a *App) startup(ctx context.Context) {
	a.stateMu.Lock()
	a.ctx = ctx
	a.stateMu.Unlock()
	a.window.setContext(ctx)
	a.debugLog("startup.begin")
	config, err := LoadConfig("")
	if err == nil {
		a.stateMu.Lock()
		a.config = config
		a.stateMu.Unlock()
		a.debugLog("config.loaded", "path", config.Path, "hotkey", config.Hotkey.Binding, "roots", len(config.Scanner.Roots), "watch", config.Scanner.Watch)
	} else {
		a.debugLog("config.error", "error", err)
		_, config, _ = a.snapshot()
	}
	a.projects.configure(ctx, config)
	store, err := OpenStore(filepath.Join(config.Cache.Dir, "lyn.db"))
	if err == nil {
		a.projects.setStore(store)
		a.debugLog("store.opened", "dir", config.Cache.Dir)
	} else {
		a.debugLog("store.error", "error", err)
	}
	a.installSystemToolsScript()
	if a.window.windowMode() == SettingsWindowMode {
		a.markSettingsWindowActive()
		placeWindow(ctx)
		a.debugLog("startup.settings.shown")
		return
	}
	a.registerHotkey()
	a.registerLaunchKeyInterceptor()
	a.startWatcher()
	configureWindowAppearance()
	installCloseToTray(a.window.isQuitting, a.minimizeToTray)
	a.debugLog("tray.start.request")
	tray.Start(a, a.debugLog)
	a.startFastRefresh()
	if config.Startup.Enabled && config.Startup.StartHidden {
		a.window.markStartupHidden(ctx)
		a.debugLog("startup.hidden")
		return
	}
	placeWindow(ctx)
	prepareWindowActivation()
	a.window.activateInitial(ctx)
	a.debugLog("startup.shown")
}

func (a *App) startFastRefresh() {
	ctx, _, store := a.snapshot()
	if ctx == nil || store == nil {
		return
	}
	go func() {
		a.debugLog("fast-refresh.begin")
		_, sourceError, err := a.projects.rescan()
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
	if a.window.windowMode() == SettingsWindowMode {
		a.clearSettingsWindowActive()
	}
	a.stateMu.Lock()
	ctx := a.ctx
	hotkey := a.hotkey
	watcher := a.watcher
	keyInterceptorCleanup := a.keyInterceptorCleanup
	a.ctx = nil
	a.hotkey = nil
	a.watcher = nil
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
	tray.Stop()
	logRuntimeError(ctx, a.projects.close())
	a.window.clearContext()
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
	launcherMode := a.window.windowMode() != SettingsWindowMode
	hotkeyChanged := old.Hotkey.Binding != loaded.Hotkey.Binding
	scannerChanged := !reflect.DeepEqual(old.Scanner, loaded.Scanner)
	a.config = loaded
	if launcherMode && hotkeyChanged {
		a.hotkey = nil
	}
	a.stateMu.Unlock()
	a.projects.useConfig(loaded)
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
			_, sourceError, err := a.projects.rescan()
			logRuntimeError(ctx, firstError(err, sourceError))
			runtime.EventsEmit(ctx, "projects-updated")
		}()
	}
}

func (a *App) snapshot() (context.Context, Config, *Store) {
	a.stateMu.Lock()
	ctx := a.ctx
	config := a.config
	a.stateMu.Unlock()
	_, _, store := a.projects.snapshot()
	return ctx, config, store
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
	return a.projects.projects()
}

func (a *App) SearchProjects(query string) ([]Project, error) {
	return a.projects.search(query)
}

func (a *App) RefreshProjects() ([]Project, error) {
	return a.projects.refresh()
}

func (a *App) Scan() ScanResult {
	return a.projects.scan()
}

var launchRequest = launch.Launch

var launchAsync = func(fn func()) { go fn() }

func (a *App) Launch(request launch.Request) launch.Result {
	return a.projects.launch(request)
}

func (a *App) SetLaunchSelection(request launch.Request) {
	a.projects.setLaunchSelection(request)
}

func (a *App) Debug(stage string, detail string) {
	a.debugLog("frontend."+stage, "detail", detail)
}

func (a *App) ChooseFolder() (string, error) {
	ctx, _, _ := a.snapshot()
	if ctx == nil {
		return "", nil
	}
	a.window.setAutoHideSuppressed(true)
	defer a.window.setAutoHideSuppressed(false)
	return runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{Title: "Add indexed folder"})
}

func (a *App) ChooseWSLFolder() (WSLRoot, error) {
	ctx, _, _ := a.snapshot()
	if ctx == nil {
		return WSLRoot{}, nil
	}
	a.window.setAutoHideSuppressed(true)
	defer a.window.setAutoHideSuppressed(false)
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
	a.window.show()
}

func (a *App) ShowSettings() {
	a.debugLog("settings.open.request")
	if err := a.OpenSettingsWindow(); err != nil {
		ctx, _, _ := a.snapshot()
		a.debugLog("settings.open.error", "error", err)
		logRuntimeError(ctx, err)
	}
}

func (a *App) Hide() {
	a.window.hide()
}

var (
	quitRuntime = runtime.Quit
	windowHide  = runtime.WindowHide
)

func (a *App) BeforeClose(context.Context) bool {
	if a.window.isQuitting() || a.window.windowMode() == SettingsWindowMode {
		a.debugLog("window.close.allow")
		return false
	}
	a.minimizeToTray()
	return true
}

func (a *App) minimizeToTray() {
	a.debugLog("window.close.minimize")
	a.window.hide()
}

func (a *App) Quit() {
	if ctx := a.window.currentContext(); ctx != nil {
		a.window.requestQuit(ctx)
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
	a.window.requestQuit(ctx)
}

func (a *App) Toggle() {
	a.window.toggle()
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
		ctx, _, _ := a.snapshot()
		logRuntimeError(ctx, err)
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
		_, sourceError, err := a.projects.rescan()
		logRuntimeError(ctx, firstError(err, sourceError))
		runtime.EventsEmit(ctx, "projects-updated")
	})
	if err != nil {
		logRuntimeError(ctx, err)
		return nil
	}
	return watcher
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
