# 05 — The one-time "who's in each project" setup

**Type:** prototype
**Status:** resolved
**Blocked by:** None

## Question

After the upgrade, how does an Owner's first visit present the projects × roles grid, plus hiding public
channels from chosen roles? What do non-Owners see before it is filled in? Can the Owner come back to it later?
Lock the owner's pick.

## Answer

Locked on 2026-10-11. The owner delegated the pick and can swap to the runner-up below.

**Shape.** The setup is one page with two grids stacked: Projects, then Public channels, and a single Finish setup at
the foot. Channels are a second grid on the same page rather than a second step, so the Owner sees the option to
hide a channel without having to look for it, and can leave it alone. Later the same page becomes the workspace's
**Access** page, with the two grids as tabs.

**Where it lives.**
- The route is `/:workspace/access`, a workspace page with its own `Access` row (ShieldCheck icon) in the sidebar's
  Workspace section, between Automations and Configuration. The row shows for anyone who can manage project
  access. It is not a Configuration section: from 1024px that page's 12rem side nav leaves the grid about 30rem,
  which is too narrow for five role columns.
- The page is a grid page, so it uses `Container` wide (`max-w-7xl`), built at 768px and checked at 1024 and 1440px.
- On an Owner's first visit after the upgrade, the workspace home redirects once to `/:workspace/access` in its
  setup state. Until Finish, the sidebar's Project section shows one `brand` row, **Set up access**, in the same
  slot and style as Continue setup. That gets the Owner back if they leave.
- The setup state is per workspace and shows while no project in the workspace lists anyone and the setup has not
  been finished. Finish records that it was done, and the page never shows its setup state again.

**Header (setup state).** `PageHeader` with the workspace crumb, the title "Who's in each project", and a mono meta
line "8 projects · 5 roles". Under the header's rule, one muted paragraph (`max-w-2xl`, 14px):
"This update rebuilt permissions, so every project starts with nobody in it and only Owners can see them. Choose
who's in each project; you can change it any time on the Access page." There is no banner, icon or step counter.

**The grid (one component, used for both cards).**
- **Card.** A `SettingsCard`. The grid bleeds to the card's side edges (`-mx-6`) and its last row sits on the footer
  rule (`-mb-6`).
- **Columns.**
  - A sticky name column on the left, at least 10rem wide, that takes the spare width. Its head is a microheader
    ("Project", "Channel").
  - One tick column per role, 72px wide. The head is the role name (12px medium, centred, up to two lines, full
    name in a tooltip) over a mono count of the people who hold it, so a tick says how many people it lets in.
  - On Projects, a trailing **People** column, 96px wide.
  - When the grids sit stacked on one page, Public channels gets an empty trailing column of the same width so the
    role columns line up.
- **Lead row.** The first body row sits on the `bg-muted/40` strip of the Permission rows pattern: "Every project"
  in medium, with "Tick a role everywhere" under it in muted 12px.
  - Each role gets one box. It is ticked when every row has that role, shows a dash (indeterminate) when only some
    do, and is empty when none do.
  - A click ticks the role on every row shown, or clears it from every row if all of them are already ticked.
  - Under a filter, the row reads "Every project shown" and acts only on the matching rows.
- **Rows.** Hairline rows about 45px tall with the `bg-accent/40` hover. Each name is led by a mark:
  - a project gets its `ProjectMark` at 28px;
  - a channel gets a 28px `surface-2` tile with a muted `#`.
- **Names.** A name truncates on one line with the full name in its tooltip: at most 9rem below a 42rem card and
  14rem above. A muted mono line under the name flags a row that needs a look:
  - "Owners only" on a project with no role and no person;
  - "Hidden from 1 role" or "Hidden from 2 roles" on a channel.
- **Cells.** Each cell is the standard `Checkbox` (brand when ticked, the spring pop), centred in a 36px hit area.
  Its label reads "Developer in Storefront".
- **People cell.**
  - From a 42rem card, it shows up to three overlapping 24px avatars and a +N, with every name in a tooltip.
  - Below 42rem, it shows a mono count instead of the avatars.
  - Then comes a ghost `UserPlus` icon button with the tooltip "Add a person". It opens a 18rem popover: a "Find a
    person" field, a "People in Acme" microheader, and checkable menu rows (avatar, name, a check when added). Its
    foot reads "A person added here sees the project whatever roles they hold."
- **Width.** When the role columns don't fit, they scroll sideways under the sticky name column. That happens with
  10 roles at 768 and 1024px; 5 roles fit at 768px.
- **Long lists.**
  - Past 12 rows, the grid scrolls inside the card (`max-h: min(60vh, 40rem)`) with the head row sticky, so the role
    names stay in view over the thirtieth project.
  - Past 10 projects, a "Filter 30 projects" search field sits above the grid.

**Projects card.**
- Title "Projects". Description: "Tick the roles that belong in each project. Everyone with a ticked role sees the
  project and everything in it."
- The footer holds a read-back as a `SettingsStatus`: a `warning` "2 projects list nobody yet · only Owners see
  them", or a `success` "Every project lists someone".
- That read-back is a signal, not a gate. Finish works with projects left empty.

**Public channels card.**
- Title "Public channels". Description: "Every member sees a public channel. Untick a role to hide a channel from it.
  Private channels keep their own members."
- Ticks mean "can see", the same direction as on Projects, so every box starts ticked. Unticking writes a deny rule
  for that role on that channel.
- The lead row is "Every channel", with the same tri-state.
- The footer reads, in muted text, "Leave every box ticked to keep channels as they are.", with the `brand`
  **Finish setup** on the right.

**Saving and done.**
- Ticks are held on the page until Finish. Finish saves both grids in one request and shows its loading state.
- After Finish:
  - the toast "Access is set" appears, with "People see their projects now. Change it any time on the Access
    page.";
  - the sidebar's Set up access row goes;
  - the page turns into the Access page in place.
- The Access page:
  - Its title is "Access" and its meta reads "8 projects · 4 public channels · 5 roles".
  - It has `PageTabs` for Projects and Public channels. Public channels is a path tab, `/access/channels`.
  - Each card saves from its footer with `SettingsSaveBar` ("Changes apply to everyone once saved.").
  - The Projects card shows the `SettingsStatus` "Set up · by <name>, <when>" in its `aside`.
- Leaving the setup page with unsaved ticks asks "Discard changes?", as the Team person dialog does.

**What a non-Owner sees until it's done.**
- No project anywhere: the switcher, the board, docs and memories are all empty.
- The sidebar's Project section shows one muted row, an hourglass with "Projects being set up", in the Being set up
  style.
- The workspace home, and any page that needs a project, shows `EmptyState` with a `FolderLock` icon:
  - title: "Your projects are on their way";
  - message: "This update rebuilt permissions, and an Owner is now choosing who's in each project. Yours will show up
    here once they're done. Chat works as usual.";
  - action: an outline **Open chat** button.
- When the Owner finishes, the live push refreshes the member's project list and the state goes by itself.

**Motion.** Nothing new animates. Ticking is a dozens-a-minute action, so it keeps the checkbox's existing spring
pop. The page swap after Finish uses `PageTabs`' and the toast's existing motion.

**Runner-up, and how to swap.** Variant A puts the same two cards on two steps:
- The meta line reads "Step 1 of 2" or "Step 2 of 2".
- The Projects footer has the read-back and a `brand` "Continue to channels →".
- The Channels footer has a ghost Back and a `brand` Finish setup.
- To swap, render one card at a time behind a `?step=` value and move Finish to the second card. Nothing else
  changes. It lost because it splits the two groups, so an Owner can finish step 1 without ever seeing that a
  client can still read #backend.

**Rejected.** A row of role tags per project, with an "Every project" menu. With five roles the tags wrapped to two
lines at 1024px, and the bulk helper was hidden in a menu.

**Local files** (outside the repo):
- Screenshots, the reference, scores and notes: `~/Code/Images/permission-rework/05-setup/` (`README.md`). The pick
  is `2026-10-11-pick-b-{768,1024,1440}-{light,dark}.png`, the worst case (30 projects × 10 roles) is
  `…-pick-b-worst-30x10-*.png`, the member state is `…-member-waiting-*.png`, the Access page is
  `…-later-access-page-*.png`, and the runner-up is `…-runnerup-a-step{1,2}-1024-dark.png`.
- Prototype: local branch `proto/permission-setup`, route `/acme/proto/project-setup` (`?v=b` the pick, `?v=a` the
  runner-up, `?as=owner|member|settings`, `?data=worst`, `?seed=1`), components in `web/src/components/proto/setup/`.

