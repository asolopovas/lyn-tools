# 005 System Commands and Remote Ranking

## Goal

Re-enable the Log Out / Restart / Shut Down launcher results and rank VS Code
SSH-remote connections below workspaces and WSL/local results.

## Scope

- Provide three virtual `system-command` results (`lyn:system:restart`,
  `lyn:system:shutdown`, `lyn:system:logout`) and inject them into the search
  index without persisting them to SQLite.
- Execute the real OS power command on launch (Windows `shutdown.exe`, Linux
  `systemctl`/`loginctl`, macOS `osascript`).
- Move SSH-remote VS Code entries below workspaces and WSL/local entries in the
  ranking tiebreak.
- Remove the disable-era guards that hid system commands (frontend cache filter,
  launch rejection, misleading helper name) while keeping the protective cache
  purge so a stale persisted command never resurfaces.

Out of scope: a launch confirmation dialog for destructive power actions (see
Follow-up Debt) and per-command usage persistence.

## Acceptance Criteria

- Typing `restart`, `shut`, or `log out` surfaces the matching system command.
- Pressing Enter on a system command runs the real OS command on Windows with no
  flashing console window; non-Windows builds emit the documented command.
- System commands never enter SQLite (`store.ListProjects` returns none after a
  rescan) and never enter the localStorage cache untyped state regressions.
- For equal match quality, ordering is workspace > WSL/local > SSH-remote >
  system-command.
- `just check` passes.

## Pending

- Manual run-through in the live app (launch each command, confirm SSH sorts
  below WSL/workspace visually).

## Completed

- System-command provider + in-memory injection (`lyn/system_command.go`,
  `indexProjects`); never persisted (`RecordLaunch` is UPDATE-only, 0 rows).
- Real OS launch commands (`lyn/launch/system.go`) replacing the rejection in
  `BuildLaunchCommand`; Windows ShellExecute routing excludes system paths.
- `kindRank` ranking: workspace > WSL/local > SSH-remote > system-command, with
  `isSSHRemoteProject` matching `ssh-remote+`/`%2b`.
- Removed disable-era guards: frontend `isSafeProject` cache filter; renamed
  `isDisabledSystemPath` -> `isSystemCommandPath`.
- Tests: provider, ranking, search-order, and per-OS launch-command tests added;
  two exact-count app tests filtered via `withoutSystemCommands`.
- `just check` green: gofmt, `go test ./...`, vet, staticcheck, frontend
  format/lint/typecheck, 40 vitest, 11 Playwright e2e.

## Decisions

- System commands are virtual: injected in `indexProjects` alongside live
  recents, so they ride every read path (`projects`, `rescan`, `RefreshProjects`)
  yet never reach `store` writes (callers persist `cacheItems` before indexing).
- The cache purge `DELETE ... kind='system-command' OR path LIKE 'lyn:system:%'`
  is kept as defense-in-depth, not removed: nothing should persist these, and the
  purge cleans any stale rows from older builds.
- `isDisabledSystemPath` is renamed `isSystemCommandPath`. The interceptor still
  skips system paths for the native run-admin/terminal/reveal shortcuts (those
  actions are meaningless for power commands); only the name was misleading.
- Linux logout uses `loginctl terminate-user <user>` (DE-agnostic on systemd)
  with the username resolved through an overridable package var for tests.
- Ranking gains a single `kindRank`: workspace=0, default=1, SSH-remote=2,
  system-command=3. SSH detection mirrors the frontend (`vscode-remote://ssh-remote+`
  and `%2b`). SSH workspaces stay rank 0 because workspace kind wins first.

## Validation

- `go test ./...`, `just check`.
- Manual: launch the app, confirm the three commands appear/search and that an
  SSH recent sorts below a WSL recent and a workspace.

## Follow-up Debt

- No confirmation prompt before Restart/Shut Down; an accidental Enter powers the
  machine off. Consider a guarded confirm action.
- Linux logout assumes systemd `loginctl`; non-systemd sessions are unsupported.
