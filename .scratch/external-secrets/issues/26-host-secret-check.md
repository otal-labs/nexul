# 26 — The host's secret check restarts an automation once it is idle

**Status:** resolved

**Blocked by:** 20

Read first: `practices/typescript.md`, ADR 0046, the spec (Automations hosts), ticket 08.

## What to build

- On `NEXUL_SECRET_CHECK_INTERVAL` (default 1h, minimum 5m, `off`), the supervisor re-resolves each
  workspace's references in one batch per token and compares with what it handed out.
- An automation whose value changed is restarted as soon as no run is in flight, or after 5 minutes at
  most. A config change's restart (`supervisor.ts`) stays immediate.

## Acceptance criteria

- [ ] With the fake and a fake clock: an idle automation restarts at once, a busy one when its run
      ends, and a stuck one after 5 minutes.
- [ ] One read per workspace token per interval; `off` makes none.

Verify: `bun run typecheck` and `bun run test` in `automations/`.

## Comments

Built: `automations/src/secret-check.ts` and `secret-fingerprints.ts`. The supervisor keeps an HMAC-SHA256 per reference handed in at connection, never the value, keyed by `secret-check.key` beside the credential file (0600, or in memory when that directory can't hold it). It re-reads each workspace's references on `NEXUL_SECRET_CHECK_INTERVAL` in one batch per workspace token (default 1h, refused under 5m, `off`), and restarts a changed automation at once when idle, when its run ends, or after 5 minutes. The SDK's dial-in client gains an optional `onRunState` callback so the host knows when a run is in flight. A reference that fails to resolve counts as unchanged.
