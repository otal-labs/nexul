# Wayfinder map: auto plays

Charted 2026-10-08 with the owner. Today nothing starts a play but a person
pressing it, apart from the decisions check, which is hardcoded. Teams want
plays to start by themselves when something happens to a ticket or doc:
"run Fix with AI when the ticket becomes unblocked", "no more than once per
unblock", "bugs first". Automations already do anything in code against the
SDK; auto plays are the quick, no-code version, set up in a play's own
settings by picking from dropdowns, the way a game designer composes "if
health < 5, shield".

## Destination

Built, merged, and walked through live on a paired harness: a play's
settings page has an Auto plays section where a member composes one or more
auto plays (when, if, priority, limits, run on), matching moments queue
runs on the right person's computer and start them in priority order, the
ticket shows what is queued, skipped, didn't run, or paused, the decisions
check is an ordinary play with a seeded auto play, and automations can call
`runPlay` with a play name the editor checks. This map carries the build:
build tickets are worked here once the decision tickets they wait on are
resolved.

## Notes

- Decided while charting:
  - An auto play is its own record belonging to one play; a play can have
    several. They live on the play's settings page. The Automations page is
    for SDK code only and gains nothing.
  - The composer is a stack, not a graph: **When** (one moment), **If**
    (all/any groups over existing fields, nested one level at most),
    **Priority** (rules like "High if type is Bug, otherwise Normal"),
    **Limits**, **Run on**. Anything needing more is an automation calling
    `runPlay`. A node graph was ruled out: it branches, waits and loops,
    which is the workflow engine automations already are.
  - Moments, all in the first cut: a ticket becomes unblocked (new event),
    enters a stage, is created, gets a developer or tester, fails a test; a
    doc is created or changed. "Changed" fires once edits have stopped for
    10 minutes, and never for an edit an agent's run made.
  - Conditions are dropdowns over fields that already exist. Ticket: type,
    project, stage, status column, category, labels, developer, tester, has
    a source doc, has a linked PR, blocked or not. Doc: project, folder.
    Only label names are typed.
  - Run on: the ticket's developer by default, or the tester, or whoever
    caused the moment (the mover, the assigner; the developer when an
    automation caused it).
  - Every run goes through the person's queue: one queue per person across
    all auto plays, High before Normal before Low, oldest first within a
    level. A run starts when the person has a free slot and their computer
    is online. Concurrency is capped per person and per ticket (one auto run
    on a ticket at a time by default).
  - A queued run is checked again when it reaches the front. If it no
    longer matches it is dropped, with a muted line on the ticket ("Fix
    with AI skipped: no longer unblocked").
  - "Didn't run" only when there is nobody to run it for, or that person
    may not run the play (excluded by a permission overwrite). An offline
    computer queues, it does not fail.
  - Chains are allowed (test fails → Fix with AI → review → Test with AI).
    The guard is a cap on automatic runs per ticket per day across all auto
    plays, default 5, editable per workspace. A capped ticket shows "Auto
    plays paused on this ticket" with a resume button.
  - Limits per auto play also include "once per ticket per occurrence of
    the moment" over a period.
  - Permissions: a new domain, `autoplays:read`, `autoplays:write`,
    `autoplays:delete`. The person a run lands on still needs `plays:run`
    on the play.
  - The decisions check becomes an ordinary play with one seeded auto play,
    "Ticket enters stage done", run on whoever caused it, off by default.
    Its switch leaves the Automations page; workspaces that had it on keep
    it on.
  - SDK: `runPlay(play, ticket)` where `play` is a union of the
    workspace's play names as string literals, so a typo or a renamed play
    fails the type check.
  - This changes ADR 0055 ("nothing fires a play but a person") and ADR
    0066 (the decisions check as a special case); a new ADR records it.
- Look tickets go through `design-mode` and the owner picks by looking;
  technical tickets are answered in the ticket and not grilled.
- Web only for configuring, built at 768px and verified at 768, 1024, and
  1440px. Read `practices/react-guide.md` and `practices/design-language.md`
  before any web edit, `practices/go.md` and `practices/architecture.md`
  before any Go edit, `practices/typescript.md` before any SDK edit.

## Decisions so far

- 01, the new moments: `ticket.unblocked` and `doc.settled`; see the
  ticket's answer.
- 02, the record and surfaces: `autoplays:*` backfilled from the matching
  `plays:*` bits, nothing from `automations:write`.
- 03, the run queue: one auto run per person by default; auto runs ignore
  the play's show-when stage, which only places the button.
- 04, `runPlay`: play names become unique per workspace.

## Not yet specified

- Whether the phone app shows the queued, skipped, and paused signals on a
  ticket, or only what the web already sends it; settled in the walkthrough.

## Out of scope

- A node-graph editor, nesting deeper than one level, or one auto play
  starting several plays. That is what automations are for.
- Time-based moments ("every morning", "after a ticket sits for a week").
  Not asked for; an automation can do it with `runPlay` once a scheduled
  trigger exists there.
- A workspace-wide priority order ("Bug before Feature before Task" for
  every auto play). Ruled out while charting: each auto play says its own
  priority.
- Custom ticket fields as conditions. They do not exist yet
  (`.scratch/ticket-workflow-depth/`).
- Configuring auto plays from the phone app.
