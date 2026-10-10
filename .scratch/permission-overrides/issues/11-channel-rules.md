# 11 — Channel rules

**Status:** ready-for-agent

**Blocked by:** 06

Read first: `practices/go.md` (section 17), `practices/mcp.md`, `practices/react-guide.md`,
`practices/testing.md`, ADRs 0087, 0094, 0098, 0140, 0148, the spec (Entities).

## What to build

- A channel takes Everyone, role and person rules, for example an announcements channel where only one role
  posts. Posting today needs only reading the conversation (ADR 0087), so this ticket first adds a posting
  permission, `channels:post`, and a forward-only migration that grants it to every role, integration install
  and automation that holds `chat:write` today (the pattern of migration 0046), so nobody's access changes.
- Rules set `channels:read` and `channels:post` and are written with `channels:write` on the channel. A Deny of
  `channels:read` hides the channel as a private one is hidden (ADR 0098) and the Owner still sees it. Private
  channel membership stays as it is.
- Every path that posts (the web, `message_post`, bots, integrations, automations, `@Agent` replies) answers
  through the posting check; reads follow the read check, in lists, unread counts, links, search, voice join
  and live frames.
- `permission_overwrite_list` and `_update` accept `resource_type: channel`; explain covers channels. The
  channel's settings gain the permissions panel limited to those two actions.
- The guide page for chat.

## Acceptance criteria

- [ ] `TestChannelRules_OnlyTheRolePosts` over HTTP, MCP, a bot webhook and an automation.
- [ ] `TestMigration_EffectiveAccessUnchanged` gains `chat:write` holders; the production-copy matrix diff is
      empty.
- [ ] A denied channel is not found in lists, unread counts, search and live frames, and the Owner sees it.
- [ ] The statement guards stay green.
- [ ] Verified at 768, 1024 and 1440px.
