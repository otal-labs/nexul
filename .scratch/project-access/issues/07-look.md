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

Prototyped 2026-10-01 with three variants of the project rows, judged at
768, 1024, and 1440px with worst-case names; the owner delegated the pick.

- **Team dialog, project rows: an accordion.** Under the workspace's role
  select, one bordered box: the Every project row first (muted lock icon,
  "Every project" over a one-line consequence, From role / None on the
  right, on the raised `bg-muted/40` surface the role editor's "Every domain"
  row uses). From role reads "Every project, through their role." and shows
  nothing below. None reads "Sees only the projects below. New projects
  stay hidden." and lists every project under a mono `1 of 5 projects`
  microheader and a search field: each row is a chevron, the project name,
  and a mono summary of what is granted ("None", or "tickets Write · docs
  Read · memories Read"). Opening a row shows the existing level list for
  project areas only, with an "Every area" row on top. One project is open
  at a time, so the dialog never grows past one project's areas. Rejected: a
  two-pane list and detail, which truncates names and summaries at the
  dialog's width; listing only granted projects with an Add popover, which
  hides the None rows and needs a starting level nobody chose.
- **Invitation dialog:** the same box under the workspace and role selects.
- **Role editor:** the level list splits under two mono microheaders,
  Workspace (with the instance areas) and Every project, the second with
  the line "Applies to members whose Every project is From role." and an
  "Every area" row of its own.
- **Project settings, People with access:** a settings card listing the
  Restricted members who can open the project (initials avatar, name, mono
  summary), "Change access from Team." in the footer, and an empty row ("No
  restricted member can open <project>.") when there are none.
- **Private channel:** in channel settings, a row in the same shape as Every
  project (lock icon, "Private channel", "Only members see it and read it.")
  with a switch. Turning it on opens "Who stays in #<channel>?": a mono
  count, a search field, and a checkable people list with the person
  switching checked and fixed. A private channel lists its members with a
  mono count, "Add people", a `…` per row for removal, and a destructive
  "Leave channel" action.
- **Project delete confirmation:** adds a bordered line, "Fahad and 1 other
  restricted member lose access.", with the names under it.
- **A Restricted member's sidebar:** only their projects in the project
  switcher, only their private channels (each with a trailing muted lock),
  and DMs as usual.
- **Access taken while a page is open:** the page body becomes the standard
  empty state, "You no longer have access to this project", "Someone changed
  your access. Projects you can still open are in the sidebar.", and a "Go
  to Home" action.
- **Every change** confirms with a toast naming the person and the project,
  or the channel.
- Motion follows the design-language baseline (row open and toast at 150 to
  200ms `--ease-out`, level changes at 150ms `--ease-standard`); nothing new.
