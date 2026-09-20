# 02 — The play definition and column validation

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Lock the shape of a play and the rules around its status columns.

1. Fields: label (the button text), type (`ticket` or `doc`), base
   instructions, default memories, excluded projects, and for a ticket play
   the show-when column and the move-to column. Anything missing: an icon,
   a description shown as a tooltip, an enabled/disabled switch, ordering
   among several plays on one page?
2. Show-when for a ticket play: exactly one column, or a set of columns
   ("show in In Dev and in Blocked")?
3. Column deletion: a play references columns by id, so renames are free.
   When someone deletes or re-stages a referenced column: refuse with a
   message naming the plays that use it, or let the play go dormant with a
   warning in settings? The owner asked for "a check and a validation
   message"; settle which.
4. Columns are per project (ADR 0007) but a play is per workspace. A
   workspace-wide "show when column X" therefore needs a rule: the play
   stores one column per project, or it stores a column of a template
   board, or the owner picks a stage after all and the column choice was a
   misreading. Put the concrete scenario to the owner: two projects with
   different column sets, one "Fix with AI" play.
5. The two seeded defaults: created on workspace creation, editable, and
   deletable like any play, or protected? What are their exact default
   instructions (draft them for the owner to react to)?

Recommendation going in: one show-when column set per project stored on the
play (question 4), refusal on delete (question 3), seeded plays are ordinary
plays the owner may edit or delete.

## Answer

Resolved 2026-09-16 with the owner (two grilling rounds).

- **Fields**: label (the button text), type (`ticket` or `doc`), one-line
  description shown as a tooltip, base instructions, enabled switch,
  excluded projects, and for a ticket play a single **show-when stage**.
  No icon, no manual ordering; plays sort by label.
- **No columns on the definition.** Show-when is a stage because the button
  must know when to appear before anyone clicks, and stages cannot be
  renamed or deleted (ADR 0022). Move-to is not stored on the play at all:
  the run dialog lists the project's own columns and the user picks one.
- **The dialog remembers.** For each user, play, and project, the last
  chosen move-to column and the last selected memories are pre-selected next
  time. There are no default memories on the definition; memory ids are per
  project and a workspace play cannot name them. A remembered column that
  no longer exists is simply not pre-selected.
- **Column validation is therefore unnecessary**: nothing a play stores can
  be deleted or re-staged out from under it. The ticket's questions 3 and 4
  dissolve.
- **Show-when is one stage**, not a set; the owner will ask for a set if a
  need appears.
- **Two plays ship out of the box**, "Fix with AI" (ticket, show-when
  `progress`) and "To tickets via AI" (doc): created with every new
  workspace and once by migration for existing ones, then ordinary plays
  the owner edits or deletes and no upgrade overwrites. Their instruction
  text is drafted in ticket 05.
