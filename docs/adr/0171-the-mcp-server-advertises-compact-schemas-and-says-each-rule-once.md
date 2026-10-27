# The MCP server advertises compact schemas and says each rule once

Every session loads the whole `tools/list`, and at 113 tools it had grown to about 223,000 characters, most of it
prose that repeated what the server instructions or the tool's own errors already say. The tool count sat one under
the budget, so the next capability in the parity list had nowhere to go.

Decision: `tools/list` carries what an agent needs to pick and call a tool, once, and a test caps its size.

- **The advertised schema is compact; the call is validated in full.** The adapter strips a pointer's `null`
  alternative and the closed-object marker (`additionalProperties: false`) from the schema it lists. Arguments are
  still validated against the full inferred schema, so an unknown key is refused and a `null` patch field is
  accepted exactly as before.
- **Each rule lives in one place.** Patch semantics, paging and ids are in the server instructions, never in a
  description. A failure the use-case reports with a message that says how to recover (a missing permission, a
  conflict, a cap) is not described in advance. A rule that two parameters share is said once, on the tool or on
  the parameter that carries it. What stays: sibling-tool routing, valid values, how to clear a field, side
  effects, secrets returned once, and limits other than paging.
- **A list given an id returns that one record.** `deploy_get` folded into `deploy_list`, the shape `dns_tunnel_list`,
  `automation_list`, `trail_list` and `computer_list` already had, and the instructions say so in one line.
  Migration 0138 rewrites the old name in stored play instructions.
- **A size budget.** `advertisedBudget` in `internal/mcp/surface_test.go` sits just above the current size, beside
  `toolBudget`.

The result: 222,623 to 182,991 characters (18% less), tool descriptions 26% shorter, input schemas 16% shorter, and
112 tools against the budget of 114.

Rejected: an `action` enum that merges a family's verbs into one tool. Annotations are per tool (ADR 0068), so a tool
that both reads and changes asks for confirmation on every read, and a merged delete loses its own confirmation.
Advertising a tool under its old and new name, which spends the budget the merge was meant to free; a retired name
answers as an unknown tool, and the client reconnects. Hoisting repeated nested types into `$defs`, which some
clients resolve badly.

Decided 2026-10-27.
