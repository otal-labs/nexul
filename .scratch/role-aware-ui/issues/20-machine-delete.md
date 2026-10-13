# 20: Delete a machine, refused while stacks deploy to it

Type: task
Status: open
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- `DELETE /api/machines/{id}` and MCP `machine_delete`: refused while stacks target the machine, naming them.
- Publishes `machine.deleted` with a live follower.
- The web adds Delete on the machine card.
- The ADR for the refusing deletes.
- Update `runners.md`.

## Acceptance criteria

- [ ] Use-case and integration tests for the refusal and the delete.
- [ ] The offline card goes.
