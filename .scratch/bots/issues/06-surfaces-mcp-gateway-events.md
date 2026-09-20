# 06 — Surfaces: settings, gateway, MCP, events, search

**Type:** grilling
**Status:** open
**Blocked by:** 02, 03

## Question

Walk the "hit every surface" list for bots and decide each:

1. Settings: a Bots tab on a channel's settings, like Discord's Integrations
   tab. Where the same lives for a voice channel, a DM, and the three thread
   kinds, which have no settings page today.
2. Gateway routes for create, list, get, regenerate, delete, plus the public
   execute route from ticket 03.
3. MCP tools, one per use-case: `botwebhook_create`, `botwebhook_list`,
   `botwebhook_regenerate`, `botwebhook_delete`. Whether an agent may post *as*
   a bot through MCP, or only through the URL like everyone else.
4. Events: `webhook.created`, `webhook.regenerated`, `webhook.deleted` as
   catalog rows with outbox writes; `chat.message.created` already fires for
   the post and its payload carries the author kind.
5. Live push: a bot post reaches open docks over the existing WebSocket
   path with no new work, confirm.
6. Search: whether chat messages are indexed today, and if so that bot
   posts and embed text are.
7. Permissions: `botwebhook:read` shows the tab and URLs, `botwebhook:write`
   creates and regenerates, `botwebhook:delete` deletes. Confirm the
   `write` implies `read` rule holds as for other domains.
