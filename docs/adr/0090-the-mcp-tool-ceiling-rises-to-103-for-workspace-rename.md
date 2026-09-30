# The MCP tool ceiling rises to 103 for workspace rename

A workspace's name and slug are editable in Configuration, General (ADR 0089), but no tool could change them, so an
agent could not do what the browser can (ADR 0019). The server stood at 101 tools under the ceiling of 102 that
ADR 0081 set. Folding came first and found no home: `workspace_list` is a read, and ADR 0068 keeps reading and
changing in separate tools so a client can confirm exactly the change; no other update tool is about a workspace.

Decision: `workspace_update` is a patch-style tool taking `id` plus an optional `name` and an optional `slug`, and the
ceiling becomes 103, exactly enough for the 102 tools the server now has, so the next new tool fails the surface test
again. Its description says that changing the slug breaks links that use the old one, because ADR 0089 does not
redirect them.

The trade-off: one more definition in every session, and a client that caps an agent at 100 tools across all its
servers drops one more of Nexul's. Accepted over leaving workspace naming to the browser alone.

Decided 2026-09-30, amending ADR 0081.
