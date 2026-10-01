# 13 — Web: private channels and the restricted sidebar

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 10
**Decided in:** tickets 04, 07; spec section "Private channels"

## What to build

The web side of ticket 10. The visual treatment follows ticket 07 as it
stands when you start.

1. **Private channel row** in the channel's settings (where rename lives
   today, reached from `hooks/useChannelRowActions.tsx` and
   `components/chat/RenameChannelForm.tsx`): lock icon, "Private channel",
   "Only members see it and read it.", and a switch; shown for text and
   voice channels with `channels:write`, never for `#general`.
2. **Who stays**: turning it on opens "Who stays in #<channel>?" with a
   count, a search field, and a checkable people list, the person switching
   checked and fixed. Turning it off confirms that the whole workspace will
   read its history.
3. **Members list** for a private channel: a count, "Add people" (any
   member), a `…` per row to remove (with `channels:write`), and a
   destructive "Leave channel" (hidden for the last member).
4. **Sidebar**: a private channel shows a trailing muted lock. A Restricted
   member's sidebar shows only their projects, only their private channels,
   and DMs; nothing else changes because the server already filters.
5. **Live** (`hooks/useLiveEvents.tsx`, `hooks/ChatHooks.tsx`): on
   `chat.conversation.members_changed`, refetch conversations; a channel
   removed from the caller drops from the sidebar, and an open one shows
   the not-found state.
6. Every change confirms with a toast naming the channel.

## Acceptance criteria

- [ ] Making a channel private with two people kept removes it from a third
      person's open sidebar live; making it public brings it back
- [ ] Add, remove, and leave work and the last member cannot leave
- [ ] A Restricted member (with ticket 08) sees their projects, their
      private channels, and DMs only
- [ ] Component tests for each state, error paths first
- [ ] Self-review checklist of `practices/react-guide.md` run; F1 to F7 hold
- [ ] Checked at 768, 1024, and 1440px with worst-case names
- [ ] Hit every surface: UI yes; HTTP and MCP in ticket 10; live push yes;
      reverse states (private and back, add and remove) yes; docs:
      `chat-and-voice.md` updated in ticket 10, check the screenshots and
      wording match
- [ ] `bun run lint && bun run typecheck && bun run test` in `web/`

## Read first

`practices/react-guide.md` (F1 to F7 and the self-review checklist),
`practices/design-language.md`, `practices/typescript.md`,
`practices/testing.md`, `practices/borrowed-practices.md`; ADRs 0080,
0094, 0098; this effort's `spec.md` and tickets 04 and 07.

## Files likely touched

- `web/src/components/chat/`
- `web/src/hooks/useChannelRowActions.tsx`, `ChatHooks.tsx`,
  `useLiveEvents.tsx`
- the sidebar's channel list component
- `web/src/models/` conversation model
