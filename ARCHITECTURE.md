# Architecture

Lyn is a Wails desktop launcher for projects, apps, VS Code recents, and workspaces.

## Layout

- Root Go files bootstrap the app, embed frontend assets, and enforce structure.
- `lyn/` is the backend application package.
- `lyn/hotkey/`, `lyn/launch/`, `lyn/startup/`, and `lyn/tray/` are focused backend domains imported by `lyn/app.go`.
- `frontend/` is the Vue/Vite workspace; `frontend/dist` is embedded for production.
- Add folders only for cohesive multi-file packages, build tags, generated output, assets, or tool-required paths.
- Split files only at real package, build-tag, generated, asset, test, or cohesive-domain boundaries; keep small Go application code in root files.
- Avoid `cmd`, `internal`, and `pkg` until multiple binaries, reusable internals, or public APIs justify them.

## Backend

- `lyn/app.go` owns Wails lifecycle, state, cache refresh, watcher restart, tray, hotkey, window, and frontend bindings.
- Closing the launcher window minimizes to tray instead of exiting; the only way to exit is the tray menu (or a restart/elevation hand-off). The tray Quit, restart, and elevation switch route through `App.requestQuit`, which sets the `quitting` flag before calling `runtime.Quit`. Genuine quits clear the gate; everything else hides.
- Close interception is two-layered. `OnBeforeClose` (`App.BeforeClose`) handles Wails `WM_CLOSE`-style closes (and is the Linux path). On Windows that is not enough: AutoHotkey/system "close active window" actions send `WM_SYSCOMMAND`/`SC_CLOSE`, which Wails does not route through `OnBeforeClose`. `window_windows.go` therefore subclasses the launcher window proc (`installWindowsCloseToTray`) and intercepts both `WM_CLOSE` and `WM_SYSCOMMAND`/`SC_CLOSE`, hiding to tray via `App.minimizeToTray` unless the `quitting` flag is set. The settings window is not subclassed and closes normally.
- GUI-thread deadlock avoidance for the hide paths. `runtime.WindowHide` is a `ShowWindow` that, when invoked from a background goroutine, becomes a cross-thread call that blocks until the launcher's GUI thread pumps messages. Two rules keep this from deadlocking: (1) every hide path (`Hide`, `Toggle`, `hideAfterFocusLoss`) mutates `shown`/`showSequence` under `stateMu` but releases the lock *before* calling `windowHide`, so the lock is never held across a blocking window call; and (2) the close interceptor dispatches `minimizeToTray` on a goroutine and returns immediately, so the GUI message loop never blocks inside the window proc. Without both, a close (GUI thread waiting on `stateMu`) racing the auto-hide pass (background goroutine holding `stateMu` inside a cross-thread `ShowWindow`) freezes the window as "Not Responding". `windowHide` is a package var so the discipline is regression-tested.
- Shared domains live in `config`, `cache`, `project`, `scanner`, `apps`, `vscode`, and `watcher` files.
- SQLite stores cached items and launch usage; all SQLite connections stay in Go, including VS Code `state.vscdb` recent-project reads.
- The backend owns launcher indexing, search, typo-tolerant matching, workspace shortcut filtering, and ranking.
- Ranking order: match quality, usage count, last launch time, name, path.
- Scanning is bounded by configured roots, depth, concurrency, timeout, ignored folders, and source-owned cache replacement.
- The filesystem watcher debounces events (200 ms) and throttles rescans to at most one per `watcherMinScanInterval` (5 s). The watch set includes high-churn paths such as VS Code's `state.vscdb`, so without the throttle a long-running editor session drives a full rescan (and SQLite write) every couple of seconds indefinitely; that background load is what previously starved the keyboard hook past its timeout.
- `path.go` is the single home for path include/skip controls (traversal skips, packaged-dependency and Windows `Startup` folders, Windows system-dir containment); scanner, watcher, apps, and vscode callers delegate to it instead of reimplementing path checks. Tests live in `path_test.go`.
- Ignored folders include build/dependency dirs (`node_modules`, `vendor`, `dist`, ...) and versioned package-store entries (any `@<digit>` in the name, e.g. bun/pnpm cache dirs like `accepts@1.3.8`), so third-party packages are never indexed as projects.
- Project detection markers, highest priority first: WordPress, Laravel, Go, Node, Rust, then any `.git` repository. The `vscode-workspace` kind is reserved for real `.code-workspace` files; a bare `.vscode` settings folder is not a workspace and a folder is classified by its actual project type instead.
- Project dedup keys are case-insensitive for Windows drive paths (VS Code lowercases drive letters, so `C:\...` and `c:\...` are the same folder); remote (`vscode-remote://`) and Unix/UNC paths keep their original casing.
- Unreachable configured roots (missing path, offline WSL/UNC share, permission error) are reported by `ScanProjects` and logged as `scan.root.skip`; a scan with skipped roots merges into the cache instead of replacing it, so a transient root never purges previously indexed projects.

## Frontend

- `frontend/src/App.vue` coordinates launcher UI state and Wails events.
- The frontend is a thin input and rendering layer: it sends query text to backend search and renders returned matches.
- Components render panels; small modules own backend access, cache, themes, icons, types, hotkeys, and launch actions.
- Keyboard mappings live in `frontend/src/hotkeys.ts` and are reused by input and window handlers.
- Local storage is only a warm UI cache, never search or ranking truth.

## Platform

- Windows discovery indexes Start Menu/Desktop shortcuts and GUI executables from PATH; console tools and the `Startup` auto-run folder are skipped.
- Windows launch/reveal/startup use native APIs when needed to avoid helper terminals.
- The `Win+D` global hotkey and the launch shortcut interceptor run on a `WH_KEYBOARD_LL` low-level keyboard hook (`hotkey/hotkeys_windows.go`). Windows silently disables a low-level hook whose callback exceeds `LowLevelHooksTimeout` (default 300 ms), and it never recovers until the app restarts. The hook callback (including `SetKeyInterceptor` work) must therefore do only fast, non-blocking, in-memory work: no SQLite queries, no cross-thread window calls (`runtime.Window*`/`ShowWindow`), no lock that a slow path can hold. The launch interceptor makes a cache-only decision in the callback (`hasCachedLaunchSelection`) and offloads the store lookup, `Hide`, and launch to a goroutine (`runNativeShortcut`); doing the `ListProjects` read or `Hide` inline previously let a busy scan stall the callback past the timeout and kill `Win+D` until restart.
- WSL roots may be UNC paths or Linux paths converted by `wsl.exe wslpath -w`. `wsl.exe` is a console program, so the GUI build must spawn it through `hideConsoleWindow` (`process_windows.go`, `CREATE_NO_WINDOW`); otherwise a console window flashes on every startup/watcher scan. Any new child console process started from the app must go through the same helper.
- Linux discovery uses desktop entries and common icon theme paths.
- Tray support is split by build tags; unsupported platforms use a no-op fallback.

## Harness

- `docs/QUALITY.md` is the strict development policy for Go, Vue, Wails, cross-platform behavior, and validation.
- `just check` is the local and CI quality gate: formatting, Go tests, vet, staticcheck, frontend lint/typecheck/unit/e2e.
- Structural tests enforce folder rules.
- Add focused tests or benchmarks for traversal, ranking, cache writes, launch, focus, watcher, icon, and platform behavior.
- Validate UI changes by driving the running app or Playwright path and inspecting screenshots/logs.
- Debug with `lyn --debug-log <path>` or `LYN_DEBUG_LOG`; key stages include `hotkey.*`, `window.*`, `frontend.*`, `launch.*`, and `scan.*`.
