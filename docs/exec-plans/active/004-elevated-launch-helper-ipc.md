# 004 Elevated Launch Helper IPC

## Goal
Let repeated "run as administrator" launches reuse a single UAC consent by routing them through a persistent elevated helper process, instead of prompting UAC on every `run-admin` launch.

## Background
- Today `run-admin` calls `ShellExecuteW` with verb `runAs` per launch (`lyn/launch/launcher.go`, `lyn/launch/launcher_process_windows.go`), so every admin launch shows a UAC prompt.
- `SwitchElevation` restarts the whole app elevated; it is unchanged by this plan.
- PowerToys solves the same problem with a runner that spawns an elevated process once and talks to it over a secured named pipe (`src/common/interop/two_way_pipe_message_ipc.cpp`, `src/common/utils/elevation.h`).

## Design
- Same binary, hidden helper mode: `lyn --elevated-helper=<pipe>`. `main.go` intercepts this before Wails starts and runs the helper loop, never showing a window.
- Transport is a single duplex Windows named pipe, message framed as newline-delimited JSON.
- Integrity model: the non-elevated UI creates the pipe **server** (medium integrity); the elevated helper connects as **client**. A high-integrity client writing to a medium-integrity object is a write-down (allowed), which avoids the no-write-up block that occurs if an elevated process owns the pipe.
- Pipe name is an unguessable GUID passed to the helper on the command line. Server security descriptor grants the current user SID `GENERIC_ALL` (SDDL `D:P(A;;GA;;;<sid>)`), scoping access to this user.
- Protocol:
  - `helperRequest{ id, launch.Request }`
  - `helperResponse{ id, launch.Result }`
- The helper only executes whitelisted actions and rejects `lyn:system:` targets. For `run-admin` it starts the target executable directly (inheriting the helper's elevated token) rather than going through Explorer, which would drop back to medium integrity.
- Routing: `App.Launch` sends `run-admin` requests to the session when the app is not already elevated and the platform is Windows; if the session cannot start or the call fails, it falls back to the existing per-launch `ShellExecuteW runAs` path.
- The session starts lazily on the first `run-admin` launch (one UAC), then stays connected for subsequent launches; it is torn down on shutdown.

## Scope
- Add neutral protocol + helper serve loop + admin-session manager with injectable transport seam.
- Add Windows transport (pipe server with SDDL, `runas` spawn, accept), Windows helper executor, and helper-mode entry.
- Route `run-admin` through the session with fallback.
- No frontend change: the optimization is transparent to the existing UI.

## Acceptance Criteria
- Helper-mode arg parsing starts the helper and skips Wails.
- Protocol round-trips and rejects malformed or oversized frames.
- Helper serve loop executes a request via an injected launcher and returns the matching response keyed by id; it rejects non-whitelisted actions and system targets.
- Admin-session manager: starts the transport once, reuses the connection across calls, and falls back to the direct launcher when the transport or call fails.
- `App.Launch` routes `run-admin` through the session on Windows when not already elevated and otherwise behaves as before.
- Non-Windows builds keep an explicit no-op session (direct launch only).
- Focused Go tests cover arg parsing, protocol, serve loop, manager reuse/fallback, and launch routing.
- `just check` passes.

## Pending
- Manual live UAC verification on a Windows desktop (cannot run headlessly).

## Completed
- Added neutral protocol, helper serve loop, and action whitelist in `lyn/elevation_ipc.go`.
- Added the admin-session manager with an injectable transport seam in `lyn/elevation_session.go`.
- Added the Windows named-pipe transport (server with user-scoped SDDL, `runas` spawn, timed accept), Windows helper executor, and helper-mode entry in `lyn/elevation_helper_windows.go`.
- Wired neutral seams in `lyn/elevation.go`, Windows seam registration and a show flag on `shellExecuteProcess` in `lyn/elevation_windows.go`, `--elevated-helper=` interception in `main.go`, and `run-admin` routing plus shutdown teardown in `lyn/app.go`.
- Added Go tests for protocol round-trip, whitelist, malformed/oversized frames, session reuse/fallback/reconnect/id-mismatch, routing predicate, and helper-mode arg parsing.

## Decisions
- No new dependency: use `golang.org/x/sys/windows` (already direct) for the pipe and `github.com/google/uuid` (already indirect, promote to direct) for the pipe name, consistent with the repo's no-extra-dependency stance.
- UI-as-server / helper-as-client to avoid mandatory-integrity write-up instead of relabeling an elevated-owned pipe.
- Limit the session to `run-admin`. `run-user` keeps the existing per-launch path; it cannot inherit elevation by token and is rare.

## Manual Verification (live UAC path, run on Windows desktop)
- Trigger a `run-admin` launch; accept the single UAC prompt; confirm the target starts elevated.
- Trigger a second `run-admin` launch; confirm no second UAC prompt and the target starts elevated.
- Cancel the UAC prompt; confirm the launch reports the cancel and the app stays responsive.
- Inspect `lyn --debug-log` for `elevation.helper.*` stages and no errors.

## Validation
- `just check` passed: gofmt, `go test ./...`, `go vet ./...`, `staticcheck ./...`, and frontend format/lint/typecheck/unit/e2e.
- Live UAC path is unverified in headless CI; run the Manual Verification steps on a Windows desktop.

## Follow-up Debt
- Optional hardening: verify the connected client PID/image path on the server side before trusting the connection.
