# 05 — How a bot message and its embeds look in the Mono Console

**Type:** prototype
**Status:** resolved
**Blocked by:** 01, 02

## Question

Discord embeds carry a colour bar, author line, title with link,
description, fields in a grid, image, thumbnail, footer, and timestamp. The
Mono Console reserves colour for status signal, so a sender's arbitrary
`color` cannot simply paint the bar. Build a throwaway render of a bot
message in the chat dock for the three real payloads ticket 01 collected
plus a plain `content`-only post, and let the owner react:

- The author row: bot avatar, bot name, a BOT tag, timestamp.
- The embed card: which fields render, in what order, how `color` is
  treated (dropped, mapped to the status palette, or shown as a neutral bar).
- Markdown in `content` and `description`: same renderer as human messages.
- Long payloads: what collapses, what truncates.

Run `design-mode`; verify at 320/375/414/768px. Link the prototype from
the answer.

## Answer

Picked by the owner from live variants over two rounds. Prototype on branch
`proto/bots-embed` (commit 3832e2c8; `?variant=1` on any conversation's
route mocks six bot posts under its real messages), the primary source for
the build. Checked at 768, 1024 and 1440px, the web widths since ADR 0080;
the phone widths above predate it, and chat is a page, not a dock.

- **Author row**: the avatar (the post's `avatar_url`, else the bot's,
  else the Nexul glyph), the name (the post's `username`, else the bot's
  name), a `BOT` outline tag like the Agent's `App` tag, `via <bot name>`
  whenever the post overrides the name so the sending bot always shows, and
  the relative time. Consecutive posts from one bot group like a person's.
- **One bubble**: the text and every embed share the message bubble other
  people's messages use, widened to 38rem when it carries an embed.
  Rejected: embeds hung under the bubble (heavier in a busy channel), and
  no bubble with each embed folded to one line (everything a click away).
- **The embed**: a 2px neutral rule on its left; the sender's `color` is
  dropped, because mapped onto status hues it misreads (GitHub's purple
  would say "merged"). Top to bottom: author line, title (a link when `url`
  is set), description, fields, the fold bar, image, footer; a thumbnail
  sits right at 64px.
- **Fields as a framed grid**: a two-column table with a hairline on every
  cell edge, the label cell shaded, the label column 40%, `inline`
  ignored. The cell hairline and label shade become tokens in `index.css`
  and the design-language table. Rejected: rows ruled without a frame, and
  Discord's own name-over-value tiles three across (Uptime Kuma's posts
  grew tall).
- **Footer**: 11px medium muted text, a 16px round icon, `•`, and the time
  in Discord's calendar form ("Today at 21:11", "Yesterday at 21:11", else
  date and minute), the exact time on hover. Rejected: the full mono date
  with seconds, which read louder than the content.
- **Long payloads**: past six fields, or a description over about 420
  characters or eight lines (clamped to six), one hairline bar ("Show 19
  more fields", "Show the rest") opens the rest and turns into "Show less".
  Past two embeds, the rest fold behind "Show N more embeds".
- **Markdown**: chat has no markdown renderer today, so a person's `**`
  shows as typed. A bot's content, descriptions and field values render
  Discord's subset: bold, italics, inline code, `[text](url)` links, list
  lines, quotes, and `<t:…>` time codes (`F` as "Tuesday, 6 October 2026 at
  21:11"). People's messages are unchanged. The build needs this renderer.
