# 07 — Clarification storage, routes, events, and agent tools

**What to build:** The storage decided in 03. A new numbered migration adds
`doc_clarification_rounds` and `doc_clarification_questions` in the docs
domain. Use-cases in `internal/docs` to read a doc's clarification, save,
skip, or clear one answer, save a round's "Anything else?", close, and the
seam the play runner calls to open and end a round (08 wires it). Gated on
the doc: `docs:read` to see, `docs:write` to answer, `docs:write` plus
`plays:run` on the Clarify play to close. Answers ignore the doc's lock.
Each change writes an outbox event with a catalog row (question and author,
never the answer) and pushes live to everyone who can read the doc; saving
the last pending answer of a round also writes the round-answered event
naming its starter. HTTP routes for the page. MCP: `doc_get` returns the
clarification; `doc_update` takes `answers` and `clarification_closed`, and
`questions`, `anything_else_reply`, and `no_gaps` with a body only from the
running round's starter while it runs (the body write through the lock per
ADR 0121). A clone starts with none; deleting the doc deletes it.

**Blocked by:** None — can start immediately

**Status:** resolved

- [x] Migration tested by upgrading from the previous schema
- [x] Use-case tests: permission refusals first (a Restricted member without
      the doc, a reader answering, a non-starter posting a round, a post with
      no round running), then the paths
- [x] Lock tests: answers saved on a locked doc; the no-gaps body accepted
      from the running round's starter and refused from anyone else
- [x] Catalog rows, automation scope, live audience entries
- [x] MCP surface test still under the ceiling; tool descriptions updated
- [x] `CONTEXT.md` and `practices/mcp.md` checked
