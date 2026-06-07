# Lyn Launcher Foundation

## Completed

- Wails v2 desktop launcher with Go backend and Vue TypeScript frontend.
- Config, SQLite cache, scanner, app discovery, VS Code recent discovery, watcher, hotkey, tray, startup, and launch actions.
- Usage ranking, lazy app icons, settings UI, workspace filtering, and WSL root support.
- Repository harness: `AGENTS.md`, `ARCHITECTURE.md`, `justfile`, and execution-plan docs.

## Decisions

- Keep root Go files limited to executable bootstrap and root checks.
- Keep application/backend Go files in `lyn/`.
- Keep `frontend/`, `frontend/src`, and `frontend/dist` because Wails/Vite require them.
- Use SQLite WAL cache and source-owned replacement for refreshes.
- Bound scanner work by roots, depth, concurrency, timeout, and ignored folders.
- Load application icons lazily.

## Validation

- `go test ./...`
- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend build`
- `go build ./...`
- `just check`
- `go test -bench BenchmarkScanProjects -run '^$' ./...`
