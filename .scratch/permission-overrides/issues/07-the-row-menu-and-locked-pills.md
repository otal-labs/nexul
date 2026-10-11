# 07 — The row menu, and locked pills

**Status:** done

**Blocked by:** 06

**Branch:** master, a normal PR, before the switch.

Read first: `practices/react-guide.md` (F1 to F7, UI components, Accessibility, the self-review),
`practices/design-language.md`, `practices/native.md` (section 3), ADR 0028, ADR 0148, the spec (Pills, UI),
`.scratch/permission-rework/issues/03-the-permissions-panel.md` (The row menu), the research (sections 1.8, 7.2 and
7.3).

## What to build

- `components/ui/context-menu.tsx` through the shadcn CLI, restyled to the house menu anatomy as rework ticket 03
  locks it, with the `cn` import changed to `@/lib/utils` and the npm package `cn` kept out. Confirm the `radix-ui`
  package already carries it; no new dependency.
- The menu on doc, folder, project, channel and play rows in the sidebar and lists, opening at the pointer, with the
  row keeping `bg-accent` while it is open. `RowActions` ⋯ wherever a row lacks one, showing the same items.
- Items: the row's existing actions, a hairline, **Permissions…** with a shield icon, a hairline, then Delete in
  `destructive` where the row has one. Permissions… shows on docs and plays only, to people who can manage the
  entity's access; projects get it in slice 14, folders in 18, channels in 20.
- The "You don't have access" tooltip on locked doc and ticket pills on the web (`MentionChip.tsx:39-82`). Check how
  the phone renders a locked pill and note it in the PR.

## Acceptance criteria

- [x] Right-click opens at the pointer, and the row keeps `bg-accent` while the menu is open.
- [x] ⋯ shows the same items, and the keyboard works.
- [x] "Permissions…" shows only to people who can manage the entity's access.
- [x] A locked pill shows the tooltip and stays inert.
- [x] Verified at 768, 1024 and 1440px, and the self-review passes.
