# 07 — Stored answers

**What to build:** The answers from 03. A new numbered migration adds a
table in the memories domain: project, question text, round (0 for the
template's questions, 1, 2, … for each round of follow-ups), the follow-up's
options and why-line when it came from the agent, picked values, free text,
skipped, who answered, and when; unique on project, round, and question.
Use-cases save one answer, skip one, clear one, and list a project's
answers, gated by `memories:read` and `memories:write` through project
access. Each save writes an outbox event with a catalog row (question and
author, never the answer) and pushes live to the project's audience. HTTP
routes for the page. MCP: reading the interview memory also returns the
parsed questions and the answers; updating it also accepts answers. Deleting
the interview memory keeps the answers; deleting the project removes them.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] Migration tested by upgrading from the previous schema
- [ ] Use-case tests: permissions, project access, skip, clear, re-answer
- [ ] Event catalog row, outbox write, live push
- [ ] MCP read and write through the existing memory tools
