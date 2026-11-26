# 06 — The project row shows the whole answer

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, decision 7.

Each row on the Projects tab shows, for its person: the computers in effective order (the project's list, else "Your
order"), the checkout each would use, and under it each play with needs or a Run as auto play for that project, with the
computer `PreviewRun` would land it on now and the first reason on each computer passed over. One read serves the row
(`GET /api/pairing/projects/{id}/placement` or the existing links read extended, whichever keeps one request per row),
answered from stored facts without asking any computer.

Done when: a handler test covers a project following your order and one with its own list, with a play whose need moves
it off the first computer; the row is checked at 768, 1024 and 1440px, and the request count per row is guarded the way
`web/src/hooks/liveRequests.test.tsx` guards a frame. `paired-computers.md`'s Projects tab section matches the row.
