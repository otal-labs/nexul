# 03 — Security-review the integration model before the store launches

**Status:** ready-for-human

**Blocked by:** None — due before the integration store accepts third parties.

**What to build:** The integration platform hands scoped API tokens to external
services and distributes third-party code via a store. Before the store goes
live, the security model needs a deliberate review — verified in code, not just
on paper.

The design itself is sound (see ADRs 0043–0045). What's missing is confirmation that the implementation matches
it. A public store distributes executable code that touches owners' infra and
secrets, so this is the gate before anyone outside the project can publish.

Review must cover:

- [ ] Scope granularity — can a token do more than its integration needs?
- [ ] Token revocation — does revoking actually cut off access, promptly?
- [ ] Webhook HMAC signing — signatures verified, replay handled
- [ ] The audit log — every scoped call attributable to an integration
- [ ] `verified` vs `community` trust tiers — what each actually gates
- [x] Bot webhook tokens — 32 random bytes, encrypted at rest, never logged
- [x] Bot webhook rate limit — per bot and per IP, enforced in code not prose
- [x] Bot webhook 404 — a wrong token and a missing bot are indistinguishable
- [x] Bot webhook revoke and regenerate — the old URL stops working at once
- [ ] Bot embed images — they load straight from the sender's host, so it sees every reader's IP; decide whether to proxy them as Discord does

## Surface when

- Integrations-domain primitives start (webhooks-out, API tokens).
- The integration store / registry / SDK work starts.
- The first third-party (non-maintainer) integration publisher is onboarded.
- Anyone asks "is the integration store ready / safe?".
