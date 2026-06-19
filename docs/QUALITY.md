# Quality Policy

Applies to every change. Prefer mechanical enforcement over reminders.

## Principles

- Small, reversible, test-covered changes; every bug fix adds a regression test or notes why it can't.
- Encode decisions in docs, tests, schemas, or plans — no hidden knowledge.
- No dead code, unused abstractions, speculative layers, or duplicated launch/hotkey logic.
- Extract repeated logic into named helpers before a second copy; reuse nearby helpers first.
- Validate external data at boundaries; structured-log lifecycle, platform, launch, scan, watcher, and UI-bridge failures.

## Go

- Idiomatic, simple Go; no clever generic or interface-heavy designs.
- `context.Context` first for cancellable work; bound goroutines with cancellation, timeouts, or owned shutdown.
- No sleeps for test correctness — use channels, contexts, polling with deadlines, or fakes.
- Wrap errors with operation context; log before swallowing platform API failures.
- Small consumer-owned interfaces; return concrete types unless boundaries need otherwise.
- Protect mutable package state with locks or atomics; prefer immutable tables.
- Build tags for platform behavior; non-target fallbacks are explicit no-ops or equivalent.
- Table-test parsing, ranking, path conversion, launch-command building, and platform decisions.
- Generic tests: no real user/host/company/domain/SSH-alias/project/filesystem names — use `example`, `examplehost`, temp paths.

## Vue / TypeScript

- Vue 3 Composition API, `<script setup>`, strict TypeScript; components presentational, reusable state in composables/modules.
- Route Wails calls through `frontend/src/backend.ts`, not generated bindings directly.
- Centralize and test keyboard/action mapping; share helpers, no divergence across mouse, keyboard, and native fallback.
- Local storage is a warm render cache, not truth.
- Vitest for pure logic; Playwright for user journeys.

## Wails / Cross-platform

- Windows, Linux, and unsupported platforms are explicit targets; use native APIs for shortcuts, startup, foreground focus, reveal, and launch.
- Derive user/install/app/distro names from config, env, registry, or discovery — never hardcode.
- Keep helper terminals hidden except for interactive actions (`code`, terminal); give critical launcher actions a native fallback, don't rely only on WebView keyboard/focus.
- Mark interactive regions `--wails-draggable: no-drag`; avoid broad draggable regions in frameless windows.
- Startup, shutdown, tray, watcher, and hotkey registrations must be idempotent and tear down cleanly.
- LF pinned via `.gitattributes` (`* text=auto eol=lf`); without it `core.autocrlf=true` checks out CRLF and breaks `gofmt`/`oxfmt`. If files show renormalized, run `git add --renormalize .`.
- Keyboard-hook key state misses events during secure desktop and hook timeouts; reconcile against `GetAsyncKeyState` before suppressing input, or a missed Win keyup permanently swallows the hotkey letter.

## Validation

Run `just check` before handoff:

```bash
just check
```

For UI, launch, focus, hotkey, watcher, or platform changes, also:

1. Run the real app.
2. Drive the user path (Playwright, DevTools, or OS automation).
3. Screenshot before/after when visual behavior matters.
4. Check debug logs for expected stages and no errors.
5. Restart and repeat if the fix changes lifecycle.

## Docs & cleanup

- One fact, one place; state the current rule, not how it was found or the old behavior. Edit in place — rewrite, never append a new version.
- One sentence of "why" max per non-obvious rule; the regression test is the incident record.
- Keep rationale where enforced — test, lint, or schema over prose; docs only when no mechanical home exists.
- Scannable bullets; a bullet past ~3 lines belongs in a test name, commit, or exec-plan. Split or trim a doc that grows past its purpose, same change.
- Convert repeated review comments into tests, lints, docs, or examples; remove obsolete docs and stale artifacts in the change that makes them stale; track larger cleanup in `docs/exec-plans/active/`.
