# 18: Edit a stack's environment, with secrets masked for readers

Type: task
Status: open
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- The Environment section on the stack page: applies on the next deploy, with Deploy now.
- Reveal for writers.
- The server masks secret values for readers.
- Check the automations SDK and the phone for value reads.
- Update `stacks-and-deploys.md`.

## Acceptance criteria

- [ ] A reader's `GET` carries no secret values.
- [ ] A writer can reveal and edit.
- [ ] Removing or blanking a key works.
