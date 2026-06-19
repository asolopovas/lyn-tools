# Backend

Part of the [architecture map](../ARCHITECTURE.md).

[`lyn/app.go`](../lyn/app.go) owns the Wails lifecycle: state, cache refresh, watcher restart, tray, hotkey, window, and frontend bindings. Domains live in `config`, `cache`, `project`, `scanner`, `apps`, `vscode`, `watcher` files under [`lyn/`](../lyn).

- The backend owns all indexing, search, typo-tolerant matching, workspace filtering, and ranking. The frontend never ranks.
- SQLite stores cached items and launch usage. All SQLite access stays in Go, including VS Code `state.vscdb` reads.

## Close-to-tray

- Closing the launcher window minimizes to tray. The only exit is the tray menu (or a restart/elevation hand-off).
- Quit/restart/elevation route through `App.requestQuit`, which sets `quitting` before `runtime.Quit`.
- `App.BeforeClose` (`OnBeforeClose`) handles Wails closes and is the Linux path.
- Windows `WM_SYSCOMMAND`/`SC_CLOSE` bypasses `OnBeforeClose`, so `installWindowsCloseToTray` ([`lyn/window_windows.go`](../lyn/window_windows.go)) subclasses the window proc to intercept `WM_CLOSE` and `SC_CLOSE`.
- The settings window closes normally.

## Hide-path threading (Windows)

`runtime.WindowHide` called from a goroutine blocks on a cross-thread `ShowWindow` and can deadlock. Two rules prevent it:

- Hide paths (`Hide`, `Toggle`, `hideAfterFocusLoss`) release `stateMu` before calling `windowHide`.
- The close interceptor dispatches `minimizeToTray` on a goroutine and returns immediately.

`windowHide` is a package var so this stays regression-tested.

## Ranking

- Order: match quality, then kind tier, then usage count, last launch time, name, path.
- Kind tiers (`kindRank`): `.code-workspace` first, then everything else (local projects, apps, WSL/non-SSH recents), then VS Code SSH-remote (`vscode-remote://ssh-remote+`), then system commands last.
- Empty query lists workspaces first, power commands last.
- Log Out / Restart / Shut Down are virtual `system-command` results (`lyn:system:{logout,restart,shutdown}`), added in memory by `indexProjects` and never saved. Opening the store also clears any stale rows as a safeguard.
- Launch maps them to the real OS command (`shutdown.exe`, `systemctl`/`loginctl`, `osascript`). Windows runs `shutdown.exe` hidden.

## Scanning and detection

- Limited by configured roots, depth, concurrency, timeout, and ignored folders. Each scan replaces the cache it owns.
- The watcher waits 200ms after events settle and runs at most one rescan per `watcherMinScanInterval` (5s). The watch set includes busy files like `state.vscdb`, so this limit stops constant rescans.
- [`lyn/path.go`](../lyn/path.go) is the single home for path include/skip controls (traversal skips, packaged-dependency and Windows `Startup` folders, system-dir containment). Scanner/watcher/apps/vscode delegate to it. Tests in [`lyn/path_test.go`](../lyn/path_test.go).
- Ignored folders: build/dependency dirs (`node_modules`, `vendor`, `dist`, `wp-includes`, `uploads`, ...) and versioned package-store entries (any `@<digit>`, e.g. `accepts@1.3.8`).
- Finding a project does not stop the scan inside it. The scan keeps going (skipping ignored folders) to find nested `.code-workspace` files. Project markers in nested folders are not added as separate projects.
- Detection markers, highest priority first: WordPress, Laravel, Go, Node, Rust, then any `.git` repo. Only real `.code-workspace` files count as the `vscode-workspace` kind. A bare `.vscode` folder is not a workspace.
- Dedup keys are case-insensitive for Windows drive paths (`C:\` == `c:\`). Remote (`vscode-remote://`) and Unix/UNC paths keep original casing.
- Unreachable roots (missing, offline share, permission error) are reported by `ScanProjects` and logged `scan.root.skip`. A scan with skipped roots merges into the cache instead of replacing it.
