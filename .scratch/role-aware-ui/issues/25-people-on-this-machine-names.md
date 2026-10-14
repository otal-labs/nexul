# 25: People on this machine shows ids to someone who can't list people

Type: bug
Status: resolved
Blocked by: None

Read first: `AGENTS.md`, `CONTEXT.md` (Machine), ADR 0151, `practices/react-guide.md`, `web/src/hooks/usePanelTargets.tsx`.

## Problem

On the Runners page, the **People on this machine** row summarises a machine's rules. Found on 2026-10-14 as a DevOps
person who holds Manage on the machine but not Team or Roles:

- The row reads `01a13b1e-5b1f-7275-87d9-ef232fdbf653, Unknown role`: the Owner's user id, and the viewer's own role
  with no name.
- The Owner sees the same row as `DevOps in Default`, so the names exist; this viewer's lists just don't carry them.
- `usePanelTargets` resolves a machine rule's targets from the Team list, which needs an instance permission this
  viewer doesn't hold, so every target falls to `unknownPerson` and "Unknown role".

A raw id in a sentence tells the reader nothing, and an Owner's id is not something Manage should show.

## What to build

Have the machine rules answer carry a display name for each target, as explain's sources already do
(`target_name`), and use it in the summary and the panel when the viewer's own lists don't resolve one. Don't widen what
the viewer can list.

## Acceptance criteria

- [x] A Manage holder who can't read Team sees role names and person names in the summary and the panel.
- [x] A person's id is never shown as their name.
- [x] The viewer's access to the Team and Roles lists is unchanged.

## Answer

The rules list (`GET /api/permissions` and `permission_overwrite_list`, for every entity kind) now carries
`target_name` on each rule: the role's name or the person's login, resolved by the same namer explain's sources use,
and left out when nothing resolves (an id is never put there). `usePanelTargets` takes `rulesOn`, the entity whose
rules name the targets, and names a target its Team and Roles lists can't resolve from that field, else "Someone" for
a person and "Unknown role" for a role. The machine row and the permissions panel both read it, so a Manage holder
without Team reads `dev, DevOps` instead of an id and "Unknown role". The Team and Roles checks are unchanged.
