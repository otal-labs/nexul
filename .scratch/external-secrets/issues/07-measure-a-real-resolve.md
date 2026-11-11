# 07 — Measure a real resolve with a test service account

**Status:** resolved
**Type:** task
**Blocked by:** 01

## Question

Not a decision. With a throwaway service account scoped to a test vault holding 20 fields, resolve them
from the Go SDK and from `@1password/sdk` under Bun, and record: the reads one `ResolveAll` costs
(`op service-account ratelimit` before and after), latency cold and warm, the error type and text for a
missing item, a missing field and a vault the token cannot reach, and what a call does with the network
cut. Also record which 1Password plan the account is on, since that sets the daily budget. Tickets 04
and 05 decide on caching from these numbers.

## Answer

Measured 2026-11-12 with the fleet service account against one temporary item of 20 dummy fields in
the Fleet vault (the token cannot create vaults: 403). The item was deleted afterwards. Read counts
come from `op service-account ratelimit` before and after each step.

- **Plan:** the account is on the individual or Families tier: 1,000 reads an hour per token, and
  1,000 reads and writes a day for the whole account, shared by every token. The fleet had used 25
  that day before the test.
- **Cost:** one `ResolveAll` is one read, whether it holds 20 references or a mix of good and bad
  ones. Twenty single `Resolve` calls cost 40 reads (two each). Creating a client costs no read.
- **Go SDK v0.4.1:** builds with `CGO_ENABLED=0`; a binary that does nothing else is 28 MB. Creating
  a client takes about 2.6 s, mostly CPU spent compiling the wasm core. `ResolveAll` of 20 took 844 ms
  cold and 594 ms warm. Twenty single resolves took 11.7 s.
- **JS SDK 0.5.0 under Bun 1.3.11:** works with a real token, `node_modules` is 14 MB. Creating a
  client takes 1.2 s, and `resolveAll` of 20 took 830 ms cold and 657 ms warm.
- **Errors, both SDKs:** a missing item gives `{"type":"itemNotFound"}`, a missing field gives
  `{"type":"fieldNotFound"}`, and a vault the token cannot reach gives `{"type":"vaultNotFound"}`,
  exactly like a vault that does not exist, so a token never learns which vaults it lacks. A batch
  with bad entries still resolves the good ones. A single `Resolve` returns or throws a plain error:
  "no item matched the secret reference query", "the specified field cannot be found within the
  item", "no vault matched the secret reference query".
- **No network:** creating a client fails at once with a plain dial error naming `my.1password.com`.
  There is no offline cache.

What this means for 04 and 05: a deploy that resolves every reference in one batch costs one read, so
the daily budget is not the constraint (about 900 such calls a day on this plan). Client start-up
(2.6 s of CPU in Go) is the cost worth avoiding, by keeping one client for the life of the process.
