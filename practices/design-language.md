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
the active nav item, selection, your own messages, checked controls and
progress. Focus is ink, not the ember. Status keeps its own hues as a dot or icon beside plain text.
Technical data (ids, repositories, targets, counts, timestamps) is set in a
monospace face, and page titles in a soft serif. Dense where the work is
dense, calm everywhere else.

Why one accent and only those roles: an accent everywhere stops meaning
"here", and it competes with status colour; held to action and selection it
reads as the app's voice while a red, amber or green dot still reads as state
(ADR 0133). Focus left the accent in round four: the ember sits a few degrees
from the destructive coral, and an ember ring on a field read as a field in
error.

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
| `brand` | `oklch(0.68 0.2 35)` | `oklch(0.565 0.19 35)` | the ember accent: primary button, active nav marker, selection, own messages, checked controls, progress, prose links |
| `brand-foreground` | `oklch(0.16 0.03 35)` | white | ink on `brand`, 6.2:1 dark and 5:1 light |
| `success` | `#4ade80` | `#15803d` | health, success, and open pull requests |
| `warning` | `#fbbf24` | `#b45309` | in-flight states |
| `info` | `#5cc8f5` | `#0369a1` | open and informational states |
| `merged` | `#a371f7` | `#8250df` | merged pull requests only |
| `destructive` | `#f97066` | `#dc2626` | errors, danger zone, closed pull requests |
| `destructive-fill` | `oklch(0.55 0.2 25)` | `#c81e1e` | the fill behind a destructive button's white label, deeper than the text hue so it never reads as the ember |
| `border` / `input` | white at 8% / 12% | ink at 10% / 16% | hairlines, the same on every surface |
| `ring` | `brand` | `brand` | canvas selection and the console glow |
| `focus` | `foreground` at 80% | `foreground` at 80% | the one focus indicator: a 2px outline, 2px out (1px on a field) |
| `panel-ring` / `panel-highlight` | white at 7% / 5% | ink at 8% / white at 90% | a panel's hairline and its inner top edge |
| `field-warm` / `field-pink` / `field-cool` | `brand` at 34%, pink, blue | `brand` at 30%, pastel pink, pastel blue | the light field's three glows; the warm one follows the palette's accent |
| `font-display` | Fraunces Variable | same | page titles, empty-state and showcase headlines, through `type-display` |
| `font-sans` | Inter Variable | same | everything else: UI, body, card and section headings |
| `font-mono` | JetBrains Mono Variable | same | technical data |

A token that a design needs and this table lacks is added to `index.css` and
to this table in the same change. A one-off class is drift.

Palettes (Appearance settings, `web/src/lib/themePalettes.ts`) override the
surface and text roles; a palette's `primary` becomes its `brand` (and so its
the field's warm glow) unless it names a brand of its own, and
its panels follow its `card`. The default palette is labelled Nexul (id
`console`). Brutalism's zero radius applies to panels too.

## Type

- Fraunces Variable is the display face: page titles, empty-state headlines
  and the showcase screens (signed out, onboarding, the error page), nothing
  else. The `type-display` utility sets it: weight 560, `"SOFT" 50, "WONK" 1`
  (soft terminals, the irregular alternates), optical sizing on, `-0.025em`
  tracking, `1.12` leading. A page title is 28px, 24px past 50 characters
  and 20px past 100 (`pageTitleClassFor`), and holds three lines
  (`ClampedTitle`): past them its last line fades out at the end and a
  muted "Show full title" opens the rest; editing the title opens it too. A
  headline takes `displayTitleClass` and its own size, never under 20px,
  where the soft serif turns muddy.
- Inter Variable for everything else: UI, body, card and section headings,
  dialog titles. Headings use tight tracking (`-0.022em`) and
  `text-wrap: balance`.
- JetBrains Mono Variable for technical data: ids, repositories, targets,
  counts, timestamps, code, terminal output. Use `.technical` or `font-mono`.
- All three are bundled locally through `@fontsource-variable` (Fraunces as
  its `full` build, the one that carries the SOFT and WONK axes). No CDN,
  because a font request to a third party is a render-blocking dependency the
  product does not control.
- A title in a script Fraunces lacks (Japanese, Arabic) falls back to the
  system serif for that script; the 1.12 leading, looser than the 1.08 a Latin
  serif wants, is what keeps two lines of Japanese from touching.
- No script face, and no fourth family.

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
  (`web/src/lib/avatarGradient.ts`) with white initials, your own profile's
  preview included; under 24px it shows
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
  The one exception is the live field of a showcase surface (Pattern spec),
  which replaces the frame instead of sitting behind it and runs only on the
  signed-out home and sign-in.
- Reduced motion is gentler, not none. The global block in `index.css`
  flattens every CSS transition and keyframe to its end state; what tells the
  reader something arrived keeps a 150ms fade instead (page and list
  entrances, chat arrivals, the send, the done wash), and nothing travels,
  scales or loops.

## Canvas (topology)

React Flow reads the same tokens: a `surface-2` well with a hairline ring for
the field, a faint `muted-foreground` dot pattern, hairline edges, and `ring`
for selection and connection lines. Every card on it is a `canvas-card`: the
card colour with the panel's hairline ring and lit top edge, no blur (nodes
move on every pan), lifting 2px on hover and outlined in `ring` when
selected. A service card is a 28px `surface-2` tile holding its runtime icon,
the name, and the status (icon and word) trailing on the same line, over a
rule and one mono line per fact (machine and strategy, replicas, volume); a
gateway card has the same header with its kind and name. Status colours only
the status; the cards carry no coloured edge.

Traffic reads left to right as a sentence: hostname pill, gateway row,
service, inside one box per docker network: `foreground` at 2.5% over the
field with a hairline ring and the large radius, a mono microheader
(`network · <name>`) inside its top edge. The gateway card is titled in plain
words ("Cloudflare tunnel", "Reverse proxy") and lists one mono row per route,
`→ service:port (address:port)`, each row with its own handle on both sides.
A hostname whose gateway is not on the canvas sits left of its service's
network box, level with the service, and its wire carries the port.
Route wires are bare 1.5px bezier curves in `muted-foreground`; only
hand-drawn relation edges keep the dashed smoothstep and the label pill. The
hostname pill is the one `rounded-full` chip on the canvas. Nothing truncates:
pills and cards are `w-max` and grow to their text. Nothing overlaps: a stored
arrangement whose cards or boxes come within 12px of each other is laid out
again. The controls sit bottom left as one `canvas-card` strip; there is no
minimap (at the canvas's size it covered a fifth of the map and showed grey
blocks) and no library attribution.

## Do and don't

Do: float content in panels and keep the sidebar on the canvas; keep the
accent to action, active nav, selection, own messages, checked
controls and progress; status as a coloured dot or icon next to plain text;
mono for technical data; hairline rings that read on any surface; dense but
controlled spacing; check both modes, which are each designed, not inverted.

Don't: the accent on anything else in the chrome (headings, icons at rest,
borders, chips, badges); a second accent hue; a panel inside a panel, or a
border where a surface step already separates; tint a chip with the accent; a
filled or tinted-background chip for status (type and label may be tinted
pills, `pillClass` in `web/src/components/board/ticketTypeColor.tsx`); the
serif anywhere but a page title or a showcase or empty-state headline; script
type; pill buttons or inputs; heavy shadows for co-planar depth; a
second ambient animation or anything animating layout behind the panels.

## Decision ledger

| Decision | Why |
|---|---|
| Glass over a light field, dark first (ADR 0133) | The monochrome console read flat after every page was cleaned up; floating panels over a soft light field give depth and a recognisable look without decorating the content |
| One ember accent, held to action, active nav, selection, own messages, checked controls and progress | Used everywhere an accent stops meaning "here" and fights status colour; held to these roles it is the app's voice |
| Panels at 85% with a 20px blur, not the mock's lower opacity | The field glows through the edges while body and muted text keep at least 6:1 on the panel |
| Light mode as its own identity | A soft grey canvas, pastel field and white panels read intentional; an inversion of the dark look did not |
| Ink on the dark accent, white on the light accent | White on the bright dark-mode ember is 3:1; dark ink holds 6:1 there, and the deeper light-mode ember holds 5:1 with white |
| Status as icon or dot plus text, never a tinted chip | Readable in both themes and keeps status from competing with the accent; tinted fills washed out once several hues appeared together |
| Board cards: type and label as tinted pills | The 15% tint with an 800/400 text shade holds 4.5:1 in both themes. Type and label may use the same pill wherever they show as a tag group; status never does |
| Board cards: the key as an eyebrow, the person opposite the pills | Against the avatar-beside-title card and a card with a ruled footer: with the 28px avatar gone from the title row the title wraps a line less, the key reads first the way people quote it, and the footer rule made every card taller for a line that spacing already separates |
| Others' chat messages as plain text, yours as the bubble | Against bubbles for everyone: a channel of bubbles was a column of boxes on a panel; plain text under the name reads as a conversation and leaves the ember bubble to mean "you" |
| Chat in a 48rem column | Against the full panel width: at 1440 your reply sat over 1000px from the message it answered |
| Stack status as one well split in three | Against three separate stat tiles: the tiles were three more boxes inside the panel and had no room to list the services, which are what the stack is |
| Deploy steps beside the log | Against the steps over the log: beside it the timeline stays in view while the log scrolls, and the log gets the panel's height |
| Doc body on a sheet, the ticket body open | Against an open doc body and the boxed card: the sheet with page margins makes the doc read as the thing being written; a ticket's body is short and sits beside its rail, where a box only framed the empty editing space |
| Board header: a project mark and the stage bar beside the title | Against a full-width stage strip and a stat row of stages under the header: both cost the board 50 to 80px of height on the page where height is cards; beside the title the summary is free and still reads first |
| Palettes theme the accent and the field | A palette's primary becomes its brand, so Ocean is blue and Grove is green everywhere the ember was, field included |
| Fraunces for titles over Inter and JetBrains Mono | A serif display face over a neutral sans is contrast, not resemblance: the titles get a voice no other developer tool has while every control, row and number stays in the precise technical pair. Soft and a little wonky (SOFT 50, WONK 1) at 560, because a sharp high-contrast serif read as editorial and a thin one vanished at 21px |
| 7px controls, 9px cards, 12px panels | Soft but precise; pills stay badge-only so controls and tags never look alike |
| Terminal-window motif, neutral glow | Code, log, and hero surfaces read as consoles |
| The live light field on showcase surfaces only | Picked over a fluted-glass refraction (busy vertical bands fought the text and the frosted vocabulary, and no CSS still could stand in for it) and an aurora (a band across the top only, nearly invisible in light mode); the field is the app's own light field moving, so its fallback still is exact and every palette retints it |
| The live field on the signed-out home and sign-in only | It repaints every frame while the page is open and the library's frame cap is out of reach from React; a wizard or an invitation stays open for minutes of form work, so there the still does the job at under 1% CPU |
| Empty states: the orbit mark | Picked over a placeholder card grid (generic, implied an add action the callers do not have, and looped) and a tile of the live field (a GPU canvas inside an everyday panel, grainy at 96px, impossible at compact size); the static orbit carries the field's colours at any size and costs nothing |
| Loading: the orbit at spinner size | Over the plain spinner and a gradient arc; it ties loading to the empty mark, keeps the ember to progress, and stays one small SVG |
| Long page titles step down to 24 and 20px and clamp at three lines | Against stepping alone (a sentence-long ticket title still ran six lines at 20px and pushed the body below the fold, and nothing bounds a title's length) and clamping at 28px alone (three lines held about half as many words); the step keeps short titles at full voice and the clamp bounds the rest |
| Bot embeds as status-edged cards with facts | Against a header strip over hairline key and value rows (scanned well for long values but read as the old table again, nine summary rows tall) and a 2px status bar across the top (the same card, but the bar read as the accent's decoration rather than state); the leading edge is the topology node's status signal reused |
| List pane placeholder: icon, count and the palette shortcut, compact | Against the icon alone (said nothing about the list) and the full empty-state size with a Fraunces headline (read as an empty page beside a full list) |
| A doc's first heading that repeats its title stays | The title and the body are separate fields; hiding a matching heading in a collaborative editor would put the caret in invisible text and show readers and writers different docs, so the author's content is shown as written |
| Runners: one card per machine, actions in its header, dashed when offline | Against a facts grid band per machine (a second header's worth of height, truncated the host and stack root, and hid the actions in a menu) and one table with machine group rows (aligned, but machines stopped reading as units and an offline machine looked like any other); greying a whole row to 60% made offline runners hard to read, a dashed unlit card says offline at full contrast |
| Topology cards: an icon tile, the name and status on one line, facts under a rule | Against a dot-and-mono-lines card (compact, but the status hue sat on a 6px dot and the word was a muted mono line) and a title strip with a coloured leading edge (a third band of height and colour on every card); the tile and trailing status echo the runner cards, and network boxes became faint filled regions instead of dashed outlines, which read as unfinished next to the lit cards |
| Focus as one ink outline, not the ember | The ember is a few degrees from the destructive coral: an autofocused field in the project wizard read as a validation error before anything was typed. Ink at 80% reads as focus on every palette and keeps 3:1 on every surface |
| Overlays on transitions, instant from a key | Keyframes restart when an overlay reopens mid-close; a transition turns back from where it is. Opened tens of times an hour, a menu or dialog driven from the keyboard must not wait on motion |
| Gradient avatars for people without a photo | A seeded gradient tells people apart at a glance where flat initials circles all looked the same |
| Sidebar: places before conversations, one scroll | With the channels first, Board and the project's pages sat below the fold at 860px and the docked workspace pane took a sixth of the height; with fixed-length pages first and the workspace section in the same scroll, every page is visible at a glance and the variable lists grow downwards |
| Permission levels as a segmented strip per domain, projects listed the same way | The owner found the trailing level dropdowns harder to read and set than the strip, where every rung up to the level fills and the whole list reads at a glance; Project access uses the same list so a role and a person read alike |
| Settings cards: a quiet header, settings as rows | Against the old header band (an 18px title over a full-width rule, so every section read as the same generic form) and the title outside the card (heading on the glass, content in the card: two surfaces for one unit, and a paired card's heading wrapped out of line with its neighbour's); a 15px title flowing into its rows keeps the unit whole and saves the rule |
| Settings save from a strip that is always there | Against a strip that opens when something changes (it pushed every card below it 52px on the first keystroke), a bar floating over the page (detached from the card it saves and covering the next one) and a Save beside the field (fits one field, not a card of them) |
| Save answers in its button, not a toast | Against the toast (it lands a panel away from the click) and a Saved line at the strip's left (opposite the pointer); the button the pointer is on turns into Saved |
| Theme and mode tiles are the app in miniature | Against swatch dots, which named a palette without showing what it changes; the miniature shows the canvas, the panel, text and the accent in that palette and mode |
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
renders per the badge rule above, in a fixed-width column when it leads the
row, so the field after it never shifts between Success and Failure. A page
that needs bulk actions uses a left
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
Group labels are 11px uppercase mono over a hairline. Before a row is picked, the open pane says
so with the compact orbit around the list's own icon (the one its sidebar
link uses), "Select a doc", a muted count of what the list holds ("27 docs
in Atlas Platform"), and, where the command palette finds these records,
its shortcut ("Ctrl K to search"); `ListDetailPlaceholder` is the
reference. Docs groups its rows by
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

Inbox. One row per doc however many notifications it has: the title with
the newest time trailing in mono on its line, over a muted summary of what
happened ("created · 2 updates"), unread while any of them is. Rows hover and select like the
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
whatever the filters show, and moves as the Motion baseline's Board entry says. A card is the
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
middle crumbs shrink and truncate first so the last stays readable), then the title left-aligned in the display face at 28px
(`pageTitleClassFor`, which steps a long title down to 24 or 20px; never
centered, an editable title takes the same class, and `ClampedTitle` holds
either to three lines), its actions top-aligned on the right,
one muted meta line 8px under the title (status as a dot plus text, counts,
who and when), and a `border-b border-border` hairline closing the header.
A mark that names the record (the board's project mark) goes in the
`leading` slot, centred on the title and meta pair. Breadcrumbs replace back links everywhere; no page renders "← Back to …" or
an arrow icon to leave. A workspace page leads with the workspace crumb
(`useWorkspaceCrumb`), a project page adds the project (`useProjectCrumb`).
A list pane (Docs, Memories, Inbox) keeps its pane title bar instead of a page
header, and the open record beside it takes the page header with crumbs back
to its list and folder (Docs › Runbooks), no workspace crumb. Its body is
a sheet (`bg-card`, hairline ring, `shadow-card`, 48px side margins once the
page is 48rem wide) holding a reading measure (`max-w-3xl`), and the title
stays out of the sheet. From a 48rem page the table of contents (a 12rem
column) sits to its left. A memory's body takes the same sheet.

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

Menu. Every menu, custom on a popover or a dropdown, is one anatomy: a
`glass-menu` panel (the frosted recipe on `popover` at 88%, the elevated
shadow) at 9px with 4px padding; rows of at least 32px in `text-sm` with
`px-2 gap-2.5` on a 5px corner, concentric in the panel's 9px, a muted 16px
icon column, the label, then a trailing mono hint (`DropdownMenuShortcut`)
or a check (`menuItemClass` in `web/src/components/MenuItem.tsx`); groups
split by an edge-to-edge hairline (`MenuSeparator`) and headed by a
microheader (`DropdownMenuLabel`). The highlighted row is `bg-accent`, never
an outline. A destructive row is `destructive` text and icon over a
`destructive` 10% hover, after a rule, last. Selects, popovers and hover
cards take the same panel. The sidebar's nav rows keep their own height at
the same `text-sm`.

Dialog. A `glass-popover` panel (the palette's surface) at 12px over a
scrim in the canvas colour, not black, at the Appearance slider's opacity
with a 4px blur, so a light page dims to a soft wash instead of grey. Up to
`100dvh - 4rem` tall and 32rem wide by default; it is a header (the 16px
semibold title, a muted line, a 28px ghost close top right), a body that
scrolls on its own (`DialogBody`), and a footer: Cancel then the primary
action on the right, on a tray (`surface-2` 60% into `popover`, a hairline
above) that stays in view. A dialog that stacks its fields without a body
scrolls as one with the footer pinned. A form dialog wider than its fields
(New automation's scope rows) takes 36rem rather than truncating them.

Sheet. The same frosted panel floating 8px from its edge at 12px, like the
page panels; its header is the grip, and dragging it toward the edge moves
the sheet 1:1 with the scrim fading in step (Motion baseline).

Tooltip. A small `glass-menu` label (`text-xs`, `px-2 py-1`, 7px) for an
icon-only control; the first waits 500ms, the next shows at once
(`TooltipProvider` in `App.tsx`).

Toast. The menus' glass at 9px, the status hue on the icon only, the action
in `brand`; bottom right, stacked by the library.

Focus. One indicator everywhere: a 2px `focus` outline (ink at 80%), 2px
out, 1px on a field, which keeps its border; a component's own ring utility
is cleared on focus. Errors keep `destructive` on the border and message,
so a focused field in error is an ink ring around a red border. A
destructive button is white on `destructive-fill`.

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

Empty and loading state. `EmptyState` is the orbit mark (the muted icon on a
card-coloured disc, ringed by two thin orbits stroked from the field's blue
through pink to `brand`, an ember dot top right and a blue one bottom left),
then a Fraunces headline at 26px, one plain muted line, and the action. No
border around it: an empty page reads as a first-run screen, not a
placeholder box. The `compact` size (list panes, sub-sections) keeps the same
mark at 56px with a 14px Inter title, because Fraunces never goes under 20px.
The mark is SVG on every surface and still once it has arrived (its entrance
is in the Motion baseline); the live field never runs inside a panel. `NoDataDisplay` passes an `icon` through for the glyph that names the
page. Loading is the same orbit at 16px, the ember dot circling a faint ring
(1.2s a turn, still under reduced motion), plus a label, held back 300ms; no
skeleton screens. `EmptyState` is for a whole empty page; an empty list inside
a card is a single `EmptyRow` sentence where the rows would be, `flush` when it
sits in a card body and lines up with the text around it.

Showcase surfaces. The signed-out pages (home, sign in, invitation), the
wizard frames and the error page sit on `ShowcaseSurface`: the light field,
live where a page opts in, the same ember top right, pink between and blue bottom left,
drifting slowly through a flow field with a little grain (`ShowcaseField`,
built on the shaders library and coloured from the tokens at runtime, so a
palette retints it). Signed out it fills the screen; signed in it takes the
frame's place as one opaque rounded surface (`data-pane-layout`), so nothing
moves behind a backdrop blur. The live layer repaints every frame for as long
as the page is open (about half a CPU core in a headless measurement, against
under 1% for the still), so it runs only where people land, the signed-out
home and sign-in, opted into with `live`; the invitation, the wizards and the
error page show the still, which every caller gets by default. Home, sign-in
and invitation are at full strength; the wizards and the error page are
`quiet`, at about half, so form text sits on near-plain canvas. The still,
the light field's own three radial gradients, is the container background: it
shows while the shader compiles, without WebGPU, and as the whole
reduced-motion variant (the shader is never mounted then); the live layer
fades in over 700ms once ready, and pauses with the browser's animation
frames in a hidden tab and offscreen. The sign-in and invitation cards are a
`panel` at 72%; over sign-in's live field that is the one place a blur sits
over motion, for a single small card. Headlines on these surfaces
are Fraunces. Never on an everyday screen, never behind a panel.

Stat and summary row. Three or four values in a single row above a list, in
the same `surface-2` well as a stack's live status, split by hairlines,
microheader label above a 20px mono value, color-coded only when the metric is
inherently positive or negative (health counts). Only a page with a fleet or
health rollup worth reading at a glance gets one; a page with nothing to
summarise does not invent one.

Runners. Under the fleet well (Online "2 of 3", Busy, Offline, Queued), one
card per machine: a 36px server mark on `surface-2` with a status dot on its
corner (`success` when every runner is connected, `warning` when some are,
muted when none), the name with its state in words beside it ("online",
"2 of 3 online", "offline"), and its actions on the same line as ghost buttons
(Import, Add runner) that drop to icons with a tooltip in a narrow card. Under
them a mono facts line (host, runner count, last seen, the editable stack root),
each fact carrying its own leading dot under a clip so no line starts with one.
Runner rows share fixed columns across every card, the connection dot under
the machine's mark: the name (or `runner <short id>` when it only repeats the
machine's), the version under it, the mono last-seen time, then what it is
doing (`idle`, a pulsing `info` dot and the job, or `offline`), then remove. A
machine with nothing connected keeps full-contrast text but loses the card
fill and shadow and draws a dashed edge, so it reads as unplugged rather than
faded. `MachineGroup` and `RunnerRow` in `web/src/components/runner/` are the
reference.

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

Settings card. `SettingsCard` opens with its title in 15px semibold and an
optional muted description, no rule under them, then the body and the
optional footer strip (`bg-muted/30` behind a hairline). The card's state, or
one quiet action, sits top right beside the title (`aside`: Up to date,
Enabled, Registered). Inside, settings are rows (`SettingsRow` in
`SettingsRows`): the label in 14px medium over a muted one-line description
on the left, the control on the right once the card is 34rem wide and under
the text below that; a rule above the first row and between rows. A list
inside a card keeps the bordered box of hairline rows. State is a
`SettingsStatus`, a 6px status dot beside plain text with an optional muted
detail (Connected · by Alice 20d ago, Active · last used 9d ago, Revoked 8 Oct),
never a chip; an active account says nothing, only one that can't sign in
(Disabled, Removed). Destructive actions that are not the card's job are ghost
buttons that turn `destructive` on hover, kept apart from the primary action,
and ask before they act. A card that edits values saves from its footer with
`SettingsSaveBar`: the strip is always there; Save is an outline button until
something changes, then Discard and "Unsaved changes" wake and Save turns
`brand`; once a save lands Save reads Saved with a check for 1.4s. Settings
that apply the moment they are picked (Appearance) have no Save. Copying uses
`CopyButton`; a value to copy sits in a mono `surface-2` well with an icon
button. The page header's meta line carries the open section's facts
(`SettingsHeaderMeta`: Nexul theme · Dark, 2 gateways · 3 hostnames) and marks
instance sections Instance-wide.

Appearance. Mode and Theme are radio groups of tiles, each a miniature of the
app (`ThemeMiniature`: canvas, sidebar with its brand marker, a panel with a
title, two text lines, a brand button and an outline one) drawn from the
palette's own roles in the mode being shown (`themePreviewColors`); System is
light with the dark miniature clipped across its right side. The picked tile
gets a 2px `brand` ring and a check badge top right; arrow keys move the pick.
"Dim behind dialogs" is a row with a miniature of a dialog over the scrim at
the slider's strength.

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

Chat stream. The messages run in one centred column up to 48rem wide, the
composer under it at the same width, so your bubble on the right never sits a
panel away from the line it answers. Your own messages are the one bubble
(`brand`); everyone else's text runs plain under a 32px avatar and a header
of the name in 13px semibold and the mono clock time (the full time on hover),
held to 72ch, so a long thread reads as a transcript instead of a stack of
boxes. A run of one person's messages drops the avatar and header after the
first and keeps 6px between messages. Each day opens with a divider: the
microheader label ("Today", "Yesterday", "Mon 21 Sept", the year only when it
is not this one) between two hairlines. The composer is one framed field
(`bg-card`, `rounded-xl`, the `input` ring turning to the focus ring) holding
attach, the text and emoji. `MessageList` and `ChatComposer` are the
reference.

Inbox row. A 28px `bg-muted/60` tile with a muted icon for what happened
(assigned, mentioned, status, doc created or edited, questions, memory, a
play run), a `foreground` dot on its corner while unread (it shrinks to 0.6
as it fades over 150ms when the row is read), then the title with
the mono time on its first line and the summary under it.

Bot message. A bot's post sits plain like a person's, under the
name and avatar it posted with (the Nexul glyph without one) and a `BOT`
outline tag, widened to 38rem when it carries an embed. Each embed is a card
(`bg-card`, hairline ring, `shadow-card`). The server drops the sender's
color, so the card's state is read from its title's words (`embedTone`:
failed, healthy, stopped, running and their kin, worst first) and shows as a
3px leading edge and an icon before the title in the status hue, the way a
topology node shows its status; a title that names no state leaves the card
neutral. An author line that only repeats the bot's name is dropped. Fields
are facts: a mono microheader over the value, inline fields three across
(two in a narrow pane), a value past 24 characters across two, a field the
sender did not mark inline across the row, and a value that is a state word
or two gets its status dot. The footer text and time sit in a hairline strip
at the card's foot in mono 11px, with the ghost fold toggle ("Show 3 more
fields", its chevron turning) on its right, past six fields or a long
description; past two embeds the same toggle opens the rest. Only a bot's
text renders Discord's markdown subset. `BotMessageRow` and `EmbedCard` in
`web/src/components/chat/` are the reference.

Deploy page. The header names the outcome ("Deployed", "Deploy failed")
with the status, id, image and time in its meta line. From a 48rem page the
steps run down a 15rem column beside the log and stay in view while it
scrolls; narrower, they sit above it. Each step is a node on a rail: a filled
`success` check once done (the rail below it in `success` at 50%), a spinner
ringed in `info` while active, a filled `destructive` cross on failure, a
hollow ring while pending and a dashed one when skipped, the label, and the
mono duration trailing. The log names each phase above its first line
(11px uppercase mono, muted), so it reads against the steps. The steps move
rail-led (Motion baseline). `DeployProgressSection` is the reference; the instance upgrade uses the
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

Sidebar. Straight on the canvas, never a panel. Top to bottom: the logo row,
the workspace switcher (a 32px initial tile), then one scroll: Search and
Inbox, the project (its switcher leads with the project mark at 28px, then
its pages), the foldable Workspace section, and last the conversations:
Channels, Voice channels, Direct messages and Threads. Places come before
conversations because their length is fixed, so a long channel list never
pushes Board or Runners below the fold. Section labels are microheaders with
their create `+` trailing; a direct message with one other person leads with
their avatar, a group with the people icon. Counts are `UnreadBadge`s. The
account row sits under the scroll on a hairline. The icon rail (below 1024px
or collapsed) keeps every page in the same order, with hairlines between the
groups, and stands one Chat link with a dot for unread messages in for the
conversation lists.

Command palette. ⌘K (Ctrl+K elsewhere), or the sidebar's Search row, opens
one palette from any signed-in page: a frosted overlay (`glass-popover`) 40rem
wide, hung at 14% of the viewport so a changing result count only moves its
bottom edge, over the dialog family's scrim. Opened from the Search row it
runs the dialog clock (220ms in, rising 6px from 0.97) and an overlay click
closes it in 150ms; opened or closed with a key, or closed by a pick, it shows and goes in
the same frame, since it is driven a dozen times an hour and a fading scrim
would swallow the next click. A row or group that joins the list after it
opened (a search landing, a letter that matches more) fades in over 120ms,
opacity only. A combobox input heads it, grouped results below, a footer of
key hints at its foot. Empty, it leads with what changed lately in the
current project, then every page the sidebar reaches under the same
permissions, every project's board, the create dialogs that already exist,
the other workspaces and the light or dark switch; typed, pages and boards
filter by every word alongside tickets and docs from the server's search
and memories by title, each group ordered by its best match (a label that
starts with the query first), and the palettes join the theme group. Group heads are
microheaders; a row is 36px with a muted icon, the label, a mono hint
(ticket key, prefix, status) and an enter mark on the active row, which takes
the sidebar's selection marks: `bg-accent`, a 2px `brand` edge and a `brand`
icon. The highlight jumps, never slides: the palette is driven a dozen times
an hour. Arrows wrap, Enter opens, Escape closes and hands focus back to
where it was; the count is announced once a search settles. `CommandPalette` in `web/src/components/command/` is the
reference.

## Motion baseline

The primitives carry the numbers; a new surface reuses them instead of
writing its own.

- Page entrance (`usePageEntrance` in `Layout.tsx`, `enterPage` in
  `lib/motion.ts`): when the page (the path's first segment past the
  workspace) changes by a click, the blocks inside its panels rise 6px and
  fade in over 200ms `--ease-out`, the first three 30ms apart, the rest with
  the third. Blocks that mount while the page's data lands (within 400ms) rise
  as they arrive. A block holding an `EnterList` or an `EmptyState`
  (`data-enter-list`, `data-enter-own`) leaves the motion to them; a block
  that runs its own entrance keeps it. A tab, a settings
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
- Overlays (`index.css`, Overlays) move on transitions from `@starting-style`,
  never keyframes, so one reopened mid-close turns back from where it is; a
  closing overlay never takes a click; and one opened or closed from a key
  shows and goes in the same frame (the root's `data-input`, written by
  `lib/motion.ts`). Popovers, menus, hover cards and selects open in 150ms
  and close in 120ms from 0.97 and 4px toward their trigger, from the
  trigger's corner. A dialog rises 6px from 0.97 over 220ms and leaves in
  150ms to 0.98; its scrim fades on the same clock. A sheet slides in 300ms
  on `--ease-drawer` and out in 220ms, its scrim with it. A tooltip that
  waited fades in from 0.97 over 125ms; one shown at once beside another
  does not animate; it fades out in 100ms. Toasts keep the library's
  choreography at 250ms `--ease-out` (a stacked toast leaves in 200ms). A
  spinner waits 300ms before it shows. Reduced motion: every overlay fades
  over 150ms in place.
- Disclosure (`.disclosure` with `data-closed`: swimlanes, question rounds;
  `AdvancedFields` does the same with the collapsible's enter and exit):
  opening, the box snaps open and the content fades in as it settles 4px over
  200ms; closing, the content fades out in 120ms and the box shuts after.
  Chevrons turn in 150ms `--ease-standard`. The sidebar's Workspace section (`settleIn`)
  settles its pages in the same way when a click opens it, and shuts at once;
  a key opens it without motion, and it never plays as the sidebar mounts.
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
- Command palette: see its pattern above; the one keyboard surface with any
  motion, and only for a pointer open and for results landing late.
- Chat: someone else's message, and the Agent's reply as its stream starts,
  rise 8px and fade in over 200ms (`arrive`); what was on screen when the
  conversation opened never animates. A day divider that arrives with the
  day's first message draws its hairlines outward from the label (`scaleX`,
  320ms `--ease-out`, 40ms in). A confirmed message keeps its pending
  row's identity (`client_key`), so nothing replays when the server answers.
- Board: a ticket that lands in a done-stage column from a working one gets
  a success wash, the `success` hue at 15% fading out over 800ms (400ms under
  reduced motion).
- Board stage bar (`BoardStageSummary`, `stageBarMotion.ts`): the first time
  it shows, each segment grows from its own left edge, `scaleX` over 300ms
  `--ease-out`, all together, so the bar never moves as a whole. When tickets
  change stage each segment glides to its new place and share (a FLIP of
  `translateX` and `scaleX`) over 250ms `--ease-standard`, and a count that
  changed rises 6px as it fades in over 200ms. Reduced motion: no scale, the
  changed count fades in 150ms.
- Empty state (`.empty-state` in `index.css`): a whole-page one turns its
  orbit into place. The outer ring settles from 0.92 (520ms), the dashed one
  turns in from -40° and 0.9 (620ms), the ember and blue dots swing 75° along
  their orbits (700ms, 60 and 100ms in), the disc settles from 0.94 (320ms),
  and the headline, line and action rise 6px over 260ms at 90, 140 and 190ms,
  all `--ease-out`: readable by 350ms, still by 760ms, which a rare first-run
  screen can afford. A compact one fades and rises 4px over 200ms. Reduced
  motion: one 150ms fade.
- Loader hand-off (`LoadingDisplay`, `enterAfterLoader`): content that
  replaces a loader that was on screen rises 6px and fades in over 200ms where
  the loader stood, the page entrance's own numbers, so late data arrives the
  same way as data that landed inside the entrance's 400ms; never a panel,
  and never outside the page frame. The loader itself stays a linear 1.2s
  turn.
- Deploy steps (`DeployStepList`, `stepMotion.ts`): the steps cascade in as
  an `EnterList`. A step that finishes pops its mark from 0.6 on the
  checkbox's `--ease-spring-pop` (350ms) while its rail draws down to the next
  rung (`scaleY` from the top, 260ms `--ease-out`); the next node lights from
  0.85 over 200ms once the rail reaches it (180ms in), the same rail-led order
  as the DNS stepper. Reduced motion: the changed node fades in 150ms.
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

- Settings save strip (`SettingsSaveBar`): when a value changes, "Unsaved
  changes" and Discard fade in as they settle 4px from the left over 150ms
  `--ease-out` and Save's fill turns `brand` (150ms colour); undoing the change
  plays it back, since it is all transitions. On a save, Save swaps to Saved in
  the same cell (`.swap`): the old label leaves up 6px with a 2px blur as the
  new one rises 6px, 150ms `--ease-out`, the check popping from 0.6 on
  `--ease-spring-pop` (350ms); it holds 1.4s and swaps back. Reduced motion:
  the swap is a 150ms fade, nothing travels.
- Copy (`CopyButton`): the copy icon shrinks to 0.6 and fades with a 2px blur
  over 150ms as the check takes its place on `--ease-spring-pop` (350ms); held
  1.4s, announced as Copied in a status; the label never changes.
- A row leaving a list on request (a signed-out device, `rowGlide`): it fades
  and shrinks to 0.98 over 150ms `--ease-standard`; once it is gone the rows
  under it glide up from where they were, `translate` over 200ms `--ease-out`,
  while the list holds its height and then shuts in one step. Only rows on
  screen glide. Reduced motion: the fade, and the gap closes at once.
- A theme or mode change lands in one frame with every colour transition held
  (`withoutTransitions`); the picked tile's ring changes over 150ms and its
  check pops on `--ease-spring-pop`.

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

### Round-three locks

Decided 2026-10-09 without the owner in the loop: each built as two to four
live variants on the real surface with seeded data, scrubbed frame by frame
and recorded at 1x and 0.25x, and judged against the baseline above.

Board stage bar.
- Load: each segment grows from its own edge, all together (300ms). Rejected:
  the whole bar scaling from the left (the gaps and caps squashed and every
  segment slid right as it grew) and a 30ms cascade across the segments (it
  ran 340ms and read as five bars arriving, not one measure).
- Change: glide plus the changed counts rising in. Rejected: a glide alone,
  which showed the shares moving but not which stages changed when someone
  else moved a ticket.

Command palette.
- Open and close: no motion from a key (⌘K, Escape, Enter) or a pick, the
  dialog clock from the Search row. Rejected: the dialog clock for every open
  (on Escape or Enter the closing scrim covered the page for 150ms and ate
  the next click, and on a pick it faded over the arriving page) and a
  quicker 120ms clock for both (the same problem, shorter).
- Results: rows that join after opening fade in over 120ms. Rejected: no
  motion (a search landing popped six rows at once) and a pop from 0.97 (a
  full-width row scaling from its centre shifted its text sideways).
- Active row: still jumps. Rejected: a sliding highlight on pointer moves,
  which trailed the cursor and split from the brand icon and enter mark that
  jump with it.

Empty state and loader.
- Entrance: the orbit turns into place (above). Rejected: the block's plain
  fade and 4px rise (it also doubled with the page entrance's rise of the
  block around it) and a quiet stagger of mark, headline, line and action
  (correct and forgettable; the mark's motion is what says it is an orbit).
- Loader idle: unchanged, linear. Rejected: a two-dot comet tail (a smear at
  16px) and an eased turn (it slowed at the top of every turn, which read as
  the load stalling).
- Hand-off: the rise. Rejected: a snap (content popped where the loader
  was) and an opacity-only fade (a second arrival vocabulary beside the page
  entrance's rise).

Deploy steps.
- Rail-led (above). Rejected: everything at once (the check, the rail colour
  and the next spinner changed in the same frame and the timeline lost its
  sense of order). A deploy changes step a few times a day, so the 440ms
  hand-down is affordable.

Inbox.
- Unread dot: shrinks and fades over 150ms, in step with the title losing
  weight. Rejected: dropping it (nothing showed what the click changed) and
  a 300ms fade (a grey dot lingered after the title had already gone read).
- Selection: unchanged colour crossfade. Rejected: the sidebar's sliding
  highlight, which slid a full-width box across the hairline rows on an
  action repeated through a whole triage, and which the Docs list pane the
  Inbox mirrors does not use.

Chat day divider.
- Hairlines draw outward from the label. Rejected: riding in with the
  message (nothing marked that a new day had begun). It happens once a day.

Sidebar.
- Active indicator: unchanged; at 0.1x it carries the longest jump (Docs to
  Configuration, about 240px) cleanly in 200ms.
- Workspace fold: the disclosure's 4px settle on open. Rejected: a snap (the
  pages popped in under the header) and a 25ms cascade per page (a list
  entrance in the nav, a second vocabulary for opening a section).

Settings save. Decided 2026-10-09 from four variants on the real card.
- Always there: the strip never moves; a change wakes Discard, the note and
  Save. Rejected: a strip that opens as a disclosure (the box snapped 52px open
  on the first keystroke and pushed every card below it), a bar floating at the
  foot of the page (rose 12px over the next card and no longer said which card
  it saved), and a Save that pops in beside the field (works for one field, not
  for Profile's two).
- Success: Save turns into Saved in place. Rejected: the toast (lands at the
  screen's corner, a panel away from the click) and a Saved line where
  "Unsaved changes" was (the far side of the strip from the pointer).

Removing a row. Glide. Rejected: a fade then a snap (the rows under it jumped
a row's height in one frame) and a collapse that shut the row's box after its
fade (the same jump, a frame later). The first glide shrank the list's box at
once, so the last row slid up from outside it and was clipped; the box now
holds its height until the glide ends.

Copy. Icon swap. Rejected: the label turning into Copied (the button kept the
long label's width, so "Copied" floated in a half-empty button) and a Copied
bubble over the icon (covered the text above a 14px icon at the top of a card).

Theme switch. One frame. Rejected: the new palette spreading as a circle from
the picked tile through a view transition (built and tuned at 320, 400 and
480ms; it read well, but capturing the page froze 180ms and the reveal dropped
about fifteen frames in a headless measurement with no throttle, 300ms and
twelve 67ms frames under 4x) and a crossfade through the same view transition
(the same capture cost). Holding the colour transitions while the palette
changes took the instant swap from about ten frames over budget to one under
a 4x throttle.

Toasts, dialogs, sheets, popovers, menus: already on the baseline's clocks;
nothing changed but the sheet, whose open and close are animations and no
longer carry an unnamed transition list.

### Round-four locks: overlays

Decided 2026-10-09 without the owner in the loop, each built as three or four
live variants on the real surface, judged from frame strips seeked to fixed
times (0 to 300ms), interrupted open-close-open strips and 1x and 0.25x
recordings.

Mechanism: transitions with an empty hold animation. Rejected: the
keyframes (`animate-in`/`animate-out`) every overlay used, which restart from
their first frame when an overlay reopens mid-close, and whose closing scrim
caught clicks for 150ms because Radix writes `pointer-events: auto` inline.

Dialog: centred, rising 6px from 0.97 (220ms in, 150ms out). Rejected: from
the trigger (0.9 from the click point: a 32rem panel travelling diagonally
from a corner button, and no trigger left to point at when a menu item
opened it), a bounce-free spring rise (320ms; still nine tenths transparent
at 50ms, so the form read late) and materialize (a 6px blur clearing: mush
for the first 50ms and a filter on the largest surface over a 24px backdrop
blur). Tuning: the plain 0.97 scale against the 6px rise; the rise matches
the page entrance's vocabulary for arriving. The confirmation dialog runs the
same clock: it is the same family.

Menus: 0.97 and 4px toward the trigger, kept. Rejected: 0.9 from the trigger
(read as a zoom on something opened tens of times a day), a clip reveal (a
curtain with no origin point, and the clip cut the shadow) and a spring
(half transparent at 50ms).

Sheet: `--ease-drawer` (`cubic-bezier(0.32, 0.72, 0, 1)`) at 300ms in, 220ms
out. Rejected: `--ease-out` at 260ms (arrived before it read as a slide) and
the bounce-free spring at 320ms (slow first 50ms). Tuning: 400, 340 and 300ms
on the drawer curve; 300 front-loads as much and settles sooner.

Toasts: unchanged at 250ms `--ease-out`. Rejected: the library's 400ms
`ease` (a stack shuffle that lingered) and the drawer curve at 350ms (slower
than the dialog family beside it).

Look. Dialog surface: the palette's glass with a footer tray over a
canvas-coloured scrim. Rejected: hairlines under the header and over the
footer (an empty band on a dialog with no body; and the glass over a black
scrim turned grey in light mode) and an opaque card (lost the frosted
language the panels speak). Menus: the glass at 9px with 32px rows on a 5px
corner. Rejected: a solid panel with 36px rows and inset rules (a three-item
menu a quarter taller; the inset rule read as a gap) and the palette's
brand edge on the highlighted row (the accent on a pointer hover). Focus: the
ink outline. Rejected: the ember drawn as a 2px inset border (still read as
an error) and an ink border with a soft halo (a focused field in error was
barely different from one unfocused). Destructive button: white on a deeper
red. Rejected: a tinted outline (too quiet for the act it confirms) and the
coral with dark ink (the ember's twin beside the primary).
