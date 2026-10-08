# A play may start itself through an auto play

ADR 0055 says nothing fires a play but a person, and ADR 0066 makes the decisions check the one exception. Teams want
more of them: "run Fix with AI when the ticket becomes unblocked, bugs first, once per unblock". Automations can
already do anything in code, but writing SDK code for "when X, if Y, run this play" is too much for a team that only
wants to pick from dropdowns.

Decision: an **auto play** belongs to one play and starts it when a moment matches. It is composed on the play's own
settings page as a stack, not a graph: one moment, conditions over the ticket's or doc's existing fields in all/any
groups nested one level, priority rules, limits, and whom it runs on (the developer by default, the tester, or whoever
caused the moment). Anything that branches, waits, or loops is an automation calling `runPlay`.

- A run still lands on one person's own paired harness with that person's permissions, as ADR 0055 asks. The person
  needs `plays:run` on the play; if there is nobody to run it on, or they are excluded, the ticket shows "didn't run".
- A match never starts a run directly. It waits in that person's queue, High before Normal before Low, oldest first
  within a level, until they have a free slot and their computer is online. One auto run per person and one per ticket
  at a time by default; a button press never queues but takes the slot. At the front, the run is checked again, and a
  run that no longer matches is skipped with a line on the ticket.
- Chains are allowed. The loop guard is a cap on automatic runs per ticket per rolling day across all auto plays,
  default 5 per workspace; a capped ticket shows "Auto plays paused" with a resume button.
- A ticket play's show-when stage only places its button. An auto run ignores it; an auto play that wants a stage
  limit adds "Stage is" as a condition.
- The decisions check becomes an ordinary play with one seeded auto play, "ticket enters a done stage", run on whoever
  caused it, off by default. Its switch leaves the Automations page; a workspace that had it on keeps it on.
- A new permission domain, `autoplays:read`, `write`, and `delete`. Existing grants are backfilled from the matching
  `plays` bits, read from read, write from write, delete from delete, and nothing from `automations:write`: auto plays
  belong to plays, so whoever edits a play edits its auto plays. Someone who could switch the decisions check only
  through `automations:write` loses that switch.
- `runPlay` names a play by its label, so play labels become unique per workspace; existing duplicates are renamed
  with " (2)", " (3)" in creation order.

The alternatives were a node-graph editor, rejected because it grows into the workflow engine automations already
are, and auto plays on the Automations page, rejected because that page is for code. The cost is that a play can now
spend someone's agent time without them pressing anything. The queue, the per-ticket cap, the "skipped", "didn't
run", and "paused" lines on the ticket, and the off-by-default switch on every auto play are how that stays visible.

Decided 2026-10-08, amending ADRs 0055 and 0066.
