# Architecture

Lyn is a Wails desktop launcher for projects, apps, VS Code recents, and workspaces.

This file is the top-level map: package layout and the layering rules that hold the system together. Subsystem behavior lives in the linked pages.

## Layout

- Root Go files bootstrap the app, embed frontend assets, and enforce structure.
- [`lyn/`](lyn) is the backend package. [`lyn/hotkey/`](lyn/hotkey), [`lyn/launch/`](lyn/launch), [`lyn/startup/`](lyn/startup), [`lyn/tray/`](lyn/tray) are focused domains imported by [`lyn/app.go`](lyn/app.go).
- [`frontend/`](frontend) is the Vue/Vite workspace. `frontend/dist` is embedded for production.
- Add folders only for cohesive multi-file packages, build tags, generated output, assets, or tool-required paths. Keep small Go code in root files.
- Avoid `cmd`, `internal`, `pkg` until multiple binaries, reusable internals, or public APIs justify them.

## Layering

- The frontend is a thin input/render layer. The backend owns all indexing, search, matching, and ranking. The frontend never ranks.
- Dependency direction: root `main` and [`lyn/app.go`](lyn/app.go) import the domain packages ([`lyn/hotkey/`](lyn/hotkey), [`lyn/launch/`](lyn/launch), [`lyn/startup/`](lyn/startup), [`lyn/tray/`](lyn/tray)). Domains never import the parent `lyn` package and never import each other.
- [`architecture_test.go`](architecture_test.go) enforces folder rules, the layering direction, and that links in these docs resolve, so structure and docs cannot drift.

## Subsystems

- [Backend](docs/BACKEND.md): lifecycle, close-to-tray, ranking, scanning and detection.
- [Frontend](docs/FRONTEND.md): UI layer, panel sizing, WebKitGTK constraints.
- [Platform](docs/PLATFORM.md): VS Code launch, Windows, Linux, tray.

## Development

- [Quality policy](docs/QUALITY.md): Go, Vue, Wails, cross-platform, and validation rules. `just check` is the gate.
- [Execution plans](docs/exec-plans/README.md): when and how to check in plans.
- Diagnostics and debug logging: see [README](README.md).
