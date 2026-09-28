# 27 — Chat on the phone

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 25
**Decided in:** tickets 11, 12

## What to build

The Chat tab: conversations (channels, direct messages, ticket and doc
threads) as rows with last message preview and time; a thread screen with
messages (markdown rendered with react-native-enriched-markdown, images
displayed, author and mono time), a composer that sends text, live updates
over the events client, and the list scrolled to the newest message (Legend
List). Same endpoints as `web/src/hooks/ChatHooks.tsx`. No reactions, voice,
or sending images.

## Acceptance criteria

- [ ] A message sent from the phone appears on the web live, and the reverse
- [ ] Long threads scroll smoothly and open at the bottom
- [ ] Markdown and images render

## Surfaces

- UI (native)

## Read first

`AGENTS.md`, `practices/native.md` (written by ticket 24), `practices/react-guide.md` (F1–F7 and the self-review checklist, applied to React Native), `practices/testing.md`, `practices/borrowed-practices.md`, ticket 11.

## Verification

In `native/`: `bun run lint`, `bun run typecheck`, `bun run test`. Screenshots from an Android emulator or a web preview of the screens at a phone width, kept out of git.

## Files likely touched

- `native/src/app/(tabs)/chat/`
- `native/src/api/chat.ts`

**Size:** L
