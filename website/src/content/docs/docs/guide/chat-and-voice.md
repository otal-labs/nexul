---
title: Chat and voice
description: Use workspace conversations, document threads, mentions, and LiveKit voice channels.
sidebar:
  order: 8
---

## Chat

Open `/chat` to see the chat page. A selected conversation has a URL such as
`/chat/<conversation-id>`, so a refresh or shared link opens the same thread.
The conversation list has two groups:

- **Chats** contains channels, voice channels, and direct messages.
- **Threads** contains document threads. Ticket threads stay on their ticket
  page.

Use the **New conversation** menu to create a channel, a voice channel, or a
direct message. Channel names are stored in lower case. Every workspace gets a
`general` channel when it is created.

Messages are markdown. Type `@` to mention a workspace member or the fixed
`@Agent` target. Press Enter to send and Shift+Enter for a new line. The
composer accepts images pasted, dropped, or selected with the attachment
button. A message list request defaults to 50 messages.

The author of a message can edit or delete it. Deletion is soft, so the row's
place in the conversation stays visible. Unread counts are per conversation
and are marked read when the newest visible message is seen.

Document threads need the `docs:thread` permission on the document. The
permission catalog also includes the `chat` and `voice` domains. Chat routes
are available under `/api/chat`; MCP exposes `conversation_list`,
`message_list`, and `message_post`. The last two also take a `doc_id`,
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

The instance sends occupancy changes to the browser over its live event stream.
The LiveKit connector is the instance's own server connection; Nexul does not
host the media service for you.

## HTTP routes

The browser uses the HTTP gateway, not MCP, for this page. The main routes are
`GET /api/chat/conversations`, `POST /api/chat/channels`,
`POST /api/chat/voice-channels`, `POST /api/chat/dms`,
`GET /api/chat/conversations/{id}/messages`, and
`POST /api/chat/conversations/{id}/messages`. Voice uses
`POST /api/voice/{conversationID}/token`,
`POST /api/voice/{conversationID}/leave`, and
`GET /api/voice/occupancy`.
