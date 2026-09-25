# 37 — Decisions check and change context

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 26
**Decided in:** ticket 11, ticket 12

## What to build

When a card enters a done-stage column, fire the built-in decisions check on the mover's paired computer, or the developer's when a merged PR moved it. It writes, supersedes, or skips a three-line decisions-log entry. If it cannot run, the ticket shows "Decisions check didn't run" with a retry. Add `pull_request_get`: given a commit or PR number, return the PR, its tickets, their docs, bugs found after done, and citing decisions-log entries.

## Acceptance criteria

- [ ] Entering done fires exactly one check
- [ ] A run that cannot start is visible and retryable
- [ ] The change-context tool walks the chain from a PR number

## Surfaces

- Events consumer
- MCP tool

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `internal/tickets/ or internal/plays/ (consumer)`
- `internal/memories/`
- `internal/codereview/ or internal/gitprovider/ (change context)`

**Size:** M
