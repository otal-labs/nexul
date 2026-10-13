# 15: An item's own page locks by its rules

Type: task
Status: open
Blocked by: 09

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, `practices/testing.md` (section 10), the spec.

## What to build

- `useEntityActions(kind, id)` calls explain once per detail page, for docs, plays, projects, and stacks under the shipped
  rules (ADR 0148). It refetches on the rules' live frame for that item.
- A list action refused on click toasts the source sentence.
- A statement guard covers explain.

## Acceptance criteria

- [ ] A rule on one doc locks that doc page's action with the rule's sentence.
- [ ] A list keeps one request per page.
