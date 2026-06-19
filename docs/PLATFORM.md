# Platform

Part of the [architecture map](../ARCHITECTURE.md).

## VS Code launch (all platforms)

- **Retargeting:** `App.resolveLaunchTarget` opens a WordPress project's `.code-workspace` if indexed, else the most recently modified `wp-content/themes/<theme>`, never the site root. Other kinds open their root. Usage is recorded against the selected project, not the retargeted path.
- **Remote:** projects open with `--folder-uri vscode-remote://<authority><path>` (the `--remote` form opens an empty window). `.code-workspace` targets use `--file-uri`.

## Windows

- **Discovery:** indexes Start Menu/Desktop shortcuts and GUI executables from PATH. Console tools and the `Startup` folder are skipped. Launch/reveal/startup use native APIs to avoid helper terminals.
- **Elevation:** launched processes must never inherit Lyn's elevation.
  - When elevated, `launch` reparents the child to the shell (Explorer) via `PROC_THREAD_ATTRIBUTE_PARENT_PROCESS`.
  - Only `run-admin`/`run-user` keep explicit `runAs`/`runAsUser`.
  - The elevated `code` path routes through its CLI (`cmd /c code …`) because the GUI exe doesn't survive reparenting.
  - Non-elevated Lyn keeps the direct `exec`/`ShellExecute` path.
- **run-admin helper:** a lazily spawned elevated helper (`lyn --elevated-helper=<pipe>`) lets repeated admin launches reuse one UAC consent.
  - The non-elevated UI owns the named-pipe server, scoped to the current user SID. The helper connects as client and runs whitelisted `run-admin` requests via `ShellExecute open`.
  - Protocol is newline-delimited JSON ([`lyn/elevation_ipc.go`](../lyn/elevation_ipc.go)).
  - Falls back to per-launch `ShellExecuteW runAs` if the pipe fails. The helper exits when the UI closes the pipe.
- **Keyboard hook:** `Win+D` and the launch interceptor run on a `WH_KEYBOARD_LL` hook ([`lyn/hotkey/hotkeys_windows.go`](../lyn/hotkey/hotkeys_windows.go)). Windows permanently disables a hook whose callback exceeds `LowLevelHooksTimeout` (300ms), so the callback does only fast in-memory work: no SQLite, no cross-thread window calls, no slow-path locks. The interceptor makes a cache-only decision (`hasCachedLaunchSelection`) and offloads the store lookup, `Hide`, and launch to a goroutine (`runNativeShortcut`).
- **WSL paths** may be UNC or Linux paths converted by `wsl.exe wslpath -w`. `wsl.exe` is a console program, so spawn it through `hideConsoleWindow` ([`lyn/process_windows.go`](../lyn/process_windows.go), `CREATE_NO_WINDOW`). Any new child console process must use the same helper.
- **Host-only tests:** path reveal and system-dir tests call `requireWindowsHost` and skip elsewhere. Do not rewrite host-correct `filepath` code to make them pass cross-platform.

## Linux

- **Discovery:** uses desktop entries and common icon theme paths.
- **Global hotkey:** owns its X11 grab directly ([`lyn/hotkey/hotkeys_linux.go`](../lyn/hotkey/hotkeys_linux.go) + [`lyn/hotkey/hotkey_grab_linux.c`](../lyn/hotkey/hotkey_grab_linux.c)) instead of `golang.design/x/hotkey`, which fired on any keystroke on empty xmonad workspaces.
  - Grabs keycode + modifier mask, runs one filtered event loop, and fires only when `keycode` matches and `(state & realModifierMask) == binding` (ignoring Num/Caps Lock and XKB group).
  - Grabs every lock-variant of the mask since `XGrabKey` matches exactly.
  - A conflicting grab from another client yields `BadAccess`, so the WM must own no overlapping binding.
  - Unregister wakes the loop via a self-pipe.

## Tray

Support is split by build tags. Unsupported platforms use a no-op fallback.
