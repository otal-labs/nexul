# 24 — A runner's secret check notices a changed value and Nexul records it

**Status:** resolved

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

## Comments

Built: per-stack fingerprints (HMAC-SHA256 with a key made on the machine, 0600 files under the runner's `secret-check/` directory, removed with the stack), the check on `NEXUL_SECRET_CHECK_INTERVAL` (default 1h, refused under 5m, `off`) with one batch per manager across every stack, the `secrets_changed` frame carrying names only, migration 0150 (`stacks.secrets_changed`), `service.secrets_changed` to the stack's viewers, the notice on the stack read and `stack_get`, and clearing on the next healthy deploy. The record keeps each variable's reference beside its fingerprint, since the check must re-read it. Checked end to end with real 1Password on Docker: after rotating the item, the first check put `["SECRET"]` on the stack, and a redeploy delivered the new value and cleared the notice. No log held a value or a reference. This run is what exposed the SDK client lifetime bug fixed in #1281.
