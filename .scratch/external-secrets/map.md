# External secrets

## Destination

A spec at `.scratch/external-secrets/spec.md`, with ADRs for the hard calls, ready to cut into build
tickets: a secret used by Nexul can be a **secret reference** to 1Password instead of a value Nexul
stores. It covers stack environment values, the automation secrets pool, connector credentials and the
instance bootstrap secret. The map is done when nothing is left to decide before the build.

## Notes

- Planning only: tickets resolve decisions; the build comes after the spec.
- Read first: `CONTEXT.md` (Stack environment, Runner, Automations host, Connector, Secret reference),
  ADRs 0047, 0074, 0095, `practices/architecture.md`, `practices/go.md`.
- Grilling tickets call the `grilling` and `domain-modeling` skills.
- Related, separate: NEXUL-94 (Encrypt stack env values at rest) seals the literal values Nexul keeps
  and ships before this build. References are stored as references.
- Settled while charting (2026-11-11), to become an ADR in the spec:
  - Scope: stack environment, the automation secrets pool, connector credentials, the bootstrap secret.
  - One reference per variable: each value is a literal or a reference, and a stack can mix both.
  - The side that uses a secret resolves it with its own credential. The runner resolves stack
    environment references; the automations host resolves the pool; the server resolves connector
    credentials and the bootstrap secret.
  - Each runner's 1Password token lives only on its machine, scoped to the vaults it deploys from.
    Nexul never holds a runner's token or a resolved value. The server has its own token, scoped to its
    own vault. Why: a Nexul breach (database, backup, server) yields no vault access. The trust left is
    that `stacks:write` can aim a reference at anything the runner's token reads, the same trust stack
    environment carries today.
  - 1Password first; other managers come in behind the same reference shape.

## Decisions so far

- [How the runner, the automations host and the server read 1Password](issues/01-1password-from-go-and-bun.md):
  the official Go SDK on the runner and server and the JS SDK on the host, resolving in one batch per use;
  rate limits are per token hourly and per account daily.

## Not yet specified

- Rotation: what happens when a vault value changes. Nothing redeploys today; whether Nexul ever
  notices depends on the failure and resolution answers.
- What a second manager (Vault or OpenBao, a cloud manager) needs from the provider seam, beyond what
  1Password shapes. Sharpens once the reference syntax and the 1Password research land.
- Rollback and preview deploys: whether a rollback uses the reference's value now or the one it
  deployed with, and how branch deploy rule overrides carry references.
- Docs and the user guide page for external secrets, and the README row.

## Out of scope

- Building a second manager's provider. This spec defines the seam and ships 1Password only.
