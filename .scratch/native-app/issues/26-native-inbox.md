# 26 — Inbox on the phone

**Type:** implementation
**Status:** done
**Blocked by:** 25
**Decided in:** tickets 11, 12

## What to build

The Inbox tab: notifications newest first as hairline rows (subject title,
kind, relative time in mono, unread marker), pull to refresh, live updates
from the events client, tap to mark read and open the subject (ticket, doc,
or conversation screens from tickets 27 to 29; until those exist, a stub
screen with the title), Mark all read in the header. Unread count as the tab
badge. Use the same endpoints as `web/src/hooks/NotificationHooks.tsx`.

## Acceptance criteria

- [ ] Rows match the web inbox for the same user
- [ ] Mark read and mark all read persist
- [ ] The tab badge tracks unread

## Surfaces

- UI (native)

## Read first

`AGENTS.md`, `practices/native.md` (written by ticket 24), `practices/react-guide.md` (F1–F7 and the self-review checklist, applied to React Native), `practices/testing.md`, ticket 11.

## Verification

In `native/`: `bun run lint`, `bun run typecheck`, `bun run test`. Screenshots from an Android emulator or a web preview of the screens at a phone width, kept out of git.

## Files likely touched

- `native/src/app/(tabs)/inbox/`
- `native/src/api/notifications.ts`

**Size:** M
