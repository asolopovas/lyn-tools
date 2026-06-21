# Platform

Part of the [architecture map](../ARCHITECTURE.md).

## VS Code launch (all platforms)

- **Retargeting:** `App.resolveLaunchTarget` opens a WordPress project's `.code-workspace` if indexed, else the most recently modified `wp-content/themes/<theme>`, never the site root. Other kinds open their root. Usage is recorded against the selected project, not the retargeted path.
- **Remote:** projects open with `--folder-uri vscode-remote://<authority><path>` (the `--remote` form opens an empty window). `.code-workspace` targets use `--file-uri`.

## Windows

- **Discovery:** indexes Start Menu/Desktop shortcuts and GUI executables from PATH. Console tools and the `Startup` folder are skipped. Launch/reveal/startup use native APIs to avoid helper terminals.
- **Elevation:** launched processes must never inherit Lyn's elevation.
  - Non-elevated Lyn launches directly via `exec`/`ShellExecute`.
  - When elevated, `startProcessAsShellUser` duplicates the shell (Explorer) process token and launches the child with `CreateProcessWithTokenW`, so GUI and console targets run at the shell user's standard integrity through one path. It fails closed if that token is itself elevated.
  - `run-admin` is a per-launch `ShellExecuteW runAs` (UAC consent each time); `run-user` is `runAsUser`. There is no standing elevated helper or pipe, so no path elevates an arbitrary executable without a prompt.
- **Manual elevation toggle:** the user can relaunch Lyn elevated ([`lyn/elevation.go`](../lyn/elevation.go) `SwitchElevation`); even then, launched children are forced back to standard integrity by the de-elevation path above.
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
