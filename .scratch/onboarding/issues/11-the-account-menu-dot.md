# 11: Welcome in the account menu, and its red dot

Type: task
Status: open
Blocked by: 09

Read first: `AGENTS.md`, `CONTEXT.md` (Welcome wizard, Template role), `practices/react-guide.md`, `practices/design-language.md`, ticket 05's answer (Account menu).

## What to build

- The red dot on the account menu trigger, in the full sidebar and the icon rail. It reuses `UpdateDot`'s shape
  in the destructive hue, and the trigger's label says how many steps are left.
- The Welcome row at the top of the menu, one row per step that isn't done, and "6/6" when every step is done.
  Each row opens the wizard on its step.

## Acceptance criteria

- [ ] Web tests: the dot shows for a left step, not for a skipped one, and not for an ungiven `projects`.
- [ ] The dot clears live when a step is done in another tab. Verified at 768, 1024, 1440.
