# 006 Host-Bound Manual Verification

## Goal

Run the live manual checks that automated `just check` and CI cannot cover, on the hosts where each feature actually runs, and confirm the shipped launcher behaves correctly end to end.

## Scope

- Re-run the full automated gate as a baseline.
- Exercise the three deferred manual paths on real hosts: WSL project handling, single-consent elevated launch, and live system commands plus remote ranking.

## Acceptance Criteria

- `just check` passes on the verifying host.
- WSL projects display as Unix paths (no `\\wsl.localhost`) and open via `code --remote wsl+<distro>` / `wsl.exe -d <distro>` (Windows + WSL host).
- A `run-admin` launch shows one UAC prompt, a second `run-admin` reuses it with no prompt, and cancelling reports cleanly (Windows desktop).
- `restart`, `shut`, and `log out` surface their system commands; an SSH recent sorts below a WSL recent and a workspace (live GUI session).

## Pending

- Baseline: run `just check` on the verifying host.
- WSL (Windows + WSL host): confirm Unix-path display and WSL-remote open/terminal/reveal.
- Elevated helper (Windows desktop): confirm single-consent UAC reuse and clean cancel; inspect `lyn --debug-log` for `elevation.helper.*` with no errors.
- System commands (live GUI session): confirm the three commands appear and search, and confirm SSH-remote ranks below WSL/local and workspaces. Launching the real power commands is destructive, so verify the ranking visually rather than triggering shutdown.

## Completed

## Decisions

- Consolidated from the deferred sections of the now-removed plans 004 (WSL), 004 (elevated helper), and 005 (system commands). The implementation for all three shipped and is covered by automated Go + e2e tests; only the live, host-bound observation remains.

## Validation

## Follow-up Debt

- WSL-native enumeration (`wsl.exe -d <distro> find`) would be faster than 9P UNC walks.
- No confirmation prompt before Restart/Shut Down; an accidental Enter powers the machine off.
- Linux logout assumes systemd `loginctl`; non-systemd sessions are unsupported.
