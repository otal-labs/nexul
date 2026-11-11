# 20 — The host's supervisor resolves an automation's pool references at connection

**Status:** ready-for-agent

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
