# 10 — The Logs view on the stack page

**Type:** task
**Status:** ready-for-agent
**Blocked by:** 08

## What to build

Per decision 06; the design is locked, and screenshots of the prototype go
to the owner before merge.

- **Stack page:**
  - A **Logs** entry in the sub-navigation, with service tabs, reached at
    `/stacks/:id/logs/:service`.
  - A **Logs** link on each Services card row.
  - On the topology canvas, a service node opens its logs.
- **The block:**
  - One terminal-style block: a mono timestamp column plus the raw line, and
    wrapped lines indent under the text.
  - Dense rows, and a copy button in the corner.
  - stderr lines carry a thin status-colored gutter mark.
- **Toolbar:** All / Errors, Pause, Copy (the lines shown), Download (the
  same as `.log`), and a Live marker while following.
- **Scrolling:** scrolling up pauses following; "Jump to live" resumes it.
- **Connection:** the live socket opens on mount and closes on unmount or on
  a tab switch. It reconnects with backoff and a small tail.
- **Performance:** keep the last 5,000 lines in memory and virtualize the
  list.
- **Empty and error states:** "No output yet"; "Runner offline"; "You can't
  read logs for this stack" (and the whole affordance is hidden without
  `stacks:logs`).
- Obey F1 to F7, and build at 768px first.

## Acceptance criteria

- [ ] Tests at the component boundary with a fake socket: lines render, the
      Errors filter works, scrolling up pauses, and unmount closes the
      socket.
- [ ] Verified live at 768, 1024 and 1440px against a real runner on the
      debug stack.
- [ ] `bun run lint`, `typecheck`, `test` and `build` are green in `web/`.
