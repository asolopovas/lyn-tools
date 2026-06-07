# Code Review Improvements

## Completed

- Optimized scanner file handling for `.code-workspace` detection.
- Split directory project detection from file detection.
- Improved VS Code recent-history error handling.
- Returned watcher setup failures when no configured root can be watched.
- Made terminal launch use containing folders for file targets.
- Reduced frontend search work by ranking once and limiting matches.
- Validated imported themes before mutating runtime theme state.
- Removed initial empty-state flashes by gating launcher readiness on initial project load.

## Validation

- LSP diagnostics on changed files.
- `just check`
