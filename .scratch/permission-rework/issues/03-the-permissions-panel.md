# 03 — The permissions panel and the right-click menu

**Type:** prototype
**Status:** resolved
**Blocked by:** None

## Question

How do the permissions panel and the right-click menu that opens it look and behave?
- The panel has a list of targets (roles, then people, plus add) and the chosen target's permissions grouped by
  domain, each with Allow / Fallback / Deny. Under each Fallback it shows the inherited answer, and Check access
  shows a person's answers with their sources.
- On a project, the panel sits behind the "People in this project" list.
- The context menu is on docs, folders, projects, channels and plays, and the same menu sits on ⋯.

The panel is checked at 768, 1024 and 1440px. Lock the owner's pick.

## Answer

Locked on 2026-10-11 from a live prototype with mock data (roles Developer, Senior Developer, Frontend, Client;
people alice, bob, lena, sam; project Storefront; folder Sales; doc Pricing; channel #backend; a play), judged from
screenshots at 768, 1024 and 1440px in both modes. The owner delegated the pick.

**Where the panel opens**
- A dialog titled "Permissions", the muted line naming the entity and where it sits ("doc **Pricing** in
  Storefront › Sales"). 56rem wide (`sm:max-w-4xl`), up to 46rem tall; header and footer stay put and the body
  scrolls. Autofocus is skipped, so nothing in it opens with a focus ring.
- The header's right holds a segmented control (`ToggleGroup variant="segmented" size="xs"`): **Rules** |
  **Check access**. It is a view of the same panel, held in the dialog, not in the path.
- Footer: a muted "Rules apply when you save." on the left, Cancel then Save (`brand`) on the right. Changes are
  held until Save.

**Layout per width** (a container query on the panel body at 48rem)
- 1024 and 1440 (the dialog is 56rem): the targets are a 15rem column on the left behind a hairline rule, scrolling
  on their own; the chosen target's rules fill the rest with 24px of gutter and scroll separately.
- 768 (the dialog is about 46rem): the targets become one horizontally scrolling row of compact tabs above the
  rules: role tabs, a short vertical hairline, people tabs, then the add button. A tab is the mark and the name; the
  chosen one is `bg-accent` with a hairline ring. Rules sit full width under it.

**The target column** (Rules)
- A "Roles" microheader, then the entity's roles in the order they were added; a "People" microheader, then its
  people. No Everyone anywhere.
- A row is a 24px mark (a role is a muted people icon on a `surface-2` tile with a hairline ring; a person is the
  gradient avatar), the 13px medium name (truncated, full name in its tooltip), and a mono 11px line under it: a
  role's "2 rules" or "no rules"; a person's held roles then their rule count ("Client · 1 rule").
- Selected: `bg-accent` with the 2px `brand` left edge, as in the list pane. Hover: `bg-accent/40`.
- The add control sits at the column's foot: a ghost "+ Add role or person" opening a popover in the menu anatomy,
  "Roles" and "People" microheaders over the roles and people not listed yet (mark and name per row). Picking one
  lists it and opens it. With nobody left: "Every role and person is listed."
- No search field for now; Check access's people column takes one once a workspace has more than about eight people.

**The rules pane**
- Header: the target's 32px mark, its name in 15px semibold, and a muted kind line ("Role", or "Person · holds
  Client", or "Person · holds no role"). On the right a ghost "Remove from this doc" (or folder, project, channel,
  play) that turns destructive on hover; it drops the target and all its rules here.
- Then one group per domain the entity carries: a microheader ("Docs", "Tickets", "Stacks", "Access"...) over the
  permission-rows box (`rounded-md border border-input`, hairline dividers). A project carries Project (See this
  project, Edit settings, Manage this project's access), Docs, Tickets, Stacks, Memories; a doc or folder carries
  Docs and Access (Manage this doc's access); a channel carries Channel; a play carries Play (See this play, Run,
  Edit, Delete).
- A row: the permission's name in 14px on the left; on the right the three-state control.
- **The control is a compact select**, 28px tall, at least 7rem: an outline button (`bg-card`, `border-input`,
  `shadow-card`) showing the current state's icon and word, and a chevron: a check in `success` for Allow, a muted
  slash for Fallback, a cross in `destructive` for Deny. It opens a popover in the menu anatomy (9px, 4px padding,
  288px wide) with three rows, each an icon, the word and one muted line:
  - Allow: "Allowed here, whatever the role or the parent says"
  - Fallback: "No rule here: the answer comes from further out"
  - Deny: "Denied here, whatever the role or the parent says"
  The current one carries a trailing check.
- **A permission the giver doesn't hold:** its Allow row is disabled, shows a lock instead of the check, and its
  muted line reads "You can't allow a permission you don't hold". Fallback and Deny stay open.
- **Under each Fallback** (only then) a line in 12px muted text: a 6px dot (`success` allowed, `destructive` denied,
  muted not allowed) and the source sentence of what applies without this rule, for example "Denied on project
  **Storefront** for role **Client**", "Allowed by role **Frontend**", "Not allowed: role **Client** doesn't grant
  it", "Not listed on project **Storefront**". Named entities and roles are links (medium weight, a hairline
  underline): an entity opens its own panel, a role selects it in the column.
- No target chosen: one muted sentence, "Pick a role or person, or add one, to set rules on Pricing."

**Check access**
- The column swaps to a "Check a person" microheader over every member of the workspace (mark, name, held roles in
  mono, or "no role"); the chosen person gets the same selected edge.
- The pane heads with the person's 32px avatar, "What bob can do" and a muted "On doc Pricing, read from every rule
  that reaches it".
- Then the same domain groups and rows, each row the permission's name over its answer: the dot and the source
  sentence, which leads with the verdict ("Allowed on doc **Pricing** for bob", "Denied on project **Storefront**
  for role **Client**", "Allowed by role **Client**", "Not allowed: no role of bob's grants it", "Not allowed: sam
  holds no role"). There is no separate verdict column; the sentence says it.
- A person outside the entity's project sees one sentence instead of the groups: "Not found: sam isn't in project
  **Storefront**", in a bordered box, with the project as a link.

**On a project: "People in this project"**
- A `SettingsCard` in the project's settings: title "People in this project", description "Only the roles and
  people listed here can open Storefront. Open one to set what it can do."
- A "Roles" microheader over role rows, a "People" microheader over person rows. A row: the 24px mark, the name in
  14px medium, and a muted line ("3 people hold it · the role's own permissions", "Client · 1 rule here"); a
  chevron trails, and the whole row opens the panel on that target. A ghost icon `X` beside it ("Remove bob from
  Storefront") turns destructive on hover and asks before it acts.
- The footer strip holds "+ Add a role or person", the same popover as the panel's add control; adding lists the
  target, which is the rule that lets it in, and does not open the panel.
- Empty: "Nobody is in Storefront yet. Only Owners can open it."

**The row menu**
- A new `components/ui/context-menu.tsx` from the shadcn/ui primitive, restyled to the house menu anatomy: the
  `float-surface glass-menu` panel at 9px, 32px rows at a 5px corner with `gap-2.5`, a muted 16px icon column,
  microheader labels, mono shortcut hints, and no keyframe animation (the menu motion lock). The CLI resolves its
  `cn` import as an npm package called `cn`: change the import to `@/lib/utils` and keep that package out.
- Right-clicking a doc, doc folder, project, channel or play row, in the sidebar or a list, opens the menu at the
  pointer, and the row keeps `bg-accent` while it is open. The row's ⋯ (`RowActions`, added where a row lacks one) opens the
  same items as a dropdown, for keyboard and touch.
- Items: the row's existing actions first (a doc's Pin, Lock, Move to folder ›, Clone; a folder's New doc, Rename;
  a project's Project settings; a channel's Settings, Rename; a play's Run, Edit, Clone), then a hairline,
  **Permissions…** with a shield icon, then a hairline and Delete in `destructive` where the row has one.
  Permissions… shows only to someone who may manage the entity's access, and opens the dialog above on that entity.

**Motion:** none new. The dialog, the popovers, the menus and the segmented thumb use the existing overlay and
navigation locks; switching a target or a person is instant.

**Runner-up and how to swap:** the same panel with the control as an always-visible segmented strip:
`ToggleGroup variant="segmented" size="xs"` with Allow (check, `success` when on), Fallback and Deny (cross,
`destructive` when on); a locked Allow is disabled with a lock and a tooltip carrying the same sentence. Every
other part stays. It lost because a column of identical three-part strips is busier than one value per row, and a
tooltip says less than the menu's line about a locked Allow.

**Where it lives locally (outside the repo):** screenshots of the pick at 768, 1024 and 1440 in both modes (the
panel, Check access, the context menu, the ⋯ menu, the locked Allow and the project's people list), the variants
and the runner-up are in `~/Code/Images/permission-rework/03-panel/`. The prototype is on the local branch
`proto/permission-panel` (`89607488`), route `/<workspace>/proto/permissions`; `?v=select|words|icons`,
`?open=<entity>&mode=rules|check&target=role:<id>&check=<person>` open a state directly.

**Questions this raised for the owner**
- On a project, being listed and the "See this project" rule are the same thing. To keep one person out of a project
  their role is in, a rule names them with See denied, so they appear in "People in this project" as someone who is
  not in it. Should the list show such a person under their own heading ("Kept out"), or should See be hidden from
  the project's panel and keeping someone out be a separate action?
- A doc's Fallback for a role reads the folder, then the project. A role not listed on the project still gets a
  doc panel row, and its fallback sentence then says "Not listed on project Storefront". Should the panel refuse to
  add a role or person to a doc whose project they are not in?

**The owner's changes (2026-10-11), these override the parts above they touch**
- **The control is three buttons side by side**, the way Discord's permission rows used to be, not a dropdown. This
  is the prototype's `?v=icons` variant: a small segmented group of three icon buttons, check (Allow), slash
  (Fallback) and cross (Deny). The one that's on is filled: Allow `success`, Fallback neutral (`accent`), Deny
  `destructive`. Each button has its label as a tooltip and an `aria-label`, and the group is a radio group for the
  keyboard. A locked Allow is disabled with a lock and a tooltip carrying the menu's sentence ("You can't allow a
  permission you don't hold"). The source line under each Fallback and every other part stay as locked above.
- **Keeping someone out:** a person denied See on a project is shown on "People in this project" under their own
  **"Kept out"** heading, below the people who are in.
- **A doc's panel refuses** a role or person who isn't in the doc's project. The add popover lists them disabled,
  with "Add <name> to <project> first" linking to the project's list.
