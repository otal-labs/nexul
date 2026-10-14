# 19: Retry a failed deploy

Type: task
Status: resolved
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

## Answer

- A deploy now records the `ref` it was asked to build and the `commit` the runner checked out. The runner reads `git rev-parse HEAD` after the checkout and stamps it on the deploy's result frame, so even a build that fails later reports its commit.
- `POST /api/deploys/{id}/retry` (and `stack_deploy` with `retry` set to a failed deploy's id) starts a new deploy of that commit, else its ref, else its image, with the stack's current settings, needing `deploys:write`. It refuses a deploy that has not failed, and one that recorded nothing to rebuild. `deploy.retried` is written beside `deploy.requested` in one transaction.
- The deploy page shows **Retry** on a failed deploy to anyone holding `deploys:write`, and the header shows the ref and short commit.
- Rollback is hidden when the last healthy deploy has no image: a build leaves none, which was the 409.
- Not pushed live: the new deploy's own `deploy.updated` frames already bring it into the history.
- A commit id is cloned without `--branch`, then fetched and checked out detached, so a retry still builds the recorded commit when its checkout directory is gone.
