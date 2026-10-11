# 12 — Move existing computers onto a personal runner

**Status:** resolved

**Blocked by:** 06

Read first: `practices/go.md`, `practices/architecture.md` (Principles), `practices/react-guide.md`,
the deprecation-and-migration approach, the spec (Moving existing computers).

## What to build

- On a computer paired by tunnel or URL, a notice "Move this computer to the Nexul app" with the install
  command, whose code is bound to that computer's id (`POST /api/pairing/computers/enrollments` with `id`).
- When its runner connects, pair through it, keeping the id, links, defaults, setup, MCP token and trails.
  Only after that pairing succeeds: delete its tunnel, DNS record and Access app, clear its tunnel columns,
  and show the command that removes `cloudflared` from the computer.
- One inbox notification per owner of such a computer, once, when the release ships. Its receipt time is
  the start of that owner's 30 days (spec, decision 20), which ticket 14 reads.
- A computer whose runner was revoked (ticket 04) is adopted the same way.

## Acceptance criteria

- [x] On nexul-box with a tunnel computer restored from a snapshot: moving it keeps its trails, links and
      setup confirmation, and its Cloudflare tunnel is gone afterwards.
- [x] A move whose pairing fails leaves the tunnel computer working.
- [x] The notification fires once per owner, not per computer or per restart (an idempotent receipt).

## Comments

Built: `internal/pairing/move_usecase.go` (`AnnounceMoves`, `retireTunnel`), the move condition in `HandleFactsReported`
(`needsPairing`: a computer not yet on its runner pairs through it on the next report), `PairComputer` with no token
moving a tunnel computer once a runner is enrolled, `computer.move_requested` (outbox, members-only) and its inbox
consumer `HandleComputerMoveRequested`; web `MoveNotice`, `MoveComputerDialog`, `MovedStep`, `ComputerMoveDetail`.
Migration 0095 adds `pairing_move_notices`, the receipt.

- The receipt: each boot sends the notice to every owner of a tunnel or URL computer with no row in
  `pairing_move_notices`, writing the row and the outbox event in one transaction; `notified_at` is the start of that
  owner's 30 days for 14. Notifications are purged by retention, the receipt never is.
- The notice is one inbox notification (`computer.move_requested`, subject type `computer`) in the first workspace the
  owner joined, since a computer belongs to no workspace; it opens a panel pointing at Computers. The phone shows it with
  nothing to open. Recorded in ADR 0146.
- The tunnel is deleted only after `pair` saved the relay session; a teardown Cloudflare refuses keeps the tunnel on the
  row and is retried on the runner's next report. The row turns into the runner row as soon as its address is the
  relay's, and the move dialog lives above the row so that flip never closes it.
- The cloudflared removal is the service only (`sudo cloudflared service uninstall`, `cloudflared service uninstall` in an
  administrator PowerShell), shown in the dialog once the tunnel is gone; the binary stays.
- A computer whose runner was revoked is adopted through the same `POST /api/pairing/computers/enrollments {id}`; no change.
- Box check (nexul-box-move from `registered`, Nexul v0.3.32-beta, then upgraded to v0.3.99-beta.12 built from this
  branch with `nexul upgrade`): a computer made the old way (`POST .../computers/tunnel`, `tunnel.sh` as `alice`,
  T3 Code 0.0.45, paired with `t3 pair` over the tunnel), set up, with a project link, defaults, an MCP token and a trail (a failed run, no provider being signed in there,
  its computer set in the database).
  The upgrade wrote one receipt and one inbox notification; two restarts added none. With `t3 auth pairing create`
  refused on the computer, running the move command left the computer on its tunnel (projects still listed through it,
  tunnel healthy) with "Not moved yet" and the reason; with it restored, Pair now moved it in 9 s: relay address,
  tunnel columns empty, tunnel, DNS record and Access app gone from Cloudflare, setup confirmation, link, defaults,
  MCP token and trail unchanged, and `sudo cloudflared service uninstall` removed the service. Screenshots in the PR.
- For 14: read `pairing_move_notices.notified_at`; a computer still on a tunnel shows `tunnel` on the list, one paired by
  URL a `server_url` that is not the relay address.

