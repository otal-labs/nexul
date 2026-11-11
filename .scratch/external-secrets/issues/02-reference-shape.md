# 02 — How a value says it is a secret reference, and who sees the reference

**Status:** resolved
**Type:** grilling
**Blocked by:** None — can start immediately

## Question

What does a secret reference look like in a stack variable, an automation secret, a connector field and
the bootstrap secret? Decide: a URI prefix read from the value (`op://…`) or an explicit kind on the
variable; how a literal that happens to start with `op://` is kept literal; the scheme for later
managers. Then, since a reference holds no value: who may read it (stack environment masks every value
today), whether events, MCP results and the audit log may carry it, and how the web and phone apps show
a reference beside a literal.

## Answer

Decided 2026-11-12.

- **Marking:** a value is a secret reference when the whole value starts with a known scheme. The
  schemes are a closed list, `op://` for now, and each later manager adds its own. Anything else stays a
  literal, including `postgres://…` and other URLs. The data shapes don't change: stack env and branch
  deploy rule overrides stay name-to-value maps, and the automation pool, connector fields and the
  bootstrap secret stay strings. The prefix is also what makes `NEXUL_AUTH_SECRET=op://…` work in the
  env file.
- **No embedding:** a reference is the entire value, never part of one. A connection string is either
  stored whole in 1Password or built from separate variables with compose's `${VAR}` interpolation.
- **Secrecy:** a reference is treated exactly like a value. Readers without `stacks:write` get names
  with `env_masked`, pool entries stay write-only after save (ADR 0047), and events, MCP results for
  readers and the audit log never carry one. The event contract is unchanged.
- **Validation:** save rejects a malformed reference (unknown shape, missing vault, item or field).
  Whether the item exists is only known when the reference is resolved (ticket 04).
- **Web UI:** value inputs stay text, with a "1Password" tag beside any value that parses as a
  reference and save errors inline. This covers the stack env editor, branch rule overrides, the
  automation secrets form and secret connector fields. Nexul never browses a vault, and the phone app
  shows no stack env, so it is unaffected.
- **Upgrade:** an existing value that already starts with `op://` becomes a reference, with one line in
  the release notes.
