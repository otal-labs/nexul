# A message's author is no longer always a user

Every message's `author_id` pointed at the users table, even for the Agent and system messages, which carried the id of
the user they acted for. A bot posting through its webhook URL has no user behind it: the URL is the credential, and
the bot is the poster.

Decision: a message with author kind `bot` has the bot's id as `author_id`, and the foreign key from a message's
author to users is dropped.

- **Snapshot, for bots only.** Two nullable columns, `author_name` and `author_avatar_url`, are written for bot
  messages, holding the post's `username` and `avatar_url` overrides or the bot's own name at post time. A bot message
  keeps the name it showed after the bot is renamed or deleted. Users and the Agent keep resolving live, so a person's
  rename still shows on everything they wrote.
- **A bot's own avatar is not copied.** Without an `avatar_url` override, `author_avatar_url` holds
  `/api/botwebhooks/{id}/avatar?v=<the bot's last change>`, so the message follows the bot's current avatar; an override
  stays as posted. A bot avatar is a data URI of up to 10 MB, and copying it would put that into every message row and
  every `chat.message.created` payload. The route serves anyone who reads the bot's conversation, since every reader
  sees its messages.
- **The migration** is a new numbered forward-only migration that rebuilds `messages` without the constraint and keeps
  every row, tested by upgrading from the previous schema; production data is live.
- **Integrity moves to code.** Chat's post use-cases still write a real user id for kinds `user`, `agent`, and
  `system`; only the bot path writes a bot id, through the one use-case `internal/botwebhook` reaches.

Rejected: a synthetic user per bot, which would put bots in the people list, the mention picker, and every member
count, and need hiding everywhere; and resolving bots live like users, which would rewrite every old post when a bot
is renamed or deleted, and lose the per-post name a sender chose.

The trade-off: the database no longer stops a message from pointing at a user that does not exist, so a bug in a post
path shows up as an unknown author instead of a failed insert.
