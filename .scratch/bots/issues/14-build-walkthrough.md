# 14 — Walkthrough on a real instance, then close the effort

**What to build:** An end-to-end run on `nexul-box`: create bots in a
channel, a DM, and a ticket thread; point a real GitHub Actions workflow, a
Grafana contact point, and Uptime Kuma at their URLs; check the messages
render as decided, mentions and unread behave, and regenerate, delete,
restore, and the limits answer as Discord does. Fix what it finds as
follow-up tickets. Then move "Channel webhooks" to shipped in
`website/src/pages/roadmap.astro` and `ROADMAP.md`, delete the
`proto/bots-embed` and `proto/bots-section` branches, and delete
`.scratch/bots/` once the owner has reacted.

**Blocked by:** 10, 11, 12, 13

**Status:** resolved

- [x] All three senders post with only the URL swapped (Uptime Kuma and Grafana for real in the box; the Discord status action's own code run locally, since the box has no public URL)
- [x] Screenshots of each message and the Bots section for the owner
- [ ] Roadmaps, branches, and the effort directory cleaned up

## Answer

Walked 2026-10-07 on a clean box with a native install of master. Everything
in the spec held; three defects were fixed in #472 (`wait=true` embeds as
`[]`, zoneless footer times read as UTC, embeds squeezed in the narrow ticket
thread pane). Roadmaps moved to shipped and the prototype branches deleted.
The directory stays until the owner has reacted to the live UI and ticket 15
is placed.
