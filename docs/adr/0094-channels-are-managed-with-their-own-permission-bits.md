# Channels are managed with their own permission bits

Amended by ADR 0098: a channel may be private, and reading a private channel takes being one of its members.

A channel or voice channel could be created but never renamed or deleted, by anyone, through any adapter. Creating one
took `chat:write` (ADR 0087), the same bit that starts a direct message, so an owner could not let people talk in DMs
without also letting them add channels to the whole workspace's sidebar.

Decision: a `channels` domain joins the permission grid. `channels:write` creates and renames a channel or voice
channel, and `channels:delete` deletes one. `channels:read` exists only so the role editor's read, write, delete
ladder reads the same as every other domain; reading a channel still takes membership alone, as `chat:read` does.
Starting a DM stays on `chat:write`.

- Rename and delete apply to text and voice channels only. A DM or a ticket, doc, or interview thread is refused as
  invalid; someone who cannot read the conversation gets not found first, and a member without the bit gets
  forbidden. A new name goes through the same rules as creating a channel, so a duplicate text-channel name is the
  same conflict.
- Delete is hard: the channel, its messages, read state, and attachments go in one transaction. Its outbox event,
  `chat.conversation.deleted`, names the id, workspace, kind, and name, since nothing is left to fetch; internal/voice
  consumes it and ends a voice channel's LiveKit room, so everyone in the call is disconnected. A rename publishes
  `chat.conversation.updated` with the old and the new name.
- A workspace's `#general` is marked when the workspace creates it (`is_general`) rather than recognised by name,
  because it may be renamed. It is never deleted.
- Migration 0046 adds both bits to every role, integration install, and automation that held `chat:write`, so
  nothing that could create a channel before the upgrade loses the ability, and marks each existing `#general`.
- Over MCP, `conversation_update` renames and `conversation_delete` deletes. `conversation_list` is a read, and
  ADR 0068 keeps reading, changing, and deleting in separate tools, so the tool ceiling rises from 103 to 105.

The trade-offs: after the upgrade everyone who could start a DM can also rename and delete channels until an owner
trims their role, because the backfill errs toward keeping what people could do. A deleted channel has no restore;
the confirmation names what goes ("Its messages are deleted for good.") instead. Two more tool definitions sit in
every agent session.

Rejected: gating rename and delete on `chat:write`, which kept DMs and channel management inseparable; a soft delete
with restore, which the owner declined for channels; and folding delete into `conversation_update`, which would make
every rename ask for a destructive confirmation.

Amends ADR 0087, whose channel and voice channel creation now takes `channels:write`, and ADR 0090's tool ceiling.
Decided 2026-09-30.
