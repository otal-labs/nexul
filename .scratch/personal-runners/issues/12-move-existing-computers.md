# 12 — Move existing computers onto a personal runner

**Status:** ready-for-agent

**Blocked by:** 06

Read first: `practices/go.md`, `practices/architecture.md` (Principles), `practices/react-guide.md`,
the deprecation-and-migration approach, the spec (Moving existing computers).

## What to build

- On a computer paired by tunnel or URL, a notice "Move this computer to the Nexul app" with the install
  command, whose code is bound to that computer's id (`POST /api/pairing/computers/enrollments` with `id`).
- When its runner connects, pair through it, keeping the id, links, defaults, setup, MCP token and trails.
  Only after that pairing succeeds: delete its tunnel, DNS record and Access app, clear its tunnel columns,
  and show the command that removes `cloudflared` from the computer.
- One inbox notification per owner of such a computer, once, when the release ships.
- A computer whose runner was revoked (ticket 04) is adopted the same way.

## Acceptance criteria

- [ ] On nexul-box with a tunnel computer restored from a snapshot: moving it keeps its trails, links and
      setup confirmation, and its Cloudflare tunnel is gone afterwards.
- [ ] A move whose pairing fails leaves the tunnel computer working.
- [ ] The notification fires once per owner, not per computer or per restart (an idempotent receipt).
