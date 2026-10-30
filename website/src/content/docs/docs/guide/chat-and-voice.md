---
title: Chat and voice
description: Talk in channels and direct messages, bring an agent into the conversation, and jump on a voice call with screen share.
sidebar:
  order: 8
---

Conversations sit in the app sidebar in four groups: **Channels**, **Voice channels**, **Direct messages**, and **Threads**. Threads holds doc threads once one exists, and a doc thread's title links back to its doc; a ticket's thread stays on its ticket page. Click **Threads** to fold the group down to its header and again to open it; the sidebar remembers which way you left it. Each row shows its unread count, and a conversation's link opens straight to it. The browser tab's title counts your unread direct messages, such as **(2) Nexul**, so you can see one waiting from another tab.

## Start a conversation

Press **+** beside Channels, Voice channels, or Direct messages. Every workspace starts with `#general`, which can be renamed but never deleted or made private.

A direct message shows in every workspace you and the other people all belong to, not only the one you started it in, so it stays in the sidebar when you switch.

When the sidebar is folded to its icons, the conversation's name at the top of the page opens the same list, with what is unread elsewhere beside it, such as **3 elsewhere**. A channel's header also holds its **Settings** gear, and a private channel's members, which open the same settings.

Right-click a channel in the sidebar to rename it, open its **Settings** or **Permissions…**, or delete it. Deleting removes every message for good and ends a voice channel's call for everyone in it. Creating and renaming take `channels:write`, deleting takes `channels:delete`, and the items are hidden without them.

## Who sees and posts in a channel

Posting in a channel takes **Post in channels** (`channels:post`), which a role allows like any other permission. A channel's own rules can change that for a role or a person: open **Permissions…** from the channel's right-click menu, or **Who sees and posts here** in its **Settings**, pick a role or a person, and set **See this channel** and **Post in this channel** to Allow, Fallback, or Deny.

- **An announcements channel:** allow **Post in this channel** for the role that announces and deny it for the others. Everyone still reads it.
- **Hide a channel from a role:** deny **See this channel**. To them it doesn't exist, as a private channel doesn't. The Owner always sees every channel.
- Someone who reads a channel but may not post in it sees a locked line where the message box was, saying what decided it, such as "Denied on channel #announce for role Client". A member with no role reads the public channels this way, and still messages people directly.

Writing a channel's rules takes `channels:write`, and you can only allow, or lift a Deny on, what you hold there yourself. The Access page's **Public channels** tab sets the same **See this channel** rules for every role at once. Bots post only while the person who created them may post in their channel, and an integration or automation posts with the `channels:post` scope. See [People and access](/docs/guide/people-and-access/).

## Write messages

Messages are markdown. **Enter** sends, **Shift+Enter** starts a new line. Paste, drop, or attach images. Type `@` to mention a member, or `@Agent` to bring in an agent.

Hover a message to react, or to edit or delete your own. A deleted message leaves its place in the conversation.

### Times in everyone's zone

Paste a Discord time code such as `<t:1792867165:F>` and everyone reads it in their own time zone. Make one with any Discord timestamp generator. The letter after the second colon picks the format:

| Code | Shows |
|---|---|
| `t` / `T` | time, without or with seconds |
| `d` / `D` | short or long date |
| `f` (or none) / `F` | date and time, then with the weekday |
| `s` / `S` | short date and time, without or with seconds |
| `R` | relative, such as "in 2 hours", kept up to date |

Hover one to see the full date, time, and zone. Click it for **Copy code**, which copies the code itself so a re-paste stays local for every reader, and **Add to calendar**, which downloads a one-hour `.ics` event named after the channel, ticket, or doc. On the phone, tap one for the full time and the code, with the same **Copy code** and an **Add to calendar** that opens your calendar app's new-event screen with that one-hour event. A code inside backticks stays as typed.

## Bots

Outside tools such as CI, Grafana, or Uptime Kuma can post into any conversation through a webhook URL, the same JSON a Discord webhook takes. Their posts carry a **BOT** tag and show embeds as short reports. Open a channel's **Settings**, or the **Bots** item in a direct message's or thread's menu, to create one. See [Bots](/docs/guide/bots/).

## Ask the agent

Mention `@Agent` and an agent answers in the conversation, running on your own [paired computer](/docs/guide/paired-computers/) with your permissions. Its reply streams in as it works; press the stop button on it to interrupt. If it asks you something, answer the question card in the conversation. If it needs a secret, such as an API key, a secret card asks for it: **Save securely** hands the value to T3 Code on the agent's computer, so it never shows in the conversation or gets stored in Nexul, and **Decline** says no. You can answer in T3 Code instead.

A mention sent while the agent is still busy goes straight into the turn it is working on, and that reply answers your message too, so you can correct it mid-run, a play run included. To have it wait its turn instead, press <kbd>Ctrl</kbd> <kbd>Enter</kbd> (<kbd>⌘</kbd> <kbd>Enter</kbd> on a Mac); the composer reminds you while the agent works. A waiting message shows **Queued** under it until the agent reads it. Press **Steer** beside it to hand it over after all.

On a ticket's or doc's thread the agent works in that project's checkout on the first of [your computers](/docs/guide/paired-computers/#your-computers-in-order) that can take it. In a channel or direct message it works in that computer's chat folder, with your defaults. When the agent hands work to another agent, a pill under its reply opens that agent's conversation.

### Notes on a ticket

An agent can leave a note on a ticket instead of growing its description: a one-line message in the ticket's thread with a file pill. Select the pill to open the file. Once a ticket has a note, its page gains a Notes tab: the newest note opens there, and the list beside it opens any other. Anyone who can edit the ticket edits a note live, and it saves as you type; deleting the message deletes the file.

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
