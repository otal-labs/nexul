---
title: Bots
description: Let outside tools post into a conversation through a webhook URL. Anything that already posts to a Discord webhook works.
sidebar:
  order: 8
---

A bot is an outside tool posting into one [conversation](/docs/guide/chat-and-voice/) through a URL. CI runs, alerts, uptime checks and release notes land in the channel your team already reads, with their embeds shown as cards.

Nexul accepts the same JSON a Discord webhook does. A tool that already posts to a Discord webhook works by swapping the URL, which differs in two places: the host, and `webhooks` becomes `botwebhooks`.

```text
Discord: https://discord.com/api/webhooks/123/abc
Nexul:   https://nexul.example.com/api/botwebhooks/9f2c.../Tq7...
```

You never build that by hand: Nexul gives you the finished URL to copy.

## Create a bot

1. Open the conversation's bots. For a channel or voice channel, open its **Settings** from the **…** menu in the sidebar and find **Bots**. For a direct message, a ticket's thread, or a doc thread, choose **Bots** in the conversation's **…** menu.
2. Press **New bot**, pick an avatar if you like, type a name, and press **Create**.
3. Copy the **Webhook URL**.

Each bot belongs to one conversation. To post to three channels, make three bots. A conversation holds up to ten, and a name is unique within it, ignoring case. Without an avatar a bot shows a Nexul glyph.

Anyone holding `botwebhook:read` sees the list of bots. Creating one, seeing its URL, and changing it take `botwebhook:write`. Opening a ticket's or doc's thread still requires reading that ticket or doc.

## Point your tools at it

Treat the URL like a password: whoever holds it can post as the bot. Paste it where the tool asks for a Discord webhook URL.

**GitHub Actions.** Save the URL as a repository secret, say `BOT_WEBHOOK_URL`. A Discord-notify action takes it as its webhook input:

```yaml
with:
  webhook: ${{ secrets.BOT_WEBHOOK_URL }}
```

Or post with a plain step:

```yaml
- run: |
    curl -sS -H "Content-Type: application/json" \
      -d '{"content":"Deploy of ${{ github.repository }} finished"}' \
      "$BOT_WEBHOOK_URL"
  env:
    BOT_WEBHOOK_URL: ${{ secrets.BOT_WEBHOOK_URL }}
```

**Grafana.** Under **Alerting → Contact points**, add a contact point of type **Discord** and paste the URL as its webhook URL. Point a notification policy at it.

**Uptime Kuma.** In **Settings → Notifications**, set up a notification of type **Discord** and paste the URL as the webhook URL. Turn on **Default enabled** if every monitor should use it.

Most other tools that list Discord as a destination work the same way.

## What shows in chat

- **Author row.** The bot's avatar and name, a **BOT** tag, and the time. A post can carry its own `username` and `avatar_url`; Nexul shows those, and adds **via** and the bot's name so a post calling itself "GitHub" can't pass for a person. Messages keep the name and avatar they were posted with, even after the bot is renamed or deleted.
- **Text and embeds.** The text and every embed share one bubble. An embed is a card with an author line, title (a link when it has a URL), description, a table of fields, an image or thumbnail, and a footer. The sender's color is dropped, because a status hue would be misread.
- **Long posts fold.** Past six fields, a very long description, or more than two embeds, a bar such as **Show 19 more fields** opens the rest and becomes **Show less**.
- **Formatting.** A bot's text renders Discord's subset: bold, italics, inline code, `[text](url)` links, lists, quotes, and `<t:…>` time codes.

### Mentions

`@alice` in a bot's post highlights and notifies alice the way a person's mention does. The post's `allowed_mentions` is honoured: leave it out and every named person is mentioned, send `{"parse": []}` to mention nobody, or list `users` (Nexul ids or logins) to mention only those. Roles and `@everyone` are ignored. `@Agent` never starts an agent from a bot's post.

A bot's post counts as unread like anyone's. There is no per-bot mute.

## Manage a bot

Open a bot from the list to change it.

- **Rename or change the avatar.** Save the name once you edit it.
- **Regenerate URL.** Issues a new URL. The old one stops working at once with a `404`, so paste the new one into every tool that posts to it.
- **Delete bot.** The URL stops working and the name is freed. Its messages stay in the conversation.
- **Restore.** Under **Deleted** in the list, press **Restore**. The bot comes back with a new URL, never the old one, and only while the conversation holds fewer than ten.

Seeing a bot's URL takes `botwebhook:write`; deleting takes `botwebhook:delete`. Someone with only `botwebhook:read` sees the bots but no URL and no controls. Set these per role in [People and access](/docs/guide/people-and-access/). Deleting a conversation deletes its bots.

## For senders

Send `POST /api/botwebhooks/{id}/{token}` with a JSON body. No other credential is needed.

```sh
curl -sS -H "Content-Type: application/json" \
  "https://nexul.example.com/api/botwebhooks/BOT_ID/TOKEN?wait=true" \
  -d '{
    "username": "Acme Monitor",
    "content": "Checkout is down",
    "embeds": [{
      "title": "checkout.example.com",
      "description": "No answer for 3 minutes",
      "fields": [{ "name": "Region", "value": "eu-west" }],
      "footer": { "text": "Acme Monitor" }
    }]
  }'
```

### What a post may carry

`content`, `username`, `avatar_url`, `embeds`, and `allowed_mentions`. A post needs `content` or at least one embed. An embed keeps its title, description, url, timestamp, footer, image, thumbnail, author, and fields; `color` is accepted and ignored. Links and images must be absolute `http` or `https` URLs. Other Discord fields are ignored. File uploads (a `multipart/form-data` post) are not supported and answer `400`.

### Limits

| What | Limit |
|---|---|
| Posts | 30 a minute per bot |
| Body | 64 KiB of JSON |
| `content` | 2000 characters |
| Embeds | 10 a post, 6000 characters of text across them |
| Embed title, field name | 256 characters |
| Embed description | 4096 characters |
| Embed fields | 25, each value up to 1024 characters |
| Footer text | 2048 characters |
| `username` | 80 characters |

### Answers

| Status | Meaning |
|---|---|
| `204` | Posted. |
| `200` | Posted, with `?wait=true`: the body is the message. |
| `400` | The payload breaks a limit or is empty. The body names the field. |
| `404` | Unknown webhook, code `10015`. A wrong id, a wrong token, and a deleted or regenerated URL all answer alike. |
| `429` | Over 30 posts a minute, or too many wrong URLs from your address. The body holds `retry_after` in seconds, and `Retry-After` and `X-RateLimit-*` headers come with it. |

## Over MCP

Agents manage bots with `botwebhook_list`, `botwebhook_create`, and `botwebhook_update` (rename, avatar, regenerate, delete, restore). URLs appear only to a caller holding `botwebhook:write`. An agent doesn't post as a bot over MCP; it posts with `message_post` as itself, or uses a bot's URL like any sender. See [MCP server](/docs/guide/mcp-server/).
