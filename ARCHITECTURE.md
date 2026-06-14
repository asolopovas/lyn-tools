# Architecture

Lyn is a Wails desktop launcher for projects, apps, VS Code recents, and workspaces.

## Layout

- Root Go files bootstrap the app, embed frontend assets, and enforce structure.
- `lyn/` is the backend application package.
- `lyn/hotkey/`, `lyn/launch/`, `lyn/startup/`, and `lyn/tray/` are focused backend domains imported by `lyn/app.go`.
- `frontend/` is the Vue/Vite workspace; `frontend/dist` is embedded for production.
- Add folders only for cohesive multi-file packages, build tags, generated output, assets, or tool-required paths.
- Avoid `cmd`, `internal`, and `pkg` until multiple binaries, reusable internals, or public APIs justify them.

## Backend

- `lyn/app.go` owns Wails lifecycle, state, cache refresh, watcher restart, tray, hotkey, window, and frontend bindings.
- Shared domains live in `config`, `cache`, `project`, `scanner`, `apps`, `vscode`, and `watcher` files.
- SQLite stores cached items and launch usage; all SQLite connections stay in Go, including VS Code `state.vscdb` recent-project reads.
- The backend owns launcher indexing, search, typo-tolerant matching, workspace shortcut filtering, and ranking.
- Ranking order: match quality, usage count, last launch time, name, path.
- Scanning is bounded by configured roots, depth, concurrency, timeout, ignored folders, and source-owned cache replacement.
- `path.go` is the single home for path include/skip controls (traversal skips, packaged-dependency and Windows `Startup` folders, Windows system-dir containment); scanner, watcher, apps, and vscode callers delegate to it instead of reimplementing path checks. Tests live in `path_test.go`.
- Ignored folders include build/dependency dirs (`node_modules`, `vendor`, `dist`, ...) and versioned package-store entries (any `@<digit>` in the name, e.g. bun/pnpm cache dirs like `accepts@1.3.8`), so third-party packages are never indexed as projects.
- Project detection markers, highest priority first: WordPress, Laravel, Go, Node, Rust, `.vscode` workspace, then any `.git` repository.
- Unreachable configured roots (missing path, offline WSL/UNC share, permission error) are reported by `ScanProjects` and logged as `scan.root.skip`; a scan with skipped roots merges into the cache instead of replacing it, so a transient root never purges previously indexed projects.

## Frontend

- `frontend/src/App.vue` coordinates launcher UI state and Wails events.
- The frontend is a thin input and rendering layer: it sends query text to backend search and renders returned matches.
- Components render panels; small modules own backend access, cache, themes, icons, types, hotkeys, and launch actions.
- Keyboard mappings live in `frontend/src/hotkeys.ts` and are reused by input and window handlers.
- Local storage is only a warm UI cache, never search or ranking truth.

## Platform

- Windows discovery indexes Start Menu/Desktop shortcuts and GUI executables from PATH; console tools and the `Startup` auto-run folder are skipped.
- Windows launch/reveal/startup use native APIs when needed to avoid helper terminals.
- WSL roots may be UNC paths or Linux paths converted by `wsl.exe wslpath -w`.
- Linux discovery uses desktop entries and common icon theme paths.
- Tray support is split by build tags; unsupported platforms use a no-op fallback.

## Harness

- `docs/QUALITY.md` is the strict development policy for Go, Vue, Wails, cross-platform behavior, and validation.
- `just check` is the local and CI quality gate: formatting, Go tests, vet, staticcheck, frontend lint/typecheck/unit/e2e.
- Structural tests enforce folder rules.
- Add focused tests or benchmarks for traversal, ranking, cache writes, launch, focus, watcher, icon, and platform behavior.
- Validate UI changes by driving the running app or Playwright path and inspecting screenshots/logs.
- Debug with `lyn --debug-log <path>` or `LYN_DEBUG_LOG`; key stages include `hotkey.*`, `window.*`, `frontend.*`, `launch.*`, and `scan.*`.
