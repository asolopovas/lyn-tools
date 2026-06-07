# Execution Plans

Use checked-in plans for complex or risky work. Keep small work in the prompt.

Location:

- Active: `docs/exec-plans/active/NNN-topic.md`
- Closed: `docs/exec-plans/completed/NNN-topic.md`

Required sections:

```markdown
# NNN Topic

## Goal
## Scope
## Acceptance Criteria
## Pending
## Completed
## Decisions
## Validation
## Follow-up Debt
```

Rules:

- Create or update a plan before coding when the user asks for `plan`.
- Keep exactly the current task in `## Pending`.
- Move verified work to `## Completed` with commands, results, logs, screenshots, or traces.
- Record decisions and debt when they affect future agents.
- Move to `completed/` only when shipped or closed.
