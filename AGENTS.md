# Agent Map

The repo is the source of truth. Put durable knowledge in markdown, code, tests, schemas, or plans.

## Important Rules

- The user's direct commands always take precedence over any guidance in this repo or default agent behavior. When a user instruction conflicts with a rule or doc below, follow the user and update the affected doc to match.

Read:

- `README.md` for setup, commands, and diagnostics.
- `ARCHITECTURE.md` before structure or boundary changes.
- `docs/QUALITY.md` for Go, Vue, Wails, cross-platform, and validation policy.
- `docs/exec-plans/README.md` before checked-in plans.

Rules:

- Keep `AGENTS.md` short; link deeper docs instead of duplicating them.
- Follow `ARCHITECTURE.md` for layout, folder, and file-split rules.
- Keep code and scripts free of comments; put rationale in markdown.
- Docs state the current rule, not its history: capture the decision and the single reason it matters, and drop incident narratives, before/after stories, and repeated context. See `docs/QUALITY.md` "Docs".
- Update docs when workflow, architecture, validation, reliability, or performance knowledge changes; editing a fact means rewriting it in place, not appending a new version.
- On `plan`, create or update `docs/exec-plans/active/NNN-topic.md` before coding.
- Work loop: inspect -> plan -> implement -> validate -> run/observe -> self-review -> handoff.
- Use isolated worktrees/env, ports, caches, logs, and app instances for parallel or risky work.
- Run `just check` before handoff.
