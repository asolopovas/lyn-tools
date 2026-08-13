# 009 Graphify Structure Refactor

## Goal

Refactor the backend and frontend structural hotspots identified by Graphify into focused collaborators and composables while preserving every exported Wails method, wire shape, persisted format, event, launch action, and user-visible behavior.

## Scope

- Keep `App` as a shallow Wails-facing facade.
- Move project indexing, search, scan, refresh, usage, and target resolution into `projectService`.
- Move window visibility, focus, suppression, mode, and quit state into `windowController`.
- Split application discovery into common, Windows, and Linux source files without build tags.
- Move theme, window-runtime, and keyboard responsibilities from `App.vue` into composables.
- Update architecture, backend, and frontend documentation.
- Regenerate Graphify output as uncommitted validation evidence when the tool is available.

## Acceptance Criteria

- Existing exported `App` methods and Wails contracts remain compatible.
- Backend behavior is covered by focused collaborator tests, including lifecycle and failure paths.
- Frontend composables have focused unit coverage for theme, window-runtime, and keyboard behavior.
- Application discovery remains cross-platform testable on every host.
- `go test -race ./lyn/...` passes where supported.
- `just check` passes.
- App and Graphify validation are recorded when available in the current environment.

## Pending

- Complete the host-bound launcher hotkey, focus-loss, watcher, restart, and tray-quit walkthrough when native Windows computer control is available and no installed Lyn instance owns the single-instance lock.

## Completed

- Re-read repository setup, architecture, quality, and execution-plan policy.
- Extracted `projectService` and `windowController` while retaining `App` as the exported Wails facade.
- Split application discovery into common, Windows-source, and Linux-source files without implicit or explicit build constraints.
- Extracted theme, Wails-window, and global-keyboard composables and reduced `App.vue` to composition and rendering.
- Added focused backend and frontend collaborator tests.
- Updated architecture, backend, and frontend ownership documentation.
- `go test ./lyn/... -count=1` passed.
- `pnpm --dir frontend test` passed: 11 files and 57 tests.
- `pnpm --dir frontend typecheck` passed.
- `go test -race ./lyn/...` passed.
- `just check` passed, including Go formatting/tests/vet/staticcheck, frontend formatting/lint/typecheck/unit tests, and 11 Playwright journeys.
- `just build` passed and produced the Windows application.
- Ran the built settings application with isolated config/cache directories; the debug trace recorded `startup.begin`, `config.loaded`, `store.opened`, `startup.settings.shown`, `window.close.allow`, `shutdown.begin`, and `shutdown.end` with no lifecycle errors.
- Regenerated ignored Graphify output with `uvx --from graphifyy graphify update .`: 1,413 nodes, 2,938 edges, 89 communities, and no import cycles. `projectService` and `windowController` are distinct hubs; window runtime, launcher keyboard, and theme state resolve into focused communities.
- Compared exported `App` methods against `HEAD`; the method set is identical.
- `git diff --check` passed.
- Self-reviewed collaborator lock ordering, shutdown serialization, startup sequencing, frontend mount/unmount cleanup, platform-source naming, and Wails facade compatibility.

## Decisions

- Preserve package layout and dependency direction; all new Go collaborators remain unexported in `lyn`.
- Treat existing tests as the behavioral compatibility baseline.
- Use `apps_windows_source.go` and `apps_linux_source.go`; Go treats `_<goos>.go` filenames as build constraints even without build-tag declarations.

## Validation

- Baseline supplied by the task: clean worktree and passing `just check` before the refactor.

## Follow-up Debt

- The installed Lyn process already owned the launcher single-instance lock, and the Windows computer-control runtime was unavailable because its native pipe was missing. The real settings process and automated journeys were validated, but the host-bound hotkey, native focus-loss, watcher, restart, and tray-quit walkthrough remains manual.
