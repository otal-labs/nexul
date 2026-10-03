# 16 — Walkthrough on both protocols

**What to build:** End to end on a box launched from the `t3-dual` image (ticket 04), with a local
Nexul (dev login) paired to T3 `0.0.45` on 3773 and nightly `2632` on 3774 inside the box.
- On both: an @Agent mention and a follow-up; a mention mid-turn that queues and gets its own reply; a
  ticket with an image; a play question answered live and one answered after a Nexul restart; Stop on a
  running and on a queued run; a Claude setup turn.
- Protocol 2 only: an agent that uses `delegate_task` (async) — the reply waits, carries a pill, and the
  pill shows the helper's conversation; a Claude native subagent gets a pill too; Stop during the wait
  leaves nothing running in T3.
- Flip: copy the 0.0.45 home and start the nightly on 3773 — the computer switches with no re-pair and
  no drift warning, and an imported thread's first turn gets the Full prompt; start 0.0.45 again — the
  "went back" message in the turn and on the settings row, no loop; start the nightly again — it
  recovers.
- Record real provider frames ticket 04 could not, and re-record any fixture that disagrees.
- Check trails, chat and pills at 768, 1024 and 1440 px. Findings become fix-up tickets here. Delete
  the box afterwards.

**Blocked by:** 06, 09, 12, 14, 15

**Status:** ready-for-human

Needs the owner: the two host firewall rules for the box's internet, and one provider login on the box.

- [ ] Every step passes or has a fix-up ticket
- [ ] No turn waits for a timeout; nothing runs in T3 after Stop
- [ ] The box is deleted
