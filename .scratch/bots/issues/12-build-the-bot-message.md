# 12 — How a bot message renders in chat

**What to build:** The web rendering decided in ticket 05, from the
prototype on branch `proto/bots-embed` (`?variant=1` on any conversation
route shows the picked shape with the Grid field table). In
`MessageRow`, author kind `bot`: the snapshot avatar or the Nexul glyph,
the snapshot name, a `BOT` outline tag, `via <bot name>` when the post
overrode the name, the relative time, grouping like a person's. Text and
embeds share the message bubble, widened to 38rem with an embed. Each
embed behind a 2px neutral rule, the color ignored: author line, linked
title, description, fields as the framed two-column grid (label cell
shaded, 40%), the fold bar, image, 64px thumbnail, and the Discord-weight
footer with the calendar time. Folding past six fields, a long
description, or two embeds. A Discord markdown renderer for bot content,
descriptions, and field values only (bold, italics, inline code, links,
list lines, quotes, `<t:…>` codes). The cell hairline and label shade
become tokens in `web/src/index.css` and the token table in
`practices/design-language.md`. Follow the F1 to F7 commandments; the
prototype is a look reference, not code to copy.

**Blocked by:** 09

**Status:** ready-for-agent

- [ ] Worst-case fixture: 25 fields, a 4096-character description, ten
      embeds, unbreakable values, a long override name
- [ ] Markdown renderer tests, including links never opening `javascript:`
      and image URLs only over http(s)
- [ ] Verified at 768, 1024, and 1440px in both themes
- [ ] People's and the Agent's messages unchanged
