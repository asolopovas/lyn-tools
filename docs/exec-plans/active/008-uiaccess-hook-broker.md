# 008 uiAccess hotkey broker for hooks over elevated windows

## Goal

`Win+D` (and any binding that needs a `WH_KEYBOARD_LL` hook) toggles the launcher even while an elevated window is focused, without putting `uiAccess` on the WebView2 GUI process.

## Why a separate process

`uiAccess="true"` on the Wails launcher exe makes the GUI exit silently at startup: WebView2's sandboxed child processes do not initialize under a uiAccess token (see `007`, "Blocker"). A medium-integrity `WH_KEYBOARD_LL` hook is blind to input while a higher-integrity window has focus, which is why `Win+D` leaks into an elevated terminal. `RegisterHotKey` bindings (e.g. `Ctrl+Space`) already fire over elevated windows, so only the keyboard-hook path (`Win+D`, which is system-reserved and cannot be taken with `RegisterHotKey`) needs this.

PowerToys solves the same problem the same way: the hook lives in a separate privileged process that signals the GUI. Lyn mirrors that with a tiny `uiAccess` broker that owns the hook and signals the medium-IL GUI through a named event. A signal-only broker cannot run arbitrary code, so it is not the escalation surface that retired the old elevated-launch pipe.

## Design

- Broker is the same `lyn` binary run as `lyn-hook.exe --hook-broker --parent <gui-pid>`. `main` short-circuits into broker mode before `wails.Run`, so it never touches WebView2.
- `lyn-hook.exe` is a copy of `lyn.exe` carrying the `uiAccess` manifest + Authenticode signature; it lives in `%ProgramFiles%\Lyn` (secure location). `lyn.exe` stays `asInvoker` (no uiAccess) so the GUI runs.
- IPC via named events in the per-session `Local\` namespace:
  - `Local\LynHotkeyToggle` (auto-reset): broker `SetEvent` on `Win+D`; GUI waits and toggles.
  - `Local\LynHotkeyBrokerStop` (manual-reset): GUI signals the broker to exit on unregister.
- GUI creates the events, spawns the broker via `ShellExecute` (so AppInfo grants uiAccess), then waits on the toggle event in a goroutine. Broker exits when the GUI process exits (waits on the parent handle) or when the stop event fires.
- Fallback: if `lyn-hook.exe` is absent (dev `just build`) or the spawn fails, the GUI keeps the existing in-process medium hook. So `just dev` is unchanged and still works over non-elevated windows.

## Packaging

`scripts/package-windows.ps1` becomes deterministic about manifests (the build's embedded manifest is not relied on):
1. Build `lyn.exe`, then `mt.exe` stamp the `asInvoker` manifest onto it.
2. Copy `lyn.exe` -> `lyn-hook.exe`; `mt.exe` stamp the `uiAccess` manifest onto the copy.
3. Sign both exes; NSIS ships both into `%ProgramFiles%\Lyn`.

`scripts/install-windows.ps1` (the `just install` dev path) does the same with the dev cert: it copies the freshly built unsigned `lyn.exe` to `lyn-hook.exe`, then in one elevated step signs both (asInvoker / uiAccess via `dev-sign.ps1`), stops the running launcher and the higher-integrity broker, and deploys both. It targets `%ProgramFiles%\Lyn` because uiAccess is only granted from a secure location; installing elsewhere warns and the broker stays medium-IL. Requires `just dev-sign-setup` once. `dev-sign.ps1` strips any existing signature before stamping, since `mt.exe` over a signed PE corrupts it (signtool `0x800700C1`).

## Acceptance Criteria

- `Win+D` toggles the launcher while an elevated Windows Terminal is focused (real keypress; cannot be synthesized because the hook ignores injected input).
- `Win+D` still toggles over normal windows; `Ctrl+Space` unaffected.
- GUI launches and renders (no uiAccess on `lyn.exe`).
- Broker exits when the GUI exits (no orphan `lyn-hook.exe`).
- `just dev` still toggles over non-elevated windows with no broker present.
- `just check` passes.

## Validation

- Automated/self-driven: signal `Local\LynHotkeyToggle` directly and confirm the GUI logs `window.toggle`; inject `Win+D` with the broker's injected-filter relaxed in a test to confirm broker -> event -> GUI toggle; confirm broker process exits when GUI is killed.
- Manual (irreducible): one real `Win+D` over an elevated terminal.

## Hardening (post-review)

- Named events: `createNamedEvent` keeps the handle on `ERROR_ALREADY_EXISTS` (no leak) and `ResetEvent`s it, so a reused/stale toggle, stop, or ready object never poisons a fresh broker. The GUI no longer degrades to the in-process hook just because the events already exist.
- Readiness handshake: the broker signals `Local\LynHotkeyBrokerReady` only after its hook is installed; the GUI waits up to 3s and falls back to the in-process hook if the broker never readies, instead of assuming success the instant `ShellExecute` returns.
- Lifetime: the parent PID is required and `OpenProcess` failure is fatal (fail closed, never orphan a uiAccess hook); the broker and the GUI waiter both block on `WaitForMultipleObjects` (stop+parent / toggle+localStop) — no polling, instant teardown.
- Observability: the broker process honors `LYN_DEBUG`/`LYN_DEBUG_LOG` and logs `broker.start`/`broker.error`/`broker.stop`.
- The launcher window class is exported as `hotkey.LauncherWindowClass`; a `lyn` test asserts it equals `NativeWindowClassName`.
- Installer waits after `taskkill` before overwriting the in-use exes.

## Deferred (reviewed, low value)

- Single-instance lock for the GUI (shared event names assume one launcher instance).
- Rapid-press coalescing on the auto-reset toggle (3+ presses inside one ~250ms show).
- Explicit least-privilege DACL on the IPC events (same-user defense-in-depth).
- Extracting a shared `ShellExecuteW` helper (duplicated across broker/elevation/launch).

## Status

- Implemented and reviewed; pending live confirmation of Win+D toggling over an elevated terminal.
