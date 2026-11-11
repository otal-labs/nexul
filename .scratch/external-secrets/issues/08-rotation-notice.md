# 08 — Should Nexul notice a rotated 1Password value and offer a redeploy?

**Status:** resolved
**Type:** grilling
**Blocked by:** None — can start immediately

## Question

A rotated value reaches a stack only on its next deploy (ticket 04) and an automation only on its next
connection (ticket 05). Connectors re-resolve on a 401/403 (ticket 06). Should Nexul ever notice that a
referenced value changed (the SDK returns item ids, and items carry versions) and tell someone or offer a
redeploy? If so, who checks (only runners and hosts can read their vaults), how often against the
1,000-a-day budget, and what the stack or automation page shows. Or is "redeploy to pick it up" the
whole answer?

## Answer

Decided 2026-11-12. Yes, Nexul notices.

- **Detection:** at each deploy the runner keeps a fingerprint of every resolved value (a keyed hash
  whose key never leaves the machine) in a 0600 file in its own directory, removed with the stack. On a
  timer it re-resolves the references of every stack it has deployed, preview stacks included, in one
  batch (one read per runner per check), compares, and tells Nexul which variables of which stacks
  changed. It never sends values or references.
- **Interval:** `NEXUL_SECRET_CHECK_INTERVAL` on runners and automations hosts. It defaults to 1h,
  refuses anything under 5m, and `off` disables the check.
- **Stacks:** a notice ("DB_PASSWORD changed since the last deploy") on the stack page on the web and
  the phone, beside the existing Redeploy, plus a new event pushed live so automations can react. It
  names variables only, is visible to anyone who can see the stack, and gives the Redeploy button only
  to those who can deploy. Nothing redeploys automatically. The next deploy, retry or rollback clears
  it, and there is no dismiss.
- **Automations:** the host checks the same way, one batch per workspace token per interval. When a
  value changed, it restarts that automation's thread as soon as no run is in flight, or after
  5 minutes at most. A run cut off then is delivered again (ADR 0046). The restart that config changes
  use today is immediate (`supervisor.ts`); this one waits.
- **Server:** no check. Connector and sign-in secrets re-resolve on a 401/403 (ticket 06).
