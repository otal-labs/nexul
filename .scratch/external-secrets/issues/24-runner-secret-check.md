# 24 — A runner's secret check notices a changed value and Nexul records it

**Status:** ready-for-agent

**Blocked by:** 17

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/mcp.md`, `practices/react-guide.md` (The live topic contract), ADRs 0044,
0137, the spec (Runners, Noticing a changed value), ticket 08.

## What to build

- At each deploy the runner writes keyed-hash fingerprints of the resolved values (a key generated on
  the machine) to a 0600 file in its directory, removed with the stack.
- Every `NEXUL_SECRET_CHECK_INTERVAL` (default 1h, minimum 5m, `off`), it re-resolves every deployed
  stack's references, previews included, in one batch per token and reports changed variable names.
- The server stores the notice on the stack, clears it on the next deploy, retry or rollback, publishes
  `service.secrets_changed` (stack id, variable names) with a `Topics()` entry, `make event-schemas`,
  a `liveRules` audience of the stack's viewers and `make live-topics`, and adds it to `stack_get`.

## Acceptance criteria

- [ ] With the fake and a fake clock: a changed value is reported once per change, with names only.
- [ ] An interval under 5m is refused; `off` makes no reads.
- [ ] The event and MCP result never carry a value or a reference; the live-topic guards pass.

Verify: `make lint`, `make event-schemas`, `make live-topics`, `go test ./internal/... ./server/...`.
