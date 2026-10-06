# Bots: Discord-compatible webhooks that post into a conversation

**Status:** ready-for-agent

Assembled 2026-10-06 from the wayfinder map in this directory. Every
decision below is recorded in a ticket under `issues/`; this file is the
one-page view for slicing into implementation tickets with `/to-tickets`.

## Problem Statement

Teams that move to Nexul leave a trail of tools posting to Discord channel
webhooks: CI runs, Grafana alerts, Uptime Kuma, release notes. Nexul has no
way in for them. Outbound integrations listen to signed events, automations
act through the scoped API, and both need code written for Nexul. A tool
that only knows how to POST Discord's JSON to a URL has nowhere to go, so
those messages stay in Discord and the channel in Nexul is missing half its
context.

## Solution

A **bot** is a named poster bound to one conversation, driven from outside
through a webhook URL, the way a Discord channel webhook works. The URL is
the credential and the conversation is the binding. It accepts Discord's own
execute-webhook JSON (`content`, `username`, `avatar_url`, `embeds`,
`allowed_mentions`), so anything that already posts to a Discord webhook
posts to Nexul by swapping the URL. A post becomes a chat message whose
author is the bot, rendered with its embeds as a card inside the message
bubble.

People with the new `botwebhook` permissions create, rename, re-avatar,
regenerate, delete, and restore bots from a Bots section in the
conversation's settings, through the gateway, or through three MCP tools.
Bot lifecycle publishes four events; no event ever carries the token.

## User Stories

1. As a developer moving off Discord, I want to paste a Nexul URL where my CI asked for a Discord webhook URL, so that its posts land in our channel with no code change.
2. As a developer, I want a Grafana alert's embed to show its title, description, chart, and footer, so that I can read the alert without leaving the channel.
3. As a developer, I want an Uptime Kuma post's fields shown as a table, so that the service, the error, and the times line up.
4. As a channel member, I want a bot's message to say it is a bot and which bot sent it, so that a post calling itself "GitHub" cannot pass for a person.
5. As a channel member, I want long payloads folded behind one bar, so that a 25-field report does not push the conversation off screen.
6. As a workspace admin, I want to create a bot with a name and an avatar from the channel's settings and copy its URL, so that I can wire up a sender in a minute.
7. As a workspace admin, I want to regenerate a bot's URL, so that a leaked URL stops working at once.
8. As a workspace admin, I want to delete a bot and restore it later with a new URL, so that removing one is never a one-way door.
9. As a member who can read bots but not edit them, I want to see which bots post here without seeing their URLs, so that I know what feeds the channel.
10. As a member of a DM or a thread, I want a "Bots" item in the conversation's menu, so that bots work outside channels too.
11. As an agent, I want to list, create, and update bots over MCP, so that I can set up a conversation's feeds the way a person does.
12. As an automation author, I want `botwebhook.*` events, so that I can react when a bot is created, changed, deleted, or restored.
13. As a sender, I want Discord's answers back (204, the message with `wait=true`, 404 code 10015, 429 with `retry_after`), so that my existing retry logic keeps working.
14. As a member, I want `@user` in a bot's post to highlight that person the way a person's mention does, honouring the sender's `allowed_mentions`, so that alerts can name the on-call engineer.

## Implementation Decisions

### The bot (tickets 02, 03)

- **One bot, one conversation**, of any kind: channels, voice channels,
  DMs, channel threads, ticket threads, doc threads. A bot that must post to
  several places is several bots.
- A new domain `internal/botwebhook` with model, repo, use-cases, handler,
  `mcp.go`, and events. The permission domain is `botwebhook` with `read`,
  `write`, and `delete`. Chat gains one use-case, post a bot message,
  reached through an interface the new domain declares; chat never learns
  what a webhook is.
- Columns: id, conversation id (on-delete cascade, so a bot goes with its
  conversation, including a deleted channel), name, avatar (a base64 image
  like the user avatar, 10 MB cap, a built-in Nexul glyph when empty),
  encrypted token, creator, created and updated times, last-post time, post
  count, and a deleted timestamp.
- **Names** are unique per conversation, case-insensitive, among live bots,
  at most 80 characters. **Cap of ten** live bots per conversation.
- **Who may create one**: `botwebhook:write` in the workspace. On a DM, any
  participant holding it. Threads inherit the parent's rule; ticket and doc
  threads also require reading the ticket or doc, as the thread itself is
  gated.
- **Soft delete**: the URL dies at once, the bot leaves the list, its name
  is freed. Restore brings it back with a fresh token so a leaked old URL
  never revives.

### The URL, the token, and limits (ticket 03)

- **Path** `/api/botwebhooks/{id}/{token}`: Discord's shape with
  `botwebhooks` in place of `webhooks`, so a sender swaps the host and one
  path segment. Mounted before authentication like `/hooks/*`, and never
  through the audit middleware, which records the raw path.
- **Token**: the scoped-token generator, 32 random bytes as 43 URL-safe
  characters, encrypted at rest with the existing crypto helper. Readable
  to anyone with `botwebhook:write` at any time; copy-again over show-once.
- **Regenerate** issues a new token on the same bot; the old one dies
  immediately.
- **Limits**: an in-memory bucket with no new dependency. 30 posts per
  minute per bot, 60 rejected requests per minute per IP, 64 KiB JSON body.
- **Answers mirror Discord**: 404 with code 10015 "Unknown webhook" for a
  wrong id or token alike, 429 with `retry_after` and `X-RateLimit-*`, 400
  with Nexul's error body for a bad payload, 204 by default and the message
  with `wait=true`.
- **Posting is synchronous**: the message row, its outbox event, and one
  audit row (actor type `bot`, token stripped) in one transaction; the bus
  fans out after. Rejects are logged and counted, never audited.
- The four checklist lines on pre-release item 03 cover token entropy, the
  rate limit, the 404 with no oracle, and immediate revocation.

### The payload (ticket 01)

- Accepted: `content` (2000 characters), `username`, `avatar_url`, `embeds`
  (up to ten, 6000 characters of embed text in total, Discord's per-field
  limits), and `allowed_mentions`. At least one of `content` or `embeds`.
- An embed keeps title, description, url, timestamp, footer (text and
  icon), image, thumbnail, author (name, url, icon), and up to 25 fields.
  `color` is accepted and ignored. Embeds are stored with the message.
- Per-post `username` and `avatar_url` are honoured and snapshotted on the
  message.

### The author (tickets 02, 06; ADR 0129)

- A new author kind `bot`, with `author_id` the bot's id. The foreign key
  from a message's author to users is dropped, and two nullable snapshot
  columns, `author_name` and `author_avatar_url`, are written for bot
  messages only, so a message keeps what it showed after a rename, a
  per-post override, or a delete. Users and the Agent keep resolving live.
- Production data is live, so this is a new numbered forward-only migration
  that rebuilds `messages` and keeps every row, with an upgrade test from
  the previous schema.

### Mentions, the Agent, and notifications (ticket 04)

- `@Agent` never fires from a bot: the agent pipeline already runs only on
  author kind `user`.
- `@user` behaves like a person's mention, filtered by `allowed_mentions`;
  a handle the sender did not allow is stored as plain text.
- Unread counts bump as for any message; no per-bot mute. A bot post counts
  as ticket and doc thread activity.

### The message in chat (ticket 05)

Prototype on branch `proto/bots-embed`, the visual source for the build.

- **Author row**: the post's `avatar_url`, else the bot's avatar, else the
  Nexul glyph; the post's `username`, else the bot's name; a `BOT` outline
  tag like the Agent's `App` tag; `via <bot name>` whenever the post
  overrides the name; the relative time. Consecutive posts from one bot
  group like a person's.
- **One bubble**: the text and every embed share the bubble other people's
  messages use, widened to 38rem when it carries an embed.
- **The embed**: a 2px neutral rule on its left, the sender's color dropped
  (on status hues it misreads; GitHub's purple would say "merged"). Author
  line, title (a link when `url` is set), description, fields, the fold bar,
  image, footer; a 64px thumbnail on the right.
- **Fields** as a framed two-column grid: a hairline on every cell edge, the
  label cell shaded, the label column 40%, `inline` ignored. The cell
  hairline and label shade become tokens in `web/src/index.css` and the
  design-language token table.
- **Footer**: 11px medium muted text, a 16px round icon, `•`, and the time
  in Discord's calendar form ("Today at 21:11", "Yesterday at 21:11", else
  date and minute), the exact time on hover.
- **Long payloads**: past six fields, or a description over about 420
  characters or eight lines (clamped to six), one hairline bar ("Show 19
  more fields", "Show the rest") opens the rest and becomes "Show less".
  Past two embeds the rest fold behind "Show N more embeds".
- **Markdown**: a bot's content, descriptions, and field values render
  Discord's subset (bold, italics, inline code, `[text](url)` links, list
  lines, quotes, and `<t:…>` time codes, `F` as "Tuesday, 6 October 2026 at
  21:11"). People's messages are unchanged.

### The Bots section (tickets 06, 07)

Prototype on branch `proto/bots-section`, the visual source for the build.

- **Where**: channels and voice channels show it in the channel settings
  dialog, which opens for anyone with `botwebhook:read`. DMs and the three
  thread kinds get a "Bots" item in the conversation's menu that opens the
  same section in a dialog. The dialog caps its height and scrolls its body.
- **The list**: under the channel's settings card, a hairline, "Bots" with a
  mono "4 of 10", and "Outside tools post here through a webhook URL.
  Anything that already posts to a Discord webhook works." One rounded row
  per bot: avatar, name, "Last post 24m ago · made by Dev User" ("No posts
  yet" before the first), a chevron for editors. "No bots yet" as an empty
  row.
- **The detail view**: a row opens the bot in place of the list behind
  "‹ Bots": the 48px avatar with "Change avatar" and "Use the default", the
  Name field with "Save name" once it changes, "Webhook URL" in a read-only
  mono field with a joined Copy button ("Copied" for two seconds), the
  made-by and created line, and "Regenerate URL" and a destructive "Delete
  bot". No hint line under the URL; after a regenerate or restore, "New URL;
  the old one stopped working." or "Restored with a new URL." shows there.
- **Create**: "+ New bot" opens the same view empty with "Pick an avatar",
  Name, and "Create", then lands on the new bot. Names are checked as typed.
  At ten bots the button disables with "A channel holds ten bots. Delete
  one to add another."
- **Confirms** use the existing confirmation dialog, destructive, naming the
  consequence: "Regenerate CI's URL? Anything still posting to the old URL
  gets a 404 from now on, until you paste the new URL into it." and "Delete
  CI? Anything posting to its URL gets a 404. Its messages stay in the
  channel, and you can restore it later with a new URL."
- **Deleted**: a "› Deleted (n)" fold for editors, rows dimmed with "Deleted
  2d ago by Bob" and Restore, off while the conversation is full.
- **Read-only** sees the rows only: no chevron, no URL, no create, no
  Deleted fold. "Delete bot" shows only with `botwebhook:delete`.

### Gateway, MCP, events, live push, search (ticket 06)

- **Gateway**: `GET /api/conversations/{id}/botwebhooks` (`deleted=true`
  for the restore list), `POST` on the same path to create, `PATCH
  /api/botwebhooks/{id}` for name, avatar, `regenerate`, and `deleted:
  false` to restore, `DELETE /api/botwebhooks/{id}` to soft delete.
- **MCP**: `botwebhook_list` (read-only; URLs only with
  `botwebhook:write`), `botwebhook_create`, and `botwebhook_update` (name,
  avatar, `regenerate`, `deleted` true or false). An ADR raises the tool
  ceiling in `internal/mcp/surface_test.go` from 108 to 111. An agent never
  posts as a bot over MCP; it uses the URL like any sender.
- **Events**: `botwebhook.created`, `botwebhook.updated` (renamed, avatar,
  regenerated), `botwebhook.deleted`, `botwebhook.restored`, each a catalog
  row with an outbox write; no payload carries the token or the URL. A post
  fires the existing `chat.message.created`, whose message carries author
  kind `bot` and the snapshot.
- **Live push**: bot posts reach open conversations through the existing
  `chat.message.created` rule. The `botwebhook.*` topics get a live-audience
  rule scoped to the conversation's audience so an open Bots section
  refreshes.
- **Search**: chat is not full-text indexed today, so bot posts are not
  either. If chat search lands, embed title, description, and field text
  are indexed with the body.

### Permissions (tickets 02, 06)

- `botwebhook:read` sees the section and the bots without URLs;
  `botwebhook:write` sees URLs, creates, renames, changes the avatar,
  regenerates, and restores; `botwebhook:delete` deletes. No implication
  between them; every use-case checks its own. The web reads them from the
  catalog, and the Access page gets an entry.

## Testing Decisions

- The execute route end to end over real SQLite: a GitHub Actions, a
  Grafana, and an Uptime Kuma payload from the research file each become one
  message with the right author, snapshot, and embeds; `wait=true` returns
  the message.
- Rejections: a wrong token and a wrong id answer the same 404 body; the
  per-bot and per-IP limits answer 429 with `retry_after`; an oversize body
  and an empty post answer 400; a deleted or regenerated token answers 404
  at once.
- The token never appears in a log line, an audit row, an event payload, or
  a `botwebhook_list` result for a read-only caller.
- The migration: upgrade from the previous schema keeps every message, and
  a bot message survives its bot's rename and delete with its snapshot.
- Permissions: each use-case refuses the action it does not hold, including
  delete with write only.
- Web: the embed renderer against the worst-case payload (25 fields, a 4096
  character description, ten embeds, unbreakable values), the Discord
  markdown subset, and the Bots section in editor and read-only modes.
- MCP: the surface test at the new ceiling.

## Out of scope

- Outbound delivery of Nexul events to Discord (the integration store's
  Discord reference integration).
- Gateway-connected bots with a login, slash commands, interactive
  components, or anything that receives messages.
- Posting as a bot over MCP.
- Markdown in people's messages.

## Follow-ups left in the fog

- Editing and deleting a bot's own messages through the URL (Discord's
  `PATCH` and `DELETE` on `/messages/{id}`).
- File uploads through the URL (multipart with `payload_json`), which would
  take the attachments cap.
- Discord's `thread_id` query parameter against a bot bound to one
  conversation.
- Compatibility suffixes like `/github` and `/slack`.
- Other bot kinds: an MCP-driven bot with its own token, or a first-party
  bot automations post through.
