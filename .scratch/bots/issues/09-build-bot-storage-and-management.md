# 09 — Bot storage, the author migration, and management routes

**What to build:** The new `internal/botwebhook` domain (model, repo,
use-cases, handler, events) per the spec's "The bot", "The author", and
"Permissions" sections. A new numbered migration adds the bots table (id,
conversation id with on-delete cascade, name, avatar, encrypted token,
creator, created and updated times, last-post time, post count, deleted
timestamp) and rebuilds `messages` without the author-to-users foreign key,
adding nullable `author_name` and `author_avatar_url`, keeping every row
(ADR 0129). Chat gains author kind `bot` and the one post-a-bot-message
use-case behind an interface `botwebhook` declares; chat never imports the
new domain. Use-cases to list (with deleted), create, rename, set or clear
the avatar, regenerate, delete, and restore, each checking its own
`botwebhook:read|write|delete` action and the conversation's gate (DM
participant, ticket or doc read for those threads). Names unique per
conversation case-insensitively among live bots, at most 80 characters;
ten live bots per conversation. Gateway: `GET`/`POST
/api/conversations/{id}/botwebhooks` (`deleted=true` for the restore list),
`PATCH` and `DELETE /api/botwebhooks/{id}`. Events `botwebhook.created`,
`.updated` (renamed, avatar, regenerated), `.deleted`, `.restored` with
catalog rows and outbox writes, never carrying the token or URL, plus a
live-audience rule scoped to the conversation's audience. The permission
table gains the `botwebhook` domain.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] Migration tested by upgrading from the previous schema: every message
      kept, a bot message survives its bot's rename and delete
- [ ] Use-case tests, refusals first: read without write sees no URL, write
      without delete cannot delete, a non-participant on a DM, a ticket
      thread without the ticket, the eleventh bot, a duplicate name
- [ ] Restore and regenerate issue a fresh token; the old one stops working
- [ ] Catalog rows, outbox writes, live audience rule; no token in any
      payload
- [ ] `CONTEXT.md` and `practices/architecture.md` checked
