# 23 — Tell machine managers when a token is rejected

**Status:** resolved

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

## Comments

Built: the `secret_manager.token_rejected` event and `secretref.NewlyRejected`. The runner and host repos write the event in the same transaction that stores the new states, so two connections can't double-fire. The server's states are not stored, so a server restart with a still-rejected token tells the Owner again. Audience: `access.MachineManagers` and `access.InstanceOwners`, one inbox row each, filed in the recipient's first workspace. The new notification kind appears on web and phone. Checked against a real server: a runner with a bogus token put one notification in the owner's inbox, and restarting it with the same token added none.
