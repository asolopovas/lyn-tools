# 003 Backend Fuzzy Search

## Goal
Move launcher matching and ranking to the Go backend so the Vue frontend only renders state and sends query text.

## Scope
- Add a backend search API that returns the top launcher matches for a query.
- Preserve workspace shortcut filtering.
- Keep usage/last-launch ranking in Go.
- Add fast typo-tolerant matching with bounded edit distance.
- Remove frontend matching/ranking logic from the live launcher path.

## Acceptance Criteria
- Query matching is performed by Go backend code, not frontend utilities.
- Empty query returns usage-ranked projects.
- Exact substring results outrank fuzzy typo results.
- Workspace shortcut excludes apps and system commands.
- Typo queries such as `calclator` can match `Calculator`.
- Frontend calls backend search when query/projects/config changes and renders returned matches.
- Focused Go and frontend tests cover the new behavior.
- SQLite access, including Lyn cache and VS Code `state.vscdb`, remains Go-only.

## Pending

## Completed
- Added `App.SearchProjects` Wails API backed by an in-memory Go search index.
- Added Go ranking helper and typo-tolerant backend search in `lyn/project_search.go`.
- Kept VS Code recent-project ingestion in Go; existing `state.vscdb` SQLite reader remains in `lyn/vscode.go`.
- Removed frontend project indexing, filtering, and ranking from the launcher path.
- Added Go tests for ranking, typo matching, exact-before-fuzzy ordering, and workspace shortcut filtering.
- Added frontend binding test and e2e mock support for backend search.
- Validation passed: `go test ./...`, `pnpm --dir frontend lint`, `pnpm --dir frontend typecheck`, `pnpm --dir frontend test`, and `just check`.
- Cleaned stale frontend search/filter/ranking state and documented the backend-owned processing boundary in `ARCHITECTURE.md`.

## Decisions
- Use a small in-repo bounded Damerau-Levenshtein fallback instead of adding a dependency so common exact/prefix searches stay allocation-light and fuzzy work is capped.
- Keep SQLite connections exclusively in Go. Frontend only talks to Wails bindings.

## Validation
- `just check` passed, including Go format check, `go test ./...`, `go vet ./...`, `staticcheck ./...`, frontend format/lint/typecheck/unit tests, and Playwright e2e.
- LSP diagnostics found no new errors in changed frontend and Go files; workspace-wide TypeScript deprecation hints remain in existing `frontend/src/hotkeys.ts` for `keyCode`/`which` compatibility.

## Follow-up Debt
