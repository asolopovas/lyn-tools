# Agent Map

The repo is the source of truth. Put durable knowledge in markdown, code, tests, schemas, or plans.

Read:

- `README.md` for setup, commands, and diagnostics.
- `ARCHITECTURE.md` before structure or boundary changes.
- `docs/QUALITY.md` for Go, Vue, Wails, cross-platform, and validation policy.
- `docs/exec-plans/README.md` before checked-in plans.

Rules:

- Keep `AGENTS.md` short; link deeper docs instead of duplicating them.
- Do not create a folder for one file unless a tool, asset, generated output, package, or build tag requires it.
- Split files only for real package, build-tag, generated, asset, test, or cohesive domain boundaries.
- Prefer root files for small Go application code.
- Keep code and scripts free of comments; put rationale in markdown.
- Update docs when workflow, architecture, validation, reliability, or performance knowledge changes.
- On `plan`, create or update `docs/exec-plans/active/NNN-topic.md` before coding.
- Work loop: inspect -> plan -> implement -> validate -> run/observe -> self-review -> handoff.
- Use isolated worktrees/env, ports, caches, logs, and app instances for parallel or risky work.
- Run `just check` before handoff.
