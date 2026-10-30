# Design language: glass over a light field

This file is the visual spec of record for all three surfaces: the web app in
`web/`, the public site in `website/` and the phone app in `native/`. They
speak one language. The token values live in `web/src/index.css`,
`website/src/styles/tokens.css` and `native/src/global.css`; the three carry
the same token names (the phone spells them `--color-*`) and the same values,
and a change to one is made to the others in the same change. When a screen and this file disagree, the screen is the
defect.

Read the shared core before any visual change, then your surface's section.
The shared core holds for every surface unless a surface section names an
adaptation and its reason. The component grammar (F1 to F7) is in
`practices/react-guide.md`, the phone's mechanics in `practices/native.md`,
and the site's build in the contributing page `website.md`.

## Contents

- [Shared core](#shared-core)
  - [Direction](#direction)
  - [Tokens](#tokens)
  - [The accent](#the-accent)
  - [Focus](#focus)
  - [Status](#status)
  - [Type](#type)
  - [Shape and depth](#shape-and-depth)
  - [Motion](#motion)
  - [Icons over words](#icons-over-words)
  - [Do and don't](#do-and-dont)
  - [Decision ledger](#decision-ledger)
- [Web app](#web-app)
  - [What the web app adapts](#what-the-web-app-adapts)
  - [Canvas (topology)](#canvas-topology)
  - [Pattern spec](#pattern-spec)
  - [Web decisions](#web-decisions)
  - [Motion baseline](#motion-baseline)
  - [Motion locks: board, chat and ticket](#motion-locks-board-chat-and-ticket)
  - [Motion locks: lists, states and feedback](#motion-locks-lists-states-and-feedback)
  - [Motion locks: overlays](#motion-locks-overlays)
  - [Motion locks: navigation](#motion-locks-navigation)
  - [Motion locks: settings sections](#motion-locks-settings-sections)
- [Public site](#public-site)
  - [What the site adapts](#what-the-site-adapts)
  - [Site tokens and type](#site-tokens-and-type)
  - [Site chrome](#site-chrome)
  - [Docs pages](#docs-pages)
  - [Home page](#home-page)
  - [Roadmap and changelog](#roadmap-and-changelog)
  - [Site motion](#site-motion)
- [Phone app](#phone-app)
  - [What the phone adapts](#what-the-phone-adapts)
  - [Phone tokens and type](#phone-tokens-and-type)
  - [Phone surfaces](#phone-surfaces)
  - [Phone lists and chat](#phone-lists-and-chat)
  - [Phone motion](#phone-motion)

## Shared core

### Direction

Content floats over a still light field. The canvas is near-black in dark mode
and a soft grey in light mode, and behind everything sits a still, soft light
field drawn from the palette: a tint of the brand top right and bottom left, a
lift of the card between, so a palette never shows a hue that is not its own.
Content sits on raised surfaces, a step lighter than the canvas with a hairline
ring: frosted panels in the web app, solid cards and sheets on the
phone, and on the site's reading pages the canvas itself. One ember accent
marks what you act on and where you are: the primary action, the active nav
item, selection, your own messages, checked controls and progress. Focus is
ink, not the ember. Status keeps its own hues as a dot or icon beside plain
text. Technical data (ids, repositories, targets, counts, timestamps) is set
in a monospace face, and titles in a soft serif. Dense where the work is
dense, calm everywhere else.

Light mode is its own soft-grey identity with a pastel field and white
panels, not an inversion. Dark is the web app's default; the site and the
phone follow the system until the person picks a mode.

### Tokens

Themed surfaces use tokens only, never a hard-coded palette class or a
one-off hex. The full set (surfaces, status hues, shadows, shapes, curves) is
in the three token files; this table holds the roles.

| Token | Dark | Light | Role | Surface differences |
|---|---|---|---|---|
| `background` | `oklch(0.13 0.005 265)` | `oklch(0.95 0.004 265)` | canvas, behind the light field | |
| `surface-2` | `oklch(0.155 0.005 265)` | `oklch(0.965 0.004 265)` | wells inside a panel: board columns, logs, code, the topology field | phone: the board's columns |
| `well-shade` | `oklch(0 0 0 / 0.45)` | `foreground` at 7% | the board's column well, laid over the panel as a shade so it separates in every palette; a palette with a black panel (Brutalism) sets it lighter, white at 2.5% | web only |
| `edge-shade` | `oklch(0 0 0 / 0.35)` | `oklch(0.2 0.02 265 / 0.1)` | the 10px gradient a sticky grid column (the Access grids' names, a pinned People column) casts on what scrolls under it | web only |
| `media` | `oklch(0.16 0 0)` | `oklch(0.16 0 0)` | behind a camera feed or a shared screen, and the name chip over one (at 80%): it frames media, not the theme, so it stays near-black in every palette and mode | web only |
| `panel` | `card` 60% into `background` | `card` | the frosted panel, painted at `--panel-opacity` (85%) | phone: solid, never frosted (`oklch(0.178 0.005 265)` dark, the same mix precomputed) |
| `card` | `oklch(0.21 0.005 265)` | white | cards and nodes, a step lighter than their panel: in every dark palette at least 0.029 of OKLCH lightness above the panel as it renders | |
| `popover` | `oklch(0.225 0.006 265)` | white | menus, dialogs, sheets | |
| `foreground` | `oklch(0.97 0.003 265)` | `oklch(0.2 0.01 265)` | primary text | |
| `muted-foreground` | `oklch(0.71 0.01 265)` | `oklch(0.465 0.01 265)` | secondary text, meta lines; 6:1 or better on every surface | |
| `accent` | white at 7% | ink at 6% | the hover lift, reads on any surface; never the current row, whose fill is `brand` at a fixed tint (see [the accent](#the-accent)) | phone: the press tint |
| `primary` | `foreground` | `foreground` | strong ink for emphasis (`text-primary`); not the action colour | site: Fumadocs' `fd-primary` is this ink |
| `brand` | `oklch(0.68 0.2 35)` | `oklch(0.565 0.19 35)` | the ember accent, held to [its roles](#the-accent) | |
| `brand-foreground` | `oklch(0.16 0.03 35)` | white | ink on `brand`, 6.2:1 dark and 5:1 light | |
| `success` | `#4ade80` | `#15803d` | health, success, and open pull requests | |
| `warning` | `#fbbf24` | `#8a6100` | in-flight states | |
| `info` | `#5cc8f5` | `#0369a1` | open and informational states | |
| `merged` | `#a371f7` | `#8250df` | merged pull requests only | |
| `destructive` | `#f26074` | `#c41945` | errors, danger zone, closed pull requests; crimson, 20° off the `brand` ember's hue | |
| `destructive-fill` | `oklch(0.55 0.2 15)` | `#a80f39` | the fill behind a destructive button's white label, deeper than the text hue so it never reads as the ember | site: none, it has no destructive button |
| `border` / `input` | white at 8% / 12% | ink at 10% / 16% | hairlines, the same on every surface | |
| `ring` | `brand` | `brand` | canvas selection and the console glow | site: none; Fumadocs' `fd-ring` is `focus` |
| `focus` | `foreground` at 80% | `foreground` at 80% | the one focus indicator, see [Focus](#focus) | phone: a colour token, `--color-focus` |
| `panel-ring` / `panel-highlight` | white at 7% / 5% | ink at 8% / white at 90% | a panel's hairline and its inner top edge | web and site only |
| `field-warm` / `field-mid` / `field-low` | warm: `brand`'s hue, L 0.5, 70% chroma, 40% alpha; mid: `card` at 55%; low: `brand`'s hue, L 0.45, 50% chroma, 30% alpha | warm: `brand`'s hue, L 0.93, 50% chroma, 40% alpha; mid: `card` at 55%; low: `brand`'s hue, L 0.93, 40% chroma, 28% alpha | the light field's three glows, each a relative colour or a mix of palette tokens, never a literal hue; a grey brand adds no stain | web only; site and phone ship the one palette, with `field-warm`, `field-pink` and `field-cool` at fixed values (phone: `--color-field-warm`, `-pink`, `-cool`) |
| `shape-sm` to `shape-2xl` | 4, 6, 8, 12, 16px | same | the radius scale behind `rounded-sm` to `rounded-2xl`, see [Shape and depth](#shape-and-depth) | phone: `--radius-*` directly; web: Brutalism zeroes every step |
| `font-display` | Fraunces Variable | same | titles and showcase headlines, through `type-display` | site: the same family name over one pinned file; phone: `Fraunces Display`, a static cut; see [Type](#type) |
| `font-sans` | Inter Variable | same | everything else: UI, body, card and section headings | phone: `Inter`, one embedded family with its weights |
| `font-mono` | JetBrains Mono Variable | same | technical data | phone: `JetBrains Mono`, one embedded family with its weights |

How each surface spells the names: the web app defines bare names
(`--background`) and exposes them to Tailwind through `@theme inline`; the site
uses the same bare names, writes each colour once with `light-dark()`, and maps
Fumadocs' `--color-fd-*` onto them; the phone defines `--color-*` directly for
Uniwind, because it reads every colour through `useCSSVariable`. Palettes,
which retheme the surface, text and accent roles, exist only in the web app;
the site and the phone show the default palette.

A token that a design needs and this table lacks is added to the token file of
every surface that uses it and to this table in the same change. A one-off
class is drift.

### The accent

`brand`, the ember, marks what you act on and where you are, on every surface:

| Role | Web app | Site | Phone |
|---|---|---|---|
| Primary action | the primary button | the home page's copy button, the 404's Open the docs | the send button, once there is text |
| Active nav | the sidebar's and section nav's 2px edge | the header's current page edge, the docs sidebar and phone menu edge | the active tab |
| Selection | the current row's tint (`brand` at 12%, 16% in dark), selected rows, the chosen access level, an unsaved Access grid cell, canvas selection (`ring`), text selection (`brand-foreground` at 30% inside your own bubble) | text selection (`brand` at 32%) | the drop target a held card is over |
| Your own messages | the chat bubble | | the chat bubble |
| Checked controls | checkbox, switch (not the Access grids' cells, which are ink; see Access grid) | | the "Only mine" switch |
| Progress | progress fills, the loader's dot | | pull to refresh |
| Prose links | `brand` text | ink text with a `brand` underline (see [Docs pages](#docs-pages)) | |

The current row. Every list that marks the current item (the sidebar and its
channels, the section navs, the Docs, Memories, Plays and Inbox list panes, the
access panel's targets, a note rail) draws it one way, from `brand` and never
from `accent`: a fill of `brand` at 12% (16% in dark), the label in ink
(`text-foreground`) at weight 500, never `brand` and never `brand-foreground`,
and a 2px `brand` edge inset 8px from the row's top and bottom
(`CurrentRowEdge`). The recipe lives in `web/src/components/CurrentRow.tsx`, and
a sliding block (`ActiveIndicator`) takes it there; a list never carries its own
copy. The row's hover stays the `accent` lift, so hover and current read
differently. `accent` is a per-palette role and swung from invisible to a solid
block across the twelve palettes; `brand` at a fixed share is a quiet tint in
all of them, and ink on it holds 4.5:1 (`themeContrast.test.ts`).

Why one accent and only those roles: an accent everywhere stops meaning
"here", and it competes with status colour; held to action and selection it
reads as the product's voice while a red, amber or green dot still reads as
state (ADR 0133).

Never the accent on anything else in the chrome: headings, icons at rest,
borders, chips, badges, a hover. Never a second accent hue, and never a chip
tinted with the accent. Text on `brand` is `brand-foreground`: white on the
bright dark-mode ember is 3:1, dark ink holds 6:1 there, and the deeper
light-mode ember holds 5:1 with white.

### Focus

One indicator everywhere: a 2px `focus` outline (ink at 80%), 2px out, 1px on
a field, which keeps its border; a component's own ring utility is cleared on
focus. Errors keep `destructive` on the border and message, so a focused field
in error is an ink ring around a red border.

Focus is not the accent: the ember sits a few degrees from the destructive
coral, and an ember ring on a field read as a field in error. Ink at 80% reads
as focus on every palette and keeps 3:1 on every surface.

### Status

Status is a coloured dot or icon in its own hue (`success`, `warning`,
`info`, `merged`, `destructive`) beside plain text, never a filled or
tinted-background chip: readable in both themes, and it keeps status from
competing with the accent. A ticket's type follows the same rule: its icon in
the type hue leads and the name stays `muted-foreground` (`NoFillBadge` with an
icon, on the phone card and ticket screen; the web board card shows the icon alone, named by its tooltip), so on a
card the type is never a filled shape beside the palette's accent. A label is a neutral chip: a
`rounded-full` hairline pill at `foreground` 4% with `muted-foreground` text,
its own hue only on a 6px leading dot (`LabelChip` on the web, drawn once for
the ticket rail and the board's label filter, and without its pill on the
board card, where a row of pills outweighed the title; the phone card
has the same chip). A destructive button is white on `destructive-fill`.

A disabled button keeps its shape but loses its fill and shadow and takes
`muted-foreground` text (6:1 or better), never a lower opacity: a half-faded
label sits near 3.5:1 on a settings tray. The web app's `Button`
does this once for every variant. A button that is busy keeps its own fill and
text: its spinner already says it is working, and grey would read as not allowed.

A live status (a runner, an execution log, a health dot) reuses the one
`status-pulse` keyframe; there is no second pulse, and a status that is not
live (a roadmap item) stays still.

### Type

Three families, all bundled locally, no CDN: a font request to a third party
is a render-blocking dependency the product does not control. No script face,
and no fourth family.

- Fraunces is the display face, and only that: page, screen and record
  titles (except the board's, a work surface titled in the sans), empty-state headlines, and the headlines of the showcase screens
  (the web app's signed-out, onboarding and error pages) and the site's
  landing pages.
  Never under 20px (20pt on the phone), where the soft serif turns muddy.
  Its settings are weight 560, `"SOFT" 50, "WONK" 1` (soft terminals, the
  irregular alternates), optical sizing on, `-0.025em` tracking, `1.12`
  leading, set by the `type-display` utility on the web and the site.
- Inter for everything else: UI, body, card and section headings, dialog
  titles. Headings use tight tracking (`-0.022em`) and `text-wrap: balance`.
- JetBrains Mono for technical data: ids, repositories, targets, counts,
  timestamps, code, terminal output. Use `.technical` or `font-mono`.

Each surface loads Fraunces the way its platform renders it best, at the same
axis settings:

| Surface | File | Axes |
|---|---|---|
| Web app | `@fontsource-variable/fraunces`, its `full` build: the variable file that carries the SOFT and WONK axes | set at runtime by `type-display`: weight 560, `font-variation-settings: "SOFT" 50, "WONK" 1`, `font-optical-sizing: auto` |
| Site | one pinned subset, `fraunces-display-latin.woff2` (34KB against the package's 121KB), preloaded | weight 560, SOFT 50 and WONK 1 pinned into the file, optical sizing kept; any other weight renders at 560 |
| Phone | a static cut, `assets/fonts/Fraunces-Display.ttf`, family `Fraunces Display` | opsz 28, wght 560, SOFT 50, WONK 1 baked in, because Android does not apply variable axes reliably |

Title sizes are per surface: the web app in [What the web app
adapts](#what-the-web-app-adapts), the site in [Site tokens and
type](#site-tokens-and-type), the phone in [Phone tokens and
type](#phone-tokens-and-type).

### Shape and depth

- Radius: one scale of five steps. 4px for small inner shapes, rows inside a
  menu and segmented thumbs (`rounded-sm`), 6px for controls (`rounded-md`),
  8px for cards and menus (`rounded-lg`), 12px for panels, dialogs and sheets
  (`rounded-xl`), 16px for the largest floating surfaces (`rounded-2xl`); full
  pills (`rounded-full`) only for badges, chips, and avatars. No radius off the
  scale: an inner corner takes the step below its container. Never a pill
  button or input; a pill reads as a tag, not a control.
- Depth: a card is a step lighter (`bg-card`) than what it sits on, with the
  hairline `border`; a well is a step darker (`bg-surface-2`). The contained
  shadows `shadow-card`, `shadow-elevated`, `shadow-overlay` stay for cards,
  popovers and dialogs. No border where the surface step already separates,
  and no box inside a box: never a panel inside a panel, never a card inside a
  card.
- People without a photo get one flat avatar fill
  (`client-core/avatarFill.ts`, shared by the web and the phone): one of 12
  evenly spaced hues picked by the login, at a fixed `oklch` lightness 0.5 and
  chroma 0.11, with white initials that clear 4.5:1 on every hue, your own
  profile's preview included; under 24px it shows one initial at 10px. Never a
  gradient or a second hue: the one accent is `brand`. A project's mark and
  the phone's tiles use the same fill.

### Motion

Fast and fluid, never slow. This is the baseline every surface shares; the web
app's [Motion baseline](#motion-baseline) holds the numbers for each primitive,
and the site and the phone reuse them unless their section names a change.

- Duration: 150 to 250ms for anything small, up to 400ms only for the few
  moments that carry a gesture or a page. Exits run about 20% faster than
  entrances. A moment that runs longer is a named lock with its reason.
- Only `transform` (`translate`, `scale`) and `opacity` move; a box that has
  to change size snaps in one layout step and its content does the moving.
- Paired elements (overlay and dialog, drawer and backdrop, filter bar and
  result list) share identical duration and easing, or the pair reads as two
  events.

Curves, the same numbers on every surface:

| Curve | Value | For |
|---|---|---|
| `--ease-out` | `cubic-bezier(0.16, 1, 0.3, 1)` | everything that enters |
| `--ease-standard` | `cubic-bezier(0.25, 0.1, 0.25, 1)` | state changes: hover, colour, a status |
| `--ease-spring` | a bounce-free spring written as a `linear()` curve, paired with 200ms | anything that slides from one place to another |
| `--ease-spring-pop` | the one small-bounce spring (0.2), written as `linear()`, paired with 350ms | a mark that pops in: the checkbox mark, the check of a copy, a save or a picked tile, a finished deploy step |
| `--ease-drawer` | `cubic-bezier(0.32, 0.72, 0, 1)` | sheets, 300ms in |

Never `ease-in`, never from `scale(0)`. The web app keeps them in
`web/src/index.css` and, for JavaScript motion, `web/src/lib/motion.ts`; the
site in `website/src/styles/tokens.css`; the phone in `native/src/lib/motion.ts`
(`ease.out`, `ease.standard`).

- Keyboard-instant. Frequency decides: keyboard moves (Enter, Space, arrows)
  and dozens-an-hour actions get no decoration. A page reached with a key, or
  with back and forward, does not replay its entrance; a highlight moved by
  the arrows jumps instead of sliding; an overlay opened or closed from a key
  shows and goes in the same frame.
- Interruptible. Motion runs on transitions, not keyframes, wherever it can be
  reversed, so an overlay reopened mid-close turns back from where it is and a
  sliding highlight plays back from wherever it was. A click never waits on
  motion, and a closing overlay never takes a click.
- Reduced motion is gentler, not none. What tells the reader something
  arrived keeps a 150ms fade instead; nothing travels, scales or loops. A
  movement the act needs stays: a dropped card still glides to its slot, a
  drag still tracks the finger.
- Nothing moves behind a blur. The light field is still: gradients or one
  static image, no filter, no animation. Behind backdrop blur any motion
  re-blurs every panel each frame, which held 50ms idle frames under a 4x CPU
  throttle while the field drifted; nothing may animate behind the panels,
  and there is no ambient motion. The web app's live field on its showcase
  surfaces replaces the frame instead of sitting behind it. A header that text
  scrolls under is opaque, not frosted.
- A motion decision is made by building live variants on the real surface
  with seeded data, recording them at 1x and 0.25x with frame strips, and
  judging them against this baseline. The lock records the pick, the numbers,
  and every rejected variant with why it lost.

### Icons over words

A well-known destination or tool is an icon: GitHub, theme, search, menu. The
product's own places (Docs, Board, Roadmap, Changelog) stay words, because no
icon says what they are. An icon-only control carries an `aria-label` and a
tooltip of the same words. In a list of words, such as a menu, an icon gets
its word beside it: an icon alone would be the one row nobody can read, and a
phone has no tooltip. A filled brand mark sits 2px smaller than the stroked
icons beside it, or it reads heavier. Stroked icons draw at 1.75 on the web
app and the site, set once (`LucideProvider` in `web/src/App.tsx`, one
`.lucide[stroke-width="2"]` rule in `website/src/styles/tokens.css`); an icon
that sets its own `strokeWidth`, like the bold checks at 3, keeps it. The phone
keeps the library's 2: its icons import by path, and the provider that sets a
default is only exported from the barrel the app must not load.

### Do and don't

Do: float content on raised surfaces over the still field; keep the accent to
action, active nav, selection, own messages, checked controls and progress;
status as a coloured dot or icon next to plain text; mono for technical data;
hairline rings that read on any surface; dense but controlled spacing; check
both modes, which are each designed, not inverted.

Don't: the accent on anything else in the chrome (headings, icons at rest,
borders, chips, badges); a second accent hue; a panel inside a panel, or a
border where a surface step already separates; tint a chip with the accent; a
filled or tinted-background chip for status or for a label (a type is an icon in its
hue beside plain text, `NoFillBadge`); the
serif anywhere but a title or a showcase, landing or empty-state headline;
script type; pill buttons or inputs; heavy shadows for co-planar depth; a
second ambient animation or anything animating layout behind the panels;
backdrop blur on the phone; scroll-triggered reveals or parallax.

### Decision ledger

The decisions every surface inherits. Each surface section records its own.

| Decision | Why |
|---|---|
| One design language for `web/`, `website/` and `native/`, specified here (2026-10-09) | A person moving from the site to the app to the phone meets one product; one spec means a decision is made once and every surface follows it |
| Glass over a light field, dark first (ADR 0133) | The monochrome console read flat after every page was cleaned up; floating panels over a soft light field give depth and a recognisable look without decorating the content |
| One ember accent, held to action, active nav, selection, own messages, checked controls and progress | Used everywhere an accent stops meaning "here" and fights status colour; held to these roles it is the product's voice |
| Panels at 85% with a 20px blur, not the mock's lower opacity | The field glows through the edges while body and muted text keep at least 6:1 on the panel |
| Disabled text is `muted-foreground`, not half opacity | At 50% opacity a disabled label sat near 3.5:1 on a light tray, where Save for an unchanged form read as missing; muted ink holds 6:1 and the loss of fill and shadow still says it is off |
| Light `muted-foreground` at 0.465 | 0.48 held 5.65:1 on the canvas, under the 6:1 the token promises; 0.465 clears it on every light surface |
| Light mode as its own identity | A soft grey canvas, pastel field and white panels read intentional; an inversion of the dark look did not |
| The current row is `brand` at 12% (16% in dark) with an ink label and the 2px `brand` edge, in every list and palette (2026-10-26) | `accent` is also the hover colour and is set per palette, so the current row was near-invisible in some palettes (light green, deep purple, console, tiesen, zen), a saturated block in others (indigo clean dark, ember, ocean, grove light), white on yellow at 1.7:1 in Brutalism, and brand-on-brand text where a palette tints its accent label. Rejected: a per-palette `accent` fill (the spread above), `brand` label text (low contrast on a brand tint) and a solid `brand` fill (read as a button) |
| Ink on the dark accent, white on the light accent | White on the bright dark-mode ember is 3:1; dark ink holds 6:1 there, and the deeper light-mode ember holds 5:1 with white |
| Status as icon or dot plus text, never a tinted chip | Readable in both themes and keeps status from competing with the accent; tinted fills washed out once several hues appeared together |
| Fraunces for titles over Inter and JetBrains Mono | A serif display face over a neutral sans is contrast, not resemblance: the titles get a voice no other developer tool has while every control, row and number stays in the precise technical pair. Soft and a little wonky (SOFT 50, WONK 1) at 560, because a sharp high-contrast serif read as editorial and a thin one vanished at 21px |
| One radius scale, 4/6/8/12/16px: 6px controls, 8px cards, 12px panels | Soft but precise, and one step apart, so a corner inside a corner stays concentric without one-off values; pills stay badge-only so controls and tags never look alike. Rejected: 7px controls and 9px cards (menu rows and logos drifted onto off-scale 5px and 10px corners) |
| Terminal-window motif, neutral glow | Code, log, and hero surfaces read as consoles |
| Focus as one ink outline, not the ember | The ember is a few degrees from the destructive coral: an autofocused field in the project wizard read as a validation error before anything was typed. Ink at 80% reads as focus on every palette and keeps 3:1 on every surface |
| Overlays on transitions, instant from a key | Keyframes restart when an overlay reopens mid-close; a transition turns back from where it is. Opened tens of times an hour, a menu or dialog driven from the keyboard must not wait on motion |
| One flat low-chroma fill per person without a photo | One of 12 hues per login, at a chroma that reads as a colour at 20px, still tells people apart at a glance where one grey circle for everyone did not. Against the seeded multi-hue gradient (the loudest colour on a chat or board in every palette, and a blue-violet or green one competed with the brand beside it) |
| Brutalism's hairlines at 20% ink, its fields solid (2026-10-28) | Against solid ink on every `border`: every card and every label chip on the board drew a full-black outline, so a board of twenty cards was forty black rectangles and the chips outweighed the titles; the palette keeps its zero radius, its red and its solid field edges, which is where hard ink earns its place |
| Stroked icons draw at 1.75 (web and site) | Beside 13 to 14px Inter in dense chrome (the sidebar, menus, the ticket rail, toolbars) a 2px stroke carries more weight than the text it labels; 1.75 sits level with it. Rejected: the library's 2 |

## Web app

The web app in `web/` is the language's fullest form: every token, the
frosted panels, palettes, and the pattern layer below. It is built at 768px
first and verified at 768, 1024 and 1440px; phones are served by the phone
app.

### What the web app adapts

Panels. Every page's content lives in a raised, frosted panel that floats 8px
from the canvas edges and from its neighbours; the sidebar sits straight on
the canvas with no surface of its own. Cards inside a panel are a step lighter
with the same hairline ring. The panel is the `panel` utility in `index.css`,
applied once by the layout. A single page gets one panel for its whole content
(`.app-frame` in `Layout.tsx`); a page built from panes marks its root
`data-pane-layout`, which turns the frame off, and gives each pane its own
`panel` with an 8px (`gap-2`) gutter. Never a panel inside a panel.

Palettes. Appearance settings (`web/src/lib/themePalettes.ts`) override the
surface and text roles; a palette's `primary` becomes its `brand` (and so the
field's warm glow) unless it names a brand of its own, and its panels follow
its `card`. The default palette is labelled Nexul (id `console`). Brutalism's
zero radius applies to panels too, and its field edge is the same hard ink as
its panel edge; its hairlines (`border`, on cards and chips) are ink at 20%,
while its fields keep the solid ink edge. Every palette's field edge stays at least 1.1:1 against its
card, so a field never vanishes. In every dark palette a card sits at least 0.029
of OKLCH lightness above the panel it is drawn on, the panel taken as it renders
(its `card` mix over the canvas at `--panel-opacity`), so cards read as raised on
the page and never dissolve into it; light palettes keep a white or near-white
card and separate it by the ring and shadow. Every palette holds 4.5:1 for its text
roles, the label on its accent, and a highlighted row's ink on its `accent`
lift, on each surface they sit on, in both modes
(`web/src/lib/themeContrast.test.ts`); the default palette's secondary text
holds 6:1.

Prose links. The web app's prose links are `brand` text: on the page
panel they hold contrast. The site, whose text sits on the canvas, uses
ink with a `brand` underline instead.

Titles. A page title is Fraunces at 28px, 24px past 50 characters and 20px
past 100 (`pageTitleClassFor`), and holds three lines (`ClampedTitle`): past
them its last line fades out at the end and a muted "Show full title" opens
the rest; editing the title opens it too. A headline takes
`displayTitleClass` and its own size, never under 20px. Fraunces is bundled
as its `full` build, the one that carries the SOFT and WONK axes; Inter and
JetBrains Mono as their `@fontsource-variable` files. A title in a script
Fraunces lacks (Japanese, Arabic) falls back to the system serif for that
script; the 1.12 leading, looser than the 1.08 a Latin serif wants, is what
keeps two lines of Japanese from touching.

Motion, beyond the shared core:

- Panels and the sidebar never move. On a route change only the content
  inside the panels arrives.
- `transition-property` defaults to `none` (base layer), so a bare
  `duration-*` class never turns into `transition: all`; a transition names
  its properties.
- Sidebar collapse: the width swaps in one layout step, never a width
  animation, and since the rail keeps every icon where it was, only the
  labels change. Opening wipes them in from the rail's edge (`revealSidebar`,
  a clip-path over 220ms `--ease-out`); closing snaps. The sidebar starts as
  the icon rail below 1024px.
- The one exception to the still field is the live field of a showcase
  surface (Pattern spec), which replaces the frame instead of sitting behind
  it and runs only on the signed-out home and sign-in.
- Reduced motion: the global block in `index.css` flattens every CSS
  transition and keyframe to its end state; what tells the reader something
  arrived keeps a 150ms fade instead (page and list entrances, chat arrivals,
  the send, the done wash), and nothing travels, scales or loops.

### Canvas (topology)

The canvas is the page: it fills the panel edge to edge as a `surface-2`
field with a faint `muted-foreground` dot pattern, and one bar across its top
is the page's header: the title in Fraunces at 20px, then the counts in
tabular figures (services, the failed and stopped ones each after their
status dot, hostnames), then the zoom (−, the percentage, +), Fit, Back and
More (each disabled while there is no lane that way, so nothing shifts),
Lock, Arrange in lanes and "+ External" as the outline button. The failed and
stopped counts are buttons: each press centres the next service in that state
at a zoom that reads. The bar folds by its own width: under 64rem it keeps
only the failed and stopped counts and Lock and Arrange fold into a "…" menu,
and under 42rem Back and More fold too. There is no floating control strip, no minimap (at the canvas's
size it covered a fifth of the map and showed grey blocks) and no library
attribution.

Every card is a `canvas-card`: the card colour with the panel's hairline ring
and lit top edge, no blur (nodes move on every pan), lifting 2px on hover and
outlined in `ring` when selected. Connection handles show only on hover. A
service card is 288px wide: the name on its own line (wrapping after a
hyphen, never truncated), under it the status as a dot and word with the
strategy in mono at the right, then the machine in mono under a rule. Status
colours only the status, and stopped is `warning`.

Traffic reads left to right in lanes, each kind in its own column under a mono
microheader over its leftmost card: Hostnames, Gateways, Services, then the
nodes Nexul does not run. A gateway card is 360px: a 28px `surface-2` tile
with a shield, its kind in plain words ("Cloudflare tunnel", "Reverse proxy")
over the container's name, the status trailing; one mono row per route,
`→ service:port` with the bracketed `(address:port)` right-aligned in tabular
figures, each row with its own hidden handle on both sides; and a mono footer
of its machine and networks. Hostname pills are derived, never stored or
dragged: each is a `rounded-full` pill (the one on the canvas) right-aligned
one lane gap left of its gateway, level with the row it enters, so the wire in
is a straight line and the pills read as a column of row labels; a pill grows
to its hostname up to 320px, then wraps after a dot, and its row grows to stay
level. A hostname whose gateway is not on the canvas joins the same column,
level with its service, and its wire carries the port. Services stack in their
network boxes (`foreground` at 3.5% over an opaque `surface-2`, so the dot grid
stops at the box, with a hairline ring and the large radius; the mono
`network · <name>` microheader with the member count at its right, and under
it the machine when every member runs on it, which their cards then leave out) in
the order the gateway rows reach them, so the wires out fan without crossing;
a service on no network takes its place in that order without a box. A drawn
node is a 240px card under an "outside Nexul" microheader, so it never reads
as a deployed service.

Route wires are bare 1.5px bezier curves in `muted-foreground`; a failed or
stopped service colours its whole path back to the hostname (`destructive` or
`warning`): a dot leads the pill, and both wires and the row's arrow take the
hue. Hand-drawn relations are smoothstep lines, one stroke per kind (depends
on dashed with an arrow, connects to dotted, mounts solid), labelled in mono
on a knockout of the field. Hovering a card lights every route it is on, end to
end, and the relations that touch it, and dims everything else to 35% over
150ms.

The canvas opens at a zoom that reads: it fits the widest run of whole lanes
from the left at 0.8 to 1, top-aligned, never cutting a lane mid-card, and a
fade over the right edge says more lanes continue. A saved camera is restored
only at the pane width it was saved at. Nothing truncates and nothing
overlaps: an arrangement whose cards, pills or boxes come within 12px of each
other (pills stacked at their rows' pitch excepted) is laid out in lanes
again, and so, once, is a canvas arranged by an older layout.

### Pattern spec

The pattern layer above the tokens: list and table treatment, filter bars,
detail headers, empty and loading states, and the multi-step forms. New
surfaces implement against it rather than inventing a shape per page.

List and row. A row is a `border-b border-border` hairline division, not a
bordered card. Cards are reserved for units that are draggable or clickable as
a whole (the kanban `TicketCard`). Row hover is a
`bg-accent` background lift only, no shadow, no scale; lift and scale are
reserved for draggable cards. The primary field sits left in normal weight;
secondary and meta fields trail right in `muted-foreground`, and in mono with
`tabular-nums` whenever the value is a count, amount, id, or timestamp. Status
renders per the shared Status rule, in a fixed-width column when it leads the
row, so the field after it never shifts between Success and Failure. A page
that needs bulk actions uses a left
checkbox column; no page invents its own selection affordance.

List pane. A page that edits one record beside its siblings (Docs, Memories, Plays)
is the app sidebar, a list, and the open record. From 1024px the list is
dragged wider or narrower by a handle on its right border (220 to 560px,
double-click resets to the 300px default, one width shared by every page,
kept in the browser), but never so wide that the open record drops below
32rem. The list heads with its title and a mono count, a ghost `+` icon button, and a
search field (`ListPaneHeader`; a list nobody creates in or searches, the
Inbox, leaves the last two out). Rows are about 56px with 12px sides and no dividers, the row's
own padding and its hover lift separate them: a 13px normal-weight title in
`foreground` over a muted one-line preview, mono meta trailing right. A
title wraps to two lines at most, the full text in its tooltip, and sets its
own direction so an Arabic title reads right to left. A
row whose record already shows its details when open (Docs) is one line at
32px: a 14px state slot under the folder icon holding the doc's one mark (a play waiting or running, then the lock, then questions closed) that is
empty at rest, the title truncated with its full text in the tooltip, then the
mono time the list is sorted by ("14m", "20 Oct"). The
selected row is the current row (`brand` tint, medium-weight ink label, 2px `brand` edge), so it alone is heavy; a hover is
`bg-accent` lift, never a boxed border. A row's actions are its right-click menu,
opened at the pointer while the row holds `bg-accent`, never a `…` button beside
it (Lock, Clone, then Permissions… with a shield between two separators for
whoever may manage the record's access, then Delete in destructive), each item
hidden without its permission and the browser's own menu back when none is left;
Docs adds Pin first, which needs no permission and lifts the doc into a leading
Pinned group kept in the browser, and Move to folder after Lock, a submenu of the
project's folders with the current one checked. A submenu portals out of its
parent menu, which scrolls and would clip it.
Group labels are 11px uppercase mono, with no rule under them or after the group. Before a row is picked, the open pane says
so with the compact orbit around the list's own icon (the one its sidebar
link uses), "Select a doc", a muted count of what the list holds ("27 docs
in Atlas Platform"), and, where the command palette finds these records,
its shortcut ("Ctrl K to search"); `ListDetailPlaceholder` is the
reference. Docs instead opens on its overview (`DocsOverview`), read from the
set the list shows (live docs, or archived ones while that view is on) and
filtered by the list's search: a muted line of the folder count and the last
edit (mono only as a relative figure, "14m ago"; a date stays in the line's sans) with the pane's one primary `New doc` at its right, the list's `+` hidden beside it, a "Needs attention"
group of the docs a play is writing or waiting on (waiting first, at most
five), and the rest of the newest twenty under Today, Last 7 days
and Earlier as its group labels; a search reads "Matching" and its count, the archived view "Archived", each one flat set. Each overview row
is the state slot, the title in 13px medium on the baseline of its mono time,
and one muted line: the folder and either the run's state or "created by" and the creator and
the doc's first line, its blocks apart by " · ". An empty project's overview is the `EmptyState` with
`New doc`; the list then shows its header alone. Docs groups its rows by
folder below Pinned, the default folder first and the rest in creation order:
a folder row is a chevron, a muted lucide `Folder` (`FolderOpen` while open),
the folder's own name in 12px medium sans, sentence case, and a muted mono
count in the rows' time column, and it toggles on
click, collapsed folders kept per browser and project. On hover and focus a
`+` (new doc in that folder, never on the default folder, which the header's
`+` already fills) takes the count's place, and a right-click opens New doc,
Rename, then Delete (never on the default folder). Doc rows start their state slot under the folder icon, and an empty folder is
its header alone, its count reading 0. A search shows matching docs inside their folders, hides the
folders without a match, and opens collapsed ones while it runs. The header
puts a ghost menu button before them that names the sort and the view ("Created",
"Archived · Edited"), holding Sort by and Show archived, and a ghost
`FolderPlus` (New folder) beside the `+`, for writers only. Below 1024px the
list is the page, so the docs a play holds lead it under "Needs attention". No
folder carries a color. Below 1024px one pane
shows at a time. The list and the open record each scroll inside a panel
that stays still and, while more lies below, fade over their last 40px (`scroll-edge`, driven by their own scroll), with bottom
padding that clears the fade at the end, so content never meets a panel's
rounded edge on one flat line. Once scrolled, the top fades over 24px too,
growing with the first 24px of scroll so the first row at rest is never
washed out. `ListDetailLayout` and `ListPaneRow` in
`web/src/components/listpane/` are the reference.

Inbox. One row per doc or ticket however many notifications it has: the
title, over a meta line saying what happened ("created · 2 updates",
"mentioned you · 2 status changes") with the newest time
trailing it in mono, unread while any of them is. Rows hover and select like the
list pane's, with no dividers. The header is `ListPaneHeader` without New or
search: the unread count in mono, hidden at zero, and Mark all read as a ghost
check-check icon while anything is unread. Docs outside their project's
default folder sit under a folder row built from the same `FolderToggle` as
the Docs pane's, its meta the number of docs, expanded until collapsed and
kept per browser; inside it a title drops a leading folder name, and its rows
take no indent. A ticket's play runs join its row. Every other notification
is its own row, and all of them
interleave by newest activity. The open pane before a pick says how many are
unread and teaches the palette's shortcut.

Board. One grid. The header carries the project's identity: its 32px mark,
the name in the display face (`pageTitleClassFor`), and a mono 12px meta line
under it — "12 of 55 tickets · 8 people", the tickets on show and the people
on them. On the header's right edge, centred on that block, sits the stage bar
(`BoardStageProgress`): 240 × 8, one rounded segment per stage in its stage
mark's colour, 2px apart, each as wide as its share with a 4px floor; a stage
showing nothing keeps a 4px sliver at 25%, Backlog sits at 50%, Done at full
strength, and the bar is hidden when the board shows no tickets. There is no
legend — the stage row under it names and counts every stage — and no
breadcrumb.

Toolbar. Its own row 12px under the header, never sharing it: a search field,
Filter, the people facepile, Mine to test, Playing, then Create alone at the
right edge; 36px controls, one row from 768. Setting a filter drops a chip row
8px beneath it, Clear all last, so the controls never reflow. Below 900 the
toggles go icon-only with their names in tooltips; below 768 Create is a 36px
icon.

Board width. The header, toolbar and grid share one box —
`min(panel − 32px, max(the tracks at their 280px ceiling, 45.625rem))` —
centred in the panel: the chrome never scrolls with the grid, and it is told
the board's scrollbar width so Create and the last track end on one pixel. The
Board is the one page that does not take `Container`'s measure; its width
comes from its tracks, and a very wide screen leaves panel either side rather
than stretching every card.

The board scrolls on its own, both ways, under the header row. Its stage row
(`BoardStageHeader`) is drawn once and pinned at the top of that scroll: each
stage's mark, its name in 13px semibold, its count across every lane, and its
`+`; the name and count are the stage's drag handle for reordering. Done
starts folded to a "Done 10 ›" control at the end of the stage row (mark,
name, count, chevron) with no well under it in any lane; the control opens it
and takes drops while a card is held, a fold button in its header closes it, and the choice is
kept per browser, because finished work is counted, not read, on a board kept
open all day. Every lane's row and the stage row share one grid template
(`boardGridTemplate`): `minmax(180px, 280px)` for each open stage and 7.5rem
for folded Done, so a stage is one column down the whole board and no card
grows past 280; past the panel's width the board scrolls sideways and never
squeezes a title, the right edge fading only while there is more board, and
below 1024 its right edge fades over 0.75rem, short enough that a cut
column's titles stay readable, and the bottom fades into the panel's edge.
The scrollbar sits in the page's right padding, beside the columns, with its
space always kept, so opening or collapsing a lane never moves a column.
A card dropped on the stage row lands in that stage in its own lane.

A swimlane is a 32px band across every stage column, folded ones included,
pinned under the stage row while its lane scrolls: a chevron, the lane name
in 14px semibold truncating with its full name in a tooltip, and the lane's
run marks (spinner and count, the waiting mark) in muted 12px. Its right end
carries the lane's progress — a 96 × 4 line, `success` over `muted`, then
"4/11 done" in mono 12px tabular on a 200px reservation — held at the board's
visible right edge so it never scrolls away with the last stage, and the
band's sticky offset is the stage row's exact height. Each
stage's well (`well-shade`: ink at 7% in light, black at 45% in dark, a shade
rather than a colour so it reads in every palette; a black panel takes a lighter
well instead) fills each lane's cell, as tall as the lane, with the radius of a
card, so a card dropped anywhere below a cell's last card lands in that cell and
every band sits between rounded wells. An empty cell takes half of it
(`well-shade/50`) — a place to land, not a slab — on the same footprint, so the
columns never move. A cell
mounts its first 10 cards and a "Show 10 more of N" row mounts the next. While
a card is held, an empty cell and the folded Done cell show a card-tall
dashed slot as the place to land.

A card is three rows. An eyebrow in 11px mono leads: the key, then at its end
the run marks, the thread mark and the 16px avatar of whoever acts next, all
centred on the key's cap height. The run timer shows one unit ("45s", "16m",
"2h"); on a narrow card the thread mark gives way first, then the timer, and
the spinner stays. The title takes the
card's full width under it, in 13px at weight 450 (500 read as bold at that
size on a dark card), at most three lines. A blocked card adds "Blocked by" and
the key on a line of its own. Last comes the type as its icon (a bug in its
fuchsia, every other type muted) and at most two labels, each an 18px pill in its
own hue (the hue at 15% under its 800 ink, 300 in dark, 900 for lime and
amber; 6:1 or better on the card), any more as "+N" naming them in its tooltip. Labels
after the first drop out whole; a first label too long for the line truncates,
its full name in its tooltip. The card pads 8px
top and bottom and 10px at the sides, `card` with the card shadow and its
hairline, so the board reads as three surfaces: panel, well, card.

Filter bar. A horizontal row of pill controls directly under the page header:
a search field, then filter pills (`h-9 rounded-md border border-border
bg-card px-3 text-sm`, with a chevron if it opens a popover), an `×`-removable
chip per active filter, and an optional saved-view or sort control trailing
right. The board's bar is 32px controls: a 15rem text filter ("Filter tickets", so
it never reads as a second copy of the sidebar's Search) leads, then the
Filter menu, the developer filter as a stack of three 28px avatars and a "+N"
that lists the rest, and the toggles (Mine to test, Playing), ghost in muted
ink until on; below 1280 Playing is icon-only (named by its tooltip) while
Mine to test keeps its label, and the search field narrows to 160px to hold
them, wrapping inside their group when the board is narrow; the labelled Create button alone holds the right edge. `BoardFilterBar` and `FilterChipRow` in `web/src/components/board/` are
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
A list pane (Docs, Memories, Plays, Inbox) keeps its pane title bar instead of a page
header, and the open record beside it takes the page header with crumbs back
to its list and folder (Docs › Runbooks), no workspace crumb. A doc's header
is three lines with no closing hairline (`actionsOn="crumbs"`): the
breadcrumb with the actions on its right (Doc | Questions first, then a
hairline divider, Plays, watch, thread and the `…` menu; they wrap under the
breadcrumb rather than cut it), the title with the pane's full width, and
the meta line (up to three 24px faces, the live dot named by its tooltip,
the mono date). Watch is always the open eye with its count, ink while you
watch and muted while you don't. The body sits straight on the page panel
(no fill, ring or shadow, no side margin), 24px under the meta line, in a
42rem reading measure that starts at the title's left edge. Once the doc's
content is 50rem wide (a 35rem body, the 2rem gap and the rail) the table of
contents is a 13rem rail pinned to the right, its top level with the body,
once the body has two headings: one line per entry with the heading in its
tooltip, indented from the doc's top level, and the section in view (the
first before any scroll) in ink with the 2px `brand` edge. Narrower, the
rail is not drawn and an "On this page" icon in the actions opens the same
outline in a popover. A memory's body sits on the panel the same way, under
the same three-line header: the `Memories` crumb and the folder, with Save and
a `…` menu (Clone to…, then Delete) on its right; the title; and a settings
line in place of the meta line: the folder as a ghost menu (Main or Footer,
each with its one-line meaning), the Always included switch with its word, and
the mono "updated · N versions". The interview and the decisions log show a
14px lock and their state ("Always included", "Not sent every turn") instead
of the switch, with their reason as a visible muted line under it, never only
a tooltip. Above the body, the "When to use" microheader over the rule in a
`surface-2` well with a 2px rule in `foreground` at 25% on its left, square on
that side, growing with the text. Attachments and the newest five versions sit
in the 13rem right rail from a 50rem pane and under the body below it; no
tabs. In the list, a fixed memory's row carries the lock in the switch's
column, as wide as the switch so the dates stay in one column, and the date
and switch sit on the title's first line (`metaOnTitleLine`), because a title
that wraps beside them would otherwise leave them level with neither line.

Rich text (`.prose-rich` in `index.css`), the doc and ticket bodies alike:
16px at 1.75 (quoted in a chat bubble 14px, in a rail 12px, both at 1.6), headings in Inter 600 at 22, 20 and 16px for `h1` to `h3` (the
28px serif page title stays the top of the hierarchy, and no heading is
smaller than the text it opens), closer to their text than to the paragraph
above (1.9em over, 0.45em under) and balanced, paragraphs `text-wrap: pretty`,
muted list markers. A task item (`- [ ]` and `- [x]` in markdown, stored and
served unchanged) is the checkbox rule at 16px on the item's first line with
no bullet: the `input` hairline unticked, a `brand` fill with a
`brand-foreground` mark ticked, ink focus ring, and the text at full contrast
either way. Only an editable body toggles it; a read-only body and a rendered
reply show the state and ignore the pointer. Tables are ruled data, not boxed
cells: a hairline under each row, a muted 13px header over a stronger rule,
tabular figures. Images take the large radius, a black or white 10% outline
drawn inside, and the card shadow. Code blocks stay consoles at 13px. A
blockquote is a 2px rule in `foreground` at 25%.

Page width. `Container` is `max-w-7xl` for lists and grids (Runners,
Automations, Topology) and `size="page"` (`max-w-5xl`) for settings and
single-record pages (Configuration, Your settings, Project settings, Stack,
Deploy, Automation); every page pads `py-8` inside its panel. Pages built
from panes (Docs, Memories, Plays, Inbox, Ticket) run the full width with no
Container, one panel per pane; Chat is one conversation panel.

Microheader. The one small uppercase label is `microheaderClass`
(`font-mono text-[11px] font-medium tracking-[0.12em] uppercase
text-muted-foreground`): facts, group labels, rail sections, column heads.
No other size, weight, or tracking for an uppercase label.

Menu. Every menu, custom on a popover, a dropdown or a right-click, is one anatomy: a
`glass-menu` panel (the frosted recipe's ring and lit edge on solid `popover`,
since with no scrim under it the page would read through, and the elevated
shadow) at 8px with 4px padding; rows of at least 32px in `text-sm` with
`px-2 gap-2.5` on a 4px corner, concentric in the panel's 8px, a muted 16px
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

Tooltip. A small `glass-menu` label (`text-xs`, `px-2 py-1`, 6px) for an
icon-only control; the first waits 500ms, the next shows at once
(`TooltipProvider` in `App.tsx`).

Toast. The menus' solid surface at 8px, the status hue on the icon only, the action
in `brand`; bottom right, stacked by the library.

Detail page header. The page header above, with the record's mono id chip
and status in the meta line. Only a page with a genuine single-record view gets this header. In-context
inspection that does not warrant leaving a list opens a right-anchored drawer
instead, or a centered dialog when the record is a short form of its own (the
Team's person dialog, its body scrolling between a fixed header and footer); a
page that works as a full detail view is not forced into a drawer.

Calendar. A page of its own, not a list pane: the title and New meeting over
one toolbar row of previous, next and Today as outline buttons, the period's
name, and a Month | Week segmented control at the right, every control 32px;
the header drops its hairline here, and the view fills the rest of the panel.
The month is a hairline grid of seven columns, the weekend's at 0.65 of a
weekday's above 880px of grid, a week to a row; the rows share the panel's
height above a floor that fits a cell's head and its lines, so nothing clips
and a short screen scrolls. A cell holds five lines in a five-week month and
four in a six-week one, a number taken from the week count, never measured.
Its head is the day number (today in an ink disc, never the accent) and, at the
right, what had no line, counted by kind in mono (`+1 due · +3`, the total
alone under 880px), a button that opens that day's week. Days of the
neighbouring months sit in the board's `well-shade`, and the grid ends in 8px
corners. Due tickets lead as outlined
boxes (4px, `foreground` at 4%, a hairline ring, the calendar-clock icon and
the title, the key in the tooltip), giving lines to the meetings while they
need them; a meeting is one 20px line, its zero-padded start in mono on a lead
track so every title starts on one edge. Under 880px of grid a meeting's time gives
way to a 6px muted dot (the time in its tooltip) and the box drops its icon.
The week is seven days, each a heading of its number and weekday over its
entries on shared tracks: a 120px lead (the zero-padded range, a `warning` dot
before one that starts inside an earlier meeting, `+1` after one that runs
past midnight; a due ticket's icon and "Due"), the title capped at 40rem with a
due ticket's key after it in mono, and from 1100px of grid the meeting's place
and how many are coming, muted, at the right; an empty day says "Nothing
scheduled". A cancelled meeting is muted, its title struck through, and opens
like any other. A meeting opens in the right sheet, its address
`?meeting=<id>&slot=<occurrence>` so a notice can link to it: title and time in
the header, facts, the people grouped Coming, Declined and No reply, the notes
held to eight lines behind Show all, and what the viewer can do at the foot
(Cancel meeting, never a bare Cancel beside the sheet's close). The grid stands
while the next period loads, so stepping never swaps it for a spinner. New
meeting and Edit are one form dialog at 36rem.

Ticket page. The page runs the full width beside the sidebar, with no centred
cap, as one full-height panel. Its header spans the panel, so a long title
reads across it, and stays put while the rest scrolls. Its meta line is the
ticket's standing in the board card's marks, the same on every tab: the key in
mono, the status dot and word, whoever acts next as the card picks them (the
tester in a testing-stage column, else the developer) as a 16px avatar and
name, one `⦸ KEY` per blocker still open, one pull request mark per linked PR,
then the presence bar; it is read only, and the rail edits. Under the header a
line tab row holds Ticket and Thread, each tab past the first at its own path
segment, and Notes once the ticket has a note. From a 52rem panel the 21rem
rail sits beside the tabs behind a hairline rule, level with the tab row, on
Ticket and Thread alike, and the tabs and the rail scroll on their own;
narrower, the rail folds behind a Details row above the tab row, closed by
default, so the body starts under the header; open, it takes the height the
tabs leave above 10rem. Notes is laid out as Ticket: the
picked note open on the left at the body's measure, headed as its dialog is,
and a Notes rail of rows (file, time, the summary clamped to two lines) with
the picked one on the active edge, in place of the ticket's rail; the newest
opens first. Ticket holds the body, open on the panel with no card and held to
a 68ch measure (a card around it only framed the empty editing space), then,
once the ticket has a thread, its three latest lines behind a hairline: a
`Thread` microheader with an "Open thread" link, and one line each (a 20px
avatar, the name, a mono time, the text muted and truncated; a system line has
no avatar or name; a question or secret request says what it asks). Thread
fills the rest of the column with the composer at its foot. A ticket opened
inside another page (the Inbox split view) fills its detail pane with the same
tabs. The rail is a run of sections, each a mono uppercase microheader with its
one action trailing as a ghost `+`, compact rows, and a muted one-line sentence
when empty ("No bugs reported."), ordered by what decides where the ticket
stands: Properties (Status, Developer, Tester, Due, Labels wrapping inline,
Category, Type), Links, Development, Reviews, Testing, Plays, then
Attachments, Bugs and Trail, and a muted mono foot line, "created 17 Aug by
<reporter>". A play is one line, its full prompt in the tooltip, ending in its
latest run's state icon and mono time. Plays is stage-bound, so it is
absent, not empty, when no play applies to the ticket's stage, unless an auto play waits there or the daily cap paused
it: a queued run is an `info` clock over "<person> · <why it waits>", a paused ticket a `warning` pause over its runs
today, each with its one ghost action (Cancel, Resume); its person also reads each of their computers' reasons in the why,
for that person only. What an auto play already did is an event line in the Thread
at its time: a small icon in a circle under the avatar column, the play's name in medium weight, the rest muted, a mono
time, and an outline "Run it" on a didn't-run. Links leads with the Source group
when the ticket has one: a muted file icon and the doc's title as a link. A linked ticket is its
status icon, its mono key and its title, muted and truncated to one line; hovering or focusing it opens a hover
card with the title, two clamped lines of description, and the status, and a
remove `×` shows on the row's hover and focus. `TicketPageBody` is the
reference.

Empty and loading state. `EmptyState` is the orbit mark (the muted icon on a
card-coloured disc, ringed by two thin orbits stroked from `muted-foreground`
to `brand`, a brand dot top right and a muted one bottom left),
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
live where a page opts in, the same brand tint top right, card lift between and brand tint bottom left,
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

Signed-out frame. Sign-in, the invitation and first-run setup share one frame
(`SignedOutFrame`): the 20px mark and the instance's host in 12px mono in the
top-left corner, 24px in (32px across from 1024px), and the screen under it in
one 448px column: the card centred for
sign-in and the invitation, and setup's forms straight on the quiet field at
one height on every step, so a step or tab that grows never moves the header. Each opens with the same header
(`SignedOutHeader`): the 32px Fraunces title and one muted line, setup's
`Step N of 3` with three 24px segments (brand for done and current) above the
title. A refusal is one `InlineNotice` (a `destructive` hairline at 40%, the
icon on the title's cap height, a 13px semibold title over a muted line) above
what it concerns, never a centred error well; a field's own error stays under
the field. Every button is 40px and full width, and a stack of provider
buttons pins each mark 16px from the edge so the marks share one x
(`ProviderButton`). An invitation lists each workspace, its project count once
signed in, and a card of its roles, each led by a role mark, with what each
allows, and fits a 900px screen at its worst case without scrolling. A command or an address to paste
elsewhere is a `CommandRow` with its copy button, never inline code in a
sentence.

Stat and summary row. Three or four values in a single row above a list, in
the same `surface-2` well as a stack's live status, split by hairlines,
microheader label above a 20px mono value, color-coded only when the metric is
inherently positive or negative (health counts). Only a page with a fleet or
health rollup worth reading at a glance gets one; a page with nothing to
summarise does not invent one.

Runners. The header's meta line is the fleet's health, each clause led by its mark and set in the UI sans with tabular figures: `● 6 of 11 online`, `● 4 busy` (the one `status-pulse`), `○ 5 offline`, a dashed ring and `1 unused code`, a muted dot and `4 waiting`; a clause whose count is zero is left out. While a job waits, a Waiting for a runner list sits under the header: one 40px row per job on the runner list's tracks: its queue place in mono and the job under Runner, the kind under Version, and under Doing why it waits in plain words (`no runner online`, `all runners busy`, `next in line`) on its machine in mono, then how long it has waited and, where the cell is 22rem wide, who started it, wrapping rather than cut; three rows below 44rem and five from there, then Show N more. Then every machine is one list: a column head row (Runner, Version, Doing as microheaders), and per machine a band of `well-shade` over `card`, pinned to the top while its rows scroll (its status dot on the runner dots' x, `success` when every runner is connected, `warning` when some are, a hollow ring when none; the name in 14px semibold, never cut; the state in muted words, "online", "4 of 6 online · 1 unused code", "offline · seen 3d ago"; a mono facts line with the hostname and the editable stack root; trailing, a chevron that folds the machine to its band and then shows how many rows it hides (a machine whose runners are all offline starts folded), who the machine's rules reach in words as a muted ghost button for whoever manages it, a muted ghost Add runner (a bare `+` below 44rem, left out while the machine has no rows, whose empty row offers it), and the `⋯` with Import, Permissions… and Delete). Under each band its runners as one-line 40px rows on tracks the whole list shares: the dot and the mono name, cut by measured width so the ellipsis touches the last dash segments (10 characters at most), the whole name in a tooltip; the version in mono, with a muted `↑` and the release it moves to in the tooltip when it is behind; what it is doing (a pulsing 8px `info` dot hung in the column gap and the job in the UI sans cut on whole words, then muted how long it has run and, where the cell is 22rem wide, who started it, `idle`, or `offline · seen 6h ago`, worded as the band words it and the only place a runner's last seen shows); a `⋯` with Remove runner. A code made for the machine and not used yet is a row there too, a dashed hollow dot and `unused code · until 11:47`, Revoke in its `⋯`; runners and codes no machine holds come last, under a No machine band two lines tall like the others. A status phrase is set in one face, its figures tabular; mono is kept for values that stand alone (names, versions, queue places, paths). Every row sets its texts on one baseline and its dots on the text's cap height. `FleetFeed`, `MachineBand` and `FleetRunnerRow` in `web/src/components/runner/` are the
reference.

Wizard frame. Every wizard (Welcome, the project wizard and its import door,
the owner onboarding, DNS setup) is one frame on the quiet showcase surface,
`WizardShell`: a rail column on `surface-2` (13rem, 15rem from a 56rem
container) holding the logo and the way out ("Finish later" where progress
resumes, "Exit" where it does not, "Skip for now" on DNS, none on the owner
onboarding) above the rail, and beside it the step in a `max-w-[40rem]`
column that scrolls on its own, with the action bar pinned under it: a 64px
band with a top hairline whose content sits on the step column's tracks, Back
on its left edge, then a ghost "Skip for now" and the primary ending on its
right edge. A step's footer (`WizardFooter`) portals into that bar, so a long
step never pushes its primary out of view; a submit button there carries
`form={id}`, since the portal takes it out of its form, and a primary that
another flow renders too wraps in `InWizardBar`, which moves it into the bar
only inside a wizard. The rail (`WizardRail`) is the flow's only progress
widget: a microheader naming the flow, the object's name in mono (two lines
at most), and "N of M done" over its top-level steps, where the open step
counts by what its run recorded and Done counts on its own screen. Then one
`WizardRailRow` per step: a 20px node (current: the brand ring with its dot;
done: a `success` check; skipped and later: a dashed ring with a dash; to do:
a hollow ring; locked: a fainter ring) over a connector that runs node edge to
node edge, the label, and under it an 11px mono caps line on every row, `Step
4 of 6` on the current one, its ordinal (`05`, `5.2` for a sub-step) on one
to do, otherwise the mark's word (Done, Skipped, Later, Locked). A row is a
button only where its flow can open that step: every Welcome and project step,
every unlocked DNS rung, no owner or import step (a later owner step needs the
owner made, a later import stage needs the discovery). A step opens with
`WizardStepHeader`, a Fraunces title at 2rem that asks the step's question
over one muted line. Lists and choices inside a step are a card per row
(`WizardCardList`, `wizardCardClass`, `wizardOptionClass`): `bg-card` at 60%
with a hairline border, the whole card the control, `bg-accent` on hover,
a picked choice ringed in the brand; a list that navigates ends each card in
a chevron, and a level reads as a mono chip. `WizardShell` is the reference.

Setup stepper. A multi-step setup inside the instance setup flow is a vertical rail
of `DnsStep` rungs (hollow dot upcoming, filled dot active, check done), with
content straight on the canvas, never a card inside a card. One rung is open
at a time; a finished rung collapses to a one-line summary with a `Change`
ghost button. Choices are radio rows with hairline dividers
(`EntryPathChoice`); a form shows its two or three real decisions and folds
the rest under an `AdvancedFields` disclosure with defaults; a mono preview
line reads the form back (`app.example.com → http://web:80`). Motion is rail
led: the finished rung's rail draws down (260ms `--ease-out`, `scaleY` from
the top), then the next body rises in (200ms, 4px, with a 180ms delay only
when it follows a drawn rail). `SetupDomainStep` is the reference. The DNS
setup after first run (`/wizard/onboarding/dns`) is a wizard instead: its
rungs (Path, Tunnel once that path is picked, Hostname, Go live) are rail
rows, one rung on screen at a time; a finished rung opens to its summary in a
card, and only the path's Change starts over (`DnsSetupWizard`).

Project wizard. The `/wizard/project/<step>` flow sits in the wizard frame.
The rail's microheader names the door ("New project", "Project setup", "New service", "Attach repository") over
the project's name once there is one, then one row per step (Name, People, Repository, Deploy,
Service, Environment when the scan found keys, Domain, Branches, Done).
A row shows what the project records for its step (ADR 0143), Deploy reading
Later when setup skipped the service. In the setup door the steps after Deploy are its sub-steps, indented one step, shown only
while Deploy or one of them is open or a service exists, the main connector running past them; the rail
is an `EnterList` with the rise arrival: it cascades in once, and sub-steps that appear when Deploy opens rise 8px over
200ms; rejected were no motion (the rows jumped in under Deploy with nothing saying where they came from) and the pop
arrival (a 0.97 scale is invisible on 14px text and read as a plain fade). The add-a-service and attach doors have no Name,
People or Deploy, and show only what that run passed, never the project's recorded marks. A step is the step header
and its body, which slides 8px in the direction of travel over 180ms; there is no door title, step heading or step
description. Every step ends in the frame's bar: Back (the first step of a door leaves the wizard, every other returns
to the row above it), a ghost "Skip for now" where the step allows it, then the primary, which says what happens
("Continue", "Create & deploy", "Save & deploy", "Expose", "Finish"). A step opened before its groundwork says so in
one muted line with a "Go to <step>" primary; a step that already made something shows it with Continue. People is a
card per role with its checkbox. Repository asks two things in order, each a small heading over its
control: "Deploys from", then "Tests live in" with a segmented "The same repository | A separate one". A repository
either answer already holds is a card (`bg-card`, hairline border, the provider mark in a `surface-2` tile, the name over
one muted line) with an outline Change that opens the search under it and turns into Cancel. In the setup door picking
only names the repository; Continue attaches it and goes to Deploy, and a project with none takes the ghost "No
repository yet". Deploy reads the attached repository itself, on any device, and offers two choice cards:
"Deploy now" over a mono line of the repository, branch and what the scan found, and
"Later". Skipping Service, or a step after it with no service, is Later: the deploy steps are marked skipped and the
wizard goes to Done, which folds Repository and the deploy steps into one Deploy row while the project has no service
and lists Docs, Board and Invite people under "Where to next" as hairline rows that finish setup on the way out. The URL
step is the whole navigation state. The import door (`/wizard/project/import`) is three stages on the same frame:
Source (the machine and the project the adopted stacks join, Discover off until both are chosen), Adopt (each compose
project, lone container and gateway as a card, ticked by default, "Import N"), Done.

Welcome wizard. `/wizard/welcome?step=<key>` sits in the wizard frame, the
rail a `WELCOME` microheader over "N of M done" and one row per step, each a
button. The step slides 8px over 180ms like the project wizard's and ends in
the frame's bar. A step done before it opened shows what it found in a
`bg-card` row with Continue alone; one done while open settles its live check
and unlocks Continue. Your projects is a card per project that opens its
board, the person's level as a chip and the areas only when they differ. The
finish view rises over 200ms: "N things left for later" (or "You're set up"),
one hairline row per step not done with an Open, then "Go to <workspace>".
The owner onboarding (`/wizard/onboarding/owner`) is the same frame with a
"Get started" rail of Profile, Workspace and Tools that moves only through
its bar.

Stack detail page. An ops view first: the daily jobs (is it healthy, what
runs, what was deployed when, what it prints) get the page's full width, and
configuration sits behind one Settings tab. The header keeps the detail-page
shape (breadcrumb of the workspace and project, title, actions top right),
and its meta line sits under the title row at the page's full width rather than
in the column the actions leave; it says what the stack is, in mono: the slug, the runner, the
strategy with its compose file's name (the whole path in the tooltip), the
repository with its branch (or the network when no repository is attached), and
a run stack's image. Each fact carries the dot before it and shortens on its
own, so a wrap never leaves a dot at a line's end. The actions are the page's
one primary, **Deploy** (`brand`, a rocket), then the per-person Watching
toggle (an outline button with an eye), then the row menu. Deploy opens a
popover with the stack's one way to deploy (a build stack: the ref field and
Build & deploy; a run stack with no repository: the image and Redeploy) and,
under a hairline, the last healthy image with Rollback; a stack with no way to
deploy and nothing to roll back to shows Deploy in its disabled look with the
reason in its tooltip, still reachable by keyboard.

Under the header is one live status strip (`StackLiveHero`), a `surface-2`
well with a microheader over each value: Health, Last deploy, Requests and
Hostnames, four columns from a 48rem strip and two by two below. Health is "No
issues" behind a `success` dot over the mono "N up", or one line per problem
behind a dot in its status hue: an open outage is `destructive` ("api down",
or "3 of 8 down" with the names in its tooltip), an error rate over its rule is
`warning` ("5xx 7.2% · last 5 min" in the line's own weight, then the time it
began in muted mono), and a recovery within the hour reads `success` ("5xx
recovered"). When the runner's newest observation of the stack is over an hour
old, Health closes with a muted "last observed 17d ago". Last deploy is the
newest attempt's `HealthDot` and outcome in deploy words (Succeeded, Failed,
Running or Pending; never "Healthy", which is how a service is doing), a
running one with its elapsed time ("Running · 27m"), over that attempt's own
facts in muted mono (its age and who started it; a running one only who), then
the image that runs now, always marked "live" so it never reads as the
attempt's: the tag by its last path segment, cut in the middle so the tag stays
whole, the full reference in its tooltip. Requests is the hour's mono total
("70,448 / h") over the 5xx share naming its window ("5xx 1.6% · hour"), "Not
counted" while the error-rate rule is off. Hostnames is the first as a link
with an arrow, cut in the middle so its domain stays whole, then a "+2 more"
that opens every hostname in a popover. The strip names a problem; the card
below it explains it, so no incident list or banner sits between them.

Under the strip, line tabs on a hairline: Overview, Deploys (with its mono
count), Logs (for whoever holds `stacks:logs`) and Settings, each its own path
(`/stacks/:id`, `/deploys`, `/logs/:service?`, `/settings/:section?`).
Overview is Services, then Requests (a stack with a hostname: the mono totals
for the last hour with the 5xx share behind its hue, per-minute bars in a
`surface-2` well with the 5xx part in the status hue, a quiet minute a 2px mark
on the floor and a minute nobody counted a faint full-height track, and the error-rate rule
in the footer). The Services card has no column headers and one line per
service, rows sorted down first, then on their way, healthy, then stopped: the
name, the image by its last segment cut in the middle, how it is reached (the
first public hostname as a link, else `service:port`, then "+N" with every
address in its tooltip; dropped below a 48rem card, where the strip carries the
hostnames), the status (an icon in its hue and the word) in a fixed column, and
a ghost Logs icon at the end. The name's tooltip holds the container name and
every address. The 14px name sits 1px higher than its box centre so the row's
texts share one baseline. A service's open outage replaces its status and is
one line under the row: a `destructive` icon and "Down since 10:42" in plain
text, then the exit code, "out of memory" and the restart count. One muted line
under the rows names the count and the networks; it is information, so it is
not a footer strip. Deploys is one card titled Deploys with All / Failed in its
`aside`, then one hairline row per deploy straight on the card: the outcome (a
running one with its elapsed time) in a fixed column, what was deployed (the
image by its last segment, or a branch icon with the ref and the short commit
for a build) with who and why under it in muted mono ("current" on the deploy
whose image runs now, then actor, pull request, rule; nothing, not a dash, when
there is none), and the time ("29 Oct 14:02") right, with the relative age in
its tooltip. The newest 25 show,
the rest behind "Show 11 older". Logs is the Logs view below.

Settings is the settings shell: `SettingsSectionNav` driving a `/:section`
path segment (Environment, Alerts, Exposures, Branch deploys, Danger zone), one
or two `SettingsCard`s per section. Alerts holds a Crash tickets card (one
switch row), a Crash notices card (a Channel select row, Off first, then a Post as select row of the channel's bots
after System message while a channel is named, both applying on pick) and an Error rate card (one switch row stating the rule, its
recovery in the footer, greyed with the reason when the stack has no hostname
or the instance can't read analytics), and under the switch row the rule's three numbers as small number fields saved from the
footer, whose hint states how it recovers. A stack whose rule is off still shows the Requests card, with one line saying nothing
is counted and a link to Alerts, and no footer. The settings nav
is a scrolling top row below 1024px and a side column from it, on every
settings-style page (Your settings, Configuration, Project settings, a stack's Settings);
a nav with two jobs (Your settings' You and Instance settings)
labels each group with a mono microheader in the side column and a hairline
between the groups in the top row, which fades over its last 24px, and a group
with nothing the viewer may open shows no label. Danger zone is set apart the
same way: a hairline above it in the side column, with even space either side,
and before it in the top row. Each label holds its medium-weight width, so the
current item turning medium never moves the others, and focus draws inside
the item, clear of its neighbours. A sideways row (this nav's top
row, a tab row) never draws a scrollbar under its labels (`scrollbar-none`);
its fade says there is more. Environment is one card of mono name and
value rows, values dots until the card's Reveal toggle, with Deploy now in the footer beside Save; someone who can't change
the stack sees the names over dots and nothing to edit. A card's action lives in its footer strip (`footer` prop: Save,
Expose, Add rule), never floating in the body, and a form that is not the
section's main job stays collapsed behind that footer button. Lists inside a
card are hairline rows straight on the card, under the rule that opens its body (`settingsListClass`), never a bordered box
inside the card; permission and access rows (Permission rows, below) and a list whose rows each fold open an editor (Run settings) keep their bordered list.
Nothing nests a card inside a card.
A project's Services card lists each attached repository no service builds
from under a Not deployed microheader, after the services or in place of the
empty line, as project settings rows: the GitHub mark, the mono name and
"Attached, not deployed yet", and an outline Deploy with a rocket. No banner,
no dismiss; the row goes once a service builds from it.

Permission rows. Wherever access is set (a role, an automation's scopes) the
domains sit in one list led by an "Every domain" or "Every area" row on a
`bg-muted/40` strip that sets them all and selects no rung once the rows
differ. Each domain row is the name on the left, its extra verbs (Run, Clone,
Thread) as small outline toggles, then the level as a segmented strip of None,
Read, Write, Delete in four fixed 56px slots, None first, so each level holds
one x on every row and a shorter ladder leaves its last slots empty. Every rung
below the chosen one takes `brand` at 10% with `foreground` text, so it reads
as "this much access", and the chosen rung is the one selection state in both
modes: a 1px `brand` hairline, the 10% wash and a `brand` label. The role
editor (Role dialog, below) puts its lists on one tab per catalog area, its
rows straight on the dialog with hairlines (no box), each tab led by its own
bulk row, and sets the extra verbs as checkboxes on a second line under the
area's name, hung on a hairline rail and disabled until it holds Read (See on
plays, whose first rung is named so), and leads each tab with its yes-or-no
permissions as switch rows (the name over a muted line), which the bulk row
never sets. A viewer who can't change a list sees it disabled.
`PermissionLevels` and `PermissionLevelControl` in `web/src/components/access/`
are the reference.

Access grid. The Projects and Public channels tabs of the Access page are one
grid each, rows by roles, and the card fills the panel's height: the grid is
the one scroller, both ways, with a stable scrollbar gutter so nothing moves
when rows start to overflow, and it snaps sideways a whole role column at a
time. Its head row is sticky on top and its name column (240px, 200px below
1024px, the name truncating inside it) sticky on the left, casting its edge
(`edge-shade`) once scrolled sideways; the save strip sits under it, always in
view. A role column is 72px: the name at 12px medium in up to three lines, cut
at a whole word with its ellipsis when longer (`headName`), over its holder
count in mono ("nobody" at none). Spare width goes to a filler between the role
columns and the right edge, so the ticks start beside the names and People
stays flush right. Rows are one 44px line, the project's 24px mark and name.
The whole 72x44 cell is the box: on Projects a tick is a `foreground` check on
a `foreground` 6% tint inset 2px, and an empty cell is unticked; on Public
channels, where every member sees a channel, the exception is marked instead,
a muted eye-off on a role the channel is hidden from. The ticks are ink, not
the accent: a grid of them would put the ember on most of the page, so the
ember rings an unsaved cell's tint (2px, inset) and the footer counts them
("3 changes"); keyboard focus rings the cell's own edge, outside the tint. The
lead row on the muted strip sets a role on every row shown and says how many
have it, in every state ("9/15", "15/15", "0/15"). People sits in a column pinned right, casting its edge while
columns lie under it, its add button showing on the row's hover or focus; a
grid with nothing pinned ends in a 16px column that fades what passes under
it. The bottom of every scroller fades over 24px while rows lie past it.
`AccessGridTable` and `AccessGridCell` in `web/src/components/accessgrid/` are
the reference.

Settings card. A card scans in three levels of type, set once in the shared
components and never per page. The title (`SettingsCard`, the same 16px
semibold as a dialog title) is `foreground`. A setting's label (`SettingsRow`,
and a field's label in a form) is 14px medium, `foreground`. A description or
hint under either (`SettingsHint`: a card's description, a row's, a field's,
a footer strip's line) is 12px at normal weight in `muted-foreground`, set
`text-wrap: pretty` at a 1.5 leading. A page, a stepper's step or a dialog tab
that sits among settings cards takes the same three levels. A label is never
muted and a hint is never medium. The card opens with its title and an
optional description, no rule under them, then the body and the
optional footer strip (`bg-muted/30` behind a hairline), which only a card with something to save or one real action carries. The card's state, or
one quiet action, sits top right beside the title (`aside`: Up to date,
Enabled, Registered). Inside, settings are rows (`SettingsRow` in
`SettingsRows`): the label over its description on the left, the control on
the right once the card is 34rem wide and under
the text below that; a rule above the first row and between rows. A list
inside a card is the same: hairline rows straight on the card, about 56px, no
box of their own and no hover lift unless the row is a link. State is a
`SettingsStatus`, a 6px status dot beside plain text with an optional muted
detail (Connected · by Alice 20d ago, Active · last used 9d ago, Revoked 8 Oct),
never a chip; an active account says nothing, only one that can't sign in
(Disabled, Removed). Destructive actions that are not the card's job are ghost
buttons that turn `destructive` on hover, kept apart from the primary action,
and ask before they act. A card that edits values saves from its footer with
`SettingsSaveBar`: the strip is always there; Save is the primary button,
disabled (the flat disabled tint) until something changes, then Discard and
"Unsaved changes" wake and Save fills `brand`; once a save lands Save reads
Saved with a check for 1.4s. Settings that apply the moment they are picked
(Appearance) have no Save. Copying uses
`CopyButton`; a value to copy sits in a mono `surface-2` well with an icon
button. The page header's meta line carries the open section's facts
(`SettingsHeaderMeta`: Nexul theme · Dark, 2 gateways · 3 hostnames) and marks
instance sections Instance-wide. On Your settings, Configuration and a project's settings the open section is the page
title ("Security" or "Labels" under a `Settings` crumb, "Mention chips" under the workspace and
`Configuration`), with a facts line on every section so the header keeps its height, and no card or tab restates it: a
card that only named the section names its job instead (Name and picture,
People, Starting questions, Ticket mentions). A template a workspace can follow or edit says which, once,
above its card's description: a status dot and word (Instance template, muted; Edited here, `info`) with
an underlined Reset to instance template beside it, and Clone to… as an outline button on the card's right.
A card whose body outgrows the view keeps its footer stuck to the bottom (`stickyFooter`), fading the text
that scrolls under it. Security is one page in the order of the job: Signed-in devices (this
device first, a ghost Sign out on each other row, and an outline Sign out everywhere else that turns
`destructive` on hover), Personal access tokens (revoked ones last), then
Connect a device as two rows, Phone app and Desktop app, the QR opening under
its row only once a code is generated, with Cancel and New code in the row's slot and the code centred on the QR. In a
settings list mono is for identifiers only (handles, IPs, token prefixes, hostnames, the pairing code); counts, dates
and the words between names stay in the UI sans. A row's state sits on its second line, never on the title's: a Team
row's presence, a connector's, which drops the word its tab already says.

Role matrix. The Roles tab of the Access page compares every role at once,
read-only, in the access grid's frame: one column per role, the Owner first
with a crown, each head the name (cut like the grids' heads, three 13-character
lines) over the holder count in mono, beside the role's `…` menu (Edit, Duplicate, Clone to workspace, Delete; the Owner's
has none); one row per area, under a muted band for each of the editor's
sections, Instance, Workspace and In projects. A cell is the level as a
`LevelGlyph` and its word, muted at None, with a mono "+1" on the word's
baseline for the extra verbs held (named in its tooltip); a yes-or-no permission reads as a check or a muted No. Deleting a role
someone holds asks first, naming how many people lose it. `RoleMatrix` and
`RoleMatrixHead` in `web/src/components/settings/` are the reference.

People in this project. A project's Settings → Access card, a section only
whoever manages its access sees: a Roles microheader over the roles it lets
in, People over the people with a rule of their own, then Kept out over the
people a rule keeps out, each a hairline list straight on the card. A row is
the 24px target mark (a role a muted people icon on a `surface-2` tile with
the hairline ring, a person their avatar) and the name in 14px medium over a
muted line (who holds a role, what a person holds, the rules here); the whole
row opens the permissions panel on that target, and its `⋯` holds Open and
Remove (Let back in on someone kept out), which asks first. The footer's "Add
a role or person" lists one, which lets it in, and opens nothing.
`ProjectPeopleSection` in `web/src/components/settings/` is the reference; the
panel's target column uses the same marks.

Project settings lists. Statuses, Categories, Ticket types, Labels,
Repositories and Services are one row (`SettingsListRow`), a hairline apart
straight on the card, at least 44px: a 16px leading slot, the name in 14px
medium with its muted meta on the same line (a template state, a machine, a
repository's role; the meta gives way before the name), a right-aligned mono
count that says what it counts ("8 tickets", "No tickets", from the project's
grouped ticket counts), and the `⋯`, shown on hover or keyboard focus where the
device can hover and always where it can't. The row lifts on hover inside its
hairlines. The leading slot is a colour's dot, or a hollow ring when none is
set (never the hue the board falls back to); a status's icon or its stage's
dot; the GitHub mark on a repository; a service's strategy icon. A ticket
type's dot and a label's pill are the colour button itself; a label has no
`⋯`, its colour being its one action. A status or type that still holds
tickets keeps Delete in its menu, disabled, with the reason under it ("Used
by 8 tickets"). Statuses group under the five stages, each a mono microheader with
its `+` in the rows' `⋯` column. The section nav groups them under Project
(General, Access), Board (Statuses, Categories, Ticket types, Labels) and
Delivery (Repositories, Services), then Danger zone.

Danger zone. Every danger zone (Configuration, project, stack, automation)
is one `DangerZone` card (`danger`, the warning icon) holding `DangerAction`
rows: the action's name in 14px medium, what it does and what is lost in one
muted sentence, what goes with it or blocks it under that (released
hostnames, "Still holds 55 tickets · 3 repos · 6 services"), and an outline
button in the destructive hue on the right from a 24rem card. The solid red
button is the confirm dialog's, whose confirm names the object ("Delete
stack"). A danger zone with nothing in it says so in one `EmptyRow`. A
page already titled Danger zone names its card for what it acts on (This
workspace). An action that can't be taken here keeps its row, the button off
and the reason as its sentence; a section whose only action the viewer's role
hides is left out of the nav.

Appearance. Mode and Theme are radio groups of tiles, each a miniature of the
app (`ThemeMiniature`: canvas, sidebar with its brand marker, a panel with a
title, two text lines, a brand button and an outline one) drawn from the
palette's own roles in the mode being shown (`themePreviewColors`); System is
light with the dark miniature clipped across its right side. Mode is three across at every width; Theme runs six across from
a 42rem card body, four from 32rem and three below, so the twelve themes sit in
two rows above the fold at 1440. The picked tile
gets a 2px `brand` ring and a check badge top right; arrow keys move the pick.
"Dim behind dialogs" is a row with a miniature of a dialog over the scrim at
the slider's strength.

Person dialog. The Team's person dialog holds one browser-style tab per
workspace the person is in: a row of tabs on a hairline that scrolls
sideways, the selected tab bordered on three sides and joined to the panel
below, no close button on a tab, and a ghost `+` after the last one that
opens a popover to pick a workspace. A tab's panel is that workspace's Roles
checklist (below), a muted line saying what follows from the roles, and
"Remove from workspace"; a workspace the viewer can't manage, or an Owner,
shows the roles as chips instead.
Every change is held until Confirm: a tab with held changes shows a small
`foreground` dot, the footer is Cancel and Confirm (disabled with nothing
held), closing with held changes asks "Discard changes?", and a refused
change shows as a destructive line above the footer while the rest stays
held. Account actions (Disable, Remove account) sit at the footer's left and
apply at once. `TeamPersonDialog` and `TeamWorkspaceTabs` in
`web/src/components/team/` are the reference.

Roles checklist. Wherever roles are given (the person dialog, each workspace of
an invitation) every role shows at once, never a dropdown, straight on the
surface: the microheader and a mono "2 of 5" trailing, one muted line ("Everything
these roles allow, combined."), then one hairline row per role you may give, at
least 44px, the whole row the checkbox's label, the name in 14px medium and the
role's tally trailing (two levels at most, as `RoleAccessSummary` draws it). Under a "Can't
give" microheader sit the Owner role and the roles holding more than you hold:
disabled boxes that keep the person's tick, muted names, and a lock with the
reason. With nothing ticked a `warning` triangle and one muted line settle in
(`settle-in`): a warning for a member, "Pick at least one role" on an
invitation, whose Create link waits for one. A role is shown elsewhere as a
chip: a neutral `rounded-full` hairline pill at 20px, `foreground` at 4%, 11px
medium, never tinted; three then a mono `+N` with the rest in its tooltip, and
a dashed muted "No roles". `RolesChecklist` and `RoleChipRow` in
`web/src/components/role/` are the reference.

Role dialog. New role and a role's Edit open one dialog, 40rem wide and
`min(52rem, 100dvh - 4rem)` tall at every width: a header with the title and,
on an edit, the holders as overlapping 20px avatars beside the first line of a
muted note of who a save reaches; the Name (and on a new role a 12rem Start from beside it from a
32rem band, under it below that); then one browser-style tab per catalog area,
Instance, Workspace and In projects, each with a mono "3/12" of the areas and
switches it allows and the person dialog's `foreground` dot while it holds changes; then
the open tab's list, the only part that scrolls; and a footer whose muted note
reads "No changes yet", "Name it to create it" or how many changes it holds ("3 changes": each area whose level moved, each extra or switch
flipped, each welcome step, the name, the description), with Cancel and Save
or Create role. It opens on Instance, the first tab, and
focuses no field. Leaving with changes asks "Discard changes?".
`RoleEditorDialog` and `RoleSectionTabs` in `web/src/components/settings/`
are the reference.

Chat stream. The messages run in one centred column up to 48rem wide, the
composer under it at the same width, so your bubble on the right never sits a
panel away from the line it answers. The column has two edges: its own left
edge holds the header's icon and name, the avatars, the day divider's rules
and the composer's frame, and the body indent (avatar and gap) holds what a
message carries, its name line, text and a bot's report. From a 50rem stream
the scroller keeps a scrollbar gutter on both sides, so the column centres on
the header and composer; narrower it draws no scrollbar, so the column spans
their width. The stream fades out over 16px under the header and 12px over the
composer, never cut on a flat line. The
conversation header is 48px, its hairline running the panel's width (in a wide
voice call its name sits over the stage instead, see Voice call): the icon
(a private channel's lock) and name, then on the right a private channel's
members as three 20px avatars and a mono count, and the channel settings gear
for whoever has settings there, both opening the dialog the sidebar row's menu
opens. While the sidebar is the icon rail, the name is a switcher: a chevron and the
unread count of every other listed conversation as a dot and "3 elsewhere", opening a 20rem popover of the
sidebar's own conversation sections. Your own messages are the one bubble
(`brand`); everyone else's text runs plain under a 32px avatar and a header
of the name in 13px semibold and the mono clock time (the full time on hover),
2px over the text and held to 72ch, so a long thread reads as a transcript instead of a stack of
boxes. A new author starts 12px under the last message; a run of one person's messages drops the avatar and header after the
first and keeps 6px between messages. Once a run's header has scrolled above the stream's top edge, a pill pinned there over the body indent names it, the sender in semibold `foreground` and the day muted ("Deploy Notifier · Thu 8 Oct", `bg-card`, `shadow-card`, a ring), until a row with its own header reaches the edge. On hover or focus a row's actions pill
sits at the row's top-right, beside the header line and clear of the 72ch text;
your own bubble keeps it on the bubble's corner. Each day opens with a divider: the
microheader label ("Today", "Yesterday", "Mon 21 Sept", the year only when it
is not this one) between two hairlines. The composer is one framed field
(`bg-card`, `rounded-lg`, the `input` ring turning to the focus ring) holding
attach, the text and emoji. `MessageList`, `ConversationHeader` and `ChatComposer` are the
reference.

Voice call. A voice channel is a chat stream with the call at its head.
Before joining, one lobby row sits above the messages in the stream's column:
a `surface-2` well at `rounded-lg`, 64px, holding a stack of up to five 28px
avatars (then a mono `+N`), the first names in medium ("Lena, Sam and Bob",
"Lena, Sam and 6 others") over the count ("3 in the call", the number in
mono), and Join at the right; an empty room leads with a dashed 28px ghost
avatar and reads "Nobody's here" with Start call, and Switch call names the
call you are in. Every other state takes the same slot, height and button
place: left after five minutes alone, connecting, voice not set up (a link
for whoever can fix it, who to ask for everyone else) and couldn't connect,
the library's own error demoted to a mono second line; so does the empty slot
shown while the voice code loads. In a call the
stage is a `surface-2` well under the header, `clamp(260px, 46vh, 460px)` tall
over the stream, and from a 1000px thread (the `thread` container query) the
full height beside it, the stream then a 380px column behind a hairline and
the header's name over the stage, not the stream. The tiles fill the well on
both axes: the column count gives a camera the largest 16:9 picture with every
cell at least 160 by 100 (`lib/stageGrid.ts`), one or two people sit in 16:9
tiles up to 480px wide, tiles sit 12px apart so two speaking rings never
touch, a last row that is not full centres, and the people who don't fit go
behind a `+N` tile opening the list of everyone. A tile is
`card` with the hairline: an avatar with the name under it in muted 12px and a
`destructive` slashed mic on the avatar's corner when muted, or a camera on
`media` with the name in a chip over it. Speaking is a 2px `success` ring
outside the tile, 2px off it on the well, so it reads the same over a camera.
A shared screen takes the stage on `media`, whole (`object-fit: contain`),
with its full screen button always shown on a dark chip; everyone is listed
beside it as 32px rows, or as 24px avatars under it in a narrower thread, their mute
marks muted, since nine red marks would drown the one that matters. The
controls are icon buttons centred under a hairline: a muted microphone is the
one state in colour (`destructive` at 12% with the slashed mic in
`destructive`), a camera or share that is on reads as pressed in ink
(`foreground` at 10% under a 25% border), never the ember, which beside the
reds reads as a third alarm, and Leave is a solid `destructive` button past a
short divider, each with a tooltip. Your own tile and row add "(you)"; while a
screen is shared, the stacked stage grows to `clamp(320px, 66vh, 640px)`, and
when people overflow, whoever starts talking swaps in from behind the `+N`
without moving the others.
`VoiceCallSection`, `ConnectedCall` and `components/voice/` are the reference.

Agent turn. Each turn of the Agent in a thread, a trail, a hand-off or a
setup transcript is a fold with no box. Its header is one line led by a 16px
outcome mark: a muted spinner while it runs, a muted check once done, a muted
dashed circle when a call never came back, and a `destructive` warning when a
step failed, the one mark with a hue. While the turn runs, the step in flight
in plain words ("Running bun run lint", "Reading BoardOrder.ts", "Thinking"
between steps) under the `shimmer-text` sweep, then the clock ("· 12s"); once
over, "Worked for 14s" in medium ink and what it did ("· explored 2 files, 1
search, ran 3 commands") muted, then the chevron. It is open while it runs,
folded once over, and opens at any time, a running turn beside it included.
Opened, the steps hang off a hairline rule dropped from the mark, every step
one 13px row of the same weight: a muted 14px icon (a file search, a file pen,
a terminal, a plug for an MCP tool, a wrench for any other) in the column the
header's words start at, the verb muted and the name in mono. Neighbouring
reads and searches list under one count ("Explored 2 files, 1 search") with
the file names and quoted patterns side by side, neighbouring edits the same
way ("Edited 2 files"), and neighbouring commands under "Ran 3 commands" one to
a line; a lone command is one row ("Ran bun run lint"), like any other tool
("Called Github · create_pull_request"). A command drops its leading `cd` into
the run's own directory, and a `cd` below it reads relative ("cd web && bun
run test"); the run's directory is the absolute one most of the turn's
commands start in (`runDirectory` in `client-core/agentSteps.ts`). What the
Agent said between steps is plain text at the rule's edge. A name, a command
or a tool row opens its time, arguments and result. A failed step adds a
`destructive` circled cross and the word "failed"; the step in flight spins a
muted loader. A transcript's scroll-to-end button sits in its scroller's
bottom-right corner (`TranscriptScrollButton`), never across the column.
`TrailTurnGroup` and `TurnStepGroup` are the reference.

Inbox row. A 6px `foreground` unread dot in a 12px leading gutter, level
with the title's first line (it shrinks to 0.6 as it fades over 150ms when
the row is read; a folder row's dot sits in the same gutter and shows only
while the folder is collapsed), so unread reads as one column down the pane.
The title is ink in both states, semibold while unread, two lines at most.
Under it the meta line: a 14px icon for what happened (assigned, mentioned,
status, doc created or edited, questions, memory, a play run), muted, or in
its status hue for a notice that carries a state (a crash or 5xx over the
limit `destructive`, their recovery `success`, a play waiting on an answer or
a ticket due tomorrow `warning`), the summary, and the mono time on the
right. Its focus ring draws inside the row (`inset-focus`), where the list's
clip cannot cut it off.

Bot message. A bot's post sits plain like a person's, under the
name and avatar it posted with (the Nexul glyph on a `muted` disc without one) and a `BOT`
outline tag, running the column's width when it carries an embed. Each embed is a report on
the panel, no box: a 3px rounded leading edge, 16px of indent, then the
heading, the description, the facts and a footer line. The server drops the sender's
color, so the report's state is read from its title's words (`embedTone`:
failed, healthy, stopped, running and their kin, worst first) and colours the
edge and an icon before the title in the status hue, the way a
topology node shows its status; a title that names no state takes its tally's
worst problem (a summary with a stopped cache reads as a warning), and with none
the edge is `muted-foreground` at 40%. An author line that only repeats the bot's name is dropped. Closed,
the facts read as sentences (`embedFacts`): short inline fields share one
13px line, each label muted then its value in mono, dot-separated; a long or
full-width field takes a line of its own, six at most; three or more fields
whose value is a state word become one tally, the problems by name worst first
and the rest counted ("● cache stopped · 2 redeployed · 6 healthy"); fewer
keep their dot on the fact line, and a long fact on one line is mono like the line's values. The last line leads with the ghost fold toggle, the footer text and time trailing right as one mono
11px line ("9 Oct 16:02", "Today 16:02"); the toggle reads "Show all 9 fields" for a tally,
"Show 6 more fields" past six long ones, "Show the rest" for a long
description, which shows its first six lines whole rather than a clamp that cuts the last or ends it in an ellipsis); opened, the fields lay out as the sender's grid, each name in
muted 12px sans over its value, as the fact line names it, inline ones three
across (two in a narrow pane). A numbered line's marker takes tabular figures,
so the lines start level.
Past two embeds the same toggle opens the rest. Only a bot's
text renders Discord's markdown subset. `BotMessageRow`, `EmbedCard` and
`EmbedFactLines` in `web/src/components/chat/` are the reference.

Reference chip. A ticket, doc or person mention is one pill (`mention-chip`):
its leading icon rides the label's baseline, lowered to the cap height's
centre, so it stays level with the text wherever the pill lands on a half
pixel. A ticket's text follows the workspace's mention format; fields side by
side read with a `·` between them and an empty one is left out
(`mentionChipText` in `client-core/`).

Time code. A `<t:…>` code in a message, a thread or a body is a live
reference chip (`mention-chip`): a clock and the time in the reader's zone,
the full date, time and zone in its tooltip, a relative one kept current. A
click opens a popover with the full time, the code in mono, then Copy code
and Add to calendar as outline buttons. In a body the code stays in the text
behind its chip and shows raw while the caret is inside it; in inline code it
is never a chip. On the phone it is an underlined time that opens a small
popover with the full time, the selectable code, then the same two outline
buttons, Add to calendar opening the system's new-event screen.
`DiscordTimeChip`, `DiscordTimeCard`, `DiscordTimes` and the phone's
`DiscordTimePopover` are the reference.

Paired computers. One row per computer straight on the Run order card, under the Your computers tab: its mono rank (`#1`) and grip, a chevron and the name, then at most one outline button for what needs doing and the `…` menu, which asks before Remove. Under the name one line of `SettingsStatus`, the state that needs doing first. A computer added with its app (a personal runner) reads each lane on its own: Online or Offline (with when it was last seen) for the computer, then T3 Code answering, Open T3 Code for a closed desktop app, Re-pairing (`info`) while Re-pair now runs, or why T3 Code is not answering; its one action is Add this computer again once its app is gone, Re-pair now when a pairing failed or ran out, else Set up or Update skills, and its menu holds Set up or Re-run setup, Re-pair now, Rename, then Remove. Until the app reports a hostname the row reads "New computer" in muted text. A computer paired through a tunnel or by URL keeps its old row until it moves: the connection, an expiry warning and the setup state, with Pair, Re-pair (the pairing-link form), Set up and Update skills. The name folds the rest open in a `surface-2` well: address, harness version, paired-until date, one line per provider, and the MCP token with its Replace and Revoke. A computer reached through its runner never shows its relay address: the well leads with T3 Code's state as a status line (port, version and install in mono after it), and its facts follow as more label rows (provider CLIs with their sign-in status, T3 projects, the computer, git identity, GitHub sign-in with a warning when there is none, free disk, versions, when they changed); `ComputerFactRows` is the reference. A computer of yours trails its name with "0 of 5 play runs" in tabular figures, names its set-up providers by their marks after its setup state, and its well opens with **Runs at once** as a 1–5 segmented control and **Chat folder** as a select over one muted line; each label sits on its value's baseline. Add a computer is one dialog in two steps. The first puts the command beside the live state, the command in the site's install snippet (a macOS / Linux | Windows segmented control, one raised row (`canvas-card`) whose line scrolls under a faded edge with the `brand` copy button inside, then a one-line muted note and the guide link; no terminal chrome) under one muted paragraph on what it needs (sudo on Linux and macOS, no administrator on Windows, and the account it installs for), and the state as the two-tile connection panel over a Checks card of three check rows (an icon tile, the name over a mono line of what it found, the state as a dot and a word): Computer connected, T3 Code found, Paired. A check that cannot pass says what to do in its mono line. Below 42rem the checks drop under the command. The footer's Set up stays disabled until Paired passes, then opens the setup step. The setup step is one column: the run's state over a 2px `brand` bar filling as providers confirm, with its one action, Set up all (brand until the computer is confirmed, then outline, while Done takes the brand); then one row per provider T3 Code lists, each a switch, the provider mark, the name over a muted line with the state glyph (Not set up yet, the running status, Confirmed, the failure, or "Off · plays leave it out"), its model picker, and one thing at the end: Set up or Retry (outline) while it waits on the person, a ghost Play icon to run it again once confirmed, nothing while it runs or is off. The first row waiting on the person is lifted (`accent` at 40%). The name opens the row's transcript in a well under it, the running or failed row opening by itself. Under the list, one muted line picks the folder the turns run in and one says what they install. The switch saves at once; Done keeps the models and folder. A tunnel computer still pairing keeps its Tunnel and Pair T3 Code step tabs, its commands in the same snippet. `ComputerRow`, `RunnerComputerRow`, `AddComputerDialog`, `ComputerChecks`, `SetupStep`, `SetupProviderItem` and `CommandSnippet` are the reference.

Run settings. The Computers page's Projects tab is one bordered list: Defaults first (a muted sliders tile), then every project of every workspace with no group headings (its `ProjectMark` at 28px), each row the name over one muted line of what its runs use ("Uses defaults" until it changes something) and a chevron. A row folds its editor open in a `surface-2` well capped at 42rem: muted labels in an 8rem column beside full-width controls, Computers as a Your order | Custom segmented control over the ranked list, and Save at the end, outline until something changes. With no computer online the model shows what is saved, never a text field. `RunSettingsRow` and `RunSettingField` in `web/src/components/settings/` are the reference.

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

Logs view. The stack page's Logs tab is a line tab row of the stack's
services, each label led by its 6px status dot so a service that is down is
found without opening every tab, over one terminal-style block per service: a
mono timestamp column and the raw line, wrapped lines indented under the text,
dense rows. The block reaches down to the panel's bottom, never shorter than
30rem, and while it has no lines its one sentence (and a Details disclosure
with the server's words) sits centred in it. Above it sit All / Errors, then
Pause, Copy and Download as 32px icon buttons with tooltips (Pause is off
while the stream is not live, Copy and Download while the block is empty, All /
Errors while no line came and none is coming), on one row with one mono status marker at the
end, always one word with the reason in its tooltip: Live while following,
Paused after the button, Scrolled up once scrolling released following, and
the connection's own state (Offline, Ended) ahead of those. Errors means a
line that looks like one by its text, from either stream. A thin `destructive`
gutter marks every stderr line and is the only color in the block. A solid
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

Navigation. Three controls move between things, each one component with one
look, and the job picks the control, never the page:

- Section nav (`SettingsSectionNav`): a page of five or more sections, each
  its own job and its own path segment (Your settings, Configuration, Project
  settings, a stack's Settings tab). The sidebar's row grammar at settings scale: muted rows,
  the `bg-accent/60` hover lift, and the sidebar's sliding block (the current
  row's tint with the 2px `brand` edge on its left) on the current one. A side column
  from 1024px, a scrolling top row below it, where the block drops its edge.
  Danger zone is a plain row set apart at the end by a hairline (a gap in the
  top row): the red lives on its card, and a red label at rest pulled the eye
  to the rarest action on every visit.
- Line tabs (`PageTabs`): two to four separate jobs inside one view or
  section, addressed by the path (below). A neutral row on a hairline with a
  `foreground` underline that slides to the active tab.
- Segmented control (`ToggleGroup variant="segmented"`, size `xs`): two to
  five views, filters or modes of the same content, held in the page and
  never in the path: Connected / Not connected, All / Errors, Doc |
  Questions, Fields / Paste .env, Follow | Question, a sort. A `surface-2`
  well with a hairline ring and a 2px inset, holding a raised thumb (`card`,
  the hairline ring, `shadow-card`; the `input` ring in dark, where some
  palettes paint `card` and the well alike) that slides to the chosen side;
  the items are muted text, the chosen one `foreground`. It is radios to a
  screen reader. The permission level strip is not one: its rungs fill up to
  the level (Permission rows).

Never two of one kind stacked: a section holds tabs, tabs hold a segmented
control, and nothing nests deeper. The one exception is a stack page, whose
Settings tab holds the section nav (Web decisions).

Tabs. A view whose cards or sections are separate jobs (two or more of them)
splits them into tabs instead of stacking them; a single-card view gets none,
and a left section nav stays as it is, the tabs live inside the section. Tabs
are `PageTabs` in `web/src/components/PageTabs.tsx`: the line tab row above,
scrolling sideways at narrow widths, the active tab the last
path segment (`/settings/connectors/github-app`) with the query left alone,
the first visible tab when the segment is missing or unknown. The first tab
has no segment of its own. A tab the viewer lacks permission for is hidden,
not disabled, and a view left with one tab drops the row, unless it has a `trailing` control: a switch that belongs to the whole record rather than one tab (a play's Enabled) sits right-aligned on the tab row and keeps the row. Anything that
deep-links into a tabbed view builds its path with `useTabPath`, never by
hand: `tabPath("versions")` names a tab and `tabPath()` the first one. A tab never opens blank: a section that renders
nothing when empty says so in an `EmptyRow` instead. A filter that narrows one
list (`ConnectorsSection`'s Connected and Not connected) is not a tab; it is a
segmented control.

Doc questions. A doc with a clarification heads its page with a "Doc |
Questions N" segmented control (not `PageTabs`: the two are views of one
record, and the choice stays in the page, never the path), N being the questions waiting in mono. Questions takes the article's
place in the same card: a "Questions" heading over one state line (the trail
icon, a medium label, a muted detail), the next action trailing right for
people who may close it, then one `QuestionSection` per round of numbered
`QuestionChecklistRow`s, the same checklist the Interview page uses. Only the
open round ends in its "Anything else?" box; an earlier round shows what was
written and its reply under its header, folded or not. The page opens on
Questions while some wait. `DocQuestionsPanel` in
`web/src/components/doc/clarification/` is the reference.

Interview. The page caps at 68rem; each run's state sits beside its own
action (the memory's with Regenerate, the audit's with Audit via AI, on its
own row once written). Two tabs, Questions and Memory.
Questions opens on a meter, one 4px segment per question grouped by round
(answered ink at 55%, skipped at 20%, a draft or suggestion waiting `warning`
at 45%, unanswered a hairline, the open one `brand`), over one mono line from
the same partition: answered, skipped, and a "need you" button that opens the
first, and a key naming each kind of segment on screen. The rows are the shared checklist: one glyph each (a check, a dash, or
the number with a `warning` dot while a draft waits), titles that wait on the
person in ink at 500, settled ones muted, and the open row as the current row.
A draft names its sources as neutral chips that open them. A suggested change
is a `surface-2` well: Now, then each pick or sentence it adds (+, `success`)
or drops (−, `destructive`). From a 60rem container the meter takes the grid's
first row and the sources sit in a 20rem rail beside the list, its header a
section header's height so the two rules meet, sticky with its own scroll, grouped Follow and Question
under microheaders, each row's citation count giving way to its actions on
hover and focus. A chip hovered lights its row there, and a row hovered lights
the questions that cite it, only while the two sit side by side. Below 60rem
the rail stacks under the list. `ProjectInterview` and `InterviewSourcesSection` in
`web/src/components/memory/` are the reference.

Sidebar. Straight on the canvas, never a panel. Top to bottom: the logo row,
the workspace switcher (a 32px initial tile), then one scroll: Search and
Inbox, the project (its switcher leads with the project mark at 28px, then
its pages, or while its setup is open one Continue setup row, a plain nav
row with a small `warning` dot trailing the label (on the rail, on the
icon's corner), and a muted Being set up line for whoever cannot change the
project; while the workspace's access setup is open, an Owner keeps the
project and its pages with a Set up access row trailing them, and a member,
who sees no project until it is done, gets one Projects being set up line), the foldable Workspace section (folded until a browser first opens it, so the conversations start above the fold), and last the conversations:
Channels, Voice channels, Direct messages and Threads. Places come before
conversations because their length is fixed, so a long channel list never
pushes Board or Runners below the fold. Section labels are 12px medium sans in
sentence case at 80% of `muted-foreground`, with their create `+` trailing; a direct message with one other person leads with
their avatar, a group with the people icon, and a private channel's lock
takes the place of its `#` or speaker. A row with unread messages sets its
name in medium `foreground` and trails the count as an `UnreadBadge`, mono
`tabular-nums` in `muted-foreground` with no disc (on the rail a dot
above the icon's corner over a canvas cut-out, the count in its tooltip). The Automations "WIP" tag
is a microheader in `muted-foreground`, no hue. Rows are `nav-row`s: muted at rest, the
`bg-accent/60` lift on hover, `bg-accent` while pressed, never a scale (a
full-width row scaling from its centre shifts its text). The scroll fades
out over its last 40px under the account row (`scroll-edge`) instead of
cutting a heading in half, and its bottom padding clears the fade at the
end; once scrolled, its top fades the same way. The account row sits under the scroll on a hairline, the gear beside it
named by a tooltip. The icon rail (below 1024px or collapsed) is 68px, so
every icon, tile and avatar keeps the x it has in the open sidebar; it keeps
every page in the same order, with hairlines between the groups, and stands
one Chat link with a dot for unread messages in for the conversation lists.
Its logo is the way back out: under the pointer or focus it turns into the
expand glyph on an accent tile (120ms crossfade), where a separate button
crowded the logo and hung over the panel's edge. Every rail item names
itself in a tooltip to its right (`RailTooltip`, on the app's tooltip clock,
so the next item along the rail names itself at once), and every badge is one 8px dot above
its icon's top-right corner, cut out by a canvas ring (`railDotClass`): `foreground` for unread, `warning` for setup still open, the count in the tooltip.
The switchers work the same on the rail: the tile opens the same menu.

Command palette. ⌘K (Ctrl+K elsewhere), or the sidebar's Search row, opens
one palette from any signed-in page: a frosted overlay (`glass-popover`) 40rem
wide, hung at 14% of the viewport so a changing result count only moves its
bottom edge, over the dialog family's scrim. Opened from the Search row it
runs the dialog clock (220ms in, rising 6px from 0.97) and an overlay click
closes it in 150ms; opened or closed with a key, or closed by a pick, it shows and goes in
the same frame, since it is driven a dozen times an hour and a fading scrim
would swallow the next click. A row or group that joins the list after it
opened (a search landing, a letter that matches more) fades in over 120ms,
opacity only. A combobox input heads it, the results below, a footer at its foot. Empty, it
leads with what changed lately in the current project (each row with its
status or age), then every page the sidebar reaches under the same
permissions, every project's board and the create dialogs that already exist,
under microheaders. Typed, it is one list: up to three local rows (pages, boards,
settings, actions, the other workspaces, the light or dark switch, the
palettes) whose label starts with a typed word, then the workspace search's
(`GET /api/search`) title hits by kind (tickets, docs, pull requests, stacks,
automations, channels, memories) in the server's order, then every other local
row that matches, then the tickets and docs found only in their text under the
one visible microheader, In text. The typed runs carry no visible heading and no
gap, one even pitch, only an `sr-only` name for the group. A row is the muted icon, the
title, and under it one 12px muted meta line: a ticket's mono key and its
stage's dot beside the column's name, a pull request's `repo#number`, the
kind in words for everything without a key (Doc, Stack, Theme), then for a doc
or memory the project's name when it is another one and its age in mono, a
settings row's place, and a check (named "current" for screen readers) on the theme
in use; on the active row the meta line takes the row's ink, since a 70%
step fell under 4.5:1 on the palettes whose accent ink is a hue. A row with no
meta (a page, an action) is one 36px line; a row with meta is 48px. The enter mark is reserved on every row and shown on the active
one, which is the cursor and not the current page: the hover lift
(`bg-accent`), a 2px `brand` edge and a `brand` icon. The highlight jumps,
never slides: the palette is driven a dozen times an hour. When a search of two
characters or more finds nothing, one active row offers New ticket with the
typed words as its title over a muted line that says nothing matched. The
footer counts the rows the typed list shows ("7 found", or "More than shown ·
add a word" when a kind had more), then the Enter and Escape keys. The list fades
an edge with more rows past it, 24px at the top and 40px at the bottom (a
scroll-driven mask, so a list that fits has none), keeps the cursor clear of
either fade, keeps its scrollbar gutter out of the right padding so a scrollbar
arriving never moves the rows, and keeps its last rows on screen while a search
is on its way. Arrows wrap, Enter opens, Escape closes and hands focus back to
where it was; the count is announced once a search settles. `CommandPalette` in `web/src/components/command/` is the
reference.

### Web decisions

The web app's pattern decisions, each with what it was picked over.

| Decision | Why |
|---|---|
| On the board card a label is a pill in its own hue, and the type stays its icon (2026-10-29) | Reverses the neutral chip for the card only, on the owner's call: with the type reduced to an icon and the key on a mono eyebrow, the labels are the card's only tinted shape, so they no longer compete with a type pill, and a hue read at a glance across a column beats a 6px dot. The tint stays at 15% under the hue's ink, so a label stays quieter than the title. The ticket page and the phone card keep the neutral chip until they are brought in line |
| The agent turn: an outcome mark over a rule, every step one row of one weight, commands listed like files (2026-10-29) | Against the fold as shipped: each command took two rows ("Ran a command" over a well holding the whole command), a leading `cd` into the run's directory pushed the command itself off the clamp, commands never grouped while reads and edits did, a tool was a bare line while a command was a box, and the muted 12px header weighed less than the body under it. One row per step and a count over neighbours keeps a run of twelve commands to about thirteen rows. The done mark stays muted rather than `success`: a long run is a column of turns, and a check in a hue on each read as a status board; failure keeps its hue. Rejected: a task card per turn (still the box the fold replaced) and a timer that moves through steps known up front (the Agent does not know its steps ahead) |
| Docs opens on an overview, and its rows are one line with a time (2026-10-29) | Against "Select a doc" beside a list of titles alone: at 1440 the open pane gave about 900px to one sentence, the list could not say what moved lately, and a doc a play waited on hid in a collapsed folder. Rejected: a column table (it lost the folder and creator at 1024, where the list beside it already shows title and time) and a narrow column led by faces (it left the wide pane empty) |
| A doc's outline is a right rail, and its actions sit on the breadcrumb's line (2026-10-28) | Against the 12rem outline to the body's left: beside the app sidebar and the docs list it was a third navigation column, it started the body about 220px right of its own title, and its entries wrapped. On the title's line the actions pushed the title onto two lines at 768 and 1024 |
| Rich-text `h2` at 20px and `h1` at 22px (2026-10-28) | Against 18 and 20px: an 18px `h2` over 16px text did not step up, so a runbook's sections read as one block |
| A label is a neutral chip with a 6px hue dot, and a ticket's type is its icon in the type hue beside plain text (2026-10-26) | Against a filled pill in each label's and each type's hue: on a card the labels were the loudest shapes, louder than the title, and a violet, green or cyan label sat beside the palette's own accent (a purple chip beside a purple Create button) so the accent stopped meaning "act here". A hairline pill on `foreground` 4% with muted text keeps the tag shape and the label's identity on the dot alone. The type is a bounded enum like status, so it takes the status shape: with the tint gone, a violet "feature" no longer reads as the accent on a chip, and the icon is the only coloured part of the card's type |
| A dark card sits at least 0.029 of OKLCH lightness above its panel (2026-10-26) | Against each palette keeping its own card step: the card sits straight on the panel, and the panel is the card mixed 60% into the canvas, so a palette whose card is only a few points above its canvas puts the card 0.012 to 0.020 above the panel and it dissolves into the page. 0.029 is the step Ember, Ocean and Grove hold (0.030), the default holding 0.039; a palette short of it lifts `card` (and `popover` where the two share a value) and keeps its hue and chroma. Light palettes are exempt: a white or near-white card has no lightness left to give, and the ring and shadow separate it. Guarded per palette in `themeContrast.test.ts` |
| A board card's type is its icon alone (2026-10-28) | Against the icon beside the type's name: on a card the name was a third kind of label in the meta row, beside the mono key and the label chips, and it took the width a second label needed; the type is a bounded set whose hue and icon already tell bug from feature, and the name stays in the tooltip and for screen readers |
| Board cards: the key as an eyebrow with the person at its end, the pills on a row of their own (2026-10-28) | Against the avatar-beside-title card, a card with a ruled footer, and the person opposite the pills: with the avatar gone from the title row the title wraps a line less, the key reads first the way people quote it, and the key's line had room the pills' row did not, where a 20px avatar pushed a third chip onto another line in a 220px column; the footer rule made every card taller for a line that spacing already separates |
| Board: every column on a well, cards raised on it (2026-10-28) | Against no well at rest, each column taking it only while a card is held: the cards and the columns shared the page panel, separated by a hairline, so in dark the board read as a wireframe and in light as white boxes on white. Three surface steps (panel, well, card) carry the depth, and the card drops its hairline in dark, where its fill already separates it. This reverses the earlier call that a well under the cards made three nested surfaces: on the board the nesting is the structure |
| Others' chat messages as plain text, yours as the bubble | Against bubbles for everyone: a channel of bubbles was a column of boxes on a panel; plain text under the name reads as a conversation and leaves the ember bubble to mean "you" |
| Chat in a 48rem column | Against the full panel width: at 1440 your reply sat over 1000px from the message it answered |
| Stack status as one well split in three | Against three separate stat tiles: the tiles were three more boxes inside the panel and had no room to list the services, which are what the stack is |
| Services rows straight on the card, the last deploy in deploy words | Against the rows in their own bordered box with a tinted footer (a box inside a card, and the strip made one muted line look like an action area), and "Healthy" for a deploy that came up (green beside a Health fact reading "db-migrate down", the strip contradicted itself) |
| Deploy steps beside the log | Against the steps over the log: beside it the timeline stays in view while the log scrolls, and the log gets the panel's height |
| Doc body straight on the page panel, like the ticket body | Against a sheet (`bg-card`, ring, shadow, page margins) inside the panel: the page is already a raised panel, so the sheet was a panel inside a panel, its frame the heaviest line on the screen, and it cost the reading column 48px a side; the body now starts at the title's left edge and the measure alone gives it its shape |
| Board: the identity header comes back, and the page stops capping the board (2026-10-29) | Reverses the two rows below, on the owner's call after living with them: the 16px sans title inside the toolbar row left the page with no name on it, so the first thing the eye met was a search box, and the cards, the stage names and the lane names all sat at 13px with nothing above them. The 1280 measure fixed the wrong end: it letterboxed a four-stage board while an eight-stage board still sat on the 180px floor, so the cap belongs on the track (`minmax(180px, 280px)`) and the page takes the panel. The project mark and the display face cost 40px above the first card, which the old board already gave back by never carrying a stage strip |
| Light `warning` is ochre, and a bug's icon is rose (2026-10-29) | Against amber-700 and red-700: in light mode both sat within 15° of the `brand` ember's hue at its lightness, so In review's dot and stage-bar segment and every bug icon read as the accent. `#8a6100` (hue 80, 5.5:1 on white) moves warning toward the dark mode's `#fbbf24`; rose-700/400 keeps a bug red without borrowing `destructive`. Rejected: a per-palette warning, since no palette overrides the status hues |
| Errors are crimson, and a bug's icon is fuchsia (2026-10-29) | Against the coral `destructive` (hue 27) that sat 8° from the `brand` ember (hue 35), so an error line read as the accent: crimson (hue 15) holds 5.4:1 on the dark panel and 5.9:1 on white. A bug's rose would then sit on the error hue at the same lightness in light mode, so the bug icon moves to fuchsia (hue 323). Rejected: rose errors with the bug moved elsewhere, and pink for the bug, which is the design type's |
| Board empty cells take half the well (2026-10-29) | Against every cell at `well-shade`: on a real board most columns are empty, so the heaviest surface was drawn mostly where no work was, and four void columns read as more of the page than the cards did. Half keeps the footprint, the drop target and the three-surface order, and leaves the full well to a cell that holds cards |
| Board lane band carries a progress line again (2026-10-29) | Against the count alone: "4/11 done" in muted 11px on a 32px band is a clause you have to read, and at 1440 a lane's progress was invisible until you found the number. 96 × 4 of `success` over `muted` is the one shape in the board that answers "how far along" without reading, and it sits on a 200px reservation at the board's visible right edge, so it reports on a collapsed lane and never scrolls away with the last stage. Rejected: 64 × 3 (too small to read as an instrument at 1440) and the old per-stage legend (it counted the project beside each lane's own count) |
| Board title in the sans on one line with its count (2026-10-28) — reversed 2026-10-29 | Against the 24px Fraunces title over a mono count line: on the one screen people keep open all day the serif was a display moment above the work, and the two lines put the first card about 225px down; one 16px line brings it near 190px and leaves the display face to the pages that are read, not worked in |
| Board header: the title and its count line only (2026-10-28) — reversed 2026-10-29 | Against the five-colour stage bar beside the title, then a stage legend without the bar, a full-width stage strip and a stat row of stages: the bar was the loudest shape on the board and competed with the project's name; the legend's project-wide counts (Backlog 28) sat beside a lane's column counts (Backlog 6) and read as a contradiction; the strip and the row cost 50 to 80px of height on the page where height is cards. The breadcrumb and the 36px project mark went too: the sidebar's switchers show both, and the green mark was the largest colour block in the header, louder than Create |
| The board is one grid: stage row once, lanes as bands, Done folded (2026-10-28) | Agreed by a review council after six score-driven rounds left it at 6: each lane had drawn its own five-column board, so the stage headers repeated per lane, a one-card stage stretched to the tallest stage in its lane, one lane filled 1440×900, and Done took a fifth of the width. Against dropping lanes for a category label (lanes carry per-category collapse and drag to recategorise) and against no wells (with lanes as bands the stages need their columns). Replaces the folded 72px and 128px rails and the per-lane column headers |
| The ticket page keeps its standing on screen: a header strip in the board card's marks, the rail beside Ticket and Thread, Details below 52rem (2026-10-28) | The blockers, the pull request and the test result sat below the fold of a rail ordered by data type, the rail fell under the whole body at 768 and left the Thread tab entirely, and the meta line repeated the status while leaving out what the card shows. Against one scroll with the thread under the body (the thread is its own virtualised scroller with its own path, so the body shows its three latest lines instead) and against moving the rail inward off the panel edge (it stays where every issue page puts it) |
| An empty column is a 72px rail with its name in two lines (2026-10-28) | Against a 160px column with its full header (two of them still took a fifth of the board to say No tickets), a 40px well of mark and count (In progress and Testing could not be told apart), a 40px strip with the name set vertically (the rotated names were the hardest text on the board to read and broke the stages' left-to-right sweep), an empty column at the full width, and narrowing a stage only when every lane is empty: with all five stages on screen, In progress and Testing were two 220px columns of "No tickets" splitting the work into islands, and a stage empty in every lane almost never happens on a real board. Narrowing per lane lets the columns of two lanes sit at different x, which the repeated column headers in each lane make readable |
| Board lanes as 32px bars and columns that fit the panel (2026-10-28) | Against a lane header ruled off with a 56px done bar and 288px columns capped at five cards: the columns scrolled inside the page, so a lane showed four cards and the fifth stage sat off the right edge at 1440. Flexible columns down to 220px fit all five stages, and the page scroll carries the cards |
| Palettes theme the accent and the field | A palette's primary becomes its brand, so Ocean is blue and Grove is green everywhere the ember was, field included |
| The light field is derived from the palette, never a fixed hue | The blue and pink glows of the first field showed in every palette: a cool wash behind the sidebar of Ember, Clay and Grove, a magenta bloom beside a teal or blue brand on sign-in, a taupe stain on Zen from a near-black brand at 30%. Each glow is now a relative colour (the brand's hue and chroma at a fixed lightness, so a grey brand adds nothing) or a lift of `card`, held to 40% alpha or less. Rejected: a per-palette field override (a value to keep in step with every palette) and a mix of the brand with the canvas (a dark brand darkens a light canvas into a stain) |
| The live light field on showcase surfaces only | Picked over a fluted-glass refraction (busy vertical bands fought the text and the frosted vocabulary, and no CSS still could stand in for it) and an aurora (a band across the top only, nearly invisible in light mode); the field is the app's own light field moving, so its fallback still is exact and every palette retints it |
| The live field on the signed-out home and sign-in only | It repaints every frame while the page is open and the library's frame cap is out of reach from React; a wizard or an invitation stays open for minutes of form work, so there the still does the job at under 1% CPU |
| Empty states: the orbit mark | Picked over a placeholder card grid (generic, implied an add action the callers do not have, and looped) and a tile of the live field (a GPU canvas inside an everyday panel, grainy at 96px, impossible at compact size); the static orbit carries the field's colours at any size and costs nothing |
| Loading: the orbit at spinner size | Over the plain spinner and a gradient arc; it ties loading to the empty mark, keeps the ember to progress, and stays one small SVG |
| Long page titles step down to 24 and 20px and clamp at three lines | Against stepping alone (a sentence-long ticket title still ran six lines at 20px and pushed the body below the fold, and nothing bounds a title's length) and clamping at 28px alone (three lines held about half as many words); the step keeps short titles at full voice and the clamp bounds the rest |
| Bot embeds as status-edged reports, facts as sentences and a state tally (2026-10-28) | As cards (fill, ring, shadow, a label grid and a footer strip) two deploy posts took about 400 of the 750px a channel shows at 1440, a summary spent a 3×3 grid saying "healthy" to report one stopped cache, and the people sat in the bottom third. Without the box and with the facts on a line a deploy is about 90px and a summary names its one problem. Against folding every healthy post to one line behind a chevron (the facts are what people read in a deploy channel, so it traded height for a click on every post) and against dropping the edge (with the box gone it is what marks where a report starts); the leading edge is the topology node's status signal reused, picked earlier over a 2px bar across the top, which read as the accent's decoration rather than state |
| The command palette is one list of two-line rows, its meta under the title, its typed runs at one pitch (2026-10-29) | Against a microheader per result kind ("deploy" spent five headers on six hits, and the row showed only a title and a key while the search also returned the status and the kind), a one-line row with the meta right-aligned at 36px or in fixed 32px columns (the status and kind squeezed long titles, and a fixed rail cut a settings row's place), and the old row with only its baseline fixed. Two lines show every title whole on the first and the key, status and kind on the second, at the cost of about two rows in the 26rem list |
| List pane placeholder: icon, count and the palette shortcut, compact | Against the icon alone (said nothing about the list) and the full empty-state size with a Fraunces headline (read as an empty page beside a full list) |
| The Inbox row: an unread gutter, the title, then the kind icon, summary and time on one meta line (2026-10-29) | Against the 28px tile with a dot on its corner, row dividers and the time beside the title (about 175px of title at the 300px default, so most ticket titles clamped mid-sentence, and unread sat in two columns with the folder dot); the icon on the meta line gives the title about 260px and a crashed service reads in its hue at a glance. Rejected: the icon beside the title (35px less title), a 16px icon centred on both lines (floated off both), a single-line title (cut most titles in half) and a full-width list until a row is picked (the list jumped to 300px on every first open) |
| A doc's first heading that repeats its title stays | The title and the body are separate fields; hiding a matching heading in a collaborative editor would put the caret in invisible text and show readers and writers different docs, so the author's content is shown as written |
| Runners: one list, a `well-shade` band per machine, pinned while its rows scroll, over one-line runner rows on shared tracks, a column head row, the fleet's health in the header and the queue above the list (2026-10-29) | Against one card per machine (a server tile, three header actions and a People row on every machine put a one-runner machine at about 200px, columns were laid out per card, and the queue sat in a side rail that fell to the bottom of the page at 768) and against bands separated by space alone or a heading over a box per machine (both read as separate pages); the stat well's four figures moved into the header's health line, the trash column and Import into `⋯` menus, and "just now" was dropped, since a connected runner's last seen always reads that |
| Topology in lanes, read at a legible zoom (2026-10-29) | The fitted canvas sat near 0.5, where the mono route rows rendered at 6 to 8px, and its automatic layout scattered the hostnames, gateways and services so the left-to-right sentence only held inside each gateway card; at its worst the fit still cut hostnames and drawn nodes off both sides. Fixed lanes, pills level with their rows and a 0.8 floor keep the sentence whole. Against a docked route index listing every hostname (hostnames shown twice, which the hub rows were chosen to avoid) and against anchoring the camera on one lane (the whole sentence fits at 1440). The gateway, service, pill, drawn-node and frame shapes were picked from screenshot variants |
| Topology service cards: the name on its own line, status and strategy under it, the machine under a rule (2026-10-29) | Replaced the icon tile with the status trailing the name, which squeezed long names at a fixed lane width; the tile stays on the gateway card, the one card whose icon says something |
| Topology network boxes: opaque over the dot grid, and the machine every member shares named once under the box label (2026-10-30) | Against the translucent box, whose dots read as texture behind the cards, and against repeating one machine on every card in the box, which cost a ruled row per card and said nothing new |
| The setup row is a nav row with a warning dot | Against an accent-filled row (it outranked every page title, read as the current page, and sat beside the destructive red on the stack page); the dot says setup is open without taking the accent's "here" |
| Sidebar rows at 13px, about 28px tall, in a 224px sidebar, with sentence-case labels (2026-10-28) | Against 14px rows near 34px tall in 240px under mono uppercase labels: the labels were the most tracked-out, most technical-looking text in the chrome for the least important words, and the tall rows pushed Voice channels to the fold at 900px; the compact rows sit level with the board's 13px card titles and give the panel 16px |
| Sidebar counts and tags are neutral | Against the solid ink pill and a warning-coloured WIP tag (the brightest things in the sidebar for the least important facts); the number being present says unread, and the sidebar keeps its colour for where you are |
| Sidebar: the Workspace section starts folded (2026-10-28) | Against it starting open: with the project's pages listed, the seven workspace pages (Calendar to Configuration, visited a few times a week) pushed every channel under the account row at 900px, where the last one faded half-drawn into the footer; folded, the project and the conversations fit, and a browser that opens it keeps it open |
| Sidebar: places before conversations, one scroll | With the channels first, Board and the project's pages sat below the fold at 860px and the docked workspace pane took a sixth of the height; with fixed-length pages first and the workspace section in the same scroll, every page is visible at a glance and the variable lists grow downwards |
| Permission levels as a segmented strip per domain | The owner found the trailing level dropdowns harder to read and set than the strip, where every rung up to the level fills and the whole list reads at a glance |
| Segmented control: a well with a raised thumb | Against joined outline buttons with an accent fill (read as a toolbar of buttons, the same box as Copy and Download beside it in the logs) and a ghost row with an accent thumb (the same grey block as the section nav's top row, so a filter looked like navigation); the well and card step is the panel's own depth vocabulary, and it reads as one control |
| Section nav in the sidebar's grammar | Against the grey block alone (the sidebar marks where you are with the ember edge, the section nav with nothing, so the two navs on one screen read as unrelated) and a hairline rail with a sliding ember segment (a table of contents, not navigation, and the rows lost their click target) |
| The Roles tab is a matrix of every area by every role (2026-10-29) | Replaces one row per role with a tally of levels ("Delete 16 · Write 6 · Read 3 · None 10") and a chevron for the areas: the counts changed which levels they listed from row to row and never named an area, and one role opened at a time, so two roles could not be compared. The owner picked the matrix over the tally kept in four fixed columns, as two lines or one, from screenshots on the worst-case workspace |
| Access grid ticks are ink and the ember marks an unsaved cell (2026-10-29) | An ember box per tick put 150 ember marks on a worst-case Public channels tab and buried its few exceptions, and an ember ring on an ember box could not show a change. Picked from screenshots over a hairline box and a bare check: an ink check on a 6% cell tint. Public channels marks the hidden role (eye-off) and leaves a seen cell empty |
| The Access grids fill the panel and scroll in one box (2026-10-29) | The card used to scroll inside the page, two scrollbars, the save strip below the fold and the names scrolling away with the boxes. A sticky head inside a box that scrolls sideways sticks to that box, so the box became the only scroller rather than syncing a separate head; it is paint-contained, because its sticky cells otherwise widened the page by 210px at 768 |
| Danger zones are rows of action, consequence and an outline button | Against the solid red button in the card (the loudest thing on the page for the rarest action, and the confirm dialog repeated it); the red fill is the dialog's, where deleting is the one thing left to do |
| Project settings lead with the project mark | The sidebar and the board already show the seeded mark; a plain initial on a square in General made the settings look like another project |
| Project settings: one nav level, the section as the title, one list row (2026-10-29) | Against the Board section with its own tab strip (two navigation levels, two at 768 stacked as scrolling rows), a "Settings" title over a card named like the nav row, and each list in its own bordered box inside the card with its colour dot left on some lists and right on others, "no colour" drawn three ways and a count only on Categories. One row with a named count on every list says what a column or type holds and why it can't be removed; its `⋯` waits for hover so a dense list reads as names. The project mark stays on General's first row, the crumb names the project |
| Settings cards: a quiet header, settings as rows | Against the old header band (an 18px title over a full-width rule, so every section read as the same generic form) and the title outside the card (heading on the glass, content in the card: two surfaces for one unit, and a paired card's heading wrapped out of line with its neighbour's); a title flowing into its rows keeps the unit whole and saves the rule |
| Settings type: 16px title, 14px label, 12px muted hint | Replaces the 15px title over 14px descriptions: the title sat one pixel off the dialog title and the description matched the label's size, so a card was two sizes and one weight told them apart, while the field hints beneath the controls were already 12px, a fourth look. Against 13px hints (a pixel under the label does not separate the levels, and it would have meant re-sizing every field hint that already reads at 12px); 12px is the size meta text already uses, and the longest hint in the settings pages (168 characters) still reads as a short paragraph at it |
| Settings save from a strip that is always there | Against a strip that opens when something changes (it pushed every card below it 52px on the first keystroke), a bar floating over the page (detached from the card it saves and covering the next one) and a Save beside the field (fits one field, not a card of them) |
| Save answers in its button, not a toast | Against the toast (it lands a panel away from the click) and a Saved line at the strip's left (opposite the pointer); the button the pointer is on turns into Saved |
| Theme and mode tiles are the app in miniature | Against swatch dots, which named a palette without showing what it changes; the miniature shows the canvas, the panel, text and the accent in that palette and mode |
| Paired computers: a folded row per computer, the setup as one column of provider rows | Against a row that showed every fact and five buttons at once (status, address, version, three provider lines, the token, two delete icons that read alike) and a setup step of three panes' worth of controls beside a transcript pane; one action for the state, the rest a fold or a menu away, keeps every capability with a fraction on screen |
| Setup: each provider row carries its own switch, model and run, providers run side by side | A separate Options disclosure hid the switches and models from the rows they belonged to, and one run at a time made a person wait on every provider to try one; a row that ends in exactly one thing (an action while it waits on you, else nothing but its state) reads at a glance |
| Add a computer: the command beside the live connection and three named checks | Against numbered install steps above a single "waiting for connection" line (one line cannot say which part is stuck, and the person only ever runs one command) and a progress bar (it says how far, never what to fix); three checks in the order they happen name the part that is stuck and what to do about it |
| Person dialog: a tab per workspace, changes held until Confirm | Several workspaces stacked in one scroll mixed their controls, and applying each change on the spot made the dialog change under the owner; tabs separate the workspaces and Confirm makes the edit one deliberate act |
| Roles as a checklist with a Can't give group, chips on the rows | Against held roles as chips with an × beside an "Add role" menu of checkable roles (the menu hid the full set and every reason until it opened) and the roles as plain muted text on Team rows (read like any other meta line); every role and why one is out of reach shows at once |
| Role dialog: a browser-style tab per area, not stacked sections | Three stacked sections made a 35-row scroll; tabs keep one area in view under a fixed name and footer. The person dialog's tabs, picked over a segmented control (it read as a filter, its 12px labels beside rows of level strips) and line tabs (the idiom for separate jobs addressed by the path) |
| A stack page is ops tabs (Overview, Deploys, Logs) with its configuration behind a Settings tab, Deploy in the header | Against the settings shell for the whole page: the three daily views shared a 12rem section nav with five configuration sections, so logs and the deploy list got about half the width, the deploy action was the last card under every service, and the history repeated a day header on each deploy. A Settings tab holding the section nav is the one place a tab holds a section nav, because a stack is a record you watch first and configure second |
| Body headings at 20, 18 and 16px, not 26, 21 and 17px | Against the old scale (an `h1` at 26px sat next to the 28px page title and the two read as one level) and 20, 17 and 15px (a 15px `h3` is smaller than the 16px text it opens, so only weight told it from a paragraph); the 1.9em lead-in does the separating the size no longer has to |

### Motion baseline

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
  section or another record inside the same page does not replay it (Tab and
  section swap, below).
- Tab and section swap (`swapIn` in `lib/motion.ts`, through
  `useSwapEntrance`; `PageTabs` and `SettingsShell` carry it): picked by a
  pointer, the new content comes in 6px from the side of the item it was
  picked from and fades in over 160ms `--ease-out`, sideways under a tab row
  or the section nav's top row, up or down beside its column. Nothing leaves:
  the old content is gone in the same frame, so a click never waits, and the
  transform never moves the layout. A list inside it that would cascade
  arrives with it instead (one entrance per container). A key press swaps at
  once; reduced motion is a 120ms fade.
- Lists (`EnterList`): the rows on screen at mount rise 4px and fade over
  200ms, the first eight 25ms apart, the rest with the eighth; a list of 50 or
  more mounts at once. A row added later (a filter or search bringing it
  back) pops in from 0.97 over 150ms; `arrival="rise"` makes it rise 8px over
  200ms instead, for news (the Inbox). A row React only moved keeps still. A
  row opts out with `data-no-enter` (a board card mounted mid-drag).
- Sliding highlight (`ActiveIndicator`): the sidebar's active row, the
  section nav's block, the line tab row's underline (`TabUnderline`, on
  `PageTabs` and the logs tabs) and every segmented control's thumb each own
  one highlight that slides to the active item, transform only, 200ms
  `--ease-spring`. It sizes to the item and plays back from wherever it was,
  so a click mid-slide carries on. Moving between the sidebar's two navs it
  fades out of one and into the other.
- Overlays (`index.css`, Overlays) move on transitions from `@starting-style`,
  never keyframes, so one reopened mid-close turns back from where it is; a
  closing overlay never takes a click; and one opened or closed from a key
  shows and goes in the same frame (the root's `data-input`, written by
  `lib/motion.ts`). Popovers, menus, hover cards and selects open in 150ms
  and close in 120ms from 0.97 and 4px toward their trigger, from the
  trigger's corner; their opacity lands in 90ms, before the scale settles,
  so the sidebar text never reads through a switcher. A dialog rises 6px from 0.97 over 220ms and leaves in
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
- Inline editors (`settle-in`: a role, a category, a branch rule's overrides,
  the new-role, new-type and secret forms): the form that takes a row's place
  or opens under a footer button fades in as it settles 4px over 200ms
  `--ease-out`, the disclosure's own numbers; it closes at once. Reduced
  motion: a 150ms fade.
- A row moved within its list from a menu (a category's Move up or down)
  glides to its new place over the row-removal glide's 200ms `--ease-out`,
  riding over the row it trades with; a drag settles on the same numbers, and a
  move made with a key lands at once.
- Nav rows (`nav-row`: the sidebar, the section nav, the switchers and the
  account row): the hover lift lands at once and lets go over 150ms
  `--ease-standard`, so the highlight is under the pointer the moment it
  arrives and leaves a short trail. Rejected: 150ms both ways (the incoming
  row lagged the pointer as it ran down the list) and no transition (the
  sweep flickered row by row).
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
- Empty state (`.empty-state` in `index.css`): a whole-page one turns its
  orbit into place. The outer ring settles from 0.92 (520ms), the dashed one
  turns in from -40° and 0.9 (620ms), the brand and muted dots swing 75° along
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

### Motion locks: board, chat and ticket

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

Agent's live step: decided 2026-10-15.
- Direction: a highlight sweeps left across the step's muted label, 2s a
  pass, linear, looping while the step runs; steps that join the open fold
  settle in 4px over 200ms (`settle-in`); the loader on the step's row spins.
- Reduced motion: the label sits still in `muted-foreground`.
- Rejected: task cards (a bordered card per turn with the live step as its
  title and the steps on a rail) put a box around every turn of a long run;
  a timeline (the Agent's sentences as nodes, the steps behind an icon count
  line, a dotted orb for the live node) hid the commands a reader checks
  first. Both were built on seeded data beside this one and judged by the
  owner.

### Motion locks: lists, states and feedback

Decided 2026-10-09 without the owner in the loop: each built as two to four
live variants on the real surface with seeded data, scrubbed frame by frame
and recorded at 1x and 0.25x, and judged against the baseline above.

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
fade (the same jump, a frame later). The list's box holds its height until
the glide ends: shrinking it at once let the last row slide up from outside it
and be clipped.

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

Toasts, dialogs, sheets, popovers, menus: on the baseline's clocks, moving as
Motion locks: overlays sets out; no overlay carries an unnamed transition
list.

### Motion locks: overlays

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
language the panels speak). Menus: the glass at 8px with 32px rows on a 4px
corner. Rejected: a solid panel with 36px rows and inset rules (a three-item
menu a quarter taller; the inset rule read as a gap) and the palette's
brand edge on the highlighted row (the accent on a pointer hover). Focus: the
ink outline. Rejected: the ember drawn as a 2px inset border (still read as
an error) and an ink border with a soft halo (a focused field in error was
barely different from one unfocused). Destructive button: white on a deeper
red. Rejected: a tinted outline (too quiet for the act it confirms) and the
coral with dark ink (the ember's twin beside the primary).

### Motion locks: navigation

Decided 2026-10-09 without the owner in the loop, each built as live
variants on the seeded app, recorded at 1x and 0.25x with frame strips at
0.1x, and judged against the baseline.

Sliding highlight (underline, segmented thumb, section nav block).
- Unchanged: transform only, 200ms `--ease-spring`, interruptible from where
  it is; a key moves it at once. Rejected: a crossfade (the old mark vanished
  and the new one faded in, so nothing said where the selection went) and a
  stretch that reaches the new item before letting go of the old (it
  underlined every tab in between for 260ms, ran past the label's own
  change, and on a two-sided control read as both sides chosen).

Tab and section swap.
- Directional: 6px from the picked item's side and a fade, 160ms
  `--ease-out`, enter only. Rejected: none (the card's content popped with
  nothing tying it to the row above), an opacity fade alone (every switch
  blinked to an empty card for a frame or two), a 4px rise (the page
  entrance's vocabulary on an act that is not a page change), and a 2px blur
  crossfade (a filter on a whole card for no gain). Tuned against 8px over
  200ms and 12px over 220ms, which read as travel on an act repeated through
  a whole settings session.

Sidebar collapse.
- A 68px rail that keeps every icon in place, layout swapped in one step,
  labels wiped in by a clip-path on open (220ms `--ease-out`), close snaps.
  Rejected: animating the width (every frame relaid the board and re-truncated
  the switcher names; 4x CPU throttle p95 16ms against 2.5ms for the wipe)
  and gliding the page beside it (moved the frosted panels, which never move,
  and its backdrop re-blur peaked at 130ms frames). Measured on the board at
  1440 under a 4x throttle: the toggle's own render and layout costs 165 to
  640ms whichever variant runs, and is the thing to make cheaper next.

Nav row hover: lands at once, lets go over 150ms (Motion baseline).

Switchers: the overlay clock (Motion locks: overlays), from the
trigger's corner, with the trigger holding its pressed fill while the menu
is open so the menu reads as coming from it.

### Motion locks: settings sections

Decided 2026-10-09 without the owner in the loop, each built as live
variants in Configuration's Roles card and the Categories card with seeded
roles and categories, recorded at 1x and 0.25x with frame strips at 0.1x.

Opening a role's areas.
- The disclosure (box snaps, content fades in settling 4px over 200ms, out in
  120ms). Rejected: a clip-path wipe from the top (a curtain on a short
  reveal, and the lower rows popped in at its end), a 12ms cascade of the
  domain rows (37 rows trickled in over 340ms, a list entrance inside a
  disclosure) and a snap (only the chevron said anything opened). Tuned
  against 2px over 160ms and 8px over 240ms, which were indistinguishable at
  1x and 8px read as a drop at 0.25x.

Opening an inline editor.
- Settle in (above). Rejected: a pop from 0.98 out of the row's top (a
  full-width form scaling shifted its text at the edges), a 4px blur fade (a
  blurred field read as loading) and a snap (a form 600px taller than its
  row appeared in one frame with nothing tying it to the row).

Moving a category from its menu.
- Glide (above). Rejected: a snap (the two names traded places in one frame
  and the eye had to re-read the list to find the row it moved) and a fade of
  the two rows that traded (said which rows changed, not where the row went,
  and the row under the menu vanished from under the pointer).

Removing a row, and saving a card: the shared locks (Removing a row,
Settings save, in Motion locks: lists, states and feedback) apply to roles, plays, categories, repositories, exposures,
secrets and the workspace, template and automation configuration cards. A
Motion `layout` glide on a bounce-free spring was built first and lost to the
shared glide only for being a second vocabulary for the same act.

## Public site

The public site in `website/`: the home page, the roadmap and changelog, the
docs pages and the 404. It is built mobile first at 320, 375 and 414px and
verified at 320, 375, 414 and 768px, the docs pages at 1024 and 1440px too,
in both schemes.

Every site decision below was made on 2026-10-09 without the owner in the
loop: each built as live variants on the real, built pages (the roadmap and
changelog with real release data), shot at 320 to 1440px in both schemes, and
the motion judged from frame strips at 0.1x and recordings at 1x and 0.25x.

### What the site adapts

The site shares the tokens under the app's names, the `type-display`
settings, the accent's roles, ink focus, status hues, the radii, and the app's
sheet, dialog, palette, sliding-highlight and copy clocks. It adapts these,
each for a reason recorded below:

- Reading text sits on the canvas, with no panel and no sheet, and the light
  field shows only beside the reading column from 1280px.
- Prose links are ink with a `brand` underline, not `brand` text.
- The header is the canvas colour with a hairline foot, not frosted.
- Fraunces ships as one pinned file, and the landing pages set their section
  titles in it.
- Code blocks are the app's console in both modes.
- Each docs page is a full page load, so the docs sidebar's active row stays
  still.

### Site tokens and type

Tokens. `website/src/styles/tokens.css` holds the app's values under the app's
names (`--background`, `--surface-2`, `--card`, `--popover`, `--foreground`,
`--muted-foreground`, `--accent`, `--brand`, `--brand-foreground`, `--border`,
`--input`, `--focus`, the status hues, the panel and field values, the
shadows, `--shape-*` and the curves) and maps Fumadocs' `--color-fd-*` onto
them: `fd-primary` is the ink, as the app's `primary`; `fd-ring` is `focus`;
`fd-overlay` is the dialog scrim (the canvas at 60%). Colours are written once
with `light-dark()`; shadows, which change geometry between modes, are
restated for light. The scheme follows the system until the header's toggle
sets `.light` or `.dark` on `<html>`, which a head script restores before
first paint. Selection is `brand` at 32%, as in the app. The ember holds the
app's roles here too: the primary action (the home page's copy button, the
404's Open the docs), the active nav marker (header and docs sidebar),
selection, and the link underline. Pages add no tokens of their own.

Fraunces. Docs titles, the 404 and the landing headlines use `.type-display`
(the app's settings).

- A docs title is 28px on a phone and 32px from 768px. Rejected: 28px
  everywhere (small against the 18px lede under it) and 40px (a showcase size
  on a page people read in sequence).
- The home page's headline is 40px at 320, 44 from 414, 56 from 768, 68 from
  1024 and 76 from 1280; its section titles are Fraunces at 30, 36 and 40px.
- A roadmap or changelog hero headline is 36px rising to 56px from 1100px.
- The 404's "Page not found" is 32px, 40px from 768px.

The face ships as one file, `fraunces-display-latin.woff2`: the Latin build
with weight 560, SOFT 50 and WONK 1 pinned and optical sizing kept, 34KB
against the package's 121KB, so it renders exactly as the app's settings do.
Mobile Lighthouse on the built home page, three serial runs each, median:

| Variant | Score | LCP | Other |
|---|---|---|---|
| Full file, preloaded | 94 | 3.08s | |
| Full file, not preloaded | 94 | 2.93s | FCP 1.95s, CLS 0.035 from the swap |
| Full file, `font-display: optional` | 93 | 3.08s | |
| No Fraunces at all | 98 | 2.25s | |
| Pinned file, not preloaded | 97 | 2.40s | CLS 0.035 |
| Pinned file, preloaded with `swap` | 98 | 2.40s | FCP 1.20s, CLS 0 |

The pinned file, preloaded, is the pick: the same score as no display face,
and no layout shift. Its cost: any weight but 560 renders at 560, which the
type rule already forbids.

### Site chrome

Header. One header on every page: the wordmark, then from 768px Docs, Roadmap
and Changelog as words, docs search (an icon below 1024px, the field from
there), GitHub as an icon button and the theme toggle. No button in `brand`:
the home page's install block is the call to action, and a header button
repeated it on pages where nobody was about to install. The current page is
ink with a 2px `brand` edge along the header's foot. Rejected: the section
nav's `bg-accent` block (read as a button) and ink text alone (the current
page barely differed from the rest). The header is the canvas colour with a
hairline foot, not frosted: text scrolls under it all the time, and nothing
may move behind a blur.

Header icons, per [Icons over words](#icons-over-words). GitHub, theme,
search and menu are each a 44px ghost icon button, muted until hovered, with
an `aria-label` and a native `title` tooltip of the same words. The filled
GitHub mark is 16px beside the 18px stroked icons. GitHub goes before the
theme toggle: the toggle is a setting and ends the row. Rejected: GitHub
after the toggle (the setting landed between two destinations), the GitHub
icon in the phone bar too (four icons beside the wordmark on a docs page at
320px; it fit, but the menu already lists it), and the mark in ink (the
filled shape outweighed every other control).

Phone menu. Below 768px every page shows the wordmark, search on docs pages,
the theme toggle and a menu button. The menu is one sheet everywhere: the
app's sheet, frosted, 8px from the right edge at 12px, up to 320px wide, over
the dialog scrim. On the landing pages it is a `<dialog>` holding Docs,
Roadmap, Changelog and GitHub, as tall as its links; on docs pages it is the
sidebar drawer, the page tree first and the same links at its foot. GitHub is
a row like the others with its mark before the word. Rows are 44px, the
current one `bg-accent` with the `brand` edge; its close mark is an X on both.

Theme toggle. A 44px ghost icon button showing the mode you are in (moon in
dark, sun in light). The palette lands in one frame with every colour
transition held, as in the app.

404. The app's error page: the light field at half strength fading out
toward the footer, the orbit mark around a compass, "Page not found" in
Fraunces, one muted line, then Open the docs in `brand` and Home as an
outline button.

### Docs pages

Reading surface. Docs text sits on the canvas with no panel and no sheet. The
light field shows only from 1280px, at half strength and with its warm and
cool glows only, where they fall beside the reading column (behind the
sidebar and the on-this-page rail). Rejected: the field at every width (at
375px its blue and pink glows sat behind body text), a sheet
behind the article (narrowed the phone measure by 16px a side, and put the
console code blocks in a box inside a box) and a frosted panel for the whole
page (the panel stopped at the rail and doubled the edges on a page that is
all reading).

Links. Prose links are ink with a 1px `brand` underline 3px below, 2px on
hover. Rejected: `brand` text, as the app's prose links (4.3:1 on the light
canvas, under the 4.5:1 body text needs; it passes only on the app's white
sheet) and a 2px underline at rest (heavy in a paragraph with several links).

Docs sidebar. The app sidebar's grammar on the canvas: the docs home as the
first row, above the groups, so it is marked when open; mono microheader
groups, 14px muted rows with the `bg-accent/60` hover that lands at once and
lets go over 150ms, the open page `bg-accent` with a 2px `brand` edge 8px in
from its top and bottom.

Code. The app's console in both modes: `#0b0b0b`, a `#232323` hairline, 8px,
13px mono at 1.65, emphasis from a grey ramp (keywords `#f5f5f5`, strings
`#a3a3a3`, comments and punctuation `#737373`, comments italic). Rejected:
ember keywords (the accent on something that is not an action; YAML keys read
as links) and the app's editor hues (green strings read as success and indigo
as a second accent on a reading page). Astro's own highlighter is off so the
docs highlighter sees each block's language; with it on, every block rendered
as plain text.

Tables, inline code, cards. Tables are the app's ruled data: a hairline under
each row, a muted 13px header over a stronger rule, tabular figures. Inline
code is mono at 0.85em on `muted` with a hairline. The docs home cards are
the app's card (`card`, the panel ring and lit top edge, 8px) under mono
microheader groups. A group's grid takes the most columns, up to two at 768px,
three at 1024px and four from 1280px, that leave no row holding one card; at
two columns an odd last card spans the row. The docs content has no callouts or blockquotes; a
blockquote takes the app's 2px rule in `foreground` at 25%.

Copy page. A quiet outline control (`border` hairline, muted text, 6px) beside
the title.

Search. The app's command palette surface (the popover glass at 12px over the
dialog scrim), 40rem wide, hung at 14% of the viewport from 768px.

### Home page

The home page (`website/src/pages/index.astro`, `website/src/styles/home.css`)
reads the site's tokens and adds none.

Showing the product.
- Real 2x screenshots of the seeded example workspace (Northwind), one per
  scheme, served by `ProductShot.astro` through Astro's image pipeline as AVIF
  and WebP with width and height set; the page's `.light` or `.dark` class
  picks the scheme, so the header toggle swaps them.
- Every shot follows one feature, passkey sign-in, in the Accounts project:
  the hero is its board, and the four steps are its spec, ticket ACC-3, the
  accounts-api 1.4.0 deploy and the deploy bot's card in #engineering with
  the team's replies, each cropped to 640x480 so they show close to 1:1 on a
  desktop. A sign-in feature needs no explaining, and its spec has the edge
  cases (a lost device, rate limits, the email code fallback) that make the
  tickets read like real work. Rejected: unrelated records from the busiest
  project (a runbook, a retry bug, an older deploy, a failed health check),
  which showed each screen but no thread between them. Passkeys won over
  sign-in with an email code alone: here the code is the passkey flow's
  fallback, so one feature carries both.
- People show their seeded avatar fill. The seeded logins carry an underscore and
  a two-letter suffix (`bob_nw`), which no GitHub account can have, so the
  app's `github.com/<login>.png` fallback finds nothing and no real face
  ships.
- The hero shot sits on a field panel (`surface-2` with the light field's
  three glows, the panel ring) and bleeds off its right and bottom edges, the
  screenshot's top-left corner rounded at 8px. Below 768px it switches to a
  480x600 crop of the board's top-left so cards stay legible.
- Weight: the hero is about 16KB on a phone and 36KB on a 1x desktop; the four
  lazy step shots add about 65KB on a 1x desktop and 105KB on a 2x phone.
- Rejected: a muted loop of the app (board, palette, ticket; 85KB at 1x and
  soft on a retina screen, about three times that at 2x, unreadable at 375px,
  motion beside the headline, and the reduced-motion still is the same
  screenshot anyway) and the old three-pane illustration re-skinned in the
  tokens (no bytes and crisp, but it drew a layout the app does not have, and
  at 375px it stacked into a 1300px column). Frames rejected: the shot bare on
  the page (in dark mode its canvas met the page's and the frame vanished)
  and inset on all four sides (smallest text, and read as a panel inside a
  panel).

Accent. The ember is the copy button, the one primary action on the page;
the platform switcher is the app's xs segmented control (well and raised
thumb, 32px tall, a 44px hit area), links stay ink, and status keeps its hues
(the connection and beta dots).

Hero install block. It sits in the headline's column, left-aligned with it,
32px under the lede and at most 480px wide, so the curl command fits on one
line from 768px and the block reads as part of the copy. Rejected: a second
column to the right of the lede (from 1024px it started 240px below the
headline's top, the eye went from the lede into empty space, and the block
floated away from the text it belongs to).

Install command. One line always, the copy button inline. The command is mono
12px under 414px, 12.5px to 768 and 13px above; when it is wider than its box
it scrolls sideways with no scrollbar and its cut edge fades over 32px until
the end is reached. Rejected: shrinking it to fit (about 8px at 320, below
reading size) and wrapping it (breaks the one line people select and paste).

### Roadmap and changelog

`website/src/styles/pages.css` holds the rules.

Page hero. A microheader eyebrow, the headline in Fraunces, a muted lede whose
links take the prose-link role (`--color-fd-primary`). The hero matches the
width of the content under it, so both start on one edge.

Roadmap.
- Grouped by status, In progress, then Planned, then Shipped, each newest
  first, under an Inter heading with its status dot and a mono count; a row of
  the same three as jump links sits under the lede. Rejected: one interleaved
  list (nothing said what was next).
- Items are hairline rows: the name in 16px Inter medium (a feature name is
  not technical data, so not mono), the summary muted under it at 62ch.
  Shipped runs in two columns from 1024px, about 1700px shorter at 1440.
  Rejected: one column at every width (5900px tall at 1440) and cards for In
  progress and Planned (a box per item where rows already separate, and two
  vocabularies on one page).
- Status: `warning` dot for in progress, `success` for shipped, a hollow muted
  ring for planned, the deploy steps' pending mark. No accent anywhere on it.

Changelog.
- One row per release: a leading chevron, the version in 15px mono, the date
  and the change count in 12px mono, the newest marked Latest and open; the
  rest are closed `<details>`. From 768px the version holds a 9.5rem gutter
  and the changes align under the date. Below it, the date and count drop to a
  second line. Rejected: each release as a card (a box per row, against the
  row rule) and the full open list (one long wall).
- Changes keep the pull request titles; Dependabot's bumps sit last under a
  "Dependency updates" microheader, muted. Grouping by verb was rejected: only
  5 of 469 titles start with Fix, so a Fixes group would have been mostly
  wrong.
- Each release is a permalink (`/changelog/#v0.3.31-beta`), from its Link in
  the meta row; landing on one opens it.

### Site motion

Shared layer: the header, menu, docs and 404.
- Phone menu: the app's sheet clock, sliding in from its edge 300ms on
  `--ease-drawer` and out in 220ms `--ease-out`, the scrim fading on the same
  clock; instant when opened or closed from a key. The dialog runs it as
  transitions with `@starting-style`, so a menu reopened mid-close turns back;
  the docs drawer runs the same numbers as keyframes, which its library waits
  for before it unmounts. Rejected: the menus' pop (0.97 and 4px from the top
  right, 150ms; it read as a dropdown, and a 0.97 scale on the full-height
  docs drawer read as a zoom) and a fade alone (nothing said where the sheet
  came from).
- Theme toggle: the page in one frame; the icon swaps as the app's copy
  button does (the old icon shrinks to 0.6 and blurs 2px out over 150ms as
  the new one pops in on `--ease-spring-pop`, 350ms). Rejected: an instant
  icon (nothing confirmed the click on a page that changed all at once) and
  the library's view-transition crossfade (the whole page passed through a
  grey wash for 250ms, and the app measured its capture freezing 180ms).
- Docs sidebar active row: still. Each docs page is a full page load, so the
  app's sliding highlight has no element to carry. Rejected: a cross-document
  view transition sliding the ember edge (200ms `--ease-spring`; the edge
  travelled while its row's block landed at once, so the two split, and a
  snapshot covered the page while it ran) and the edge growing in on arrival
  (replayed on every page load, decoration on the most frequent act in the
  docs).
- Search: from a pointer, the app's dialog clock (rising 6px from 0.97 over
  220ms `--ease-out`, out in 150ms to 0.98, the scrim on the same clock);
  from a key (Ctrl K, /) it shows and goes in the same frame. Rejected: the
  library's default (from 1.06 over 300ms: the box bloomed toward the reader,
  half transparent over a page not yet dimmed) and no motion from a pointer
  (the palette popped over an undimmed page).
- Reduced motion: the menu, drawer and search fade over 150ms in place;
  everything else the docs move lands at its end state; the theme icon swap
  keeps its fade without the scale and blur; the 404's orbit becomes one
  150ms fade.
- Under a 4x CPU throttle at 375px the landing menu held 16.7ms frames; the
  docs drawer and search hit a 33ms p95 from mounting their React trees in the
  first frame, not from the slide.

Home page.
- Hero arrival: the text never moves; the board shot waits until it has
  decoded, then rises 12px and fades in over 400ms `--ease-out`, once per
  session (a 150ms fade under reduced motion). Rejected: the page entrance
  on the text (headline blank for the first 30 to 90ms, so reading waited),
  no motion (the image popped in whenever it finished decoding, sometimes
  after the text had been read) and a clip-path wipe (a curtain with no
  origin, finished within 100ms on the ease-out so it read as a flash). It
  cost about 0.1s of lab LCP on a throttled phone.
- Platform switch: the app's sliding highlight for the thumb (200ms
  `--ease-spring`) and its tab swap for the command (6px from the picked
  tab's side, 160ms `--ease-out`); a key switch moves at once.
- Copy: the app's icon swap (the copy icon shrinks to 0.6 and blurs out over
  150ms as the check pops on `--ease-spring-pop`), held 1.4s, announced as
  Copied; instant from a key, a 150ms fade under reduced motion.
- No scroll-triggered reveals, no parallax: the step shots are lazy images
  that simply appear.

Roadmap and changelog.
- A release opening: the disclosure (the box snaps open, the content fades in
  as it settles 4px over 200ms `--ease-out`; closing, it fades out in 120ms and
  the box shuts after), built on `::details-content` with `content-visibility`
  as a discrete transition; a browser without it snaps. Chevron turns 90° in
  150ms `--ease-standard`. Rejected: a snap (only the chevron said anything
  opened) and a height animation with `interpolate-size` (the rows below glide,
  but every frame re-lays the page: 89 layouts across six toggles of the
  33-change release under a 4x CPU throttle against 7 for the disclosure, 2.5
  times the layout time, growing with the page; the app's rule is that a box
  changing size snaps). A grid-rows height animation was not built: it costs
  the same layout per frame.
- A jump to a roadmap group or a release lands at once and its heading row
  takes a wash, `foreground` at 9% fading out over 900ms `--ease-out`.
  Rejected: native smooth scrolling (about 800ms to travel 1850px at 375px,
  its curve out of our hands, and it delays reading on every keyboard jump)
  and smooth scrolling plus the wash (slower still).
- Status markers stay still. Rejected: the `status-pulse` loop on the in
  progress dots (it says live, and a roadmap is not live; ambient motion on a
  reading page) and a one-time ring on load (decoration with nothing behind
  it).
- Reduced motion: the release content fades in 150ms with no travel, and the
  wash shortens to 400ms.

## Phone app

The phone app in `native/` speaks this language on a phone. Its mechanics
(Uniwind, `useCSSVariable`, Reanimated worklets, the navigation shape) are in
`practices/native.md`; this section holds how it looks and moves.

Every decision below was built as live variants on the Android emulator with
the seeded workspace, judged from screenshots, 1x and 0.25x recordings and
frame strips, and measured with `gfxinfo`; the emulator judges whether motion
runs and degrades, not its final feel.

### What the phone adapts

The phone shares the tokens, the accent's roles, ink focus, status hues, the
three families, the radii and the web's motion numbers. It adapts these:

- No backdrop blur anywhere: solid surfaces on a still field, which shows
  behind tab roots only (Phone surfaces).
- No palette picker, so it shows the default palette; light and dark follow
  the system until the Appearance setting picks one.
- Fraunces is a static cut, and titles are sized in points.
- Push, tab switch and sheets move as the platform does.
- Press feedback lands on touch, and every gesture has a second way to do the
  same thing.
- Rows on a tab root are separated by space, not hairlines.

### Phone tokens and type

Tokens. `native/src/global.css` holds the web's values under the same names,
dark and light. `focus` and the light field's three glows are colour tokens
there (`--color-focus`, `--color-field-warm`, `-pink`, `-cool`), because the
phone reads every colour through `useCSSVariable`.

Type. Fraunces is a static cut at the web's axes (opsz 28, wght 560, SOFT
50, WONK 1), `Fraunces Display`, for screen titles, record titles and
empty-state headlines, never under 20pt. A screen title is 30pt, 25pt past 44
characters and 22pt past 90, three lines at most. Rejected: the static
Fraunces SemiBold Google Fonts serves, which sits at opsz 9 with SOFT 0 and
read wide and sharp next to the web's soft titles.

### Phone surfaces

A tab's root sits on the canvas with the still light field behind it and its
rows straight on the field; pushed screens are the plain canvas; cards, wells
and sheets are solid. Picked over a raised panel inset 8pt (the field only
showed in an 8pt frame and the panel cost 16pt of a 412pt width) and the plain
canvas everywhere (correct and anonymous). The field is one static SVG: Board
scrolling measured p50 16ms and p90 16ms with it, the same as without it.
Backdrop blur was tried on the board's drop bar, over a board that holds still
while a card is held, and lost: frames went to a 95th percentile GPU time of
4.95s, so the phone has no blur.

Rejected and not built: a theme picker (the phone has none) and blur anywhere.

### Phone lists and chat

Lists. Rows on a tab root are separated by space, not hairlines: a 36pt tile
or avatar with its state as a dot on the corner, the title, mono meta
trailing; read titles turn muted. Rows grouped in a card keep hairlines.
Board columns are `surface-2` wells of cards (the mono key as an eyebrow, the
title, the type icon and name and label chips, whoever acts next), under a header with the
project mark, the stage bar and an "Only mine" switch that takes the ember
when on. The tab bar's active tab is the ember; its unread count is the ink
pill.

Search. More's first row opens a pushed screen: one field at the top, then
the web palette's groups as microheaders over plain rows (a muted kind icon,
the title, the mono key trailing), tickets and docs found only in their text
under In text, and a quiet line under the list when a kind had more.

Chat. Your messages are the ember bubble on the right (12pt corners, the
bottom-right one 2pt); everyone else's text runs plain under a 32pt avatar
with the name and mono clock time; day dividers are the microheader between
hairlines. Bot embeds are the web's status-edged cards with facts. The
composer is one framed field with the send button inside it, ember once there
is text. A hand-off folds each turn of the helper agent into one
row: the web's outcome mark (the orbit while it runs, a muted check, a
`destructive` warning when a step failed, a muted dashed circle when a call
never came back), the step in flight or "Worked for" in medium ink over what it
did ("Explored 2 files, ran 3 commands"), folding open to the web's step rows
on a hairline rule dropped from the mark, a listed group's names one to a
line. Open while it runs, folded once it ends.

### Phone motion

The numbers are the web's [Motion baseline](#motion-baseline) unless named.

- Screen push, tab switch and sheets: the platform's. Rejected for push:
  slide from the right (a 300ms slide on an act repeated all session) and
  fade from the bottom (the board and the ticket showed through each other,
  the reason the web dropped its title morph). Sheets take a 12pt corner.
- Sending a message: Rise, 12px over 240ms from 0.96 out of the bubble's
  bottom-right corner; the server's copy keeps the pending row's key, so the
  rise is never cut. Rejected: rising from the composer (64px; on a phone it
  doubled the list's own scroll to the end and read as two moves), a pop from
  0.9 (the bubble's text shifted as it scaled) and none (the bubble appeared
  with the list's jump and nothing said it was yours).
- Moving a card: hold 280ms to pick it up; it lifts to 1.03 with the
  elevated shadow in 150ms (a shadow layer drawn once and faded, never an
  animated shadow), follows the finger 1:1, and the columns rise in as drop
  targets along the bottom; over one the card shrinks to 0.55 around the
  finger so the target stays in view and the target takes the ember edge.
  Dropped, it glides onto the target in 220ms and shrinks into it while the
  board moves the card at once; let go anywhere else, it springs back
  (300ms, no bounce). Under reduced motion nothing scales and the glide
  stays. Drag frames measured p50 16ms, p90 27ms on the emulator, whose input
  injection paces the frames. Each card also offers "Move to" accessibility
  actions.
- Inbox: a row swipes left to flip it between read and unread (its "Mark read"
  or "Mark unread" action is the alternative); the unread dot shrinks to 0.6 as it fades over 150ms.
- Pull to refresh: the platform's spinner in the ember, since it is progress.
  Rejected: a muted spinner, which read as disabled.
- Press: the `accent` tint on rows and cards, on touch; a labelled button
  also presses to 0.97 over 100ms and lets go over 150ms; an icon button
  answers with its background alone. Rejected: the Android ripple (no iOS
  counterpart, the same at rest, and it doubled the tint where both ran).
- Empty state: the orbit turns into place as on the web; a compact one fades
  and rises 4px. Loader: the orbit at 16pt, a linear 1.2s keyframe turn,
  held back 300ms. Hand-off: content that replaces a loader rises 6px over
  200ms where it stood. A folded group (a hand-off's turn) opens in one layout
  step and its rows fade and rise 4px over 200ms, the compact empty state's
  numbers; its chevron turns at once.
- Reduced motion (Android's Remove animations): every moment is a 150ms
  fade or nothing; the loader holds still; a drag still tracks the finger.
