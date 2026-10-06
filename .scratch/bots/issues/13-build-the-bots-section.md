# 13 — The Bots section in conversation settings

**What to build:** The section decided in ticket 07, from the prototype on
branch `proto/bots-section` (`?variant=3&settings=<channel>`; its last
commit drops the paste hint). In the channel settings dialog for channels
and voice channels, opening for anyone with `botwebhook:read`; a "Bots"
item in the menu of DMs and the three thread kinds opening the same section
in a dialog. The list with the "n of 10" count and the one-line
explanation; rows with avatar, name, and the last-post and made-by line; a
row opens the detail view behind "‹ Bots" with the avatar (change, use the
default), the name with "Save name", the URL in a read-only field with a
joined Copy button, Regenerate URL and Delete bot behind the
consequence-naming confirms, and the "New URL" or "Restored" line after
those actions. Create opens the same view empty and lands on the new bot.
A Deleted fold with Restore. Read-only sees rows only; Delete bot needs
`botwebhook:delete`. The dialog caps its height and scrolls. The Access
page gets the `botwebhook` entry from the catalog. Live updates from 09's
events.

**Blocked by:** 09

**Status:** ready-for-agent

- [ ] Tests for editor, read-only, and write-without-delete views; the cap
      at ten; a duplicate name; the confirms
- [ ] Worst case: ten long-named bots and long creator names
- [ ] Verified at 768, 1024, and 1440px in both themes
- [ ] Every way in has its way out: delete and restore, regenerate, the
      default avatar
