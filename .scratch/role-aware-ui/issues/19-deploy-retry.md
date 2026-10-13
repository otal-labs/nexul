# 19: Retry a failed deploy

Type: task
Status: open
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- A deploy records the commit it built.
- Retry rebuilds that commit with the stack's current settings, publishing `deploy.retried`.
- The web shows Retry on a failed deploy.
- Rollback is hidden without an image.

## Acceptance criteria

- [ ] Retry after fixing a missing environment variable deploys the same commit with the new value.
- [ ] Rollback no longer returns 409 from the UI.
