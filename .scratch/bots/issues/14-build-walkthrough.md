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

**Status:** ready-for-human

- [ ] All three senders post with only the URL swapped
- [ ] Screenshots of each message and the Bots section for the owner
- [ ] Roadmaps, branches, and the effort directory cleaned up
