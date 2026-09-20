# 02 — The bot model and its author identity

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Charting settled that a bot is its own author: a new author kind `bot`, with
`author_id` pointing at the bot. Today `author_id` is always a real user id,
even for Agent and system messages (`internal/chat/model.go`). Settle the
model around that:

1. The bot record: name, avatar, the conversation it binds to, who created
   it, when, and the token (hashed or not is ticket 03's call). Is a bot
   bound to exactly one conversation, as Discord's are, or may one bot post
   to several?
2. Where it lives: a `webhooks` package beside chat, or inside the chat
   domain. A bot posting is a chat use-case call; domains must not import
   each other's internals, so the seam decides the package.
3. Avatar: a URL the owner types, an uploaded attachment, or both. Per-post
   `avatar_url` overrides come from outside and are always URLs.
4. What a message keeps when the bot is later deleted or renamed: the
   display name and avatar at post time need to live on the message (per-post
   overrides need that anyway), or history reads wrong.
5. Binding to a DM, a channel thread, a ticket thread, or a doc thread: who
   may create one there, and whether the settings surface for those kinds
   exists at all (a DM has no settings page today).
6. Name uniqueness per conversation, and a cap on bots per conversation, if
   any.

Recommendation going in: one bot, one conversation; a `webhooks` package
that calls chat's `PostMessage` through a narrow interface; avatar is a URL;
message rows carry a display name and avatar column populated for every
author kind; DM bots need both participants' `webhooks:write`, threads
inherit the parent's rule.

## Answer

Grilled 2026-09-17, two rounds.

- **One bot, one conversation.** The URL is the binding. A bot that must
  post to several places is several bots. Any conversation kind may hold
  one (channels, voice channels, DMs, channel threads, ticket threads, doc
  threads); the owner considered per project and ruled it out.
- **Package and permission domain are both `botwebhook`.** A new
  `internal/botwebhook` domain with model, repo, use-cases, handler, MCP
  tools, and events. Chat gains one use-case, post a bot message, reached
  through an interface the new domain declares; chat never learns what a
  webhook is. Permissions are `botwebhook:read`, `botwebhook:write`,
  `botwebhook:delete`, replacing the `webhooks` name from charting.
- **The bot is the author.** The foreign key from a message's author to the
  users table is dropped; `author_id` is the bot id when the author kind is
  `bot`. Two nullable snapshot columns, `author_name` and
  `author_avatar_url`, are written for bot messages only, so a message keeps
  what it showed after a rename, a per-post override, or a delete. Users and
  the Agent keep resolving live. No production data, so the migration is
  breaking.
- **Avatar** reuses the user avatar mechanism: a base64 image on the bot
  row, the same 10 MB cap and file picker as the first-login step, and a
  built-in Nexul bot glyph when none is set. Per-post `username` and
  `avatar_url` from the sender are both honoured.
- **Soft delete.** A deleted timestamp; the URL dies immediately; the bot
  leaves the Bots tab; its name is freed among live bots; a restore action
  brings it back with a fresh token so a leaked old URL never revives.
- **Who may create one.** `botwebhook:write` in the workspace. On a DM, any
  participant with it. Threads inherit the parent's rule, and ticket and doc
  threads also require the creator to read the ticket or doc, matching how
  the thread itself is gated.
- **Names unique per conversation**, case-insensitive, among live bots.
  **Cap of ten** bots per conversation.

ADR candidate for the spec ticket: a message's author is no longer always a
user (the foreign key drop and the snapshot columns).
