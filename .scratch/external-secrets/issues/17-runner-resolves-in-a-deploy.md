# 17 — A runner resolves a deploy's references before anything changes

**Status:** resolved

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

## Comments

Built: `internal/runner/secrets.go`. Before any checkout, pull, build or start, the runner resolves the env's references in one batch per manager (duplicates asked once), substitutes them on a copy, and masks the values in command output and errors. Any failure fails the deploy naming variables and plain reasons only. Checked end to end on Docker with a real 1Password item: the container received the value, the deploy log said "Resolved 1 secret reference in 0.7s", and no log, database or file held the value or the reference.
