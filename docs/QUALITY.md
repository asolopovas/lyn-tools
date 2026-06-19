# Quality Policy

Applies to every change. Prefer mechanical enforcement over reminders.

## General

- Keep changes small, reversible, and covered by tests.
- No hidden knowledge: encode decisions in docs, tests, schemas, generated files, or plans.
- No dead code, unused abstractions, speculative layers, or duplicated launch/hotkey logic.
- Extract repeated logic into named functions or small modules before adding a second copy; during refactors, scout nearby files for reusable helpers before creating new logic.
- Parse and validate external data at boundaries.
- Use structured logs for lifecycle, platform, launch, scan, watcher, and UI bridge failures.
- Any bug fix must add a regression test or document why it cannot.

## Go

- Use idiomatic, simple Go; avoid clever generic or interface-heavy designs.
- Pass `context.Context` first for cancellable work.
- Bound goroutines with cancellation, timeouts, or owned shutdown paths.
- Never use sleeps for correctness in tests; use channels, contexts, polling with deadlines, or fakes.
- Wrap errors with operation context; do not swallow platform API failures unless logged.
- Keep interfaces small and consumer-owned; return concrete types unless tests or boundaries need otherwise.
- Protect mutable package state with locks or atomics; prefer immutable tables.
- Use build tags for platform behavior; non-target fallbacks must be explicit no-ops or equivalent behavior.
- Add table tests for parsing, ranking, path conversion, launch command building, and platform decisions.
- Keep tests generic and user-independent: never use real user, host, company, domain, SSH alias, project, or filesystem names; use examples such as `example`, `examplehost`, and temp paths.

## Vue / TypeScript

- Use Vue 3 Composition API with `<script setup>` and strict TypeScript.
- Keep components presentational when possible; put reusable stateful logic in small modules/composables.
- Keep Wails calls behind `frontend/src/backend.ts`; do not call generated bindings directly from components.
- Keep keyboard/action mapping centralized and tested.
- Use shared helpers for repeated event consumption, key predicates, and UI action wiring.
- Avoid divergent behavior between mouse, keyboard, and native fallback paths.
- Do not depend on local storage as truth; it is only a warm render cache.
- Add Vitest coverage for pure logic and Playwright coverage for user journeys.

## Wails / Cross-platform

- Treat Windows, Linux, and unsupported platforms as explicit targets.
- Use native APIs for OS behavior that shells handle poorly: shortcuts, startup, foreground focus, reveal, and launch.
- Do not hardcode user paths, install paths, app names, or distro names; derive them from config, env, registry, or discovery.
- Keep helper terminals hidden except for intentionally interactive actions (`code`, terminal).
- Do not rely only on WebView keyboard/focus delivery for critical launcher actions; provide native fallback where practical.
- Avoid broad draggable regions in frameless windows; mark interactive regions `--wails-draggable: no-drag`.
- Startup, shutdown, tray, watcher, and hotkey registrations must be idempotent and cleanly tear down.
- Line endings are pinned to LF via `.gitattributes` (`* text=auto eol=lf`). `core.autocrlf=true` would otherwise check Go/TS files out as CRLF on Windows and make `gofmt`/`oxfmt` reject them even though the committed blobs are clean. Do not remove this; if files ever show as renormalized, run `git add --renormalize .`.
- Low-level keyboard hooks miss events during secure desktop (lock screen, UAC) and hook timeouts; any key state tracked from hook events must reconcile against `GetAsyncKeyState` before suppressing input, or a missed Win keyup permanently swallows the hotkey letter.

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

- One fact, one place. State the current rule or constraint; do not narrate how it was discovered or what the old behavior was.
- Give each non-obvious rule at most one sentence of "why" — the durable reason it holds, not the bug that prompted it. The regression test is the incident record; the doc is the rule.
- Edit in place. Changing a fact means rewriting that line, never appending a newer version beside the old one.
- Prefer scannable bullets over paragraphs; a bullet that runs past ~3 lines is a sign it carries narrative that belongs in a test name, commit message, or exec-plan.
- Keep rationale where it is enforced: prefer a regression test, lint, or schema over prose; reach for docs only when no mechanical home exists.
- When a doc grows past its purpose, split or trim it in the same change rather than letting it accrete.

## Continuous Cleanup

- Convert repeated review comments into tests, lints, docs, or examples.
- Remove obsolete docs and stale generated artifacts in the same change that makes them stale.
- Track larger cleanup in `docs/exec-plans/active/` or follow-up debt.
