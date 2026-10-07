# Design language: the Mono Console

This file is the visual spec of record for the web app in `web/`. The token
values live in `web/src/index.css`; the component grammar (F1 to F7) lives in
`practices/react-guide.md`. When a screen and this file disagree, the screen
is the defect.

## The direction in one paragraph

The app looks like the deployment console it is: a strictly monochrome canvas,
near-black in dark mode and near-white in light mode, the two being true
mirror inversions of each other, with no accent color anywhere in the chrome.
Depth comes from a stepped surface stack (canvas, sidebar, card, popover) and
1px hairlines, never from blur shadows or a colored glow. Technical data (ids,
repositories, targets, counts, timestamps) is set in a monospace face. Code
and log surfaces are framed as terminal windows with a contained monochrome
glow. Color is reserved for status signal: a colored icon or dot next to plain
text, never a filled or bordered chip, with one exception: board cards carry
type and label as a low-opacity tinted pill (an owner decision). Everything else is black, white, and
gray. The whole thing is dense, precise, and quiet.

The reason behind the single loudest choice, no accent hue: once more than a
couple of hues are on screen, a tinted chip fill reads as washed out, and a UI
accent competes with status color for attention. Dropping the accent entirely, rather than swapping
it, is what keeps status color legible.

## Token quick reference

Dark is the default and the brand. Light is a true black and white inversion
of it, not a separate paper identity. The full set (surfaces, status hues,
shadows, motion) is in `web/src/index.css`; themed surfaces use tokens only,
never a hard-coded palette class.

| Token | Dark | Light | Role |
|---|---|---|---|
| `background` | `#050505` | `#ececec` | canvas |
| `surface-2` | `#0a0a0a` | `#f1f1f1` | sidebar, section breaks, canvas field |
| `card` | `#111111` | `#f6f6f6` | panels, rows, nodes |
| `popover` | `#161616` | `#fcfcfc` | menus, dialogs, sheets |
| `foreground` | `#f5f5f5` | `#0a0a0a` | primary text |
| `muted-foreground` | `#9a9a9a` | `#656565` | secondary text, meta lines |
| `primary` | `#f5f5f5` | `#0a0a0a` | near-white in dark, near-black in light; no hue |
| `primary-foreground` | `#0a0a0a` | `#f5f5f5` | ink on primary, inverted per theme |
| `success` | `#4ade80` | `#15803d` | health, success, and open pull requests |
| `warning` | `#fbbf24` | `#b45309` | in-flight states |
| `info` | `#5cc8f5` | `#0369a1` | open and informational states |
| `merged` | `#a371f7` | `#8250df` | merged pull requests only |
| `destructive` | `#f97066` | `#dc2626` | errors, danger zone, closed pull requests |
| `border` / `input` | `#262626` / `#333333` | `#d9d9d9` / `#cccccc` | hairlines |
| `ring` | `#f5f5f5` | `#0a0a0a` | focus and selection |
| `cell-line` | `foreground` at 10% | `foreground` at 12% | a field grid's cell edges (a bot embed's fields) |
| `cell-label` | `foreground` at 4% | `foreground` at 4% | a field grid's shaded label cell |

A token that a design needs and this table lacks is added to `index.css` and
to this table in the same change. A one-off class is drift.

## Type

- Inter Variable for everything: UI, body, headings, display. Headings use
  tight tracking (`-0.022em`) and `text-wrap: balance`; page titles are
  `font-semibold tracking-tight`.
- JetBrains Mono Variable for technical data: ids, repositories, targets,
  counts, timestamps, code, terminal output. Use `.technical` or `font-mono`.
- Both are bundled locally through `@fontsource-variable`. No CDN, because a
  font request to a third party is a render-blocking dependency the product
  does not control.
- No serif, no script, no display face. Two families is the whole type system.

## Shape and depth

- Radius: 6px for interactive elements (`rounded-md`), 8px for cards
  (`rounded-lg`), full pills (`rounded-full`) only for badges, chips, and
  avatars. Never a pill button or input; a pill reads as a tag, not a control.
- Depth: surface stepping plus 1px hairlines (`border`), and the contained
  shadows `shadow-card`, `shadow-elevated`, `shadow-overlay`. No blur glow on
  a product surface. The one glow is the terminal window's, token-driven
  through `color-mix(in oklab, var(--ring) ...)` and monochrome.
- Interactive text on `primary` is ink on white in dark mode and ink on black
  in light mode. It holds AA contrast without a hue.

## Motion

- One standard ease for state changes (`--ease-standard`) and `--ease-out`
  for entrances. Microinteractions run 150 to 250ms; a page transition never
  exceeds 400ms. Animate `transform` and `opacity` only; width and height
  trigger layout on every frame.
- Sidebar collapse is instant: a width swap, no layout animation.
- `prefers-reduced-motion` disables everything through the global block in
  `index.css`. No per-component override is needed; each new animation is
  checked to land in its correct final state under it.

## Canvas (topology)

React Flow reads the same tokens: a `surface-2` field, a faint
`muted-foreground` dot pattern, hairline edges, and `ring` for selection and
connection lines. Service nodes are `bg-card` cards with a status-colored
leading edge.

Traffic reads left to right as a sentence: hostname pill, gateway row,
service, inside one dashed hairline box per docker network with a mono
microheader (`network · <name>`). The gateway card is titled in plain words
("Cloudflare tunnel", "Reverse proxy") and lists one mono row per route,
`→ service:port (address:port)`, each row with its own handle on both sides.
Route wires are bare 1.5px bezier curves in `muted-foreground`; only
hand-drawn relation edges keep the dashed smoothstep and the label pill. The
hostname pill is the one `rounded-full` chip on the canvas. Nothing truncates:
pills and cards are `w-max` and grow to their text. No fills and no per-kind
accent color; only the status dot carries color.

## Do and don't

Do: strictly monochrome in both themes as true mirror inversions; mono for
technical data; hairline borders; status, type, and label rendered as a
colored icon or dot next to plain text; dense but controlled spacing.

Don't: any single accent hue on chrome (buttons, links, active states, focus
rings); a parchment or cream canvas; serif or script type; decorative
gradients; heavy shadows for co-planar depth; pill buttons or inputs; a filled
or tinted-background chip for status, type, or label anywhere but the board
card's own pill row (`pillClass` in `web/src/components/board/ticketTypeColor.tsx`).

## Decision ledger

| Decision | Why |
|---|---|
| Monochrome, dark first | A black and white exploration on the board read better than a tinted identity; adopted app-wide so every page shares one voice |
| No accent color anywhere | A UI accent competed with badge and status color for attention; dropped rather than swapped |
| Light mode as a true inversion | Both themes stay strictly monochrome instead of light becoming a second, paper-like identity |
| Ink-on-primary buttons | AA-safe and distinctive without a hue |
| Stepped surface stack | Depth by stepping and hairlines, not shadows, keeps dense screens readable |
| Inter plus JetBrains Mono only | A precise technical voice; two families is enough |
| 6px interactive radius, 8px cards | Soft but precise; pills stay badge-only so controls and tags never look alike |
| Terminal-window motif, neutral glow | Code, log, and hero surfaces read as consoles; a neutral glow no longer implies an accent |
| Status and label as icon or dot plus text | Readable in both themes; tinted fills washed out once several hues appeared together |
| Board cards: type and label as tinted pills | Re-tested on the card layout where the pill row sits alone under the title with the id opposite; the 15% tint with a 700/400 text shade stayed legible in both themes, so the board keeps pills while every other surface stays icon-or-dot |
| Permission levels as a segmented strip per domain, projects listed the same way | The owner found the trailing level dropdowns harder to read and set than the strip, where every rung up to the level fills and the whole list reads at a glance; Project access uses the same list so a role and a person read alike |
| Person dialog: a tab per workspace, changes held until Confirm | Several workspaces stacked in one scroll mixed their controls, and applying each change on the spot made the dialog change under the owner; tabs separate the workspaces and Confirm makes the edit one deliberate act |

## Pattern spec

The pattern layer above the tokens: list and table treatment, filter bars,
detail headers, empty and loading states, and the multi-step forms. New
surfaces implement against it rather than inventing a shape per page.

List and row. A row is a `border-b border-border` hairline division, not a
bordered card. Cards are reserved for units that are draggable or clickable as
a whole (the kanban `TicketCard`). Row hover is a
`bg-accent/40` background lift only, no shadow, no scale; lift and scale are
reserved for draggable cards. The primary field sits left in normal weight;
secondary and meta fields trail right in `muted-foreground`, and in mono with
`tabular-nums` whenever the value is a count, amount, id, or timestamp. Status
renders per the badge rule above. A page that needs bulk actions uses a left
checkbox column; no page invents its own selection affordance.

List pane. A page that edits one record beside its siblings (Docs, Memories)
is the app sidebar, a list, and the open record. From 1024px the list is
dragged wider or narrower by a handle on its right border (220 to 560px,
double-click resets to the 300px default, one width shared by every page,
kept in the browser), but never so wide that the open record drops below
32rem. The list heads with its title and a mono count, a ghost `+` icon button, and a
search field. Rows are about 56px with 12px sides and hairline dividers: a
13px medium title over a muted one-line preview, mono meta trailing right. A
row whose record already shows its details when open (Docs) is the title alone
at about 40px, with a muted state icon after it (a lock) and no meta. The
selected row is `bg-accent` with a 2px `muted-foreground` left edge, a hover
`bg-accent/40`, never a boxed border. A row's actions are one `…` menu (Lock,
Clone, then Delete in destructive after a separator), each item hidden without
its permission and the `…` gone when none is left; Docs adds Pin first, which
needs no permission and lifts the doc into a leading Pinned group kept in the
browser, and Move to folder after Lock, a submenu of the project's folders with
the current one checked. The menu takes the meta's place on hover and focus and
stays on the selected row.
Group labels are 11px uppercase mono over a hairline. Docs groups its rows by
folder below Pinned, the default folder first and the rest in creation order:
a folder row is a chevron, a muted lucide `Folder` (`FolderOpen` while open),
the name in the group-label style, and a muted mono count, and it toggles on
click, collapsed folders kept per browser and project. On hover and focus a
`+` (new doc in that folder, never on the default folder, which the header's
`+` already fills) and a `…` (Rename, then Delete, never on the default
folder) take the count's place. Doc rows lead with a muted `FileText`
icon aligned under the folder icon, and an empty open folder says "No docs" in
one muted line. A search shows matching docs inside their folders, hides the
folders without a match, and opens collapsed ones while it runs. The header
puts a ghost `FolderPlus` (New folder) beside the `+`, for writers only. No
folder carries a color. Below 1024px one pane
shows at a time. `ListDetailLayout` and `ListPaneRow` in
`web/src/components/listpane/` are the reference.

Inbox. One row per doc however many notifications it has: the title over a
muted summary of what happened ("created · 2 updates"), the newest time
trailing in mono, unread while any of them is. Docs outside their project's
default folder sit under a folder row built from the same `FolderToggle` as
the Docs pane's, its meta "4 docs · 7 updates", expanded until collapsed and
kept per browser; inside it a title drops a leading folder name. Every other
notification is its own row with the same trailing time, and all of them
interleave by newest activity.

Filter bar. A horizontal row of pill controls directly under the page header:
a search field, then filter pills (`h-9 rounded-md border border-border
bg-card px-3 text-sm`, with a chevron if it opens a popover), an `×`-removable
chip per active filter, and an optional saved-view or sort control trailing
right. The board's bar centres its search field with a labelled Create button
beside it, the filters on its left, and stacks the two under each other when
the board is narrow. `BoardFilterBar` and `FilterChipRow` in `web/src/components/board/` are
the reference; every list page converges on this shape. A filter popover is a
`bg-popover` panel with a checkable row list (icon, label, checkmark) anchored
below its trigger.

Detail page header. Back link (`font-mono text-xs text-muted-foreground
hover:text-foreground`), then a row of mono id chip plus status badge, then
the title, then a muted meta line, then a `border-b border-border` hairline
before the content. `DocDetail.tsx` and `TicketDetail.tsx` are the reference.
Only a page with a genuine single-record view gets this header. In-context
inspection that does not warrant leaving a list opens a right-anchored drawer
instead, or a centered dialog when the record is a short form of its own (the
Team's person dialog, its body scrolling between a fixed header and footer); a
page that works as a full detail view is not forced into a drawer.

Ticket page. The page runs the full width beside the sidebar, with no centred
cap: the Thread is a pane on the left, the body in the middle, and an 18rem
rail flush right. The pane is sticky and the viewport tall with the composer at
its foot, and its width is dragged like the list pane (handle on its right
edge, 288 to 640px, arrow keys, double-click back to its share of the page,
saved once on release, kept in the browser). The body text holds a readable
measure inside its card. Breakpoints follow the width of the page, not the
screen: the pane from 736px, the rail beside the body from 1120px, and below
them the Thread and then the rail drop under the body. A ticket opened inside
another page (the Inbox split view) keeps the Thread under the body. The rail
is a run of sections, each a mono uppercase microheader with its one action
trailing as a ghost `+`, compact rows, and a muted one-line sentence
when empty ("No bugs reported."): Properties, Plays, Development, Reviews,
Attachments, Links, Testing, Bugs, Trail. Plays is stage-bound, so it is
absent, not empty, when no play applies to the ticket's stage. Links leads with the Source group
when the ticket has one: a muted file icon and the doc's title as a link. A linked ticket is its
status icon and mono key only; hovering or focusing the key opens a hover
card with the title, two clamped lines of description, and the status, and a
remove `×` shows on the row's hover and focus. `TicketPageBody` is the
reference.

Empty and loading state. A centered icon in a `bg-muted/60` square (not a
circle), a title, an optional message, an optional action, inside a
dashed-border container; this is `EmptyState.tsx`. One muted icon, no
illustration. Loading is the existing spinner plus a label; no skeleton
screens. `EmptyState` is for a whole empty page; an empty list inside a card is
a single `EmptyRow` sentence where the rows would be.

Stat and summary row. Three or four values in a single row above a list,
label above value, value slightly larger, color-coded only when the metric is
inherently positive or negative (health counts). Only a page with a fleet or
health rollup worth reading at a glance gets one; a page with nothing to
summarise does not invent one.

Setup stepper. A multi-step setup inside the wizard shell is a vertical rail
of `DnsStep` rungs (hollow dot upcoming, filled dot active, check done), with
content straight on the canvas, never a card inside a card. One rung is open
at a time; a finished rung collapses to a one-line summary with a `Change`
ghost button. Choices are radio rows with hairline dividers
(`EntryPathChoice`); a form shows its two or three real decisions and folds
the rest under an `AdvancedFields` disclosure with defaults; a mono preview
line reads the form back (`app.example.com → http://web:80`). Motion is rail
led: the finished rung's rail draws down (260ms `--ease-out`, `scaleY` from
the top), then the next body rises in (200ms, 4px, with a 180ms delay only
when it follows a drawn rail). `DnsSetupStepper` is the reference.

Project wizard. The `/wizard/project/<step>` flow (info, repository, service,
environment when the scan found keys, reach, deploy branches, done) is a
horizontal progress row above the active step, not the vertical rail. The row
is an `ol` up to `max-w-3xl` of 20px nodes evenly spaced on a 1px connector;
segments up to the current step fill with `foreground` (a `scaleX` over 200ms),
the rest stay `border`. A done node is a check in the `success` token and is a
button back to that step only when revisiting has no side effect (never Info,
and none once the stack exists); the current node is a filled ring with
`aria-current="step"`; future nodes are hollow, muted, and disabled. Labels are
`text-[11px]` mono under each node from a 42rem container up; narrower, one line
under the row names the current step with its `n / total` counter. The step
content sits in a `max-w-xl` column beneath the title and slides 8px in the
direction of travel over 180ms. Every step ends with one footer: Back on the
left (Info leaves the wizard, Service returns to Repository until its stack
exists), then a ghost "Skip for now" where the step allows it, then the
primary action; Info's reads "Continue to <next step>". No new stepper chrome
beyond this row exists. The URL step is the whole navigation state.

Stack detail page. The header keeps the detail-page shape (back link, mono
slug, title, actions top right) and adds a facts grid: a mono microheader over
each value (Status, Image, Runner, Strategy, Hostnames, Repository, or Network
when no repository is attached); a compose stack runs several images, so it shows
Services, a count with how many are not running, in Image's place. Below it the page is the settings shell:
`SettingsSectionNav` driving a `/:section` path segment (Overview, Logs, Exposures, Branch deploys,
Deploy history, Danger zone), one or two `SettingsCard`s per section. The nav
is a scrolling top row below 1024px and a side column from it, on every
settings-style page (Your settings, Configuration, Project settings, Stack);
a nav with two jobs (Your settings' You and Instance settings)
labels each group with a mono microheader, and a group with nothing the
viewer may open shows no label. A card's action lives in its footer strip (`footer` prop: Rollback,
Expose, Add rule), never floating in the body, and a form that is not the
section's main job stays collapsed behind that footer button. Lists inside a
card are hairline rows in one bordered box, except permission and access rows
(Permission rows, below), which sit straight on the card or dialog surface.
Nothing nests a card inside a card. The Services card has no column headers: each row is the service name and
status over its image and a muted `container <name>` line, with how it is
reached trailing right in mono (public hostnames as links, then
`service:port` on the stack's network, then `host :port` when published on the
machine); its footer names the networks and when the runner last looked.

Permission rows. Wherever access is set (a role, a person's overrides and
Project access, an invitation, an automation's scopes) the domains sit in one
bordered list (`rounded-md border border-input`, hairline dividers), led by an
"Every domain" or "Every area" row on a `bg-muted/40` strip that sets them
all and selects no rung once the rows differ. Each domain row is the name on
the left, its extra verbs (Run, Clone, Thread) as small outline toggles, then
the level as a segmented strip of None, Read, Write, Delete where every rung
up to the chosen one fills, so it reads as "this much access". The role
editor splits its lists under Workspace (with the instance areas) and Every
project, from the catalog's area. Project access is the same list: an Every
project lead row (a muted icon, "Every project" over its one-line
consequence, and a From role / Chosen projects segmented pair), then one row
per project with the same level strip and an "Areas" toggle that opens that
project's area rows indented under it; a project whose areas differ selects
no rung and shows a mono summary under its name ("tickets Write · docs
Read"). Under From role the project rows are disabled. A viewer who can't
change a list sees it disabled. `PermissionLevels`, `PermissionLevelControl`,
and `ProjectAccessBlock` in `web/src/components/access/` are the reference;
deny overrides keep the checkbox grid (`PermissionGrid`).

Person dialog. The Team's person dialog holds one browser-style tab per
workspace the person is in: a row of tabs on a hairline that scrolls
sideways, the selected tab bordered on three sides and joined to the panel
below, no close button on a tab, and a ghost `+` after the last one that
opens a popover to pick a workspace and a role. A tab's panel is that
workspace's role, overrides, Project access, and "Remove from workspace".
Every change is held until Confirm: a tab with held changes shows a small
`foreground` dot, the footer is Cancel and Confirm (disabled with nothing
held), closing with held changes asks "Discard changes?", and a refused
change shows as a destructive line above the footer while the rest stays
held. Account actions (Disable, Remove account) sit at the footer's left and
apply at once. `TeamPersonDialog` and `TeamWorkspaceTabs` in
`web/src/components/team/` are the reference.

Bot message. A bot's post sits in the same bubble as a person's, under the
name and avatar it posted with (the Nexul glyph without one) and a `BOT`
outline tag, widened to 38rem when it carries an embed. Each embed hangs
behind a 2px `muted-foreground` rule; the sender's color is never painted.
Fields are a framed two-column grid, `cell-line` on every edge and the label
cell in `cell-label` at 40%. Past six fields, a long description, or two
embeds, one hairline fold bar opens the rest. Only a bot's text renders
Discord's markdown subset. `BotMessageRow` and `EmbedCard` in
`web/src/components/chat/` are the reference.

Logs view. The stack page's Logs section is a line tab row of the stack's
services over one terminal-style block per service: a mono timestamp column
and the raw line, wrapped lines indented under the text, dense rows, and a
copy button in the block's corner. Above it sit All / Errors, Pause, Copy and
Download, with one mono status marker at the end: Live while following, Paused
after the button, Scrolled up once scrolling released following, and the
connection's own state (reconnecting, ended) ahead of those. Errors means a
line that looks like one by its text, from either stream. A thin `destructive`
gutter marks every stderr line and is the only color in the view. A solid
Jump to live control, never a translucent one, resumes following.
`LogsView` in `web/src/components/logs/` is the reference.

Model choice. Wherever a model is picked, it is one frame holding the model
button, a hairline, then the model options button ("High · 1M"); a model with
no options shows the model button alone. The model button opens a searchable
list with a left rail (favourites, then one mark per provider), rows of name,
a mono New mark, and the provider line under it, and legacy models grouped
last under a mono microheader. The options button opens one section per
option the model supports, the harness default marked. Provider marks keep the
brand's own fills (a light and dark pair where the brand has one), an exception
to color being status signal. `ModelChoice` in `web/src/components/model/`
is the reference; a settings row puts it right of the label and description.

Tabs. A view whose cards or sections are separate jobs (two or more of them)
splits them into tabs instead of stacking them; a single-card view gets none,
and a left section nav stays as it is, the tabs live inside the section. Tabs
are `PageTabs` in `web/src/components/PageTabs.tsx`: a monochrome line tab row
on a hairline that scrolls sideways at narrow widths, the active tab the last
path segment (`/settings/connectors/github-app`) with the query left alone,
the first visible tab when the segment is missing or unknown. The first tab
has no segment of its own. A tab the viewer lacks permission for is hidden,
not disabled, and a view left with one tab drops the row. Anything that
deep-links into a tabbed view builds its path with `useTabPath`, never by
hand: `tabPath("versions")` names a tab and `tabPath()` the first one. A tab never opens blank: a section that renders
nothing when empty says so in an `EmptyRow` instead. A filter that narrows one
list (`ConnectorsSection`'s Connected and Not connected) is not a tab.

Doc questions. A doc with a clarification heads its page with a "Doc |
Questions N" segmented switch (an outline `ToggleGroup`, not `PageTabs`: the
two are views of one record, and the choice stays in the page, never the
path), N being the questions waiting in mono. Questions takes the article's
place in the same card: a "Questions" heading over one state line (the trail
icon, a medium label, a muted detail), the next action trailing right for
people who may close it, then one `QuestionSection` per round of numbered
`QuestionChecklistRow`s, the same checklist the Interview page uses. Only the
open round ends in its "Anything else?" box; an earlier round shows what was
written and its reply under its header, folded or not. The page opens on
Questions while some wait. `DocQuestionsPanel` in
`web/src/components/doc/clarification/` is the reference.

## Motion baseline

The two eases in `web/src/index.css` cover every role; no new duration or
easing token is added.

- Entrances and exits (list mount, filter results appearing or leaving, empty
  and loading states, drawer, panel, popover, dialog): `--ease-out`, 150 to
  200ms. Start from `opacity-0 translate-y-1` (4px), or `scale-[0.97]` for a
  popover with `transform-origin` at the trigger edge. Never `scale(0)`: a
  real object always has a visible shape. Exit about 20% faster than entrance.
- On-screen movement and state change (filter reflow while items stay
  visible, hover lift, a status value changing): `--ease-standard`, 120 to
  150ms.
- Stagger: a list entrance staggers at most 8 items at 20 to 30ms each, about
  200ms in total; beyond 8, the remaining rows appear together. Never stagger
  a virtualised or 50-plus-row list.
- Live updates (runner status, execution log, `HealthDot`): reuse the
  `status-pulse` keyframe in `index.css`; no second pulse. A status change
  gets one 150ms `--ease-standard` background cross-fade on the affected chip
  or dot, never a full-row re-entrance.
- Paired elements (overlay and dialog, drawer and backdrop, filter bar and
  result list) share identical duration and easing, or the pair reads as two
  events.
- The one exception to the 150 to 250ms rule is the phone-connected hero on
  the Devices tab, because it happens once per phone and the list has to
  answer too: the QR content crossfades to a check tile at 800ms `--ease-out`
  with a 4px blur and 0.98 scale; the new row enters Other devices with a 4px
  rise over 800ms `--ease-out`; and a `bg-accent` glow behind it fades out
  over 5600ms after an 800ms hold. Nothing else adopts these numbers.
