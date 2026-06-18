# 004 WSL Unix Paths

## Goal

Index and open WSL projects using their Unix paths (e.g. `/home/you/src/app`) instead of
`\\wsl.localhost\...` Samba UNC paths, with reliable Windows↔WSL communication, a dedicated
WSL folder picker, and Windows/WSL folder separation in settings. Multi-distro, system
distros (e.g. `docker-desktop`) hidden.

## Scope

- Carry the WSL distro on each project so a Unix display path can still resolve to a distro.
- Scan WSL roots via `\\wsl.localhost\<distro>\...` for filesystem access but store Unix paths.
- Launch/open/reveal/terminal WSL projects through proper WSL mechanics (`code --remote
  wsl+<distro>`, `wsl.exe -d <distro> ...`).
- Settings: separate Windows and WSL folder lists; WSL picker via native dialog at
  `\\wsl.localhost`, stored/displayed as Unix; auto-migrate existing WSL roots.

## Acceptance Criteria

- WSL projects appear and are stored as Unix paths; no `\\wsl.localhost` shown in the UI.
- Opening a WSL project (code/terminal/reveal/open) uses the correct distro and Unix path.
- Adding a WSL folder via the picker stores `{distro, unixPath}` and shows the Unix path.
- `just check` passes.

## Pending

- Verify in the running app: WSL projects show as Unix paths and open via the WSL remote.

## Completed

- `wsl.go`: UNC↔Unix mapping, distro enumeration (`WSL_UTF8=1`), system-distro filter; tests.
- `Project.Distro` + `distro` SQLite column (additive migration) + stale `\\wsl%` purge.
- `ScannerConfig.WSLRoots` + legacy-root migration in `NormalizeConfig`; tests.
- Scanner scans WSL roots via UNC and records Unix paths + distro (`wslScannedProject`).
- `launch` threads distro: `code --remote wsl+<distro>`, `wsl.exe -d <distro> ...`; tests.
- `App.ChooseWSLFolder` (native dialog at `\\wsl.localhost`) + `App.WSLDistros`.
- Startup + folder-change now trigger a full rescan so WSL (unwatched) indexes promptly.
- Frontend: types, backend bindings, settings "WSL folders" section + picker, launch distro.
- `just check` green (Go quality + lint/typecheck/40 unit/11 e2e).

## Decisions

- Distro is stored as a new `Project.Distro` field and a `distro` SQLite column (additive
  migration), keeping `path` as the Unix path and primary key.
- Config gains `Scanner.WSLRoots []WSLRoot{Distro, Path}`; legacy WSL entries in `Roots`
  (Unix or `\\wsl.localhost\` UNC) migrate into `WSLRoots` during normalization.
- WSL scanning keeps using UNC traversal from Windows (same cost as today); a WSL-native
  `find` scan is possible later (see debt).

## Validation

## Follow-up Debt

- WSL-native enumeration (`wsl.exe -d <distro> find`) would be much faster than 9P UNC walks.
- `path` primary key means identical absolute Unix paths in two distros collide (rare).
