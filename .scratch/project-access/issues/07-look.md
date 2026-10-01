# 07 — How it looks: role editor, Team dialog, project settings, channels

**Type:** prototype
**Status:** resolved
**Blocked by:** 02, 04, 05

## Question

What do the owner and a Restricted member see?

- The role editor's two sections, Workspace and Every project.
- The Team dialog's workspace row: the Every project row (From role or
  None) and one row per project opening into area levels (ticket 05), at
  768px and up; the same rows in the invitation dialog.
- Project settings' read-only list of who has access.
- Making a channel private (the who-stays picker), managing its members,
  and leaving.
- The project delete confirmation's line naming who loses access.
- What a Restricted member's sidebar and project list look like.

Run through `design-mode`; the owner picks from live variants.

## Answer

Prototyped 2026-10-01 in two rounds; the owner delegated both picks. The
first round (an accordion of None/Read/Write/Delete strips per area) was
rejected: a wall of identical controls, boxes nested three deep, rows
doubling at 768px, the one-level common case costing as much as per-area
detail, and the Every project control the smallest thing on screen. The
second round, below, replaced it.

- **A level is a compact trailing dropdown**, never a strip of buttons: the
  row's name on the left, the current level as muted text with a chevron on
  the right ("Write", "Read + Run"). Its menu lists None "No access", Read
  "Open and read", Write "Create and edit", Delete "Also delete", each with
  that one-line description, then the area's verbs as checkable items under
  "Also allow", then, where a grant can go, a destructive "Remove access"
  last. Rows are about 44px hairline rows on the surface they sit on, under
  mono microheaders; no bordered box inside a card or dialog.
- **Team dialog workspace entry:** the role select, then the Every project
  row (muted icon, "Every project" over a one-line consequence, a full-size
  select on the right: "From role", reading "Every project, at their role's
  level.", or "Only chosen projects", reading "Sees only the projects below.
  New projects stay hidden."). Under Only chosen projects: a `Projects · 1 of
  5` microheader with a search field from five projects, then one row per
  project with one dropdown for the whole project, its value the uniform
  level or "Custom" with a mono summary under the name ("tickets Write ·
  docs Read"). The menu's "Customize areas…" opens that project's area rows
  inline, indented, one project at a time, each area with its own dropdown,
  closed again by a "Hide areas" link under them. Focus rings show for
  keyboard focus only. Rejected in the second round: area rows in a popover
  (two layers deep, capped at eight of eleven areas) and a second dialog
  step (loses sight of the list and the Every project row).
- **Invitation dialog:** the same block under the workspace and role
  selects.
- **Role editor:** every domain is a row with the trailing dropdown, under
  two mono microheaders, Workspace (with the instance areas) and Every
  project ("Applies to members whose Every project is From role."), each
  with an "Every domain" / "Every area" row on top reading "Custom" when
  rows differ.
- **Project settings, People with access:** a settings card with hairline
  rows straight on its surface (initials avatar, name, mono summary), "Change
  access from Team." in the footer, an empty row when there are none.
- **Private channel:** one card with the "Private channel" row (lock icon,
  "Only members see it and read it.", a switch) over the member rows (count,
  "Add people", a `…` per row with "Send a message" and, after a separator,
  "Remove from channel"), and a destructive "Leave channel". Turning it on
  opens "Who stays in #<channel>?": a count, a search field, a checkable
  people list with the person switching checked and fixed.
- **Project delete confirmation:** a line "Fahad and 1 other restricted
  member lose access." with the names under it.
- **A Restricted member's sidebar:** only their projects, only their private
  channels (trailing muted lock), DMs as usual.
- **Access taken while a page is open:** the standard empty state, "You no
  longer have access to this project", "Someone changed your access.
  Projects you can still open are in the sidebar.", "Go to Home".
- **Every change** confirms with a toast naming the person and the project,
  or the channel.
- Motion follows the design-language baseline; nothing new.
- `practices/design-language.md` gains a "Permission rows" pattern entry
  with the above, and its card rule ("Lists inside a card are hairline rows
  in one bordered box") gains the exception for permission and access rows,
  which sit straight on the card or dialog surface.
