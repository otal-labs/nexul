# 17 — A runner resolves a deploy's references before anything changes

**Status:** ready-for-agent

**Blocked by:** 11, 16

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, ADR 0181, the spec (Runners, A deploy), ticket 04.

## What to build

- In `internal/runner/executor.go`, before pull, build or `compose up`: collect the references in the
  deploy's env (with the rule's overrides), resolve them in one batch, and substitute the values.
- Any failure fails the deploy, leaves running containers alone, and reports each variable and reason
  (`DB_PASSWORD: field not found`), never a reference or a value. No token means every reference fails
  the same way.
- Resolved values go into the 0600 `.env` and `-e` flags exactly as literals do (`writeComposeEnv`).
  Nothing is kept after the deploy.
- One deploy-log line: the count resolved and the time taken.

## Acceptance criteria

- [ ] With the fake: a failing reference stops the deploy before the first compose or docker command,
      and the error and log hold no reference or value.
- [ ] A mixed env deploys with literals unchanged and references substituted, for compose and run stacks.
- [ ] A rollback and a preview deploy resolve the same way.

Verify: `make lint`, `go test ./internal/runner/...`, `make coverage`.
