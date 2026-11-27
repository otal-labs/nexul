# Agent page rework

**Status:** ready-for-human

## Problem

The Agent page grew one section at a time and reads as a pile:

- One left list mixes four play groups, your computers, computers shared with
  you, and the Projects run settings.
- An auto play can only be reached inside its play's page.
- The workspace's daily cap, and every computer's runs-at-once, are edited
  inside a single ticket play's page, and runs-at-once is editable in two places.
- The switch that stops a computer taking plays sits deep inside its details
  box, so the owner could not find it.
- An auto play has no way to keep its runs off particular computers. The owner
  tried "Label is not windows, linux", which tests the ticket's labels, not the
  computer's.
- Add a computer saves a blank computer before the install runs. When the
  install never finishes, the row stays for good as "New computer", in the
  limits list and the run order.

## Decisions (picked by the owner from prototypes, 2026-11-27)

1. **Tabs, one per play type.** The page opens on Ticket plays. Tabs: Ticket
   plays, Doc plays, Stack plays, Interview plays, each with its count. There
   are no filter tabs (no All, no "Start by themselves", no "Only when
   pressed"). The header has a Computers button (a status dot and "2/4
   online") that opens the Computers dialog, then the primary button, which
   follows the tab: New ticket play, New doc play, and so on.
2. **A play list is a tree.** Each play is a row with its name, then one line
   with its button stage and description, its runs in the last 7 days, its
   last run, and its enabled switch. Under it, one indented ⚡ row per auto
   play, worded as a sentence ("When it enters In progress, developer is
   Onik"), with its own 7-day count and on/off switch. After the sentence come
   pills naming each computer it would run on, each with its state dot. When
   the person is only known at run time, one muted pill says so ("Whoever
   moved it: their computers").
3. **Rows open in place.** Clicking an auto play row unfolds its editor under
   the row, with Save and Cancel. Clicking a play row unfolds the play's own
   settings (instructions, button stage, excluded projects, needs, repository)
   the same way. There is no side sheet, no dialog, and no separate play page.
4. **The auto play editor is a flat form.** It has When, Only if (one level
   of conditions, all or any), Runs as, **On computers**, Priority, Limits and
   Memories. Nested condition groups stay in the model, but the form shows
   them only when one exists.
5. **On computers: by label or by name.** By label: "Labelled" / "Not
   labelled" plus labels, which works whoever the auto play runs as. By name:
   a list of computers, offered only when it runs as a fixed person. Under the
   field, a live "Right now that's ..." line names the computers it reaches;
   a computer it excludes is struck through.
6. **The Computers dialog is a list beside a detail pane.** It is a large
   dialog with "Computers" as its title and Add a computer in its header. It
   replaces today's computer rows in the left list.
   - **Left:** each computer in run order: state dot, name, #rank, short
     state, and its providers as the app's own marks
     (`HarnessProviderMark`), not written names. Computers shared with you
     sit in the same list, marked as shared. Unfinished setups sit at the
     foot, with Clear.
   - **Right:** the picked computer:
     - the header, with Open terminal and Update skills;
     - switch rows for Plays you start, Auto plays and Terminal;
     - Runs at once, Models (a mark plus the model), and Chat folder;
     - "On this computer" facts: T3 Code, Nexul app, free disk, T3
       projects, labels.
7. **Each setting has one home.** Runs at once is edited only on its computer,
   and it is the only limit on how much runs. The workspace's daily cap ("auto
   runs per ticket a day") is removed.
8. **A play's History is a dialog.** It opens from the unfolded play's
   History button: versions on the left, each with its date, a template note
   and its words added and removed; the picked version's instructions on the
   right, with Restore this version.
9. **Where runs go lives in the Computers dialog.** Under the computers, the left
   list has a "Where runs go" group: Defaults first, then each project, each
   with one muted line of what it uses ("Uses defaults"). Picking one opens its
   run settings in the right pane: Computers (Your order or Custom), Send runs
   to, Model, and Threads start in. A project's pane has Use defaults.

## Schema changes

- `pairing_computers.auto_plays_off INTEGER NOT NULL DEFAULT 0` sits beside
  `plays_off`. Placement passes a computer over for queued auto runs when it is
  set, and for every run when `plays_off` is set.
- `auto_plays.computer_rule` is JSON
  `{"labels_in":[],"labels_out":[],"ids":[]}`. Placement filters the run-as
  person's ordered computers by it; `ids` is accepted only when `run_on` is
  `person`. This partly reverses ADR 0197's "auto plays name no computers":
  write a new ADR that supersedes that line.
- Add a computer no longer saves a `pairing_computers` row before enrolment:
  the enrolment code carries the pending setup, and the row is created when
  the runner enrols. A migration deletes rows with an empty name, no token
  and no runner that are older than one day.

## Surfaces to reach

- **Web:**
  - the Agent page;
  - the ticket rail's Plays section, which keeps working;
  - Your settings, whose Computers link opens the Computers tab.
- **MCP:**
  - `play_update` takes `computer_rule` on an auto play;
  - the computer tools take `auto_plays_off`;
  - `computer_list` shows both switches.
- **Events and live push:** add the new fields beside the existing ones, then
  run `make event-schemas`.
- **Phone:** `native/` reads plays and auto plays (`PlayHooks`, `NowSection`),
  so confirm it tolerates the new fields.
- **Docs:**
  - `practices/design-language.md`, its Plays and Paired computers
    paragraphs and the decision ledger;
  - the guide pages `plays.md` and `paired-computers.md`;
  - `CONTEXT.md` if a term changes;
  - the new ADR.

- The daily cap goes:
  - drop `workspaces.auto_play_daily_cap`;
  - drop the cap check in `dispatchItem`, along with `autoRunsToday`;
  - drop the `auto-play-limits` routes and the cap on `workspace_list`;
  - drop the ticket rail's "the daily cap paused it" state.

  Queued auto runs wait only for a computer with room.

## Open

- The owner signs off the clickable prototype before the build starts.

Prototype screenshots and the clickable prototype (not committed) are in `~/Code/Images/nexul-agent-rework/`.
