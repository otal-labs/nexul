# 09 — What the code that resolves references must look like so a second manager slots in

**Status:** open
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
