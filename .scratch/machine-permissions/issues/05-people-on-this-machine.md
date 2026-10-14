# 05: People on this machine, in the web and on the phone

Type: task
Status: open
Blocked by: 02, 03

Read first: `AGENTS.md`, `CONTEXT.md`, `practices/react-guide.md` (F1 to F7 and the self-review),
`practices/design-language.md` (the shared core and the web app section), `practices/native.md`,
`practices/testing.md`, ADRs 0148, 0151, the spec (UI).

## What to build

- Each machine card on the Runners page gets **People on this machine**: the permissions panel
  (`web/src/components/access/PermissionsPanel.tsx`) limited to See, Deploy to, Manage and Delete, targeting a role of
  any workspace or a person. Shown to the instance Owner and the machine's Manage holders; the same menu on a
  right-click and the ⋯ button, as other rule-carrying rows.
- A machine with no one but the instance Owner says so on its card, with the action to add people.
- The "machines" instance area's Access card in Settings shows Add machines.
- The phone's Runners screen shows only what the server returns and refreshes on the same topics; no editing there.
- Guide: `runners.md` (People on this machine), `people-and-access.md`.

## Acceptance criteria

- [ ] Component tests for the panel's four actions, both targets, and the empty state.
- [ ] Verified at 768, 1024 and 1440px, light and dark.
