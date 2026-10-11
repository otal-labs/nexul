# 05 — Channels read their rules

**Status:** ready-for-agent

**Blocked by:** 02, 03

**Branch:** master, a normal PR, before the switch. Read only; nothing writes a channel rule yet.

Read first: `practices/go.md` (section 17), `practices/testing.md` (sections 3, 10), ADRs 0087, 0094, 0098, 0135,
0140, 0148, the spec (Channels and plays stay workspace-wide, Performance plan), the research (sections 1.6 and 9).

## What to build

- `readsChannel` (`internal/chat/usecase.go:145-154`) consults `channel` rules for the person and their roles,
  batched across the conversation list, unread counts, search, voice join and live frames, under the memo fence. A
  public channel's read starts held; a Deny makes the channel not found, as a private one is.
- The Owner is never denied. Private channels and Restricted behave as today. No writer.

## Acceptance criteria

- [ ] An empty `access.Matrix` diff on a production copy, in the PR.
- [ ] A channel denied to a role in a test is not found for its holders in lists, unread counts, search, voice join
      and live frames, and the Owner still sees it.
- [ ] The "chat conversation list" and "chat unread counts" rows of `TestStatements_ListsCostTheSameAtAnyLength`, with
      a rule on every channel, and the live-frames guard keep their counts.
