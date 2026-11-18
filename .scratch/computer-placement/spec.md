# Computer placement: one order, plays name what they need

**Status:** ready-for-agent

## Problem

Three lists decide where a run goes, and the one that wins is not on the page you are looking at.

- **Your computers**, the person's order (ADR 0165).
- **A project's computers**, which follow your order or name their own subset, with a checkout picked per computer.
- **An auto play's computers** (Run on a chosen computer, ADR 0156), which replace both for that auto play's runs.

Seen on 2026-11-18: Fix with AI's auto play named mini alone, while Acme's own list was desk and
laptop. The queued run waited on "mini has several checkouts of this project; pick one", and **Pick a checkout**
opened the Projects tab, whose Acme row listed only desk and laptop. The computer the run waited on was not on the page,
and nothing there said the auto play had overridden the project.

The "several checkouts" was a second problem. Acme has two repositories, the app (`acme-app`) and its tests
(`acme-e2e`). Every computer had a clone of each, both matched the project's remotes, and placement treated that as
ambiguous on every computer.

## Decisions (owner, 2026-11-18)

1. **One order.** Your computers, in Settings, is the order for every run. A project may narrow it (fewer computers, or a
   different first one), as it can today. Nothing else orders computers.
2. **Plays name needs, never computers.** A play that must run where something is (Docker, a GPU, a Windows VM, staging
   keys) names that need. Its runs go to the first computer in the order that meets every need. The reason a play runs
   elsewhere is always that a machine has something, not load.
3. **Run as a person.** Run on a chosen computer becomes **Run as \<person\>**: the run takes that person's identity and
   their order, narrowed by the project and the play's needs. The person is the one picked on the auto play, checked as
   ADR 0156 checks the picker today.
4. **No shared computers for auto plays** in this effort. A computer shared with someone is not in their order (ADR 0165),
   and with the computer list gone an auto play has no way to name one. Adding a shared computer to your own order is a
   later effort.
5. **The app repository wins a checkout tie.** When a computer has clones of several of the project's repositories, the
   run uses the clone of the repository whose role is `app`. It asks only when two folders hold the same repository.
6. **Ask where the run waits.** A waiting run's fix (pick a checkout, add the project) is done on the waiting run, for
   the computer it waits on, never by sending the person to a page that may not list it.
7. **The project row shows the whole answer.** For each project: the computers in effective order, the checkout each would
   use, and for each play with needs, the computer that would take it now.

ADR 0197 records decisions 1 to 5. It amends ADR 0156 (the chosen-computer list goes) and ADR 0165 (needs, the app-repo
tie-break, the end of the auto play's own list).

## Placement after this effort

One function still answers "where does this run go" (`internal/pairing/placement.go`, `place`). What changes:

**Candidates.** The run's person's own computers, in their order, narrowed by the project's list when it has one. A run
no longer carries its own computer list. The one exception left is a pressed run whose person picks a computer in the run
dialog for that one run (ADR 0165, Change), which is a list of one.

**Ready**, in this order, so the first gap is the reason shown. The new row is the needs check:

| Check | Passed over as |
|---|---|
| Online, T3 Code answering | offline |
| Session current | needs pairing again |
| Set up for the run's provider | not set up for \<provider\> |
| Offers the model | doesn't offer \<model\> |
| Has every need of the play | doesn't have \<need\> (`missing_need`) |
| Has a checkout of the project | doesn't have this project / pick which checkout |
| Has room, when room counts | full (n of n) |

**Needs** are labels. A computer's labels are the ones its owner sets (free text, lowercase, `[a-z0-9-]`, up to 32
characters, up to 20 per computer) plus the ones Nexul derives from its facts: `docker` when Docker is available, and
`linux`, `macos` or `windows` from its OS. A play's needs are a list of labels, all of which must hold. An empty list
needs nothing. Needs belong to the play, so a pressed run and an auto run of the same play land under the same rule.

**Checkout**, for a project on a computer:

1. The checkout picked for this person, project and computer, while the computer still lists it. A pick is kept for any
   computer, in the project's list or not.
2. Else the computer's T3 projects whose remote matches a repository of the project. One match wins. Several: keep the
   ones whose repository's role is `app`; one left wins. More than one left, or several matches with no app among them,
   is ambiguous.
3. Else clone automatically, as today (computer-failover ticket 04).
4. Else "doesn't have this project".

## Moving over

A migration, then `make sqlc`, run against a seeded copy in the migration test.

- `pairing_computers.labels TEXT NOT NULL DEFAULT '[]'`.
- `plays.needs TEXT NOT NULL DEFAULT '[]'`, versioned with the play like its instructions.
- `auto_plays.run_on` value `computer` becomes `person`. `computer_ids` and `computer_id` are dropped after the move;
  `run_as` stays and is the person.
- `play_queue.computer_ids` and `computer_id` are dropped after the move. A queued item keeps its person.
- Each auto play that named computers: `run_on = person`, `run_as` kept. An auto play whose list named only computers
  shared with its picker is switched off, with "it ran on a shared computer, which auto plays no longer use" in its
  history, so it is not silently moved onto the picker's own machines.
- Fix with AI's auto play on mini becomes Run as its picker, which lands on mini first, since it heads the order.

## Surfaces

- **HTTP.** `PATCH /api/pairing/computers/{id}` takes `labels`. `GET /api/pairing/computers` adds `labels` and
  `derived_labels`. Plays take and return `needs`. Auto plays lose `computer_ids` and `computer_id`; `run_on` takes
  `person` with `run_as`. Queue items lose `computer_ids`. A new read for the project row's answer (ticket 06). Each
  writing route keeps its permission in `route_permissions.go`, then `make permission-catalog`.
- **MCP.** `computer_pair` takes `labels` on the caller's own computer; `computer_list` returns them. `play_create` and
  `play_update` take `needs`; `play_list` returns them. The auto play fields follow the HTTP change. The tool budget and
  server instructions are at their caps: every new description is paid for by trimming another.
- **Events.** `play.updated` carries `needs` (additive). `auto_play.updated` keeps `computer_ids` in the contract, tagged
  `deprecated`, always `[]`. A pass-over's `why` gains `missing_need`. `make event-schemas`.
- **Waking.** A change to a computer's labels, or to a play's needs, publishes `computer.placement_changed` or wakes the
  queue the way a play update does, so a run waiting on a need tries again.
- **Web.** The Computers tab edits a computer's labels and shows the derived ones. The play form gets **Needs**. The auto
  play's Run on section offers Run as \<person\> in place of picked computers. A waiting run's fix is an inline picker
  for the computer it waits on (ticket 02). The Projects tab row shows the full answer (ticket 06). 768, 1024 and 1440px.
- **Phone.** Nothing: the phone app has no computer settings or queue.
- **Docs.** `paired-computers.md` (choosing where runs go, the reasons table, labels), `plays.md` (Run on, Needs, Run on
  a chosen computer becomes Run as a person), `mcp-server.md`, `CONTEXT.md` (need, label, run as), and the amendment
  lines on ADRs 0156 and 0165. Each ticket updates the pages its change makes wrong.

## Rules that always hold

- A run goes to a computer in its person's order, or one they picked for that one press, and nowhere else.
- A computer missing a need of the play never takes its run.
- A waiting run names every computer it passed over and why, and each reason that a person can fix has its fix on the
  waiting run.
- Saving anything placement reads (order, project list, checkout, labels, needs) wakes that person's waiting runs.

## Out of scope

- Shared computers in a person's order, or in an auto play (decision 4).
- Load balancing beyond today's runs at once and spread.
- Moving a run that is already going (ADR 0165).
- The clone and Add flow itself, which computer-failover ticket 04 owns.
- Needs derived from anything but Docker and the OS. More derived labels (a GPU, a tool on PATH) can join later without a
  schema change.
