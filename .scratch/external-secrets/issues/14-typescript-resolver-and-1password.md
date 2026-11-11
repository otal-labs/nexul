# 14 — The TypeScript resolver interface, its 1Password implementation and a fake

**Status:** ready-for-agent

**Blocked by:** 11

Read first: `practices/typescript.md`, ADR 0182, the spec (The seam for a second manager), ticket 07.

## What to build

- In `automations/src/`: the same `Resolver` shape as ticket 13 (scheme, batch resolve, typed errors),
  a registry by scheme, a 1Password resolver on `@1password/sdk` pinned at exactly `0.5.0`, and a fake.
- The scheme check reuses `client-core/secretReference.ts` if the host can import it, otherwise a copy
  with a test asserting both agree on ticket 11's table.

## Acceptance criteria

- [ ] The fake and the error mapping are tested, including a batch with good and bad entries.
- [ ] The package pins `@1password/sdk` exactly, and `bun install` adds no native step.

Verify: `bun run typecheck`, `bun run test` in `automations/`.
