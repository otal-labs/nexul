# 07 — Sources and drafts: storage and surfaces

**What to build:** The storage and every non-page surface from 01, 03, and
05. One new numbered migration adds `interview_sources` and
`interview_drafts` in the memories domain, with the columns, uniqueness,
and cascades in 01 and 03. Use-cases: list, add, update (stance, label),
and remove a source; list drafts; dismiss a draft; and save drafts from a
run, which drops a draft whose picks and text equal the stored answer
(05). Gated by `memories:read` and `memories:write` through project
access. Adding a doc, memory, or project source also takes read on what it
points at, through the access gate and a new doc-to-project lookup port.
Labels of doc, memory, and project refs are resolved with the reader's
access, so an unreadable one comes back as kind only, flagged not visible,
and a missing one comes back flagged gone. Outbox events with catalog rows
for source added, changed, removed and draft saved, dismissed (no bodies,
no ref content) and live push to the project's audience. HTTP routes for
the page. MCP: `memory_get` on the interview memory also returns sources
(text bodies included) and drafts; `memory_update` on it also takes
sources to add, remove, or change the stance of, and drafts. A
permissions catalog entry if the Access page needs one.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] Migration tested by upgrading from the previous schema
- [ ] Use-case tables: every refusal in 01 (absolute path, `..`, a ref the
      adder cannot read, the 50 cap, the 32,000 text cap), not visible and
      gone resolution, equal-draft drop
- [ ] Integration tests on real SQLite for the repo and the cascades
- [ ] MCP surface test updated; tool count unchanged
