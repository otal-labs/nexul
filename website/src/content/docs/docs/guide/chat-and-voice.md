---
title: Chat and voice
description: Use workspace conversations, document threads, mentions, and LiveKit voice channels.
sidebar:
  order: 8
---

## Chat

Open `/<workspace>/chat` to see the chat page, where `<workspace>` is the
workspace's slug. A selected conversation has a URL such as
`/<workspace>/chat/<conversation-id>`, so a refresh or shared link opens the same
thread.
The conversation list has two groups:

- **Chats** contains channels, voice channels, and direct messages.
- **Threads** contains document threads. Ticket threads stay on their ticket
  page.

Use the **New conversation** menu to create a channel, a voice channel, or a
direct message. Channel names keep the case they are typed in, and two text
channels can't share a name in any case. Every workspace gets a
`general` channel when it is created.

A channel's or voice channel's `…` menu in the sidebar renames or deletes it.
Creating and renaming take `channels:write`, deleting takes `channels:delete`,
and each item is hidden without its permission. Deleting removes the channel
and every message in it for good, ends a voice channel's call for everyone in
it, and sends anyone viewing it back to the chat page. The workspace's
`general` channel can be renamed but not deleted.

## Private channels

A text or voice channel can be private: only its members see it and read it.
To anyone else it is gone, from the conversation list, unread counts, links,
messages, attachments, and a voice channel's call and occupancy. The
workspace Owner sees every private channel, messages included.

- **Create one private** from the **New conversation** menu and pick its
  starting members. A member who sees only some of a workspace's projects can
  create only private channels.
- **Make a channel private** from its settings with `channels:write`. You stay
  in it and pick who else stays; everyone else loses it at once.
- **Make it public again** with `channels:write`. The member list is dropped
  and the whole workspace can read its history, including what was said while
  it was private.
- **Members.** Anyone in a private channel can add people from the workspace.
  Removing someone else takes `channels:write`. Anyone can leave, except the
  last member.

The workspace's `general` channel is always public.

Messages are markdown. Type `@` to mention a workspace member or the fixed
`@Agent` target. Press Enter to send and Shift+Enter for a new line. The
composer accepts images pasted, dropped, or selected with the attachment
button. A message list request defaults to 50 messages.

The author of a message can edit or delete it. Deletion is soft, so the row's
place in the conversation stays visible. Unread counts are per conversation
and are marked read when the newest visible message is seen.

Document threads need the `docs:thread` permission on the document. The
permission catalog also includes the `chat`, `channels`, and `voice` domains.
Chat routes are available under `/api/chat`; MCP exposes `conversation_list`,
`conversation_update`, `conversation_delete`, `message_list`, and
`message_post`. `conversation_list` marks each channel `private` and lists a
private channel's `member_ids`. `conversation_update` renames a channel,
switches it with `private` (and `member_ids` naming who stays), and adds or
removes members with `add_member_ids` and `remove_member_ids`, where your own
id leaves. `message_list` and `message_post` also take a `doc_id`,
`ticket_id`, or interview `project_id` instead of a conversation id, and
posting starts that thread the first time.

## Voice channels

A voice channel is a conversation with a LiveKit room attached. Create one
from the **New conversation** menu. Its row shows live occupancy. Select the
row to join the call, or select its message button to open the channel's text
chat without joining.

Joining has these states:

- **Join call** requests a short-lived LiveKit token.
- **Connecting** can be cancelled.
- **Voice needs a LiveKit connector** means a workspace owner must configure
  LiveKit under **Settings → Connectors**.
- A connection error offers **Retry** or **Dismiss**.
- A connected call shows the participants and controls for microphone,
  camera, screen sharing, and **Leave call**.
- A shared screen has a **Full screen** button in its corner (or double-click
  it) that fills your monitor with just that screen; press Esc to exit.
- If you are the only person in a call for 5 minutes, you leave it
  automatically, and the channel says why with a **Rejoin** button.

The instance sends occupancy changes to the browser over its live event stream.
The LiveKit connector is the instance's own server connection; Nexul does not
host the media service for you.

## HTTP routes

The browser uses the HTTP gateway, not MCP, for this page. The main routes are
`GET /api/chat/conversations`, `POST /api/chat/channels`,
`POST /api/chat/voice-channels` (both take `private` and `member_ids`), `POST /api/chat/dms`,
`PATCH /api/chat/conversations/{id}` (rename), `DELETE /api/chat/conversations/{id}`,
`PUT /api/chat/conversations/{id}/private`, `POST /api/chat/conversations/{id}/members`,
`DELETE /api/chat/conversations/{id}/members/{userID}`, `POST /api/chat/conversations/{id}/leave`,
`GET /api/chat/conversations/{id}/messages`, and
`POST /api/chat/conversations/{id}/messages`. Voice uses
`POST /api/voice/{conversationID}/token`,
`POST /api/voice/{conversationID}/leave`, and
`GET /api/voice/occupancy`.
