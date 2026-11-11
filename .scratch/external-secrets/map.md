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
- [How a value says it is a secret reference, and who sees the reference](issues/02-reference-shape.md):
  a whole value starting with a known scheme (`op://`), treated exactly like a value, syntax checked at
  save, a "1Password" tag in the web editors.
- [Measure a real resolve with a test service account](issues/07-measure-a-real-resolve.md): one
  batch resolve is one read whatever its size; the account's daily limit is 1,000 across all tokens;
  client start-up (about 2.6 s in Go) is the real cost; errors name the missing part, never the vault.
- [When a runner resolves references, and what a failed resolution does to a deploy](issues/04-deploy-resolution.md):
  every deploy in one batch with no cache, before pull or build; any failure fails the deploy, naming
  variables only; values go into `.env` as today; a rollback uses each reference's value now.
- [How a runner gets its 1Password token and tells Nexul it can resolve](issues/03-runner-setup.md):
  an install flag plus `nexul runner 1password <name>`; the token is checked at start and reported as
  ready or rejected; deploys with references go only to a ready runner; a tag on the runner list.

## Not yet specified

- Rotation: a rotated value takes effect on the next deploy (ticket 04). Whether Nexul should ever
  notice a change and offer a redeploy is still open.
- What a second manager (Vault or OpenBao, a cloud manager) needs from the provider seam, beyond what
  1Password shapes. Sharpens once the reference syntax and the 1Password research land.
- Docs and the user guide page for external secrets, and the README row.

## Out of scope

- Building a second manager's provider. This spec defines the seam and ships 1Password only.
