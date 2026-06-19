# Lyn

Fast desktop launcher for Windows and Linux.

Lyn indexes project folders, apps, VS Code workspaces, and VS Code recent projects, then opens them from a global hotkey.

Version: `0.1.7`

## Features

- Project, app, VS Code workspace, and VS Code recent indexing.
- SQLite cache, filesystem watching, and usage-based ranking.
- Global hotkey, tray, startup, and hidden-background support.
- Open, reveal, VS Code, and terminal launch actions.
- Windows WSL root support.

## Requirements

- Go 1.26+
- pnpm 11+
- Wails v2.12+
- `just`
- `staticcheck`
- WebView2 runtime on Windows
- NSIS for Windows installers
- Docker for Windows-built `.deb` packages

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

## Usage

```bash
just tidy
just start
```

Config: `~/.config/lyn/lyn.json`. Use `lyn.example.json` as a starter. On Windows, WSL roots may be UNC paths like `\\wsl.localhost\Ubuntu\home\you\src` or Linux paths like `/home/you/src`.

## Diagnostics

```bash
lyn --debug
lyn --debug-log <path>
```

Default debug log: `lyn/debug.log` under the user cache directory (`%LOCALAPPDATA%\lyn\debug.log` on Windows). Trace `hotkey.*`, `window.*`, `frontend.*`, `launch.*`, and `scan.*` entries.

## Development

```bash
just format
just check
just build
just scan
```

`just install` installs locked frontend dependencies, builds the app, and installs the resulting binary. On Linux it first installs any missing native build libraries (GTK3, WebKit2GTK, Ayatana AppIndicator) via the system package manager.

Read `AGENTS.md` and `ARCHITECTURE.md` before structural changes. Use checked-in execution plans for complex work.

## Release

```bash
just bump patch
just release-patch
just release-minor
just release-major
```

Packages are built under `releases/` and published through GitHub CLI.

## License

MIT. See `LICENSE`.


