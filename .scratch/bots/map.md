# Wayfinder map: bots

Charted 2026-09-16 (grilling rounds 1–2 with the owner). A **bot** is a
named poster inside a conversation that outside systems drive through a
webhook URL, the way a Discord channel webhook works: the URL is the
credential, the conversation is the binding, and the payload is Discord's own
execute-webhook JSON, so anything that already posts to a Discord webhook
posts to Nexul by swapping the URL. Not a login, not a gateway connection,
not an integration.

## Destination

A spec at `.scratch/bots/spec.md` with every product decision locked, ready
to slice into implementation tickets with `/to-tickets`: the bot model and
its author identity, the URL and secret lifecycle, the accepted payload and
how embeds render, mentions and notifications, permissions, and the settings,
MCP, gateway, and event surfaces. Planning only; the build is its own effort
after this map.

## Notes

- Grilling tickets: invoke `/grilling` + `/domain-modeling`. Research
  tickets follow `/research`; findings land in `.scratch/bots/research/`.
- Any ticket that touches a screen (the Bots tab, a bot message, an embed
  card) runs `design-mode` and verifies at 320/375/414/768px.
- Plain language with the owner; no section codes or shorthand without
  saying what it means in the same sentence.
- Grounding: `CONTEXT.md` (Chat, Integration, Permission, Agent, Harness),
  ADR 0029 (a turn runs on the mentioning user's harness, and a bot has
  none), 0043 (integrations are external services with scoped tokens; a bot
  is deliberately not one), 0044 (event catalog), 0051 (chat is a dock).
  Code: `internal/chat/model.go` (`Kind`, `AuthorKind`, `Message`),
  `internal/chat/usecase.go` (`PostMessage`), `internal/chat/handler.go`,
  `internal/chat/mcp.go` (`message_post`),
  `internal/integrations/webhook.go` (the outbound signed webhook, the thing
  a bot is not), `internal/platform/permissions/permissions.go`.
- Pre-release item 03 (security review of the integration model) lists
  "before any third party gets a credential" among its triggers. The URL
  ticket surfaces it if the design grows past "the URL is the credential".
- Related, read but do not act on: `.scratch/integration-store/` holds the
  outbound Discord reference integration, which is the mirror image of this
  effort and stays there.

### Settled at charting (grilling rounds 1–2)

- **Direction**: inbound. Outside systems post into Nexul conversations.
  Nexul posting out to Discord is the store's job, not this effort's.
- **Payload**: Discord-compatible. The URL accepts Discord's execute-webhook
  JSON (`content`, `username`, `avatar_url`, `embeds`, and whatever else the
  research ticket finds matters) as the contract, so existing senders work
  with zero code. Embeds render as a card.
- **Binding**: any conversation kind. Channels, voice channels, DMs, channel
  threads, ticket threads, doc threads. Who may bind one to a DM or thread is
  the model ticket's question.
- **Name**: **Bot** is the vocabulary term; a webhook URL is its one
  mechanism today, leaving room for other bot kinds without renaming. The
  permission domain is `botwebhook`, with `read`, `write`, and `delete`
  (renamed from `webhooks` in ticket 02; the Go package matches).
  Posting through the URL needs no permission: the URL is the credential.
- **Author**: the bot itself. A new author kind `bot`, with `author_id`
  pointing at the bot rather than at a user, and Discord's per-post
  `username` and `avatar_url` overrides honoured.
- **Destination shape**: a spec, then `/to-tickets`, like the plays effort.
- From repo rules, not asked: MCP tools ship with the feature; bot lifecycle
  is catalog events with an outbox write; every way in has its way out
  (delete, regenerate).

## Decisions so far

<!-- one line per resolved ticket: gist, then the link for the detail -->
- [What exactly is Discord's execute-webhook contract?](issues/01-discord-webhook-contract.md) — `POST /webhooks/{id}/{token}`, one of content/embeds/components/file/poll required, 2000 chars, ten embeds, 6000 embed chars; `wait` flips 204 to the message body; multipart for files; per-webhook rate limits with no published quota; edit and delete on the same URL; three real payloads quoted.
- [The bot model and its author identity](issues/02-bot-model-and-author.md) — one bot, one conversation of any kind; package and permission domain `botwebhook`; the bot is the author with the users foreign key dropped and name/avatar snapshot columns on bot messages; avatar reuses the user mechanism with a Nexul glyph default, per-post overrides honoured; soft delete with restore on a fresh token; unique names, cap of ten.
- [The URL, the secret, and abuse](issues/03-url-secret-and-abuse.md) — `/api/botwebhooks/{id}/{token}`, 43-character token readable to write-holders and encrypted at rest; 30 posts/min per bot, 60 rejects/min per IP, 64 KiB body; Discord-shaped 404/429, synchronous write, one audit row per accepted post and none for rejects; bot cascades with its conversation; four lines on the pre-release security checklist.

## Not yet specified

- **Editing and deleting a bot's messages through the URL** (Discord's
  PATCH and DELETE on a webhook message). Waits on the contract research
  and the author model.
- **File uploads through the URL** (Discord's multipart form). Waits on the
  contract and on how attachments render in chat today.
- **Discord's `thread_id` query parameter** against a bot bound per
  conversation: honour it, ignore it, or map it onto channel threads. Waits
  on the research.
- **Other bot kinds**: an MCP-driven bot with its own token, or a first-party
  bot that automations post through instead of a user's token. The name
  leaves the door open; nothing to decide until the webhook kind ships.
- **Compatibility suffixes** the way Discord accepts `/github` on a webhook
  URL (they would hang off `/api/botwebhooks/{id}/{token}`) to translate GitHub's own payload. Whether Nexul mirrors those waits
  on the research listing which exist and what they cost.

## Out of scope

- Outbound delivery of Nexul events to a Discord channel. That is the
  Discord reference integration in `.scratch/integration-store/`.
- Gateway-connected bots with a login, slash commands, interactive
  components, and anything that needs a bot to receive messages.
- Redesigning the chat dock or message rendering beyond what a bot message
  and an embed card need.
- Deleting a channel. It does not exist today; a bot cascades away when its
  conversation is deleted, and the delete itself is chat's missing way out.
