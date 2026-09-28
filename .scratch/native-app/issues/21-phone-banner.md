# 21 — Banner for phones on the web app

**Type:** implementation
**Status:** done
**Blocked by:** None — can start immediately
**Decided in:** ticket 04

## What to build

Below 768px the web app shows a dismissible banner, never a block. On an
Android user agent: "Nexul is built for tablet and desktop." with a "Get the
Android app" link to the project's GitHub releases; everywhere else: "Nexul
is built for tablet and desktop." Dismissal persists per device through a
zustand persist store. Built from the shared display components, monochrome.

## Acceptance criteria

- [ ] Appears below 768px only, on every signed-in page and the login page
- [ ] The Android variant links to the releases page; other phones get no link
- [ ] Dismissed stays dismissed after reload on that device
- [ ] Verified at 320, 375 and 768px (it exists for phone widths)

## Surfaces

- UI only

## Read first

`practices/react-guide.md`, `practices/design-language.md`, `practices/testing.md`, ADR 0080.

## Verification

In `web/`: `bun run lint`, `bun run typecheck`, `bun run test`; screenshots at 320, 375, 768px.

## Files likely touched

- `web/src/Layout.tsx`, a new `web/src/components/PhoneBanner.tsx`, a new store in `web/src/stores/`

**Size:** S
