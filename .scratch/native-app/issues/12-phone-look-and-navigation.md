# 12 — Phone look and navigation

**Type:** prototype
**Status:** resolved
**Blocked by:** 10, 11

## Question

How does the Mono Console translate to a phone? Navigation shape (bottom tabs, and which), sheets versus pushed screens, how the theme library and light/dark carry over, type and spacing at phone sizes. Prototype through `design-mode` on a device or emulator for the owner to react to.

## Answer

Decided 2026-09-28 without a prototype round, on the owner's delegation. The
owner reacts to screenshots of the built app instead.

- **The Mono Console carries over:** the monochrome surface stack (background,
  surface-2, card, popover), 1px hairlines, color only for status, Inter for
  text, JetBrains Mono for ids, counts and timestamps. Dark by default, and it
  follows the system appearance. The twelve web palettes come later.
- **Navigation:** a native bottom tab bar (the Expo Router tabs) with the
  five tabs from ticket 11. Detail screens push onto a stack with the
  platform back gesture. Pickers (status, project, workspace) and confirms
  open as native form sheets.
- **Lists** are hairline rows at 44px or more of touch height: the primary
  field left, meta right in muted mono. There are no cards except where the
  web uses one.
- **Motion:** the platform defaults for push and sheet. Row press uses the
  pressed-state tint, and nothing else animates in the first cut.
