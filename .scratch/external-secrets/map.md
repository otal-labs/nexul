# External secrets

## Destination

A spec at `.scratch/external-secrets/spec.md`, with ADRs for the hard calls, ready to cut into build
tickets: a secret used by Nexul can be a **secret reference** to 1Password instead of a value Nexul
stores. It covers stack environment values, the automation secrets pool, and the secret fields of
connectors and sign-in providers. The map is done when nothing is left to decide before the build.

## Notes

- Planning only: tickets resolve decisions; the build comes after the spec.
- Read first: `CONTEXT.md` (Stack environment, Runner, Automations host, Connector, Secret reference),
  ADRs 0047, 0074, 0095, `practices/architecture.md`, `practices/go.md`.
- Grilling tickets call the `grilling` and `domain-modeling` skills.
- Related, separate: NEXUL-94 (Encrypt stack env values at rest) seals the literal values Nexul keeps
  and ships before this build. References are stored as references.
- Settled while charting (2026-11-11), to become an ADR in the spec:
  - Scope: stack environment, the automation secrets pool, and connector and sign-in provider secrets
    (the bootstrap secret was later ruled out, ticket 06).
  - One reference per variable: each value is a literal or a reference, and a stack can mix both.
  - The side that uses a secret resolves it with its own credential. The runner resolves stack
    environment references; the automations host resolves the pool (with a token per workspace,
    ticket 05); the server resolves connector and sign-in provider secrets.
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
- [Automation secrets by reference on the automations host](issues/05-automation-pool.md): one token
  per workspace per host, held by the supervisor and never by an automation's thread; resolved once per
  connection; a failed entry is left out and named in the automation's log.
- [Connector credentials and the bootstrap secret by reference on the server](issues/06-server-references.md):
  connector and sign-in provider secrets resolve at start and on save with the server's own token; a
  1Password outage degrades only those credentials; a 401/403 re-resolves.
- [Should Nexul notice a rotated 1Password value and offer a redeploy?](issues/08-rotation-notice.md):
  yes. Runners and hosts check hourly by default against on-machine fingerprints; a stack shows a
  notice with Redeploy, and an automation restarts once it is idle.
- [What the code that resolves references must look like so a second manager slots in](issues/09-provider-seam.md):
  nothing published names a manager; a `Resolver` interface and registry in Go and TypeScript; a CLI
  subcommand per manager.
- [The remaining scope, build and docs calls](issues/10-scope-build-and-docs.md): AGPL; hellos batched
  per workspace; one guide page; fake-only tests; three ADRs; a rejected token notifies machine
  managers; an optional install prompt.

**Destination reached 2026-11-12:** [spec.md](spec.md) with ADRs 0181–0183. Next: cut it into build
tickets.

## Not yet specified

Nothing left: every area became a ticket and every ticket is resolved.

## Out of scope

- Building a second manager's provider. This spec defines the seam and ships 1Password only.
- The bootstrap secret (`NEXUL_AUTH_SECRET`) as a reference ([ticket 06](issues/06-server-references.md)):
  backups already lack it and the server's token would sit beside it, so it adds almost no protection
  and would stop the server starting while 1Password is down.

The spec also carries the user guide page for external secrets and the README row; they are writing,
not decisions.
