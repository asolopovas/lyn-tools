# 008 Linux hotkey requires CGO (cross-build fails with CGO_ENABLED=0)

## Goal

Building for Linux with `CGO_ENABLED=0` must not fail confusingly; the CGO requirement is explicit and cross-compiles either work or fail with a clear signal.

## Scope

- `lyn/hotkey/hotkeys_linux.go` (X11, `import "C"`, `#cgo LDFLAGS: -lX11`) is the only Linux definition of `registerHotkeyBinding`.
- `lyn/hotkey/hotkeys_default.go` is `//go:build darwin` only, so there is no non-CGO Linux fallback.

## Acceptance Criteria

- `GOOS=linux CGO_ENABLED=0 go build ./...` either compiles or fails with an explicit, documented message.
- Native Linux build (`scripts/build-linux.sh`, CGO on) keeps working unchanged.

## Pending

- Decide the fix: (a) document that Linux builds require `CGO_ENABLED=1` and have CI/build set it explicitly, and/or (b) add a `//go:build linux && !cgo` stub `registerHotkeyBinding` that returns an error so cross-builds compile.

## Completed

- (none)

## Decisions

- (none yet)

## Validation

- Symptom on Windows host: `GOOS=linux CGO_ENABLED=0 go build ./lyn/hotkey/` -> `lyn/hotkey/hotkeys.go:52:9: undefined: registerHotkeyBinding`. Confirmed pre-existing on clean `main` (v0.1.9), unrelated to the elevation refactor.

## Follow-up Debt

- Pre-existing; surfaced while cross-validating the launcher elevation work ([[007-non-elevated-launcher-uiaccess]]).
