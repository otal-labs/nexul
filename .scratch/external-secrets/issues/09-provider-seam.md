# 09 — What the code that resolves references must look like so a second manager slots in

**Status:** resolved
**Type:** grilling
**Blocked by:** None — can start immediately

## Question

1Password ships first and Vault or a cloud manager come later (Out of scope for this effort), resolved
by the same runners, hosts and server. Decide the shape the first version must already have so a second
manager is an addition, not a rewrite: one scheme per manager in the closed list (ticket 02), one
resolver interface in Go (runner, server) and TypeScript (host) with batch resolve and typed errors, how
a runner or host reports several managers' states, and how the `nexul … 1password` commands generalise
(per-manager subcommands or a manager argument). Only the shape is decided here; no second manager is
built.

## Answer

Decided 2026-11-12.

- **Everything published is written for any manager.** The runner's and host's connect report, the
  stored runner and host state, events and MCP fields carry a list of managers, each with its state
  (`ready`, `token rejected`), never a 1Password-named field. The UI tag and error messages take the
  manager's name from the reference's scheme. Nothing published says 1Password except the `op://`
  scheme, so a second manager adds to the contract and never migrates it.
- **Code:** one small `Resolver` interface in Go (runner, server) and one in TypeScript (host). Each
  has a scheme, a batch resolve, and typed errors (not found, field missing, no access, unreachable).
  A registry picks a resolver by a reference's scheme. 1Password is the only implementation in v1.
- **CLI:** a subcommand per manager, since each manager's credential has its own shape:
  `nexul runner 1password <name>`, `nexul automations 1password <host> --workspace <slug>`,
  `nexul server 1password`. A later `nexul runner vault <name>` takes Vault's own inputs.
