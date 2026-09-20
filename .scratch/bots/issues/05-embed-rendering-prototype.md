# 05 — How a bot message and its embeds look in the Mono Console

**Type:** prototype
**Status:** open
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
