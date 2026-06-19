# Architecture

Lyn is a Wails desktop launcher for projects, apps, VS Code recents, and workspaces.

## Layout

- Root Go files bootstrap the app, embed frontend assets, and enforce structure.
- `lyn/` is the backend package; `lyn/hotkey/`, `lyn/launch/`, `lyn/startup/`, `lyn/tray/` are focused domains imported by `lyn/app.go`.
- `frontend/` is the Vue/Vite workspace; `frontend/dist` is embedded for production.
- Add folders only for cohesive multi-file packages, build tags, generated output, assets, or tool-required paths. Keep small Go application code in root files.
- Avoid `cmd`, `internal`, `pkg` until multiple binaries, reusable internals, or public APIs justify them.

## Backend

- `lyn/app.go` owns Wails lifecycle, state, cache refresh, watcher restart, tray, hotkey, window, and frontend bindings.
- Shared domains live in `config`, `cache`, `project`, `scanner`, `apps`, `vscode`, `watcher` files. SQLite stores cached items and launch usage; all SQLite access stays in Go (including VS Code `state.vscdb` reads).
- The backend owns indexing, search, typo-tolerant matching, workspace shortcut filtering, and ranking.
- **Close-to-tray:** closing the launcher window minimizes to tray; the only exit is the tray menu (or restart/elevation hand-off). Quit/restart/elevation route through `App.requestQuit`, which sets the `quitting` flag before `runtime.Quit`. `OnBeforeClose` (`App.BeforeClose`) handles Wails closes and is the Linux path. On Windows, `WM_SYSCOMMAND`/`SC_CLOSE` bypasses `OnBeforeClose`, so `window_windows.go` subclasses the launcher window proc (`installWindowsCloseToTray`) to intercept `WM_CLOSE` and `SC_CLOSE`. The settings window closes normally.
- **Hide-path threading (Windows):** `runtime.WindowHide` from a goroutine is a blocking cross-thread `ShowWindow`. Two rules avoid deadlock: hide paths (`Hide`, `Toggle`, `hideAfterFocusLoss`) release `stateMu` before calling `windowHide`; the close interceptor dispatches `minimizeToTray` on a goroutine and returns immediately. `windowHide` is a package var so this is regression-tested.

### Ranking

- Order: match quality, then kind tier, then usage count, last launch time, name, path.
- Kind tiers (`kindRank`): `.code-workspace` first, then everything else (local projects, apps, WSL/non-SSH recents), then VS Code SSH-remote (`vscode-remote://ssh-remote+`), then system commands last. Empty query lists workspaces first, power commands last.
- Log Out / Restart / Shut Down are virtual `system-command` results (`lyn:system:{logout,restart,shutdown}`) injected in-memory by `indexProjects`, never persisted (store open purges stale rows as defense). Launch maps them to the real OS command (`shutdown.exe`, `systemctl`/`loginctl`, `osascript`); Windows runs `shutdown.exe` hidden via background launch.

### Scanning & detection

- Bounded by configured roots, depth, concurrency, timeout, ignored folders, and source-owned cache replacement.
- The watcher debounces events (200 ms) and throttles rescans to one per `watcherMinScanInterval` (5 s). The watch set includes high-churn paths like `state.vscdb`, so the throttle prevents constant rescans.
- `path.go` is the single home for path include/skip controls (traversal skips, packaged-dependency and Windows `Startup` folders, system-dir containment); scanner/watcher/apps/vscode delegate to it. Tests in `path_test.go`.
- Ignored folders: build/dependency dirs (`node_modules`, `vendor`, `dist`, `wp-includes`, `uploads`, ...) and versioned package-store entries (any `@<digit>`, e.g. `accepts@1.3.8`).
- Detecting a project does not stop the scan inside it: traversal continues (skipping ignored folders) to collect nested `.code-workspace` files, but nested directory project-markers are not added as separate projects.
- Detection markers, highest priority first: WordPress, Laravel, Go, Node, Rust, then any `.git` repo. `vscode-workspace` kind is reserved for real `.code-workspace` files; a bare `.vscode` folder is not a workspace.
- Dedup keys are case-insensitive for Windows drive paths (`C:\` == `c:\`); remote (`vscode-remote://`) and Unix/UNC paths keep original casing.
- Unreachable roots (missing, offline share, permission error) are reported by `ScanProjects` and logged `scan.root.skip`; a scan with skipped roots merges into the cache instead of replacing it.

## Frontend

- `frontend/src/App.vue` coordinates launcher UI state and Wails events. The frontend is a thin input/render layer: it sends query text to backend search and renders matches.
- Components render panels; small modules own backend access, cache, themes, icons, types, hotkeys, launch actions. Keyboard mappings in `frontend/src/hotkeys.ts`, reused by input and window handlers.
- Local storage is only a warm UI cache, never search or ranking truth.
- Launcher panel height is `min(<configured>px, 100vh)` (`width: 100%`) so it never exceeds the live viewport; the results list owns its scroll (`overflow-y: auto`). The `100vh` cap is required because under fractional display scale WebKitGTK locks the viewport to creation-size and `DisableResize` keeps `WindowSetSize` from changing it. Do not use `overflow-y: overlay` — WebKitGTK treats it as `visible` (no scrolling).

## Platform

- **Windows discovery:** indexes Start Menu/Desktop shortcuts and GUI executables from PATH; console tools and the `Startup` folder are skipped. Launch/reveal/startup use native APIs to avoid helper terminals.
- **Open in Code retargeting:** `App.resolveLaunchTarget` opens a WordPress project's `.code-workspace` if indexed, else the most recently modified `wp-content/themes/<theme>`, never the site root. Other kinds open their root. Usage is recorded against the selected project, not the retargeted path.
- **Remote projects** open with `--folder-uri vscode-remote://<authority><path>` (the `--remote` form opens an empty window); `.code-workspace` targets use `--file-uri`.
- **Elevation:** launched processes must never inherit Lyn's elevation. When elevated, `launch` reparents the child to the shell (Explorer) via `PROC_THREAD_ATTRIBUTE_PARENT_PROCESS`; only `run-admin`/`run-user` keep explicit `runAs`/`runAsUser`. The elevated `code` path routes through its CLI (`cmd /c code …`) since the GUI exe doesn't survive reparenting. Non-elevated Lyn keeps the direct `exec`/`ShellExecute` path.
- **run-admin helper:** routes through a lazily spawned elevated helper (`lyn --elevated-helper=<pipe>`) so repeated admin launches reuse one UAC consent. The non-elevated UI owns the named-pipe server (scoped to the current user SID); the helper connects as client and runs whitelisted `run-admin` requests via `ShellExecute open`. Protocol is newline-delimited JSON (`lyn/elevation_ipc.go`). Falls back to per-launch `ShellExecuteW runAs` if the pipe fails; the helper exits when the UI closes the pipe.
- **Windows keyboard hook:** `Win+D` and the launch interceptor run on a `WH_KEYBOARD_LL` hook (`hotkey/hotkeys_windows.go`). Windows permanently disables a hook whose callback exceeds `LowLevelHooksTimeout` (300 ms), so the callback must do only fast in-memory work — no SQLite, no cross-thread window calls, no slow-path locks. The interceptor makes a cache-only decision (`hasCachedLaunchSelection`) and offloads the store lookup, `Hide`, and launch to a goroutine (`runNativeShortcut`).
- **WSL paths** may be UNC or Linux paths converted by `wsl.exe wslpath -w`. `wsl.exe` is a console program, so spawn it through `hideConsoleWindow` (`process_windows.go`, `CREATE_NO_WINDOW`); any new child console process must use the same helper.
- **Linux discovery:** uses desktop entries and common icon theme paths.
- **Linux global hotkey:** owns its X11 grab directly (`hotkey/hotkeys_linux.go` + `hotkey/hotkey_grab_linux.c`) instead of `golang.design/x/hotkey` (which subscribes to all root key presses and fired on any keystroke on empty xmonad workspaces). The replacement grabs keycode + modifier mask, runs one filtered event loop, and fires only when `keycode` matches and `(state & realModifierMask) == binding` (ignoring Num/Caps Lock and XKB group). It grabs every lock-variant of the mask since `XGrabKey` matches exactly. A conflicting grab from another client yields `BadAccess`, so the WM must own no overlapping binding. Unregister wakes the loop via a self-pipe.
- **Tray** support is split by build tags; unsupported platforms use a no-op fallback.

## Harness

- `docs/QUALITY.md` is the strict development policy for Go, Vue, Wails, cross-platform behavior, and validation.
- `just check` is the local and CI quality gate: formatting, Go tests, vet, staticcheck, frontend lint/typecheck/unit/e2e. It installs `staticcheck` on first run; `.github/workflows/ci.yml` runs it on Linux and Windows.
- Windows-host-only tests (path reveal/system-dir) call `requireWindowsHost` and skip elsewhere; do not rewrite host-correct `filepath` code to make them pass cross-platform. Structural tests enforce folder rules.
- Add focused tests/benchmarks for traversal, ranking, cache writes, launch, focus, watcher, icon, and platform behavior. Validate UI changes by driving the running app or Playwright path and inspecting screenshots/logs.
- Debug with `lyn --debug-log <path>` or `LYN_DEBUG_LOG`; key stages: `hotkey.*`, `window.*`, `frontend.*`, `launch.*`, `scan.*`.
