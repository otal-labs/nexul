# Design language: glass over a light field

This file is the visual spec of record for the web app in `web/`. The token
values live in `web/src/index.css`; the component grammar (F1 to F7) lives in
`practices/react-guide.md`. When a screen and this file disagree, the screen
is the defect.

## The direction in one paragraph

Content floats. The canvas is near-black in dark mode and a soft grey in light
mode, and behind everything sits a still, soft light field: an ember glow top right,
a cool blue glow bottom left, a hint of pink between. Every page's content
lives in a raised, frosted panel that floats 8px from the canvas edges and from
its neighbours; the sidebar sits straight on the canvas with no surface of its
own. Cards inside a panel are a step lighter with the same hairline ring. One
ember accent marks what you act on and where you are: the primary action,
focus, the active nav item, selection, your own messages, checked controls and
progress. Status keeps its own hues as a dot or icon beside plain text.
Technical data (ids, repositories, targets, counts, timestamps) is set in a
monospace face. Dense where the work is dense, calm everywhere else.

Why one accent and only those roles: an accent everywhere stops meaning
"here", and it competes with status colour; held to action, focus and
selection it reads as the app's voice while a red, amber or green dot still
reads as state (ADR 0133).

## Token quick reference

Dark is the default. Light is its own soft-grey identity with a pastel field
and white panels, not an inversion. The full set (surfaces, status hues,
shadows, motion) is in `web/src/index.css`; themed surfaces use tokens only,
never a hard-coded palette class.

| Token | Dark | Light | Role |
|---|---|---|---|
| `background` | `oklch(0.13 0.005 265)` | `oklch(0.95 0.004 265)` | canvas, behind the light field |
| `surface-2` | `oklch(0.155 0.005 265)` | `oklch(0.965 0.004 265)` | wells inside a panel: board columns, logs, code, the topology field |
| `panel` | `card` 60% into `background` | `card` | the frosted panel, painted at `--panel-opacity` (85%) |
| `card` | `oklch(0.21 0.005 265)` | white | cards and nodes, a step lighter than their panel |
| `popover` | `oklch(0.225 0.006 265)` | white | menus, dialogs, sheets |
| `foreground` | `oklch(0.97 0.003 265)` | `oklch(0.2 0.01 265)` | primary text |
| `muted-foreground` | `oklch(0.71 0.01 265)` | `oklch(0.48 0.01 265)` | secondary text, meta lines; 6:1 or better on every surface |
| `accent` | white at 7% | ink at 6% | hover and selected-row lift, reads on any surface |
| `primary` | `foreground` | `foreground` | strong ink for emphasis (`text-primary`); not the action colour |
| `brand` | `oklch(0.68 0.2 35)` | `oklch(0.565 0.19 35)` | the ember accent: primary button, focus ring, active nav marker, selection, own messages, checked controls, progress, prose links |
| `brand-foreground` | `oklch(0.16 0.03 35)` | white | ink on `brand`, 6.2:1 dark and 5:1 light |
| `success` | `#4ade80` | `#15803d` | health, success, and open pull requests |
| `warning` | `#fbbf24` | `#b45309` | in-flight states |
| `info` | `#5cc8f5` | `#0369a1` | open and informational states |
| `merged` | `#a371f7` | `#8250df` | merged pull requests only |
| `destructive` | `#f97066` | `#dc2626` | errors, danger zone, closed pull requests |
| `border` / `input` | white at 8% / 12% | ink at 10% / 16% | hairlines, the same on every surface |
| `ring` | `brand` | `brand` | focus and canvas selection |
| `panel-ring` / `panel-highlight` | white at 7% / 5% | ink at 8% / white at 90% | a panel's hairline and its inner top edge |
| `field-warm` / `field-pink` / `field-cool` | `brand` at 34%, pink, blue | `brand` at 30%, pastel pink, pastel blue | the light field's three glows; the warm one follows the palette's accent |
| `cell-line` | `foreground` at 10% | `foreground` at 12% | a field grid's cell edges (a bot embed's fields) |
| `cell-label` | `foreground` at 4% | `foreground` at 4% | a field grid's shaded label cell |

A token that a design needs and this table lacks is added to `index.css` and
to this table in the same change. A one-off class is drift.

Palettes (Appearance settings, `web/src/lib/themePalettes.ts`) override the
surface and text roles; a palette's `primary` becomes its `brand` (and so its
focus ring and the field's warm glow) unless it names a brand of its own, and
its panels follow its `card`. The default palette is labelled Nexul (id
`console`). Brutalism's zero radius applies to panels too.

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

- Radius: 7px for controls (`rounded-md`), 9px for cards (`rounded-lg`), 12px
  for panels (`rounded-xl`, the `panel` utility), full pills (`rounded-full`)
  only for badges, chips, and avatars. Never a pill button or input; a pill
  reads as a tag, not a control.
- Panels: the `panel` utility in `index.css`, applied once by the layout. A
  single page gets one panel for its whole content (`.app-frame` in
  `Layout.tsx`); a page built from panes marks its root `data-pane-layout`,
  which turns the frame off, and gives each pane its own `panel` with an 8px
  (`gap-2`) gutter. Never a panel inside a panel.
- Depth inside a panel: a card is a step lighter (`bg-card`) with the hairline
  `border`; a well is a step darker (`bg-surface-2`). The contained shadows
  `shadow-card`, `shadow-elevated`, `shadow-overlay` stay for cards, popovers
  and dialogs. No border where the surface step already separates.
- People without a photo get a seeded conic-gradient avatar
  (`web/src/lib/avatarGradient.ts`) with white initials; under 24px it shows
  one initial at 10px.

## Motion

Fast and fluid, never slow: 150 to 250ms for anything small, up to 400ms only
for the few moments that carry a gesture or a page. Only `transform`
(`translate`, `scale`) and `opacity` move; a box that has to change size snaps
in one layout step and its content does the moving.

- Curves, all in `web/src/index.css`: `--ease-out` (`cubic-bezier(0.16, 1,
  0.3, 1)`) for everything that enters; `--ease-standard` for state changes
  (hover, colour, a status); `--ease-spring`, a bounce-free spring written as
  a `linear()` curve and paired with 200ms, for anything that slides from one
  place to another; `--ease-spring-pop`, the one small-bounce spring (0.2,
  paired with 350ms), only on the checkbox mark. JavaScript motion uses the
  same numbers from `web/src/lib/motion.ts`. Exits run about 20% faster than
  entrances. Never `ease-in`, never from `scale(0)`.
- Frequency decides: keyboard moves (Enter, Space, arrows) and dozens-an-hour
  actions get no decoration. A page reached with a key, or with back and
  forward, does not replay its entrance, and a highlight moved by the arrows
  jumps instead of sliding.
- Panels and the sidebar never move. On a route change only the content
  inside the panels arrives.
- `transition-property` defaults to `none` (base layer), so a bare
  `duration-*` class never turns into `transition: all`; a transition names
  its properties.
- Sidebar collapse is instant: a width swap, no layout animation. The
  sidebar starts as the icon rail below 1024px.
- The light field is still: gradients only, no filter, no animation. Behind
  backdrop blur any motion re-blurs every panel each frame, which held 50ms
  idle frames under a 4x CPU throttle while the field drifted; nothing may
  animate behind the panels, and the app has no ambient motion.
- Reduced motion is gentler, not none. The global block in `index.css`
  flattens every CSS transition and keyframe to its end state; what tells the
  reader something arrived keeps a 150ms fade instead (page and list
  entrances, chat arrivals, the send, the done wash), and nothing travels,
  scales or loops.

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

Do: float content in panels and keep the sidebar on the canvas; keep the
accent to action, focus, active nav, selection, own messages, checked
controls and progress; status as a coloured dot or icon next to plain text;
mono for technical data; hairline rings that read on any surface; dense but
controlled spacing; check both modes, which are each designed, not inverted.

Don't: the accent on anything else in the chrome (headings, icons at rest,
borders, chips, badges); a second accent hue; a panel inside a panel, or a
border where a surface step already separates; tint a chip with the accent; a
filled or tinted-background chip for status (type and label may be tinted
pills, `pillClass` in `web/src/components/board/ticketTypeColor.tsx`); serif
or script type; pill buttons or inputs; heavy shadows for co-planar depth; a
second ambient animation or anything animating layout behind the panels.

## Decision ledger

| Decision | Why |
|---|---|
| Glass over a light field, dark first (ADR 0133) | The monochrome console read flat after every page was cleaned up; floating panels over a soft light field give depth and a recognisable look without decorating the content |
| One ember accent, held to action, focus, active nav, selection, own messages, checked controls and progress | Used everywhere an accent stops meaning "here" and fights status colour; held to these roles it is the app's voice |
| Panels at 85% with a 20px blur, not the mock's lower opacity | The field glows through the edges while body and muted text keep at least 6:1 on the panel |
| Light mode as its own identity | A soft grey canvas, pastel field and white panels read intentional; an inversion of the dark look did not |
| Ink on the dark accent, white on the light accent | White on the bright dark-mode ember is 3:1; dark ink holds 6:1 there, and the deeper light-mode ember holds 5:1 with white |
| Status as icon or dot plus text, never a tinted chip | Readable in both themes and keeps status from competing with the accent; tinted fills washed out once several hues appeared together |
| Board cards: type and label as tinted pills | The 15% tint with an 800/400 text shade holds 4.5:1 in both themes. Type and label may use the same pill wherever they show as a tag group; status never does |
| Board cards: the key as an eyebrow, the person opposite the pills | Against the avatar-beside-title card and a card with a ruled footer: with the 28px avatar gone from the title row the title wraps a line less, the key reads first the way people quote it, and the footer rule made every card taller for a line that spacing already separates |
| Stack status as one well split in three | Against three separate stat tiles: the tiles were three more boxes inside the panel and had no room to list the services, which are what the stack is |
| Deploy steps beside the log | Against the steps over the log: beside it the timeline stays in view while the log scrolls, and the log gets the panel's height |
| Doc body on a sheet, the ticket body open | Against an open doc body and the boxed card: the sheet with page margins makes the doc read as the thing being written; a ticket's body is short and sits beside its rail, where a box only framed the empty editing space |
| Board header: a project mark and the stage bar beside the title | Against a full-width stage strip and a stat row of stages under the header: both cost the board 50 to 80px of height on the page where height is cards; beside the title the summary is free and still reads first |
| Palettes theme the accent and the field | A palette's primary becomes its brand, so Ocean is blue and Grove is green everywhere the ember was, field included |
| Inter plus JetBrains Mono only | A precise technical voice; two families is enough |
| 7px controls, 9px cards, 12px panels | Soft but precise; pills stay badge-only so controls and tags never look alike |
| Terminal-window motif, neutral glow | Code, log, and hero surfaces read as consoles |
| Gradient avatars for people without a photo | A seeded gradient tells people apart at a glance where flat initials circles all looked the same |
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
title wraps to two lines at most, the full text in its tooltip, and sets its
own direction so an Arabic title reads right to left. A
row whose record already shows its details when open (Docs) is the title alone
at about 40px, with a muted state icon after it (a lock) and no meta. The
selected row is `bg-accent` with a 2px `brand` left edge, a hover
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
trailing in mono, unread while any of them is. Rows hover and select like the
list pane's. Docs outside their project's
default folder sit under a folder row built from the same `FolderToggle` as
the Docs pane's, its meta "4 docs · 7 updates", expanded until collapsed and
kept per browser; inside it a title drops a leading folder name. Every other
notification is its own row with the same trailing time, and all of them
interleave by newest activity.

Board. The header leads with the project mark (`ProjectMark`: the prefix in
mono on a gradient seeded from it, 44px, 9px radius, the same family as a
person's avatar) beside the title, a mono "55 tickets · 8 people" line under
it, and the stage summary as the header's action (`BoardStageSummary`): one
6px bar split by stage in the stage hues, backlog at half strength, over a
legend of dot, stage and mono count. It counts every ticket in the project,
whatever the filters show, and grows in like a progress fill. A card is the
mono key as an eyebrow (with the run timer and thread mark trailing on its
line), the title at full width, the blocked line, then the pills with the
20px avatar of whoever acts next opposite. A swimlane header is a leading
chevron, the lane name, and a 56px done bar beside the mono "4/11 tickets".
An empty column is a dashed slot saying "No tickets".

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

Page header. Every page opens with `PageHeader`: a breadcrumb of the
ancestors (`PageBreadcrumb`, mono `text-xs`, muted, each crumb a link; the
middle crumbs shrink and truncate first so the last stays readable), then the title left-aligned at `text-2xl font-semibold
tracking-tight` (`pageTitleClass`; never centered, never another size, an
editable title takes the same class), its actions top-aligned on the right,
one muted meta line under the title (status as a dot plus text, counts,
who and when), and a `border-b border-border` hairline closing the header.
Breadcrumbs replace back links everywhere; no page renders "← Back to …" or
an arrow icon to leave. A workspace page leads with the workspace crumb
(`useWorkspaceCrumb`), a project page adds the project (`useProjectCrumb`).
A list pane (Docs, Memories, Inbox) keeps its pane title bar instead of a page
header, and the open record beside it takes the page header with crumbs back
to its list and folder (Docs › Runbooks), no workspace crumb. Its body is
a sheet (`bg-card`, hairline ring, `shadow-card`, 48px side margins once the
page is 48rem wide) holding a reading measure (`max-w-3xl`), and the title
stays out of the sheet. From a 48rem page the table of contents (a 12rem
column) sits to its left.

Rich text (`.prose-rich` in `index.css`), the doc and ticket bodies alike:
16px at 1.75, headings closer to their text than to the paragraph above
(1.9em over, 0.45em under) and balanced, paragraphs `text-wrap: pretty`,
muted list markers. Tables are ruled data, not boxed cells: a hairline under
each row, a muted 13px header over a stronger rule, tabular figures. Images
take the large radius, a black or white 10% outline drawn inside, and the
card shadow. Code blocks stay consoles at 13px. A blockquote is a 2px rule in
`foreground` at 25%.

Page width. `Container` is `max-w-7xl` for lists and grids (Board, Runners,
Automations, Topology) and `size="page"` (`max-w-5xl`) for settings and
single-record pages (Configuration, Your settings, Project settings, Stack,
Deploy, Automation); every page pads `py-8` inside its panel. Pages built
from panes (Docs, Memories, Inbox, Ticket) run the full width with no
Container, one panel per pane; Chat is one conversation panel.

Microheader. The one small uppercase label is `microheaderClass`
(`font-mono text-[11px] font-medium tracking-[0.12em] uppercase
text-muted-foreground`): facts, group labels, rail sections, column heads.
No other size, weight, or tracking for an uppercase label.

Menu row. Every popover menu row, custom or a dropdown item, is `text-sm`
with `px-2 py-1.5 gap-2` and a muted 16px icon (`menuItemClass` in
`web/src/components/MenuItem.tsx`), in a `p-1` panel; the sidebar's nav rows
keep their own height at the same `text-sm`.

Detail page header. The page header above, with the record's mono id chip
and status in the meta line. Only a page with a genuine single-record view gets this header. In-context
inspection that does not warrant leaving a list opens a right-anchored drawer
instead, or a centered dialog when the record is a short form of its own (the
Team's person dialog, its body scrolling between a fixed header and footer); a
page that works as a full detail view is not forced into a drawer.

Ticket page. The page runs the full width beside the sidebar, with no centred
cap: the Thread is its own full-height panel on the left with the composer at
its foot, and the body is a second panel that scrolls on its own. Its header
spans the whole panel, so a long title reads across it, and under it sit the
body and a 19.5rem rail behind a hairline rule. The thread panel's width is dragged like the list pane
(handle in the gap on its right, 288 to 640px, arrow keys, double-click back to
its share of the page, saved once on release, kept in the browser). The body
sits open on the panel, no card, held to a 68ch measure; a card around it only
framed the empty editing space. Breakpoints follow widths, not
the screen: the two panels from 736px of page width, the rail beside the body
once the body panel is 52rem wide (under a hairline above it until then), and below 736px one panel holds the body,
then the Thread, then the rail. A ticket opened inside
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
a single `EmptyRow` sentence where the rows would be, `flush` when it sits in a
card body and lines up with the text around it.

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
segments up to the current step fill with `brand` (a `scaleX` growing from
the left, see the Motion baseline),
the rest stay `border`. A done node is a check in the `success` token and is a
button back to that step only when revisiting has no side effect (never Info,
and none once the stack exists); the current node is a filled ring with
`aria-current="step"` in `brand`; future nodes are hollow, muted, and disabled. Labels are
`text-xs` mono under each node from a 42rem container up; narrower, one line
under the row names the current step with its `n / total` counter. The step
content sits in a `max-w-xl` column beneath the title and slides 8px in the
direction of travel over 180ms. Every step ends with one footer: Back on the
left (Info leaves the wizard, Service returns to Repository until its stack
exists), then a ghost "Skip for now" where the step allows it, then the
primary action; Info's reads "Continue to <next step>". No new stepper chrome
beyond this row exists. The URL step is the whole navigation state.

Stack detail page. The header keeps the detail-page shape (breadcrumb of the
workspace and project, title, the mono slug in the meta line, actions top
right), then the live status (`StackLiveHero`): one `surface-2` well split in
three by hairlines from a 36rem page. Last deploy is the deploy's state in
20px semibold behind its `HealthDot` with the mono "last deploy 4h ago" under
it; Services is the mono count with how many are up or not running, over one
line per service (a status dot, the mono name, its state trailing, four at
most then "+N more"); Hostnames is the first as a link with an arrow, then
"+N more". Under the well a facts grid holds the rest: a mono microheader
over each value (Image for a run stack, Runner, Strategy, Repository, or
Network when no repository is attached). Below it the page is the settings shell:
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

Deploy page. The header names the outcome ("Deployed", "Deploy failed")
with the status, id, image and time in its meta line. From a 48rem page the
steps run down a 15rem column beside the log and stay in view while it
scrolls; narrower, they sit above it. Each step is a node on a rail: a filled
`success` check once done (the rail below it in `success` at 50%), a spinner
ringed in `info` while active, a filled `destructive` cross on failure, a
hollow ring while pending and a dashed one when skipped, the label, and the
mono duration trailing. The log names each phase above its first line
(11px uppercase mono, muted), so it reads against the steps.
`DeployProgressSection` is the reference; the instance upgrade uses the
same step list.

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
are `PageTabs` in `web/src/components/PageTabs.tsx`: a neutral line tab row
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

The primitives carry the numbers; a new surface reuses them instead of
writing its own.

- Page entrance (`usePageEntrance` in `Layout.tsx`, `enterPage` in
  `lib/motion.ts`): when the page (the path's first segment past the
  workspace) changes by a click, the blocks inside its panels rise 6px and
  fade in over 200ms `--ease-out`, the first three 30ms apart, the rest with
  the third. Blocks that mount while the page's data lands (within 400ms) rise
  as they arrive. A block holding an `EnterList` leaves the motion to its
  rows; a block that runs its own entrance keeps it. A tab, a settings
  section or another record inside the same page changes in place.
- Lists (`EnterList`): the rows on screen at mount rise 4px and fade over
  200ms, the first eight 25ms apart, the rest with the eighth; a list of 50 or
  more mounts at once. A row added later (a filter or search bringing it
  back) pops in from 0.97 over 150ms; `arrival="rise"` makes it rise 8px over
  200ms instead, for news (the Inbox). A row React only moved keeps still. A
  row opts out with `data-no-enter` (a board card mounted mid-drag).
- Sliding highlight (`ActiveIndicator`): the sidebar's active row, the
  settings section nav, the line tab row's underline (`TabUnderline`, on
  `PageTabs` and the logs tabs) and the Doc | Questions switch each own one
  highlight that slides to the active item, transform only, 200ms
  `--ease-spring`. It sizes to the item and plays back from wherever it was,
  so a click mid-slide carries on. Moving between the sidebar's two navs it
  fades out of one and into the other.
- Popovers, menus, hover cards and selects open in 150ms and close in 120ms
  from 0.97 and 4px toward their trigger; a dialog and its overlay open in
  200ms and close in 150ms; a sheet and its overlay in 250 and 200ms. Toasts
  keep the library's choreography at 250ms `--ease-out` (a stacked toast leaves
  in 200ms). A spinner waits 300ms before it shows.
- Disclosure (`.disclosure` with `data-closed`: swimlanes, question rounds;
  `AdvancedFields` does the same with the collapsible's enter and exit):
  opening, the box snaps open and the content fades in as it settles 4px over
  200ms; closing, the content fades out in 120ms and the box shuts after.
  Chevrons turn in 150ms `--ease-standard`.
- Hover and press: a draggable card lifts 1px with a soft elevated shadow
  (an opacity fade on a pseudo layer), 150ms, on hover-capable pointers only;
  a strip below the card keeps the vacated pixel inside it so the hover never
  flickers. A labelled button presses to 0.97 over 150ms `--ease-out`; an icon
  button answers with its background alone, 120ms. Row hover is a background
  lift only.
- Controls: the switch thumb slides on `--ease-spring`, 200ms. The checkbox
  mark pops in from 0.6 on `--ease-spring-pop` (350ms) and fades out in 150ms;
  a box that loads ticked shows still.
- Progress fills grow from the left with `scaleX`, 300ms `--ease-out` the
  first time they show, then follow a change in 250ms `--ease-standard`.
- Live updates (runner status, execution log, `HealthDot`): reuse the
  `status-pulse` keyframe in `index.css`; no second pulse. A status change
  gets one 150ms `--ease-standard` cross-fade on the affected chip or dot,
  never a full-row re-entrance.
- Chat: someone else's message, and the Agent's reply as its stream starts,
  rise 8px and fade in over 200ms (`arrive`); what was on screen when the
  conversation opened never animates. A confirmed message keeps its pending
  row's identity (`client_key`), so nothing replays when the server answers.
- Board: a ticket that lands in a done-stage column from a working one gets
  a success wash, the `success` hue at 15% fading out over 800ms (400ms under
  reduced motion).
- Paired elements (overlay and dialog, drawer and backdrop, filter bar and
  result list) share identical duration and easing, or the pair reads as two
  events.
- The one exception to the 150 to 250ms rule beyond the heroes below is the
  phone-connected moment on the Devices tab, because it happens once per
  phone and the list has to answer too: the QR content crossfades to a check
  tile at 800ms `--ease-out` with a 4px blur and 0.98 scale; the new row
  enters Other devices with a 4px rise over 800ms `--ease-out`; and a
  `bg-accent` glow behind it fades out over 5600ms after an 800ms hold.
  Nothing else adopts these numbers.

### Hero locks

Each was built as three live variants on the real surface, recorded at 1x
and 0.25x, and picked against the motion character above.

Board drop: decided 2026-10-08.
- Direction: Settle. Picked up, the card lifts to 1.03 with the elevated
  shadow in 150ms (no tilt). Dropped, dnd-kit's overlay glides onto the slot
  in 220ms `--ease-out`, coming down to 1 and losing its shadow on the way, and
  hands over to the card already sitting there in the same frame (the card
  never transitions its opacity, so the handover cannot blink).
- Reduced motion: no lift scale; the glide stays (the card has to reach its
  slot), the done wash stays as a fade.
- Rejected: Land (a tilted card that levels as it glides, then a 1.04 and
  -1.2° settle on a bouncy spring) took two stages and about 440ms and read as
  playful on the gesture the board is used for all day. Snap and ring (a 160ms
  snap, then a brand ring fading over 600ms) put the accent on something that
  isn't action, focus or selection, and the ring read as the card having
  focus.

Sending a chat message: decided 2026-10-08.
- Direction: Rise. Your message rises 12px into place over 240ms
  `--ease-out` while its bubble grows from 0.96 out of its bottom-right
  corner; nothing waits on it, the composer clears at once.
- Reduced motion: a 150ms fade.
- Rejected: composer to bubble (the bubble leaves from where the text was
  typed and travels to its place) crossed the whole panel, over 1000px at
  1440, on an action repeated dozens of times an hour. Slide and glow (a
  180ms slide, then a brand halo fading over 600ms) lingered on every message
  and animated a paint property.

Opening a ticket from the board: decided 2026-10-08.
- Direction: the standard page entrance; nothing extra.
- Rejected: title morph (a view transition carrying the card's title into
  the page title) cross-faded the whole page meanwhile, so the board and the
  ticket's two panels showed through each other for 200ms, and it froze input
  for the transition and depended on the ticket being cached. Panels grow
  (the ticket's panels scale from 0.98 out of the card's spot) moved the
  panels, which never move on a route change, and 0.98 was too small to read
  as coming from the card.
