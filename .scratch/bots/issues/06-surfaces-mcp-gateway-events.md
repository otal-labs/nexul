# 06 — Surfaces: settings, gateway, MCP, events, search

**Type:** grilling
**Status:** resolved
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

## Answer

Decided 2026-10-05 as technical calls; the look of the settings surface is
ticket 07's.

- **Settings.** Channel settings is one dialog with one card today, not
  tabs, so bots get a **Bots section** component. Channels and voice
  channels show it in that dialog, and the dialog opens for anyone with
  `botwebhook:read` even when they could not otherwise edit the channel.
  DMs and the three thread kinds have no settings page; they get a "Bots"
  item in the conversation's menu that opens the same section in a dialog.
  No settings page is built for them.
- **Gateway.** `GET /api/conversations/{id}/botwebhooks` (with
  `deleted=true` for the restore list), `POST` on the same path to create,
  `PATCH /api/botwebhooks/{id}` for rename, avatar, `regenerate`, and
  `deleted: false` to restore, and `DELETE /api/botwebhooks/{id}` to soft
  delete. The public execute route stays ticket 03's
  `/api/botwebhooks/{id}/{token}`, mounted before authentication.
- **MCP: three tools**, shaped per task the way `doc_update` carries
  archive and restore. `botwebhook_list` (read-only; URLs only for
  `botwebhook:write`), `botwebhook_create`, and `botwebhook_update` (rename,
  avatar, `regenerate`, `deleted` true or false). The ceiling at 108 has one
  slot left, so the build ships an ADR raising it to 111: no existing tool
  can carry this without chat learning what a webhook is, which ticket 02
  ruled out. **An agent never posts as a bot through MCP**; `message_post`
  stays the caller's own voice, and an agent that needs a bot's voice uses
  the URL like any sender.
- **Events**: `botwebhook.created`, `botwebhook.updated` (what changed:
  renamed, avatar, regenerated), `botwebhook.deleted`, `botwebhook.restored`,
  each a catalog row with an outbox write. **No payload ever carries the
  token or the URL.** A post fires the existing `chat.message.created`, whose
  message now carries author kind `bot` and the name and avatar snapshot.
- **Live push.** A bot post reaches open conversations through the existing
  `chat.message.created` live-audience rule with no new work. The four
  `botwebhook.*` topics get a live-audience rule scoped to the
  conversation's audience so an open Bots section refreshes.
- **Search.** Chat messages are not full-text indexed today (only ticket
  notes are), so bot posts are not either. If chat search lands, embed
  title, description, and field text are indexed with the body.
- **Permissions.** The table has no implication between actions; roles grant
  each one explicitly and every use-case checks its own. `botwebhook:read`
  sees the section and the bots without URLs; `botwebhook:write` sees URLs,
  creates, renames, changes the avatar, regenerates, and restores (a restore
  issues a token, like a create); `botwebhook:delete` deletes. The web reads
  these from the catalog and gets an entry on the Access page.

Two corrections to earlier tickets, carried into the spec:

- Ticket 02 called the author migration breaking because there was no
  production data. There is now, so dropping the messages-to-users foreign
  key is a new numbered forward-only migration that rebuilds the table and
  keeps every row, with an upgrade test from the previous schema.
- The map lists deleting a channel as missing. It exists now; a bot goes
  with its channel through ticket 03's cascade.
