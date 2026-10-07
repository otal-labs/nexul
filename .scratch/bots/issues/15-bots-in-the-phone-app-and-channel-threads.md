# 15 — Bot messages in the phone app, and Bots on channel threads

**What to build:** Two gaps the web build left. The Android app's chat
(`native/`, its `MessageRow`) resolves a message's author as a person, so a
bot's post shows the bot's raw id; render author kind `bot` with the
snapshot name, the avatar (the served avatar path needs the session, as on
the web), a `BOT` tag, `via <bot>`, the text, and the embeds in the
phone's own shape. And channel threads have no "Bots" menu item because the
web has no screen that shows a channel thread; add it when that screen
exists.

**Blocked by:** 12, 13

**Status:** needs-triage

- [ ] A bot post on the phone shows its name and BOT tag, never an id
- [ ] Embeds readable at phone width
