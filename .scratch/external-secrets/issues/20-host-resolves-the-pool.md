# 20 — The host's supervisor resolves an automation's pool references at connection

**Status:** resolved

**Blocked by:** 19

Read first: `practices/typescript.md`, ADRs 0046, 0047, 0182, the spec (Automations hosts).

## What to build

- An automation's hello passes through the supervisor, which resolves the workspace's references with
  that workspace's token and hands the values into the thread, once per connection.
- Hellos of one workspace that arrive within about a second are resolved in one batch.
- A failed entry is left out of `ctx.secrets`; the automation's log and its page say
  `<NAME>: field not found`, never the reference. A workspace with no token fails each reference so.
- Web: an automation whose pool has references its host cannot resolve shows a warning on its page.

## Acceptance criteria

- [ ] With the fake: an automation gets only its own workspace's values, even when it asks for another's.
- [ ] Five automations of one workspace connecting together make one resolve call.
- [ ] A failed entry is absent from `ctx.secrets` and named, without the reference, in the log.

Verify: `bun run typecheck` and `bun run test` in `automations/`, `sdk/` and `web/`;
`go test ./internal/automations/...`.

## Comments

Until a host holds tokens for one workspace only (ADR 0182, as amended), the pool refuses a reference on save and `nexul automations 1password` refuses to store a token (`--remove` still works). Lifting both, and their two tests, is part of this ticket.

A host now holds one workspace's token and runs only that workspace's automations, and `nexul automations 1password` stores tokens again, keyed by workspace id (`WorkspaceManagers.resolverFor` takes the assignment's `workspace_id`). The pool's refusal of references and its test (`internal/automations/secrets.go`) are still this ticket's to lift.

Built: a worker thread asks the supervisor, over its message channel, to resolve the references in its hello pool, and the supervisor answers with the values of that worker's own workspace's token only, since the thread never names a workspace. Requests of one workspace within a second share one `resolveAll`. A failed name is left out of `ctx.secrets` and logged as `<NAME>: <reason>`. A resolve also moves the reported state (unreachable to ready and back), sent at the next poll. The pool accepts references again. The automation read carries `pool_has_references`, and the automation page warns when its host has no ready manager for the workspace. A bring-your-own SDK automation has no supervisor, so it still receives the pool as stored.
