---
title: Chat and voice
description: Talk in channels and direct messages, bring an agent into the conversation, and jump on a voice call with screen share.
sidebar:
  order: 8
---

Conversations sit in the app sidebar in four groups: **Channels**, **Voice channels**, **Direct messages**, and **Threads**. Threads holds doc threads once one exists; a ticket's thread stays on its ticket page. Each row shows its unread count, and a conversation's link opens straight to it.

## Start a conversation

Press **+** beside Channels, Voice channels, or Direct messages. Every workspace starts with `#general`, which can be renamed but never deleted or made private.

A channel's **…** menu in the sidebar renames it, opens its **Settings**, or deletes it. Deleting removes every message for good and ends a voice channel's call for everyone in it. Creating and renaming take `channels:write`, deleting takes `channels:delete`, and the items are hidden without them.

## Write messages

Messages are markdown. **Enter** sends, **Shift+Enter** starts a new line. Paste, drop, or attach images. Type `@` to mention a member, or `@Agent` to bring in an agent.

Hover a message to react, or to edit or delete your own. A deleted message leaves its place in the conversation.

## Bots

Outside tools such as CI, Grafana, or Uptime Kuma can post into any conversation through a webhook URL, the same JSON a Discord webhook takes. Their posts carry a **BOT** tag and show embeds as cards. Open a channel's **Settings**, or the **Bots** item in a direct message's or thread's menu, to create one. See [Bots](/docs/guide/bots/).

## Ask the agent

Mention `@Agent` and an agent answers in the conversation, running on your own [paired computer](/docs/guide/paired-computers/) with your permissions. Its reply streams in as it works; press the stop button on it to interrupt. If it asks you something, answer the question card in the conversation.

On a ticket's or doc's thread the agent works in that project, through your project link. In a channel or direct message it uses your defaults. When the agent hands work to another agent, a pill under its reply opens that agent's conversation.

### Notes on a ticket

An agent can leave a note on a ticket instead of growing its description: a one-line message in the ticket's thread with a file pill. Select the pill to open the file. Anyone who can edit the ticket edits it live, and it saves as you type; deleting the message deletes the file.

## Private channels

A private channel is seen and read only by its members. To everyone else it doesn't exist: not in the list, not by link, and not its call. The workspace owner sees every one. A lock marks it in the sidebar.

- **Create one private:** switch on **Private channel** in the create dialog and pick who's in it.
- **Make an existing channel private:** open **Settings** from its **…** menu and switch on **Private channel**, with `channels:write`. You choose who stays; everyone else loses it at once.
- **Make it public again:** switch it off. The whole workspace can then read its history, including what was said while it was private.
- **Members:** anyone in it can **Add people**. **Remove from channel** takes `channels:write`. Anyone can **Leave channel** except the last member.

Automations and integrations never receive a private channel's events, or a direct message's. A member who sees only some of the workspace's projects can create only private channels. See [People and access](/docs/guide/people-and-access/).

## Voice channels

A voice channel is a call with its own text chat. Its row shows who's in the call without joining.

1. Select the voice channel's row to join the call. Its message button opens the text chat without joining; **Join call** there joins.
2. While connected, use the buttons to mute your microphone, turn on your camera, **Share screen**, or **Leave call**.
3. A shared screen has a **Full screen** button; Esc exits.

If you're alone in a call for 5 minutes you leave it automatically, and the channel offers **Rejoin**.

Voice runs on your own LiveKit server. Until someone connects it in **Settings → Connectors**, joining shows "Voice needs a LiveKit connector" with a link to set it up for those who can. A failed connection offers **Retry**.
