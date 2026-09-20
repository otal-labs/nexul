# 06 — What presence can tell the button today

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

The play button is disabled with a reason when the clicking user has no
paired harness or it is offline. Establish, from the code on master, what
can actually be known per user before a click:

- How `internal/presence` (`keeper.go`) and `harness.Client.Hold` decide a
  paired harness is connected, per user and per computer, and how that
  reaches the web today (any hook, live topic, or endpoint).
- How `pairing.ResolveTarget` picks the computer and harness project for a
  given user and project, and every `NotConfiguredReason` it can return, so
  the button's disabled reasons map one-to-one onto real states.
- Whether the web already knows the current user's pairing state anywhere
  (settings page hooks) that a ticket page could reuse without a new query.
- Any gap: a state the button needs ("paired but the harness project for
  this ticket's project is unset") that nothing exposes yet.

Findings go to `.scratch/plays/research/06-harness-presence.md` with file
and line references.

## Answer

`presence.Keeper` holds one goroutine per paired computer per user with a
live `/ws/events` socket; connected/connecting is per-computer, polled by
the web via `GET /api/pairing/presence` (no live topic exists).
`pairing.ResolveTarget` resolves project link → user defaults → sole
computer, yielding four `NotConfiguredReason`s that map 1:1 onto
`replyNotConfigured`'s messages — but `ResolveTarget` is only ever called
mid-turn (`internal/agent/pipeline.go:211`), with no read-only HTTP path.
The web already has computers, presence, defaults, and raw (unresolved)
project links as hooks, but nothing that runs the resolution chain for a
given project without reimplementing it client-side.

Gaps: (1) no read-only resolve endpoint exposing `NotConfiguredReason` for
a project without a live turn attempt — the one gap that matters; (2) the
resolve response needs to return `computer_id` so the web can join it
against presence for "offline"; (3) a harness that rejects a session
(revoked pairing, token not yet expired) silently disappears from presence
with no distinguishing reason. See
`.scratch/plays/research/06-harness-presence.md` for full detail.
