# 20 — Channel rules and posting

**Status:** done

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

- [x] An `access.Matrix` diff on a production copy that is empty for posting. Replaced, by the owner's call, with no
      production copy: a production-shaped fixture at master's schema (two workspaces, Owners, roles with and without
      `chat:write`, members holding one role, two and none, public, private, voice and general channels, a setup-grid
      rule hiding a channel, DMs, an integration install and an automation), run through `TestAccessMatrix_OfACopy` on
      master, where posting is reading (ADR 0087), and on this branch. The posting diff is empty for every member
      holding a role, the role without `chat:write` included; a member holding no role loses posting in the public
      channels they read, the one change, which the owner agreed on 2026-10-13 (ADR 0149).
- [x] `TestChannelRules_OnlyTheRolePosts` over HTTP, MCP, a bot webhook and an automation.
- [x] A rule set on the panel and one set on the Access page's channel grid show the same state in both
      (`TestChannelRules_ThePanelAndTheAccessGridAgree`).
- [x] The chat rows of `TestStatements_ListsCostTheSameAtAnyLength` keep their counts.
- [x] Verified at 768, 1024 and 1440px.
