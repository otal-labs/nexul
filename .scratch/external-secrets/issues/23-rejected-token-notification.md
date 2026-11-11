# 23 — Tell machine managers when a token is rejected

**Status:** ready-for-agent

**Blocked by:** 16, 19, 21

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, the spec (States and who hears about them).

## What to build

When a runner's, host's or the server's manager state turns `token rejected`, one inbox notification
goes to the people who can Manage that machine, or to the Owner for the server. It goes once per change
into that state, not on every reconnect.

## Acceptance criteria

- [ ] A state that turns rejected notifies exactly the machine's managers, once; reconnecting while still
      rejected notifies nobody; recovering and failing again notifies again.

Verify: `make lint`, `go test ./internal/...`.
