# 20 — Channel rules and posting

**Status:** ready-for-agent

**Blocked by:** 13, 17

**Branch:** master, a normal PR, after the switch.

Read first: `practices/go.md` (section 17), `practices/mcp.md` (sections 4, 11), `practices/react-guide.md`,
`practices/testing.md` (section 10), ADRs 0087, 0094, 0098, 0129, 0140, 0148, the spec (Channels and plays stay
workspace-wide, Entities).

## What to build

- The channel panel, with role and person targets, in the channel's settings and on its row menu ("Permissions…").
  Rules set `channels:read` and `channels:post`, written with `channels:write` on the channel, under
  hold-what-you-give. A Deny of `channels:read` hides the channel as a private one is hidden; the Owner still sees it.
  Private channel membership stays as it is.
- `channels:post`, with a forward-only migration granting it to every role, integration install and automation that
  holds `chat:write` today (the pattern of migration 0046), so nobody's posting changes.
- Every path that posts answers through the posting check: the web, `message_post`, bots, integrations, automations
  and `@Agent` replies.
- `permission_overwrite_list` and `_update` cover channel posting; explain covers channels.
- The guide page `chat-and-voice.md`.

## Acceptance criteria

- [ ] An `access.Matrix` diff on a production copy that is empty for posting.
- [ ] `TestChannelRules_OnlyTheRolePosts` over HTTP, MCP, a bot webhook and an automation.
- [ ] A rule set on the panel and one set on the Access page's channel grid show the same state in both.
- [ ] The chat rows of `TestStatements_ListsCostTheSameAtAnyLength` keep their counts.
- [ ] Verified at 768, 1024 and 1440px.
