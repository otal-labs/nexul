# 08 — The role editor as a dialog

**Type:** prototype
**Status:** resolved
**Blocked by:** None

## Question

Creating or editing a role should open a **dialog with a Save button**, not a form that expands in the list. The
owner asked for this on 2026-10-11, so the editor doesn't feel overwhelming. The roles list on Configuration →
Roles stays as it is ("looks clean").

How does the dialog look and behave?
- the name
- copy-from
- the role's permissions after the rework: a workspace section, and an "In projects" section replacing "Every
  project"
- the new "Create projects", "Manage this project's access" and per-play See
- extras such as Run and Clone
- a scrolling body with the footer's Save always in view
- its widths at 768, 1024 and 1440px

Current screens: `~/Code/Images/permission-rework/current-roles/`. Lock the pick (auto design mode).

## Answer

Locked on 2026-10-11 from a live prototype with mock data (workspace Acme; roles Developer, Senior Developer,
Frontend, Client and a 60-character worst case), judged from screenshots at 768, 1024 and 1440px in both modes. The
owner delegated the pick.

**Where it opens.** The roles list stays as it is. The row menu's Edit and the card footer's New role open the same
dialog; the inline `RoleEditForm` and `CreateRoleForm` go. Duplicate, Clone to workspace and Delete are unchanged.

**The dialog.** The house dialog at 40rem (`sm:max-w-[40rem]`), `h-[min(52rem,calc(100dvh-4rem))]`, `p-0 gap-0`, in
three parts like `TeamPersonDialog`:
- Header (`px-6 pt-6 pb-4`): the title "Edit Developer" or "New role" (16px semibold, two lines at most, full name in
  its tooltip). Under it on an edit, the holders as overlapping 20px avatars and one muted line: "Held by Bob and Lena.
  Saving changes what they both can do.", "Held by Alice. Saving changes what they can do.", "Held by Alice, Bob and 2
  more. …", or "Nobody holds this role yet." On a new role: "Nobody holds a new role until you give it to someone in
  Team." Autofocus is skipped.
- Body: the only part that scrolls (`overflow-y-auto`, a hairline above it, `px-6 pt-5 pb-6`, 24px between blocks).
- Footer: the dialog tray, always in view. On the left a 12px muted note: "No changes yet" (edit), "Name it to create
  it" (new and unnamed), or, once something differs from what was loaded, a 6px `foreground` dot and "Unsaved changes"
  settling in with the existing `settle-in`. On the right a ghost Cancel, then the primary: "Save" on an edit, "Create
  role" on a new role, disabled until something changed and the name isn't blank.

**Name and copy-from.** "Name" over the input. On a new role, beside it from a 32rem body (a 12rem column, under it
below that), "Start from": a select of "Nothing allowed" (the default) and "A copy of <role>" for every role but
Owner. Picking one replaces the draft's permissions, and a muted 12px line appears under the fields: "Copied from
Developer. Change anything below; Developer stays as it is." Nothing is saved until Create role.

**Two sections.** Each is a microheader with a mono 11px tally trailing on its line ("12 of 24 allowed": rows above
None plus abilities switched on), one 14px muted line, then one bordered list (`rounded-md border border-input`,
hairline dividers):
- **Workspace**, "Across Acme, whatever project someone is in.": Create projects, then the workspace areas, then the
  instance areas, in catalog order.
- **In projects**, replacing "Every project", "Only inside the projects a holder is in. Each project lists who's in
  it.": Manage this project's access, then the project areas.

**The rows, top to bottom in each list.**
- The bulk row, one per section and named for it: "All workspace areas" / "All areas in projects" over a 12px muted
  "Sets every row below at once", on the `bg-muted/40` strip, with the four-rung strip. It selects no rung once the rows
  differ, and it never touches the switches. This replaces "Every domain" and "Every area".
- An ability row for each yes-or-no permission: the name in 14px medium over a 12px muted line, a `Switch` trailing,
  the whole row its label. "Create projects": "A new project lists its creator and the roles they pick." "Manage this
  project's access": "Choose who's in a project and what they can do there."
- An area row: the name in 14px (muted while at None, truncated with its tooltip) and the existing level strip
  trailing (`PermissionLevelControl`'s idiom: 28px rungs, 56px each, every rung up to the level filled). Plays' first
  rung reads **See** instead of Read; a play's own See rule can still hide one play.
- An area's extra verbs sit on a second line under its name, so they aren't missed: a checkbox (the house checked
  control) and a short word each: Run (plays), Threads, Clone, Lock (docs), Clone (memories, roles), Logs (stacks),
  Create (workspaces). The full label ("Read container logs") is the tooltip and the `aria-label`. At None they are
  disabled and muted with the tooltip "Needs Read first" ("Needs See first" on plays), and setting None clears them, as
  `withLevel` already does.

**Leaving.** Cancel, the ×, Escape or a scrim click with nothing changed closes at once. With changes, the existing
confirm asks "Discard changes?", "Your changes to Developer haven't been saved." (a new role: "This role hasn't been
created yet; what you set here will be lost."), confirm "Discard changes". Save shows the usual toast ("Developer
saved", "Support created") and closes.

**Per width.** The same at 768, 1024 and 1440: the dialog keeps 40rem, so at 768 it sits with 64px either side and
nothing reflows; Start from moves under Name only below a 32rem body.

**Left out on purpose.** No search over permissions: built and judged, it saves one scroll at 35 rows and costs a
field on every edit; add it when the catalog passes about 50 rows. No sticky section headers: an opaque strip over the
frosted dialog read as a band in dark mode.

**Builder notes.** The draft is one held value (name and actions) compared with what was loaded for "dirty"; a store
selector that falls back to a fresh `[]` when the dialog is closed loops React on close, so the fallback is a stable
constant. The catalog keeps area-driven sections (`domainsIn`); a domain with no ladder renders as switches.

**Motion.** Nothing new: the dialog's surface motion, the checkbox pop, the switch thumb and `settle-in` are existing
locks, with their reduced-motion variants.

**Runner-up and how to swap.** The same dialog with each section folded to one row: a chevron, the section's name in
14px medium over its muted line, and the tally trailing; opening it shows the same list under it, and only one click
per section. To swap, wrap each section in a `Collapsible` with that row as its trigger; everything else stays. It lost
because a closed section hides the role while you edit it, and the extras the brief wanted visible with it.

**Where it lives locally (outside the repo).** Screenshots of the pick at 768, 1024 and 1440 in both modes (edit with
Save live, create with copy-from and its menu open, two long-scroll positions, the discard confirm, the worst cases),
the variants, the runner-up and a short clip are in `~/Code/Images/permission-rework/08-role-dialog/` (see its
README). The prototype is the local branch `proto/permission-role-dialog` (`66b77964`, never merged), route
`/<workspace>/proto/role-dialog`: `?v=stack|rail|fold`, `?open=developer|senior|frontend|client|long|new`,
`?copy=<role>`, `?name=`, `?q=`.
