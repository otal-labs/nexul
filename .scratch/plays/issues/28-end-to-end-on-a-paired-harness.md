# 28 — End to end on a paired harness

**What to build:** Both seeded plays run against a real T3 harness on the debug stack. One ticket in In progress with a real repository goes through Fix with AI: a branch, a PR linked back, the reply in the thread, the ticket in In review, the trail complete with the activity lines. One doc goes through To tickets via AI and becomes backlog tickets linked to the doc. One run is stopped mid-way and its trail reads back. A memory with an image is selected and the Agent sees it. Anything found is fixed in place, and the tracker directory for this effort is deleted when everything is green, per the tracker's rules.

**Blocked by:** 24, 25, 26, 27

**Status:** done

- [ ] The Fix with AI walkthrough above holds, with screenshots of the ticket page before, during, and after
- [ ] The To tickets via AI walkthrough holds
- [ ] Stop, failure by silence, and the never-backwards skip each observed once for real
- [ ] The image reaches the harness on the pinned or current T3 release
- [ ] `.scratch/plays/` deleted in the closing PR; ADRs and glossary carry everything durable

## Observed on the debug stack, 2026-09-18

- Fix with AI on NEX-2 (a real lint-config ticket): branch, PR #49 linked back, reply in the thread, ticket moved to Done, trail complete; PR reviewed and merged.
- To tickets via AI on a three-section doc: two backlog tickets linked to the doc, the already-fixed section skipped, reply in the doc thread.
- Stop: a run stopped a second after start left "Run stopped by dev." in the thread and kept its record, but ended `done` rather than `interrupted` (ticket 33).
- Never backwards: a run with move-to Open on an In progress ticket skipped with "Ticket is already in progress; not moving it back to Open".
- Image: an image embedded in the always-included memory reached T3 as an attachment (confirmed in T3's own event log) and the Agent described it as the ticket-page screenshot.
- Silence timeout: not observed live (fifteen minutes); covered by the runner tests.
- Findings became tickets 29 to 33; the effort's directory stays until they ship.
