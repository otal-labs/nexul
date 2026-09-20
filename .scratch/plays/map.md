# Wayfinder map: plays

**Status (2026-09-17): COMPLETE, sliced.** Planning tickets 01–12 resolved;
the spec is `spec.md`; implementation tickets 13–28 are `ready-for-agent`
in `issues/`. Three prefactors (13, 14, 15) start the frontier.

Charted 2026-09-16 (grilling rounds 1–2 with the owner). A **play** is a
pre-configured Agent turn a user fires from a ticket page or a doc page with
one button ("Fix with AI", "To tickets via AI"). It runs on the clicking
user's own paired harness, posts into the target's thread, and on success
moves a ticket to a chosen column. Memories become their own entity, per
project, selectable per run, with their own permission.

## Destination

A spec at `.scratch/plays/spec.md` with every product decision locked, ready
to slice into implementation tickets with `/to-tickets`: the play model and
its two types, permissions, the run lifecycle and record, doc threads, the
run dialog, prompt composition, memory storage and permission, MCP and
gateway parity, and the two seeded default plays. Planning only; the build
is its own effort after this map.

## Notes

- Grilling tickets: invoke `/grilling` + `/domain-modeling`. Research
  tickets follow `/research`; findings land in `.scratch/plays/research/`.
- Screens are judged at desktop widths; the owner plans a separate mobile
  app, so narrow-width web verification is not a gate for this effort.
- Plain language with the owner; no section codes, no "memory-flagged doc"
  style shorthand without saying what it means in the same sentence.
- Grounding: `CONTEXT.md` (Agent, Harness, Memory, Stage, Permission,
  Permission overwrite), ADR 0022 (stages), 0029 (turn runs on the user's
  harness), 0042 (overwrites), 0050/0051 (page is the editor, chat is a
  dock), 0054 (harness client). Code: `internal/agent/pipeline.go` (how a
  turn runs today), `internal/agent/prompt.go` (prompt and memories index),
  `internal/docs/usecase.go` (`SetMemory`, `ListMemories`),
  `internal/chat/model.go` (`KindTicketThread`),
  `internal/platform/permissions/permissions.go` (the domain table).
- Related open efforts to read, not act on: `.scratch/branch-from-ticket/`
  and `.scratch/ticket-workflow-depth/` (both `needs-triage`).
- The owner's dogfooding goal: use a Nexul workspace to fix Nexul itself, so
  every run must leave a record good enough to debug from.

### Settled at charting (grilling rounds 1–2)

- **Name**: **Play**. Not "agent action" (`action` is already the second
  half of every permission) and not "button". "Fix with AI" is one play; the
  button label is whatever the owner typed.
- **Two play types**: `ticket` and `doc`, extensible later. A ticket play has
  a show-when status column and a move-to status column. A doc play has
  neither; it shows on every doc it isn't excluded from.
- **Condition and move-to**: settled in ticket 02 (show-when is a stage,
  move-to is chosen per run).
- **Defined per workspace**, with a **project exclusion list** on the play.
  **User exclusion is not a play field**: it is the existing per-resource
  permission overwrite (deny `plays:run` on that play).
- **Runs on the clicking user's own paired harness**, with that user's
  permissions, exactly like `@Agent` today. Button hidden without
  permission; shown disabled with a reason when the user has no paired
  harness or it is offline. Copy says **harness**, never "computer".
- **Posts into the target's thread**: the ticket's one existing thread, or a
  **doc thread**, a new conversation kind that is a real conversation
  humans use too, not a run log.
- **The server moves the ticket only when the harness reports the turn
  done**; error and interrupt leave it in place.
- **The run dialog** asks a memory multi-select, a custom instructions
  box, and for a ticket play the move-to column, each pre-selected from the
  user's last run of that play in that project. Provider and model come from pairing settings. Nothing
  else per click until the owner has seen it.
- **Selected memories are inlined in full** into the prompt; the rest of the
  index rides along as today.
- **Memories get their own permission** (`memories:read`, `memories:write`),
  split from docs. Their model is ticket 01's decision.
- **Every run is persisted** (play, target, starter, timestamps, outcome)
  and publishes catalog events.
- **Seeded defaults**: two plays ship with the workspace, "Fix with AI"
  (ticket) and "To tickets via AI" (doc), whose base instructions push a
  branch, link the PR, and, when a selected memory says so, merge it.
- From repo rules, not asked: MCP tools for plays and memories ship with the
  feature; runs are catalog events with an outbox write.

## Decisions so far

<!-- one line per resolved ticket: gist, then the link for the detail -->
- [Memories: keep them as designated docs, or make them their own thing](issues/01-memory-storage-and-permission.md) — own entity, per project with a copy action, rich-text body exported to markdown, `memories:read/write`, versioned with revert, agents write them through MCP with no approval step and a notification instead.
- [The play definition and column validation](issues/02-play-definition.md) — no columns on the definition: show-when is one stage, move-to and memories are picked per run and remembered per user, play, and project; description and enabled switch; two plays ship out of the box.
- [The run lifecycle and record](issues/03-run-lifecycle.md) — persisted from the click with the harness activity stream; one run per target, stoppable with logs kept; move-to never moves backwards past the automations; `play.run_started/finished`, actor kind `play`; timeout on harness silence, no total cap.
- [Doc threads](issues/04-doc-threads.md) — one lazily created `doc_thread` per doc, rendered in the chat dock from a header button, carries the doc as markdown, visible only to those who can read the doc, no new permission.
- [What a play run sends to the harness](issues/05-prompt-composition.md) — play material rides inside the request block; a visible "Started <play>" message is the request; custom instructions win over base; memories can be marked always included; seeded texts ship as drafted; images in memories and targets go to the harness as attachments (see tickets 11 and 12).
- [Can a harness turn carry images?](issues/11-harness-image-attachments.md) — unknown from this repo whether T3 accepts attachments; the turn path is text-only end to end today, placeholder stays until T3's own API is checked.
- [What shape does T3 Code accept for turn attachments?](issues/12-t3-turn-attachment-shape.md) — a new image attachment needs bytes inlined as base64 `dataUrl` (10 MiB cap); no variant accepts a fetchable URL, so a Nexul-authenticated link can't work.
- [Permissions and exclusions](issues/08-permissions-and-exclusions.md) — vocabulary gains domain verbs: `plays:run`, `memories:clone`, `docs:thread`; run shows buttons, read shows settings; user exclusion is an overwrite on the play; doc threads gated by `docs:thread`; no migration grants.
- [MCP tools, gateway routes, and events](issues/09-mcp-and-gateway-surface.md) — sixteen tools mirrored by routes, a `play.run` live topic, MCP runs on the token owner's harness with `:mcp` provenance, five catalog rows, nothing indexed.
- [The button and the run dialog](issues/07-run-dialog-and-button-prototype.md) — ticket page: rail Plays section under Properties with a bottom bar when the rail collapses; doc page: one Plays menu; the Trail (run history) listed under the ticket body; dialog and running signals as prototyped on `proto/plays-ui`.
- [Write the spec](issues/10-write-the-spec.md) — `spec.md` written; ADRs 0055, 0056, 0057; glossary updated. The map is complete; slice with `/to-tickets`.
- [What presence can tell the button today](issues/06-harness-presence-facts.md) — resolution and presence are both fully computed server-side but never exposed read-only; the button needs one new resolve endpoint, not new logic.

## Not yet specified

- **Auto-firing a play on a status change** (the "beauty" of chaining "In
  Review" to another play without a click). A play is user-triggered by
  definition today; whether an automation may start a run through the API,
  and whose harness it would use, waits on the run lifecycle and the
  harness question.
- **A play's harness-side project** when the ticket's project has no pairing
  link: the fallback chain exists for chat; whether a play needs stricter
  rules (refuse rather than fall back to a default project) surfaces once
  the run lifecycle is settled.
- **A per-play silence timeout override**, if a play ever needs longer than
  the instance-wide value.

## Out of scope

- A workspace-level service harness so unpaired users can fire plays. Every
  turn stays on the user's own harness (ADR 0029).
- A per-click provider or model override.
- Play types beyond `ticket` and `doc`.
- Redesigning how threads and the chat Agent surface on pages. The owner
  will run that as its own design effort; this spec uses the dock as is.
