# 09: The action state and the shared action components

Type: task
Status: open
Blocked by: 08

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/typescript.md` (section 11), `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, `practices/native.md`, `practices/react-guide.md`, the spec.

## What to build

- `client-core`: `actionState` and the "Your role can't <verb>" line from the catalog's labels, with tests.
- Web: `ActionButton`, `ActionMenuItem`, `ActionInlineEdit` and `ActionDragHandle` in ticket 04's look. That
  means `aria-disabled`, `aria-describedby` and the tooltip on focus.
- Phone: the matching components, with tap for the reason and a 44pt target.
- `practices/react-guide.md` and `practices/native.md` say every write goes through them.

## Acceptance criteria

- [ ] Table tests for live, locked and hidden across the switch, an entity answer, and the Owner.
- [ ] Component tests for keyboard focus and the announced reason.
- [ ] Verified at 768, 1024 and 1440, and on the emulator.
