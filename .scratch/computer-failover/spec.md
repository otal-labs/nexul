# Computer failover: runs go to the first ready computer in your order

**Status:** ready-for-agent

## Problem

A person with two paired computers still has one place their runs can go. A run uses the project's link, which pins
one computer, else the default computer; when that computer is offline the run waits with "offline" and the other
computer, online and idle, is never tried. Capacity is the other half: "runs at once" is a workspace setting per
person, 1 to 5, counted across every workspace (ADR 0132's amendment), so a second computer adds no room. Two
computers at five runs each should be ten.

The settings screens carry the old model. Computers lists machines; Projects pins one computer per project; Defaults
holds a default computer and one fallback T3 project, a checkout on one machine.

Since PR #838 a computer's T3 projects carry their normalized git remote, so Nexul can find a project's checkout on any
computer without a person naming it. That is what makes failover possible without a pick per computer.

## Decisions (owner, 2026-10-22)

1. **One ordered list per person: Your computers.** Drag to reorder. A run goes to the first computer that can take it.
   "Default computer" is gone; the top of the list is the default.
2. **A project follows your computers, or names its own.** A project link either follows your order (the default) or
   names its own subset in its own order. The checkout on each computer is found by git remote; a person picks one by
   hand only when that is ambiguous or missing.
3. **Runs at once belongs to the computer.** 1 to 5 per computer, set by its owner. The workspace's runs at once per
   person is removed.
4. **Placement: fill in order, or spread.** One switch on your computers, overridable on a project with its own list.
   Fill in order takes the first computer with room; spread takes the least busy, ties by order.
5. **Failover happens before a run starts, never mid-run.** A run already going on a computer that drops ends the way
   it does today.
6. **Only your own computers are in your list.** A computer shared with you is not; Run on a chosen computer (ADR 0156)
   keeps working as it does.
7. **A computer without the checkout is skipped**, unless **Clone automatically** is on, which clones the project's
   repository there first. The switch lives in the defaults part of the Computers tab, off by default.
8. **Model, provider, effort and start-in stay per project**, for every computer. A computer whose provider is not set up,
   or that does not offer the model, is skipped for that project.
9. **No "where to run?" for a pressed play** unless no computer of yours has the project. Pressed runs never wait for
   room: with every computer full they start on the first one that is otherwise ready.
10. **Only play runs fill slots.** `@Agent` chats do not, as today.
11. **An auto play on a chosen computer names an ordered list** through the same picker; one computer is still a list.
12. **Signals, not notifications.** A computer shows "2 of 5 running". A trail says which computer it ran on and, when it
    passed one over, why ("laptop was offline"). A waiting run names every computer and why each could not take it.
13. **Two tabs.** Computers (the ordered list, each computer's runs at once and chat folder, and the defaults: model,
    start-in, placement, clone automatically). Projects (each project's computers, checkouts, model, start-in).
    Defaults goes, and its path 404s like any moved route.
14. **On upgrade** the old default computer goes to the top of the list, the rest follow newest first; every link keeps
    its model, options and start-in but stops pinning its computer, and its T3 project is kept as that computer's
    checkout for the project.

ADR 0165 records the order, the failover and the end of the pressed-play question; ADR 0166 moves runs at once to the
computer. The look is not decided here: the web screens are built in auto design mode then animation mode.

## Choosing a computer

One function in `internal/pairing` answers "where does this run go" for every caller: a pressed play, a queued auto
play, an `@Agent` turn that starts a thread. It takes the person, the project (empty outside one), the run's provider
and model, and whether room counts, and returns the chosen computer and checkout, or a refusal that names every
computer and why it was passed over.

**Candidates, in order.** The project's own list when it has one, else the person's computer order. Only the person's
own computers; shared ones never appear (decision 6). A chosen-computer auto play passes its own list instead, checked
against the picker's rights as ADR 0156 does today.

**A computer is ready for the run when**, checked in this order so the reason is the first gap:

| Check | Why it is passed over (shown) |
|---|---|
| Its personal runner is connected and T3 Code answers | offline |
| Its T3 session has not expired | needs pairing again |
| Setup is confirmed, and confirmed for the run's provider | not set up for <provider> |
| The provider offers the model (from computer facts) | doesn't offer <model> |
| It has a checkout of the project (below) | doesn't have this project |
| It has room, when room counts | full (n of n) |

**The checkout on a computer**, for a project:

1. The checkout remembered for this person, project and computer, while the computer still lists it.
2. Else the computer's T3 projects whose git remote matches one of the project's repositories. Exactly one wins. Several
   are ambiguous: passed over as "pick which checkout", and the Projects tab asks.
3. Else, with Clone automatically on, a project with a GitHub repository and a computer that may clone: clone it under
   ADR 0162's folder rule, remember it, use it. A clone that fails passes the computer over with the clone's error.
4. Else passed over as "doesn't have this project". A project with no repository only ever has remembered checkouts.

Outside a project (a chat in a channel or DM) the checkout is the computer's **chat folder**, which replaces the single
fallback T3 project; a computer without one is passed over as "no chat folder".

The match reads computer facts (`Facts.Projects`, stored, refreshed when they change); the launch still checks the
checkout against T3's live list as it does now, so a stale fact fails the run rather than landing it in the wrong folder.

**Placement.** Fill in order: the first ready computer. Spread: among ready computers, the lowest running ÷ runs at
once, ties by order. A pressed run with every ready computer full takes the first ready one ignoring room (decision 9).

**Room** is the computer's active play trails (`starting`, `running`, `waiting`) against its runs at once, counted on
`play_trails.computer_id`, whoever started them, so a run someone else lands through a share takes room too.

## Callers

- **Pressed play** (run dialog, `play_run`, Continue that records no location). Resolves as above with room counting
  for placement but never blocking. The refusal that asks where (`needs_location`) now fires only when every computer
  of theirs is passed over as "doesn't have this project" or "pick which checkout"; the answer is saved as a remembered
  checkout, not a pin. Every other refusal disables the play with its reason, as a linked project does today.
- **Run dialog.** **Where to run** shows one line: the computer it will land on now and why the others were passed over
  when they were. **Change** picks a computer for this one run (any of the person's own ready computers, with a
  checkout picker when needed); it no longer saves a link. Order and project lists are edited in settings.
- **Queued auto play.** The queue item carries no computer until it starts (a chosen-computer auto play carries its
  list). At the front of the queue the dispatcher resolves with room counting; nothing ready holds the item with
  reason `no_computer_ready` and a `computers` list beside it, one `{computer_id, name, why}` per candidate. The
  person's pass no longer stops at one offline computer. A run that starts records the computers passed over.
- **Waking.** Today `HandleSlotsFreed` wakes on a run finishing and on the workspace limit changing. It now also wakes
  on a computer's runs at once, order, project list, checkout or clone switch changing, and on a computer connecting
  (`HandleComputerChanged` already covers paired and setup confirmed; add the runner connecting).
- **`@Agent` turn starting a thread.** Resolves with room ignored. A reply in an existing thread stays on its computer,
  as `follow.go` does now.
- **Decisions check and its Run again.** Same as a queued run, with room ignored, since nothing queues them.
- **Continue and started-again** keep the recorded location (ADR 0145), unchanged.

The per-person "pick a computer" for a waiting run (`PickComputer`, `POST /api/plays/queue/{id}/computer`,
`trail_update`'s `computer_id` and `t3_project_id`) goes. Its one job was naming a default, and there is none; a waiting
run's gap is fixed in settings, which wakes it.

## Storage

Migrations, then `make sqlc`:

- `pairing_computers.position INTEGER NOT NULL DEFAULT 0`, per owner; `runs_at_once INTEGER NOT NULL DEFAULT 1`
  (1 to 5); `chat_project_id TEXT NOT NULL DEFAULT ''`.
- `pairing_user_defaults`: add `placement TEXT NOT NULL DEFAULT 'in_order'` (`in_order` or `spread`) and
  `clone_automatically INTEGER NOT NULL DEFAULT 0`; drop `default_computer_id` and `fallback_project_id` after copying.
- `pairing_project_links`: drop `computer_id` and `harness_project_id` after copying; add `placement TEXT NOT NULL
  DEFAULT ''` (empty follows the person's).
- New `pairing_project_computers (user_id, project_id, computer_id, position, harness_project_id)`, primary key the
  first three, cascading from user, project and computer. `position` NULL is a remembered checkout only; a number puts
  the computer in the project's own list. A project whose rows all have NULL positions follows the person's order.
- `workspaces.auto_play_runs_at_once` is dropped.

**Data on upgrade** (production has real users; the migration test runs it on a seeded copy):

- Order: the default computer first, then the rest newest first, numbered per owner.
- Runs at once: each computer takes the highest `auto_play_runs_at_once` of the workspaces its owner belongs to, so
  nobody's room shrinks.
- Chat folder: the old fallback T3 project goes onto the old default computer.
- Links: each keeps provider, model, options and start-in; its computer and T3 project become one
  `pairing_project_computers` row with a NULL position.

## Surfaces

- **HTTP.** `GET/PUT /api/pairing/defaults` lose the default computer and fallback project and gain placement and
  clone automatically. `PUT /api/pairing/computers/order` takes the full ordered id list. `PATCH
  /api/pairing/computers/{id}` takes runs at once and chat folder beside the name it takes today. Project links take
  `placement`, `computers` (ordered ids, empty to follow) and `checkouts` (`{computer_id, harness_project_id}`).
  `GET /api/pairing/computers` adds `position`, `runs_at_once`, `running` and `chat_project_id`. The workspace
  auto-play limits route keeps the daily cap only. Each new writing route names its permission in
  `route_permissions.go` (`none`, a person's own settings) and `make permission-catalog`.
- **MCP.** `computer_list` gains `position`, `runs_at_once`, `running`. `computer_setup_update` gains `position` and
  `runs_at_once` for the caller's own computer (no new tool). `workspace_update` and `workspace_list` lose
  `auto_play_runs_at_once`. `play_run`'s `computer_id` and `t3_project_id` pick for one run and remember the checkout.
  `trail_list` with `queue` returns `computers` on a waiting run. Pairing defaults and project links stay HTTP-only;
  add that gap to `.scratch/mcp-parity/`.
- **Events.** `play.queued` and `play.queue_updated` carry the item, so `computers` joins it beside `reason`
  (additive). A trail gains `passed_over` (`[{computer_id, name, why}]`, `[]` when none). `auto_play.limits_updated`
  keeps `runs_at_once_per_person` in the contract, tagged `deprecated`, and is no longer published for it. `make
  event-schemas`.
- **Live.** A computer's `running` follows `play.run_started` and `play.run_finished`, which already reach its owner;
  the web computers query refetches on them.
- **Web.** `PairingPanel` becomes two tabs, Computers and Projects. Computers: the ordered list with drag and keyboard
  reorder, each computer's runs at once, running count, chat folder and its existing facts; the defaults (model,
  start-in, placement, clone automatically). Projects: each project row says "Follows your computers" or its own list,
  and shows per computer whether it has the checkout, with **Add to <computer>** (clone or create, the existing
  pickers). Queue rows show `computers`; `PickComputerMenu` and `no_default_computer` copy go. The run dialog's **Where
  to run** as above. Built in auto design mode, then animation mode (the reorder and a failover line are the motion
  candidates), at 768, 1024 and 1440px.
- **Phone.** Nothing; the phone app has no computer settings or queue.
- **Docs.** `paired-computers.md` (Choose where turns run, the reasons table), `plays.md` (Where to run, the queue, runs
  at once, Run on a chosen computer), `chat-and-voice.md`, `mcp-server.md`, `CONTEXT.md` (done with this spec), and
  ADRs 0145, 0102, 0132, 0156 carry their amendment lines.

## Out of scope

- Moving a run that is already going when its computer drops (decision 5).
- Shared computers in a person's list, or capacity borrowed from another person (decision 6).
- A workspace ceiling on runs at once on top of each computer's.
