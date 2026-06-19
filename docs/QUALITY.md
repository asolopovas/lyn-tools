# Quality Policy

Applies to every change. Prefer mechanical enforcement over reminders.

## General

- Keep changes small, reversible, and test-covered.
- Encode decisions in docs, tests, schemas, or plans — no hidden knowledge.
- No dead code, unused abstractions, speculative layers, or duplicated launch/hotkey logic.
- Extract repeated logic into named helpers before a second copy; scout nearby files for reusable helpers first.
- Parse and validate external data at boundaries.
- Use structured logs for lifecycle, platform, launch, scan, watcher, and UI bridge failures.
- Every bug fix adds a regression test or documents why it can't.

## Go

- Idiomatic, simple Go; avoid clever generic or interface-heavy designs.
- Pass `context.Context` first for cancellable work.
- Bound goroutines with cancellation, timeouts, or owned shutdown.
- No sleeps for test correctness; use channels, contexts, polling with deadlines, or fakes.
- Wrap errors with operation context; log before swallowing platform API failures.
- Keep interfaces small and consumer-owned; return concrete types unless boundaries need otherwise.
- Protect mutable package state with locks or atomics; prefer immutable tables.
- Use build tags for platform behavior; non-target fallbacks are explicit no-ops or equivalent.
- Table-test parsing, ranking, path conversion, launch-command building, and platform decisions.
- Keep tests generic: no real user, host, company, domain, SSH alias, project, or filesystem names — use `example`, `examplehost`, temp paths.

## Vue / TypeScript

- Vue 3 Composition API with `<script setup>` and strict TypeScript.
- Keep components presentational; put reusable stateful logic in composables/modules.
- Route Wails calls through `frontend/src/backend.ts`, not generated bindings directly.
- Centralize and test keyboard/action mapping; share helpers for event consumption, key predicates, and action wiring.
- No divergent behavior across mouse, keyboard, and native fallback paths.
- Local storage is a warm render cache, not truth.
- Vitest for pure logic; Playwright for user journeys.

## Wails / Cross-platform

- Treat Windows, Linux, and unsupported platforms as explicit targets.
- Use native APIs for shortcuts, startup, foreground focus, reveal, and launch.
- Derive user/install/app/distro names from config, env, registry, or discovery — never hardcode.
- Keep helper terminals hidden except for intentionally interactive actions (`code`, terminal).
- Provide native fallback for critical launcher actions; don't rely only on WebView keyboard/focus.
- Mark interactive regions `--wails-draggable: no-drag`; avoid broad draggable regions in frameless windows.
- Startup, shutdown, tray, watcher, and hotkey registrations must be idempotent and tear down cleanly.
- Line endings are pinned to LF via `.gitattributes` (`* text=auto eol=lf`); without it `core.autocrlf=true` checks out CRLF and breaks `gofmt`/`oxfmt`. If files show renormalized, run `git add --renormalize .`.
- Keyboard-hook key state misses events during secure desktop and hook timeouts; reconcile against `GetAsyncKeyState` before suppressing input, or a missed Win keyup permanently swallows the hotkey letter.

## Validation

Run before handoff:

```bash
just check
```

For UI, launch, focus, hotkey, watcher, or platform changes also:

1. Build/install or run the real app.
2. Drive the user path with Playwright, DevTools, or OS automation.
3. Capture before/after screenshots when visual behavior matters.
4. Inspect debug logs for expected stages and absence of errors.
5. Restart and repeat if the fix changes lifecycle behavior.

## Docs

- One fact, one place. State the current rule, not how it was discovered or the old behavior.
- At most one sentence of "why" per non-obvious rule — the durable reason, not the bug; the regression test is the incident record.
- Edit in place: rewrite a fact, never append a new version beside the old.
- Prefer scannable bullets; a bullet past ~3 lines carries narrative that belongs in a test name, commit, or exec-plan.
- Keep rationale where enforced — regression test, lint, or schema over prose; docs only when no mechanical home exists.
- Split or trim a doc when it grows past its purpose, in the same change.

## Continuous Cleanup

- Convert repeated review comments into tests, lints, docs, or examples.
- Remove obsolete docs and stale generated artifacts in the change that makes them stale.
- Track larger cleanup in `docs/exec-plans/active/` or follow-up debt.
