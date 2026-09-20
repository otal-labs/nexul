# Plays: pre-configured Agent turns fired from tickets and docs

**Status:** ready-for-agent

Assembled 2026-09-17 from the wayfinder map in this directory. Every
decision below is recorded in a ticket under `issues/`; this file is the
one-page view for slicing into implementation tickets with `/to-tickets`.

## Problem Statement

The Agent can already do the whole loop through MCP, but only when someone
types `@Agent` and a paragraph into a thread. Nothing on a ticket says "do
the obvious thing for a ticket in this state", nothing on a doc says "turn
this into tickets", and every run has to restate the team's conventions by
hand. Memories exist, but as ticked docs mixed into the docs list, with no
permission of their own and no way to force one into a turn. And when a run
misbehaves there is no record of what the Agent did, only its final reply.

## Solution

A **play** is a pre-configured Agent turn a user fires from a ticket page or
a doc page with one button. Two ship out of the box: "Fix with AI" on
tickets in the progress stage, and "To tickets via AI" on docs. A play runs
on the clicking user's own paired harness, posts into the target's thread,
inlines the memories the user picked, and, for a ticket, moves it to a
chosen column when the harness reports the turn done. Every press leaves a
**trail**: the choices made, every step the Agent took, and the outcome.

**Memories** become their own entity, per project, with versions, an
always-included flag, a clone action, and their own permission. **Docs get
a thread**, gated so a client who reads the doc need not see the work
behind it. The permission vocabulary gains three domain-declared verbs to
express all of this without over-granting `write`.

## User Stories

1. As a developer, I want a "Fix with AI" button on any ticket in progress, so that I can hand the ticket to the Agent without composing a prompt.
2. As a developer, I want to pick which memories the run reads and add a line of my own, so that the Agent follows the right guide for this change.
3. As a developer, I want the ticket to move to In review when the run finishes, so that the board reflects the work without me touching it.
4. As a developer, I want the button to tell me why I can't press it (no harness paired, harness offline), so that I know what to fix.
5. As a developer, I want a running play to be visible on the ticket, on the board card, and in the thread, and to be able to stop it, so that I can walk away and come back.
6. As a developer, I want every run to leave a trail I can open later, so that when Nexul is used to fix Nexul I can see what the Agent did.
7. As a product owner, I want to write a doc and press "To tickets via AI", so that the doc becomes backlog tickets that link back to it.
8. As a workspace owner, I want to define plays once per workspace and exclude projects or people, so that the same button works everywhere it should and nowhere it shouldn't.
9. As a workspace owner, I want memories to be a separate, permissioned thing from docs, so that agent instructions never sit in a client-facing list.
10. As a workspace owner, I want a memory that applies everywhere to be copyable to other projects and workspaces, so that a good guide spreads without a shared dependency.
11. As a workspace owner, I want memories versioned with revert and to be told when an agent changes one, so that an agent may keep them current without anyone approving each edit.
12. As a client with read access to a doc, I want to see only the doc, so that the team's working thread and play trails stay internal.
13. As an agent working through MCP, I want the same play and memory tools the browser has, so that a capability is not half shipped.
14. As an automation author, I want events when a run starts and finishes, so that I can notify people or chain further work.

## Implementation Decisions

### Plays (tickets 02, 07, 08)

- A play belongs to a workspace and has: label (the button text), type
  (`ticket` or `doc`; extensible), one-line description shown as a tooltip,
  base instructions, an enabled switch, an excluded-projects list, and for a
  ticket play exactly one **show-when stage** (`backlog`, `progress`,
  `review`, `testing`, `done`, ADR 0022). No icon, no manual ordering;
  plays sort by label.
- No status column is stored on a play. Move-to is chosen in the run
  dialog from the target project's own columns. Column deletion therefore
  needs no play validation.
- **Ticket page**: a "Plays" section in the right rail under Properties,
  one ghost button per play; when the rail collapses the plays become a
  sticky bottom bar. **Doc page**: one "Plays" menu button in the header
  beside the Thread button; each play is a row with label and description.
  Prototype on branch `proto/plays-ui`, route `/prototype/plays`.
- Button states: hidden without `plays:run`; disabled with a reason when the
  user has no paired harness ("Pair a harness in Settings to run plays") or
  it is offline ("Your harness is offline"); disabled with "a run is in
  progress" while a trail is active on the target; enabled otherwise. Copy
  says **harness**, never "computer".
- Two plays are seeded with every new workspace and once by migration for
  existing ones: "Fix with AI" (ticket, show-when `progress`) and "To
  tickets via AI" (doc). Ordinary plays afterwards; no upgrade overwrites
  them. Their instruction texts are in ticket 05.

### The run dialog (tickets 02, 07)

- Asks three things: a memory multi-select (always-included memories ticked
  and locked; titles and when-to-use lines only), a custom-instructions
  box, and for a ticket play the move-to column with a "never moves
  backwards" note. The confirm button names the destination ("Run Fix with
  AI · then In review").
- Pre-selection: for each user, play, and project, the last chosen move-to
  and the last selected memories are pre-selected next time, read from the
  user's latest trail. A remembered column that no longer exists is not
  pre-selected. No default memories live on the definition.
- A selection whose inlined memories exceed the per-run ceiling is refused
  in the dialog with the totals shown. The dialog also shows which computer,
  provider, and model the run will use, preselected from the project link
  or the user's pairing defaults, changeable per run (ADR 0058, amending
  this section's earlier "nothing else is asked per click").
- A user without `tickets:write` sees the move-to field disabled with "you
  can't move tickets in this project"; the run proceeds with no move.

### Trails (ticket 03, 07)

- A trail is created at the click in state `starting`, before the harness
  accepts, then `running`, `done`, `failed`, `interrupted`. A harness that
  refuses to start is a `failed` trail with the reason.
- Fields: play, target (ticket or doc), starter, selected memories, custom
  instructions, chosen move-to column, harness session id, started, ended,
  outcome, last error, the Agent's reply message id, and the harness
  activity stream captured on the trail, capped at a few hundred lines.
- One active trail per target, whichever play. Stop is available to the
  starter and to anyone with `plays:write`; a stopped trail keeps its
  record; continuing later is a new trail in the same thread, whose harness
  session is reused.
- **Move-to never moves backwards.** The PR-opened and ticket-finished
  default automations already move tickets during a run; on `done` the play
  applies the chosen column only if its stage is later than or equal to the
  ticket's current stage, otherwise it skips with a note in the thread. A
  chosen column deleted mid-run also skips with a note.
- Failure and interrupt: the ticket stays, any partial reply is posted, a
  system note says why it ended, and the starter gets an inbox notification
  for every terminal outcome.
- Timeout is harness silence, on the order of fifteen minutes with no
  snapshot, activity, or terminal received; no total ceiling. Chat's
  ten-minute cap stays chat's.
- Provenance: a status change made by a play carries actor kind `play` with
  the play's name and the trail id, beside `user` and `automation`; started
  from MCP it also carries the `:mcp` suffix executions use (ADR 0049).
- The "Trail" section under the ticket body (above the thread) lists the
  ticket's trails, one hairline row each (spinner, tick, or cross; play,
  summary, starter, when); a row opens the full trail. Running state shows
  as a spinner in the play button with a Stop square, a spinner beside the
  id on the board card, and a spinner line in the thread. No banner.

### Prompt composition (ticket 05, research 11 and 12)

- Block order: the fixed Agent instructions (identity, whoami check,
  memories index) as today; the target (ticket, or doc as markdown) ; the
  conversation so far; then the run request. The request holds the play's
  name and base instructions, the inlined memories under "Memories the user
  selected for this run, follow them", and the custom instructions under
  "Instructions from <user> for this run; where these conflict with the
  play's instructions, these win". Play material rides inside the request so
  the reused-session incremental prompt carries it.
- The click posts a real message from the starter, "Started <play>" plus
  their custom instructions; that message is the request. The Agent's reply
  lands under it.
- Trim order when over the harness limit: conversation history first, then
  the target body with a note; never the inlined memories or the custom
  instructions.
- Images in an inlined memory or in the target body travel to the harness
  as attachments: the server reads the stored bytes (ADR 0027) and hands
  them over through a harness-neutral `Attachments []Attachment{Name, MIME,
  Bytes}` on the turn prompts; the T3 client encodes them as base64 data
  URLs (T3 accepts up to 10 MiB each and never fetches a URL). The markdown
  keeps the image's name in place. Oversized or non-image attachments fall
  back to "[attachment omitted: <name>]".
- A memory can be marked **always included**: inlined in every run of that
  project, ticked and locked in the dialog, counted toward the ceiling
  first. Recommendation: chat turns inline them too, so chat and plays never
  disagree.
- Seeded instruction texts, editable like any play:

  *Fix with AI*: Confirm your Nexul access with the whoami tool, then read
  the ticket with `ticket_get` and its links with `ticket_get_links`. Work in
  the project checkout you are running in. Create a branch named `<ticket
  key>-<short-slug>` from the default branch, implement the fix, run the
  project's tests and linters, and commit. Push the branch, open a pull
  request whose title starts with the ticket key, then link the branch and
  the PR to the ticket with `ticket_link_branch` and `ticket_link_pr`. Do not
  merge unless a memory selected for this run explicitly permits merging; if
  one does, merge once checks pass. Reply with what changed, how it was
  verified, and the PR link. If you cannot complete the fix, say what
  blocked you instead of opening a partial PR.

  *To tickets via AI*: Confirm your Nexul access with the whoami tool, then
  read the doc with `doc_get`. List the doc's project's columns with
  `status_list` and its ticket types. Search existing tickets with
  `ticket_search` so you do not duplicate work already tracked. Split the
  doc into tickets a developer could pick up independently: one outcome per
  ticket, a title under eighty characters, a body with context, acceptance
  criteria, and a pointer to the doc section it came from. Create each with
  `ticket_create`, passing the doc id so the ticket links back. Put them in
  the backlog column. Reply with the list of tickets created and anything in
  the doc you deliberately did not turn into a ticket.

### Memories (ticket 01)

- Memory is its own entity: own table, own "Memories" page in the
  workspace, own MCP tools. The "Agent memory" switch on docs goes away, and
  memories never appear in a docs list, page, or search result.
- Fields: title, one-line when-to-use, rich-text body (same editor as docs,
  images and links allowed, exported to markdown at run time per ADR 0026),
  project, always-included flag.
- Every memory belongs to exactly one project; there is no workspace-wide
  set. A memory is **cloned** to another project or workspace through an
  explicit action; a clone is independent and may drift.
- Every memory is versioned like docs (`doc_versions` pattern: one row per
  update with author and time); the page shows the history and any version
  can be reverted to, which appends a new version.
- Agents write memories through `memory_create` and `memory_update` under
  the mentioning user's `memories:write`, with no approval step. An update
  publishes `memory.updated` with the author and version, and the inbox
  turns it into a notification for members with `memories:read`.
- Inlining ceilings: a per-memory and a per-run character cap next to the
  existing prompt limit; tuned in the build.
- Migration: none for existing data; the owner restarts the schema before
  any public deploy. The seeded defaults include one always-included memory
  per project, "Working in this project", with a starter body. The memories
  index in the chat prompt and the `nexul-memory` skill text switch from doc
  tools to memory tools.

### Doc threads (tickets 04, 08)

- New conversation kind `doc_thread`, one per doc, created lazily on the
  first message or the first play run, mirroring the ticket thread's
  get-or-create and unique index.
- Renders in the chat dock, opened from a "Thread" button in the doc header.
  No section under the editor. The owner may redesign thread surfaces later
  as its own effort.
- An Agent turn in a doc thread carries the doc's title and body as
  markdown, trimmed with a note when cut.
- Gated by `docs:thread` on that doc (see permissions): the thread, its
  messages, the Agent's replies, and the doc's trails. Listed in the chat
  page as "Doc thread" with the doc's title; no marker on the docs list.

### Permissions (ticket 08, ADR 0057)

- The vocabulary keeps `read`, `write`, `delete` for every domain and lets a
  domain declare a verb for an act that is neither: `plays:run`,
  `memories:clone`, `docs:thread` arrive with this effort. The Go domain
  table already lists actions per domain; catalog, roles grid, and token
  scopes pick them up.
- `plays:read` lists definitions in settings; `plays:run` shows the buttons
  and fires them; `plays:write` creates and edits; `plays:delete` deletes.
- `memories:read` sees the page and the picker; `memories:write` creates,
  edits, reverts, and flags; `memories:clone` copies to another project or
  workspace (plus membership of the destination workspace; attachments are
  copied as new rows); `memories:delete` deletes.
- User exclusion is a permission overwrite with resource type `play`
  denying `plays:run`, managed from the play's settings page with the doc
  sharing dialog generalised to a resource type. Project exclusion is the
  list on the play and hides it from the page and from `play_list`.
- Doc threads are the one fine-grained gate in this effort; ticket threads
  stay member-open (ADR 0023). No migration grants: a fresh workspace seeds
  the Owner role only and every new role ticks its own bits.

### Harness readiness (research 06)

- Resolution and presence are computed server-side but never exposed
  read-only. Add `GET /api/pairing/resolve?project_id=` returning
  `{ok:true, computer_id, ...}` or `{ok:false, reason}` with the four
  existing `NotConfiguredReason` values, and join `computer_id` against the
  existing `/api/pairing/presence` map for "offline". A harness that
  rejects a session silently vanishes from presence; surface a reason there
  if it proves confusing.

### MCP, gateway, events (ticket 09)

- Tools: `play_list`, `play_run`, `play_run_get`, `play_run_stop`,
  `play_list_runs`, `play_create`, `play_update`, `play_delete`;
  `memory_list`, `memory_get`, `memory_create`, `memory_update`,
  `memory_delete`, `memory_clone`, `memory_list_versions`, `memory_revert`.
  Gateway routes mirror them one to one. Live topic `play.run` carries trail
  state changes to the ticket page, the doc page, and the board card.
- Catalog rows: `play.run_started`, `play.run_finished` (with `outcome`),
  `memory.created`, `memory.updated`, `memory.deleted`, each an outbox write
  (ADR 0044). Plays and trails are not indexed for search.

### Domains

- New domain `internal/plays/` (plays, trails, the run pipeline reusing
  `internal/agent`'s turn machinery), new domain `internal/memories/`, and
  `doc_thread` added to `internal/chat`. The harness seam gains
  `Attachments`; `internal/t3client` encodes them. `internal/pairing` gains
  the read-only resolve endpoint. The permission table gains `plays`,
  `memories`, and the `docs:thread` verb.

## Out of scope

- A workspace-level service harness so unpaired users can fire plays.
- Play types beyond `ticket` and `doc`.
- Redesigning how threads and the chat Agent surface on pages.
- Narrow-width web verification; the owner plans a separate mobile app.

A per-click provider or model override was out of scope here; ADR 0058
reverses that and lets the run dialog also pick the computer.

## Follow-ups left in the fog

- Auto-firing a play on a status change, and whose harness it would use.
- A play's harness-side project when the ticket's project has no pairing
  link: refuse, or fall back to the user's default as chat does.
- A per-play silence-timeout override.
