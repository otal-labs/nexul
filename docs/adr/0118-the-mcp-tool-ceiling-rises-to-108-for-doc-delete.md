# The MCP tool ceiling rises to 108 for doc deletion

People delete docs from the doc's menu, but no tool could, so an agent asked to move research into a memory and delete
the source doc could only archive it and report the job half done (ADR 0019). The server stood at 106 tools under the
ceiling of 107 that ADR 0117 set.

Folding came first and found no home. A `delete` field on `doc_update` would make the one tool agents call for every
edit carry destructive hints, so a client confirming deletes would ask before each title change, and ADR 0068 keeps
deleting in its own tool for exactly that reason. Archiving stays a field on `doc_update`, since it is reversible.

Decision: `doc_delete` deletes a doc through the same use-case as the browser, so it needs `docs:delete` and publishes
`doc.deleted`. A doc that tickets name as their source is refused as a conflict that says to clear their source first,
for the browser and agents alike, instead of an internal error. The ceiling becomes 108, exactly enough for the 107
tools the server now has.

The trade-off: one more definition in every session, and a client that caps an agent at 100 tools across all its
servers drops one more of Nexul's. Accepted over leaving deletion to the browser alone.

Decided 2026-10-04, amending ADR 0117's tool ceiling.
