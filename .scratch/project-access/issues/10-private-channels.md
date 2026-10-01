# 10 — Private channels and a restricted member's chat (backend)

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** tickets 01, 04, 06; spec section "Private channels"; ADR 0098

## What to build

Private channels in `internal/chat` and every read that touches a
conversation, plus the chat side of a Restricted member.

1. **Migration 0051**: `conversations.private INTEGER NOT NULL DEFAULT 0`.
   A private channel's members are its `conversation_participants` rows.
   Upgrade test: every existing channel stays public.
2. **Use-cases** (`internal/chat/usecase.go`):
   - `SetChannelPrivate(id, private, keepUserIDs)`, `channels:write`: text
     and voice channels only, `#general` (`is_general`) refused as invalid.
     Going private writes the caller plus `keepUserIDs` (workspace members
     only) as members; going public drops the member rows.
   - `AddChannelMembers`: any member of the private channel, workspace
     members only. `RemoveChannelMembers`: someone else takes
     `channels:write`; yourself is leaving; the last member cannot leave.
   - Each publishes `chat.conversation.members_changed` through the outbox
     (new topic in `events.go` `Topics()` and the catalog: conversation id,
     workspace id, `private`, `added_user_ids`, `removed_user_ids`,
     `actor_id`). A switch to private names everyone who lost it as removed.
3. **Reads.** `requireRead` refuses a private channel to a non-member with
   not found; the Owner passes. `ListConversations` and its query leave out
   private channels the caller is not in; `UnreadCounts`, `ListMessages`,
   `PostMessage`, `MarkRead`, message links, conversation attachments
   (`internal/attachments/usecase.go`: a conversation owner's file goes
   through reading the conversation, not a bare signed-in check), and the
   voice join token (`internal/voice`) all go through the same read.
4. **Restricted member's chat.** A Restricted member reads no public
   channel. `readableKinds` stops asking `tickets:read` and `memories:read`
   once per workspace: ticket, doc, and interview threads are checked per
   thread against their project (`RequireProject`, or the doc check for doc
   threads), so a Restricted member keeps the threads they can read and
   loses the rest. Until ticket 08 lands this answers exactly as today.
   DMs are unchanged, both ways.
5. **Live** (`server/cmd/live_audience.go`): `conversationFrame` inherits
   through `GetConversation`. `chat.conversation.members_changed` reaches
   the channel's readers and its `removed_user_ids`.
   `chat.conversation.deleted` for a private channel carries `member_ids`
   (additive) and reaches only them and the Owner; a public one stays on
   `workspaceFrame`. Voice occupancy inherits.
6. **MCP** (`internal/chat/mcp.go`): `conversation_list` items gain
   `private` and, when private, `member_ids`. `conversation_update` is
   retitled "Update channel" and gains `private`, `member_ids` (who stays
   when going private), `add_member_ids`, and `remove_member_ids` (yourself
   to leave); the description names each permission; omitted fields keep
   their value. Update `internal/mcp/instructions.go`'s "conversation_update
   renames a channel" hint. No new tool.
7. **HTTP**: routes beside the existing channel rename for the switch, add,
   remove, and leave; conversation reads return `private` and member ids.
8. **Docs**: `website/src/content/docs/docs/guide/chat-and-voice.md` and
   `mcp-server.md` if it lists the conversation tools.

## Acceptance criteria

- [ ] Migration 0051 upgrade test from the previous schema
- [ ] Table-driven use-case tests, error paths first: `#general` refused;
      DM and threads refused as invalid; non-member gets not found; remove
      without `channels:write` forbidden; last member cannot leave
- [ ] A non-member cannot list, read, post in, fetch an attachment of, join
      the voice call of, or receive a live frame from a private channel; the
      Owner can
- [ ] Switching to private removes the channel from everyone else's open
      sidebar live; switching back shows it, with history
- [ ] A Restricted member (with ticket 08) lists DMs, their private
      channels, and only the threads they can read
- [ ] MCP tests through `Call` for every new field, including the patch rule
- [ ] Hit every surface: HTTP yes; MCP yes; events (catalog row, outbox) yes;
      live push yes; permissions yes; reverse states (private and back, add
      and remove, leave) yes; docs yes; UI in ticket 13
- [ ] `make lint`, `make coverage`, `make sqlc-check`

## Read first

`practices/go.md`, `practices/architecture.md` (sections 1 to 6),
`practices/mcp.md`, `practices/testing.md`,
`practices/borrowed-practices.md`; ADRs 0042, 0087, 0094, 0098; this
effort's `spec.md` and tickets 04 and 06.

## Files likely touched

- `internal/platform/storage/migrations/0051_*.sql`, `queries/chat.sql`,
  `migration_0051_test.go`, the chat repo in `internal/platform/storage/`
- `internal/chat/` (`usecase.go`, `model.go`, `events.go`, `handler.go`,
  `mcp.go`)
- `internal/attachments/usecase.go`, `internal/voice/`
- `internal/integrations/catalog.go`, `sdk/src/events.generated.ts`
- `server/cmd/live_audience.go`, `internal/mcp/instructions.go`
- `website/src/content/docs/docs/guide/chat-and-voice.md`
