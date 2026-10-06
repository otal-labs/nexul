# 07 — The Bots tab and the create flow

**Type:** prototype
**Status:** resolved
**Blocked by:** 05, 06

## Question

Build a throwaway Bots tab on channel settings using the settings kit from
the stack page: the list of bots with name, avatar, creator, last post;
create with name and avatar; the URL shown per ticket 03's rule with a copy
button; regenerate and delete with the confirm pattern already in settings.
Let the owner react to the flow, not the code.

Run `design-mode`; verify at 320/375/414/768px. Link the prototype from
the answer.

## Answer

Picked by the owner from three live variants: the detail view. Prototype on
branch `proto/bots-section` (commit 49adbe82; `?variant=3&settings=<channel>`
on a chat route opens the channel's settings with in-memory bots, `as=viewer`
for read-only, `worst=1` for ten long-named bots), the primary source for
the build. Checked at 1024px in the settings dialog; it is the ticket 06
Bots section, not a tab, and the phone widths above predate ADR 0080.

- **The section**: under the channel's existing settings card, a hairline,
  then "Bots" with a mono "4 of 10" count and one line: "Outside tools post
  here through a webhook URL. Anything that already posts to a Discord
  webhook works." Then one rounded row per bot: avatar, name, and a muted
  "Last post 24m ago · made by Dev User" ("No posts yet" before the first),
  a chevron for anyone who can edit. "No bots yet" as an empty row.
- **The detail view**: a row opens the bot in place of the section, behind
  a "‹ Bots" back link. The avatar at 48px with "Change avatar" and "Use
  the default"; the Name field, with "Save name" once it changes; "Webhook
  URL" in a read-only mono field with a Copy button joined to its right
  edge (it reads "Copied" for two seconds); the made-by and created line;
  and a footer with "Regenerate URL" and a destructive "Delete bot". No
  hint line under the URL (the owner cut "Paste it where the sender asks
  for a Discord webhook URL"); after a regenerate or restore the one line
  "New URL; the old one stopped working." or "Restored with a new URL."
  shows there instead.
- **Create**: "+ New bot" under the list opens the same view empty, with
  "Pick an avatar", Name and "Create"; creating lands on the new bot with
  its URL. Names are unique per conversation, at most 80 characters, and
  checked as you type. At ten bots the button disables with "A channel
  holds ten bots. Delete one to add another."
- **Confirms** use the existing confirmation dialog, destructive, naming
  the consequence: "Regenerate CI's URL? Anything still posting to the old
  URL gets a 404 from now on, until you paste the new URL into it." and
  "Delete CI? Anything posting to its URL gets a 404. Its messages stay in
  the channel, and you can restore it later with a new URL."
- **Deleted**: a "› Deleted (1)" fold under the list for editors, each row
  dimmed with "Deleted 2d ago by Fahad" and Restore (off while the channel
  is full).
- **Read-only** (`botwebhook:read` without write): the rows only, no
  chevron, no URL, no create, no Deleted fold. "Delete bot" shows only
  with `botwebhook:delete`; everything else in the detail view is write.
- **The dialog must scroll**: with ten bots it outgrows a 900px window
  and the title scrolls off, so the build caps the dialog's height and
  scrolls its body.
- Rejected: rows that expand to show the URL (one more click for the
  common copy), and the URL always under every row (twice the height, and
  every URL on screen whenever settings are open).
