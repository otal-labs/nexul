# 04 — Several roles on screen, and a person with none

**Type:** prototype
**Status:** resolved
**Blocked by:** None

## Question

How does Team show and edit several roles per person, and what does an invite with one or more roles look like?
What does a member with zero roles see when they open the app? Lock the owner's pick.

## Answer

Locked 2026-10-11; the owner delegated the pick. The roles are a checklist, every role visible at
once, never a dropdown. The look is the same at 768, 1024 and 1440px: the Team card and the dialogs keep their
existing widths, and only the sidebar changes (icon rail below 1024px).

**Team rows.** Today's `TeamPersonRow`, with the roles where "Developer in Acme" was. Under the name sits a line of
role chips: a neutral pill (`rounded-full`, hairline `border`, `foreground` at 4%, 11px medium, 20px tall, never
tinted, never the accent). The Owner chip comes first and looks like any other. Three chips show, then a mono `+N`
whose tooltip lists the rest; a chip truncates at 10rem with its full name in the tooltip. A person with zero roles
gets one dashed chip, "No roles", in muted text. Presence and the chevron stay trailing. A person in several
workspaces keeps today's "N workspaces" line; the dialog holds the rest. The card's description becomes "People in
the workspaces you manage, and the roles they hold in each. Invite someone with a link."

**Person dialog.** The workspace tab's Role select, overrides and Project access are replaced by one Roles section,
straight on the dialog surface (the Permission rows rule):
- Header: the microheader "Roles", a mono "2 of 5" trailing, then one muted line: "Everything these roles allow,
  combined."
- One hairline row per role you may give, at least 44px: a checkbox, the name in 14px medium, and the role's level
  tally trailing (`LevelGlyph`, the level, a mono count; two levels at most, highest first, as in `RoleRow`). The
  whole row is the checkbox's label.
- Under the microheader "Can't give", the roles that hold more than you hold, plus Owner. Each is a disabled checkbox
  that keeps the person's tick, the name in muted text, and a muted line with a small lock icon giving the reason:
  "Holds Deploys: Delete and Runners: Write, which you don't hold." or "The Owner role is never given or taken here."
  You can't take away a role you couldn't give.
- With no box ticked, a line appears under the list: a `warning` triangle and "With no roles, Sam sees nothing in
  Acme: no projects, channels or plays." It is a signal, not a gate: Confirm stays enabled. It settles in with the
  existing `settle-in` when the last role is unticked, and shows still when the dialog opens on a person who already
  has none (one entrance at a time).
- Under the section, one muted line: "The projects, channels and plays Lena sees follow from these roles and from the
  rules on each one." Then "Remove from workspace", as today. Changes are still held until Confirm, and the tab's
  dot marks them.

**Invitation dialog.** The same checklist sits under each workspace row, titled "Roles in Acme", with no zero-roles
person line; instead "Pick at least one role. An invitation gives a role on joining." Create link is disabled while
any workspace has no role ticked. Nothing is ticked by default. The per-grant overrides disclosure goes. The
description reads "Pick the workspaces and the roles the link gives. It works once." The invitation page lists each
workspace with its roles as the same chips, right-aligned and wrapping.

**A member with zero roles.** The workspace's page shows `EmptyState` at full size with a `KeyRound` icon, the
headline "No roles in Acme yet", the line "Your roles decide what you see here, so there are no projects, channels or
plays for you yet. An Owner can give you a role.", and as its action the microheader "Owners of Acme" over a 18rem
hairline list of each Owner (24px avatar, name, mono `@login`). The sidebar keeps the logo, the workspace switcher,
Search, Inbox and the account row. It draws no Project section, and no Channels, Voice channels or Direct messages
section: a conversation section with no rows and nothing the person may create there is left out. Leaving the header
alone read as broken.

**Your settings.** A read-only "Your roles" card: one hairline row per workspace, the name left and its role chips
right. The description reads "What you hold in each workspace. An Owner of that workspace changes them." The sidebar's
account row, which shows the role name today, lists the roles joined by commas and truncated, or "No roles".

**Motion.** Nothing new. The checkbox mark keeps its pop, the dialogs keep their surface motion, and the empty state
keeps its entrance; the only addition is the zero-roles line's `settle-in`, whose reduced-motion fade is already in
`index.css`.

**Runner-up, and how to swap.** In the dialog, the held roles are chips with an ×, then a ghost "Add role" that opens
a menu of checkable roles. A "Can't give" label sits over the greyed roles, each with its reason under its name, and
a held role you can't give has no ×. Team rows list the roles as plain muted text ("Developer, Frontend", "No roles").
It lost because the menu hides the full set and the reasons until it opens, and plain text reads like any other meta
line. To swap, build the dialog's Roles section and the invitation's from the menu version and the rows from the
text line. Everything else above is unchanged.

**Where to look (local, outside the repo).** Screenshots of the pick at 768, 1024 and 1440px in light and dark, the
runner-up at 1024px, the scored references and the motion clip:
`~/Code/Images/permission-rework/04-roles/` (see its README). The prototype is the local branch
`proto/permission-roles`, never merged: `/proto/roles` (`?v=pick` or `?v=alt`, `?view=team|zero|you`,
`?person=lena|sam`, `?invite=1`) and `/proto/invite`, on mock data with roles Owner, Developer, Senior Developer,
Frontend and Client.
