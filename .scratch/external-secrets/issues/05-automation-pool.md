# 05 — Automation secrets by reference on the automations host

**Status:** resolved
**Type:** grilling
**Blocked by:** 01, 02, 07

## Question

How does an automations host get its 1Password token (bundled host and a placed host, ADR 0074), and
when does it resolve a pool entry that is a reference: when the automation starts, on each
`ctx.secrets.NAME` read, or cached for a time? What does an automation see when a reference cannot be
resolved, and how does the pool's write-only behaviour (ADR 0047, ADR 0095) read once an entry is a
reference rather than a value?

## Answer

Decided 2026-11-12.

Found while deciding: every automation on a host runs as a `worker_threads` thread inside the host
process (`automations/src/worker.ts`) and inherits its environment, and the `instance` host runs
automations from every workspace. A token in the host's environment would let one workspace's
automation read the vault items another workspace references, breaking ADR 0095's isolation.

- **Tokens:** one per workspace per host, kept on the host machine and never in Nexul. They are set,
  rotated and removed with `nexul automations 1password <host> --workspace <slug>` from stdin, and
  `upgrade` keeps them. The server tells the host each automation's workspace (`AutomationTarget`
  carries only id, name and token today).
- **Resolver:** the host's supervisor. Tokens live only there and are stripped from the environment
  the automation threads inherit. An automation's hello goes through the supervisor, which resolves
  that workspace's references in one `resolveAll` and passes the values in. There is one client per
  token, opened at host start. An automation can only ever reach its own workspace's token, so its
  reach stays what ADR 0047 accepts.
- **When:** once per connection, exactly when literals are delivered today. A rotated value arrives on
  the next reconnect or restart, as a changed literal does now.
- **Failure:** the automation starts with every secret that resolved. The failed name is absent from
  `ctx.secrets`, and the automation's log and page say `<NAME>: field not found`, never the
  reference. A workspace with no token on the host fails each of its references this way.
- **Status:** like a runner (ticket 03), the host checks each token at start and reports `ready` or
  `token rejected` per workspace. The host list shows it, and an automation whose pool has references
  its host can't resolve shows a warning on its page.
- **Write-only:** unchanged; a saved reference is no more readable than a saved value (ticket 02).

Risk carried to the build: a restart loop costs one read per automation each time it reconnects,
against the account's 1,000-a-day budget (ticket 07). Batching the automations of one workspace that
connect together into one resolve softens it.
