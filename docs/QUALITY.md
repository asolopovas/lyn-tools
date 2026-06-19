# Quality Policy

Applies to every change. Prefer mechanical enforcement over reminders.

## Principles

- Keep changes small, reversible, and test-covered.
- Every bug fix adds a regression test, or a note on why it can't.
- Encode decisions in docs, tests, schemas, or plans, not hidden knowledge.
- No dead code, unused abstractions, speculative layers, or duplicated launch/hotkey logic.
- Before writing a second copy, extract repeated logic into a named helper, reusing nearby ones first.
- Validate external data at boundaries.
- Use structured logs for lifecycle, platform, launch, scan, watcher, and UI-bridge failures.

## Go

- Write idiomatic, simple Go, not clever generic or interface-heavy designs.
- Pass `context.Context` first for cancellable work.
- Bound goroutines with cancellation, timeouts, or owned shutdown.
- Never use sleeps for test correctness, only channels, contexts, polling with deadlines, or fakes.
- Wrap errors with operation context.
- Log before swallowing platform API failures.
- Keep interfaces small and consumer-owned, returning concrete types unless a consumer must swap implementations.
- Prefer immutable tables, protecting any mutable package state with locks or atomics.
- Use build tags for platform behavior, with non-target fallbacks as explicit no-ops or a returned unsupported-platform error.
- Table-test parsing, ranking, path conversion, launch-command building, and platform decisions.
- Keep tests generic with `example`, `examplehost`, and temp paths, never real user, host, company, domain, SSH-alias, project, or filesystem names.

## Vue / TypeScript

- Use Vue 3 Composition API with `<script setup>` and strict TypeScript.
- Keep components presentational, with reusable state in composables or modules.
- Route Wails calls through `frontend/src/backend.ts`, not generated bindings directly.
- Centralize and test keyboard/action mapping, sharing helpers.
- Keep behavior identical across mouse, keyboard, and native fallback.
- Local storage is a warm render cache, not truth.
- Vitest for pure logic, Playwright for user journeys.

## Wails / Cross-platform

- Treat Windows, Linux, and unsupported platforms as explicit targets.
- Use native APIs for shortcuts, startup, foreground focus, reveal, and launch.
- Derive user, install, app, and distro names from config, env, registry, or discovery instead of hardcoding.
- Keep helper terminals hidden except for interactive actions (`code`, terminal).
- Give critical launcher actions a native fallback, not only WebView keyboard/focus.
- In frameless windows, avoid broad draggable regions and mark interactive regions `--wails-draggable: no-drag`.
- Startup, shutdown, tray, watcher, and hotkey registrations must be idempotent and tear down cleanly.
- LF is pinned via `.gitattributes` (`* text=auto eol=lf`). Without it, `core.autocrlf=true` checks out CRLF and breaks `gofmt`/`oxfmt`. If files show renormalized, run `git add --renormalize .`.
- Keyboard-hook key state misses events during secure desktop and hook timeouts. Reconcile against `GetAsyncKeyState` before suppressing input, or a missed Win keyup permanently swallows the hotkey letter.

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

- One fact, one place. State the current rule, not how it was found or the old behavior.
- Edit in place by rewriting, never appending a new version.
- One sentence of "why" max per non-obvious rule. The regression test is the incident record.
- Keep rationale where it is enforced (test, lint, or schema over prose). Use docs only when no mechanical home exists.
- A bullet past ~3 lines belongs in a test name, commit, or exec-plan. Split or trim a doc that grows past its purpose in the same change.
- Convert repeated review comments into tests, lints, docs, or examples.
- Remove obsolete docs and stale artifacts in the change that makes them stale.
- Track larger cleanup in `docs/exec-plans/active/`.
