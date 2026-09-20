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
| `success` | `#4ade80` | `#15803d` | health and success only |
| `warning` | `#fbbf24` | `#b45309` | in-flight states |
| `info` | `#5cc8f5` | `#0369a1` | open and informational states |
| `destructive` | `#f97066` | `#dc2626` | errors, danger zone |
| `border` / `input` | `#262626` / `#333333` | `#d9d9d9` / `#cccccc` | hairlines |
| `ring` | `#f5f5f5` | `#0a0a0a` | focus and selection |

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

Filter bar. A horizontal row of pill controls directly under the page header:
a search field, then filter pills (`h-9 rounded-md border border-border
bg-card px-3 text-sm`, with a chevron if it opens a popover), an `×`-removable
chip per active filter, and an optional saved-view or sort control trailing
right. `BoardFilterBar` and `FilterChipRow` in `web/src/components/board/` are
the reference; every list page converges on this shape. A filter popover is a
`bg-popover` panel with a checkable row list (icon, label, checkmark) anchored
below its trigger.

Detail page header. Back link (`font-mono text-xs text-muted-foreground
hover:text-foreground`), then a row of mono id chip plus status badge, then
the title, then a muted meta line, then a `border-b border-border` hairline
before the content. `DocDetail.tsx` and `TicketDetail.tsx` are the reference.
Only a page with a genuine single-record view gets this header. In-context
inspection that does not warrant leaving a list opens a right-anchored drawer
instead; a page that works as a full detail view is not forced into a drawer.

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

Project wizard. The `/wizard/project/<step>` flow (project, repository,
service, env, reach, done) reuses the setup stepper wholesale: `DnsStep` rungs
inside `WizardLayout`, with the URL step deciding which rung is open. A rung's
state is its position relative to the current step, so moving forward
collapses everything before it to done with a `Change` summary and lights the
next rung. The project rung is the one departure: entering from an existing
project preselects it, so that rung renders completed from the first paint
instead of asking for something already known. Candidate choice and reach
fields are radio rows and form fields per the patterns above; no new stepper
chrome exists.

Stack detail page. The header keeps the detail-page shape (back link, mono
slug, title, actions top right) and adds a facts grid: a mono microheader over
each value (Status, Image, Runner, Strategy, Hostnames, Repository, or Network
when no repository is attached). Below it the page is the settings shell:
`SettingsSectionNav` on the left driving `?section=` (Overview, Exposures,
Branch deploys, Deploy history, Danger zone), one or two `SettingsCard`s per
section. A card's action lives in its footer strip (`footer` prop: Rollback,
Expose, Add rule), never floating in the body, and a form that is not the
section's main job stays collapsed behind that footer button. Lists inside a
card are hairline rows in one bordered box. Nothing nests a card inside a
card.

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
