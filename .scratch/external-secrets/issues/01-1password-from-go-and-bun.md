# 01 — How the runner, the automations host and the server read 1Password

**Status:** resolved
**Type:** research
**Blocked by:** None — can start immediately

## Question

What is the best way for a Go binary (runner, server) and the Bun automations host to resolve an
`op://vault/item/field` reference with a service account token? Compare the official SDKs
(Go and JS), the `op` CLI, and a 1Password Connect server: install footprint on a machine, release
and cross-compile cost for Linux, macOS and Windows, how a token is scoped to particular vaults, rate
limits for service accounts (per hour and per day) and what a deploy that resolves 20 references costs
against them, batch resolution, latency, offline and outage behaviour, and the error shapes for a
missing item, a missing field and a token without access. Recommend one path per side.

## Answer

The official SDKs, batch-resolving once per use. Runner: the Go SDK (`v0.4.1`, CGO-free, which matches
`.goreleaser.yaml`), with one `ResolveAll` per deploy. Server: the same Go SDK with one long-lived
client. Automations host: `@1password/sdk` (`0.5.0`) with `resolveAll`. The `op` CLI was rejected
(a second binary on every machine), and so was Connect (a hosted service with its own tokens and no
reference resolver). Carry forward: both SDKs are version 0, so pin exact versions. The Go SDK embeds
a 9.5 MB wasm core, so only the runner and server import it, never `cmd/nexul`. A token's vaults are
fixed when it is created. Hourly rate limits are per token (Business 10,000 reads, Teams 1,000);
daily limits are per account and shared by every token (Business 50,000, Teams 5,000). Not verified
without a real token: the reads each `ResolveAll` costs, latency, outage behaviour, exact error text,
and Bun support. Those go to ticket 07.

Findings: branch `research/1password-from-go-and-bun`, file
`.scratch/external-secrets/research/1password-from-go-and-bun.md` (commit `ce1ace1e`).
