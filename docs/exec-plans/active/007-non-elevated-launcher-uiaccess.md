# 007 Non-elevated launcher with uiAccess hotkeys and safe de-elevation

## Goal

The launcher runs at standard (medium) integrity by default so it can never start a child as administrator by accident. The global hotkey still fires over admin windows via a `uiAccess` exemption rather than elevation. The manual elevation toggle stays, and when the user opts into it, launched apps are still forced back to standard integrity. No standing elevated process or pipe exists that could elevate arbitrary software without a UAC prompt.

## Scope

- `uiAccess` manifest + signed exe + Program Files install so the non-elevated hook works over admin windows.
- Replace the fragile elevated-launch de-elevation (Explorer reparenting + `cmd /c code` workaround) with one `CreateProcessWithTokenW` primitive using the shell (Explorer) token; fail closed if that token is elevated.
- Keep the standard/admin elevation toggle; remove the consent-reuse elevated-helper pipe IPC.
- `run-admin` is a per-launch `ShellExecute runAs` (UAC each time).
- Startup guard: if launched elevated unexpectedly, do not silently keep admin for child launches.

## Acceptance Criteria

- Non-elevated launcher's hotkey opens the launcher while an admin window (e.g. admin terminal) is focused.
- Selecting a VS Code workspace/folder opens it, including paths with spaces, whether the launcher is standard or manually elevated.
- A child launched while the launcher is manually elevated runs at medium integrity (verified non-elevated token).
- No named-pipe/helper path can run an arbitrary executable elevated without a UAC prompt.
- `just check` passes.

## Pending

- uiAccess release feature (one coupled unit — do NOT land piecemeal; see "uiAccess packaging checklist" below). Needs the user's signing-cert decision and a per-machine install, so it is gated on confirmation.

## uiAccess packaging checklist

Empirically verified launch behavior of a `uiAccess="true"` exe (Win11, UAC on, `mt.exe`-stamped test exe):

- Unsigned + non-secure location -> FAILS to launch ("A referral was returned from the server").
- Signed + non-secure location -> launches.
- Signed + `%ProgramFiles%` -> launches.

Therefore the manifest must NOT be added to the shared `build/windows/wails.exe.manifest`: it would break unsigned `just dev` / `just build`. uiAccess = manifest + Authenticode signature + secure-location install, landed together.

1. Keep `build/windows/wails.exe.manifest` as `asInvoker` (no `uiAccess`) for dev.
2. Release-only manifest with `<requestedExecutionLevel level="asInvoker" uiAccess="true"/>` (trustInfo `urn:schemas-microsoft-com:asm.v2`).
3. Release packaging (`scripts/package-windows.ps1`, after `wails build`, before NSIS): stamp the uiAccess manifest onto `build/bin/lyn.exe` with `mt.exe -manifest <file> -outputresource:lyn.exe;#1`, then `signtool sign /fd SHA256` (signing must be last and is now mandatory — an unsigned uiAccess exe will not launch).
4. Installer `build/windows/installer.nsi`: `InstallDir "$PROGRAMFILES64\Lyn"`, `RequestExecutionLevel admin`, move uninstall/Run/PATH registry writes from HKCU to HKLM. This flips Lyn from per-user to per-machine install (admin install, system PATH, HKLM Run) — confirm this UX change before applying.
5. Dev signing: a self-signed `CodeSigningCert` added to LocalMachine `Root` + `TrustedPublisher`, used by `signtool sign /sm /sha1 <thumbprint> /fd SHA256`. Sign both `lyn.exe` and the NSIS installer.

## Completed

- Diagnosis: on this host VS Code runs elevated while Lyn runs non-elevated; a medium-IL `code.exe` cannot drive the elevated singleton, so nothing opens. Reproduced and isolated (direct launch opens; de-elevated/reparented launch does not).
- Spike: `CreateProcessWithTokenW` with the Explorer token launches a GUI child (Notepad) at medium integrity from an elevated caller (`non-elevated` token confirmed while caller is `ELEVATED`).
- Replaced `startProcessAsShellUser` internals with the `CreateProcessWithTokenW` + shell-token primitive; removed the `cmd /c code` branch and Explorer reparenting. Fails closed if the shell token is elevated. Verified: real `startProcessAsShellUser` launches Notepad at `non-elevated` integrity from an `ELEVATED` caller. `just check` green. `docs/PLATFORM.md` updated.
- Removed the elevated-helper pipe IPC entirely (`elevation_helper_windows.go`, `elevation_ipc.go`, `elevation_session.go` + tests; `--elevated-helper` entry in `main.go`; `adminSession`/`dispatchLaunch` wiring in `app.go`; helper hooks in `elevation.go`/`elevation_windows.go`). `run-admin` now goes straight to per-launch `ShellExecute runAs`. Kept the `SwitchElevation` toggle. `just check` green.
- Verified empirically that an unsigned `uiAccess` exe will not launch, so the manifest stays out of the shared template until the signed + Program Files release path exists.

## Decisions

- Hotkey-over-admin-windows is solved with `uiAccess`, not by elevating the launcher (chosen over a minimal elevated key-broker and over accepting the limitation). Requires Authenticode signing and install under `%ProgramFiles%`.
- The launcher stays non-elevated by default; the manual elevation toggle remains as an explicit user choice.
- Even when elevated, child launches are de-elevated to the shell user (least privilege); the de-elevation primitive fails closed if the shell token is elevated.
- The elevated-helper pipe (`elevation_helper_windows.go`, `elevation_ipc.go`, `elevation_session.go`) is removed: a same-user pipe that runs paths elevated without UAC is an escalation surface. `run-admin` reverts to per-launch UAC consent.

## Validation

- `go test ./lyn/launch/...`, `just check`.
- Manual: hotkey over an elevated terminal; open a spaced-path workspace from both standard and elevated launcher; confirm child integrity is medium.

## Follow-up Debt

- Signing certificate provisioning for `uiAccess` (self-signed for dev; real cert for release) and the Program Files installer change are external to the code and tracked here until done.
