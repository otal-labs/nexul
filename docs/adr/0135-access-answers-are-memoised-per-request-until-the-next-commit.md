# Access answers are memoised per request until the next commit

Every `RequireProject` or `Can` read the permission layers again: the project's workspace, the membership and its
role, the workspace-wide overwrite, then the project or doc overwrite, four to seven statements per check. Lists that
checked each row paid that per row, so callers grew their own batching (a per-workspace map in plays, a per-person
standing cache in chat, a per-project grouping for docs), and the paths without one stayed slow: on a heavy copy of a
real instance a member's conversation list ran 11,277 statements, and each agent-stream frame ran six per connected
person at token rate.

Decision: `access` keeps a memo of the reads it makes, keyed by what each read is about (person, workspace, project,
doc, resource), and every check consults it before reading.

- **Where it lives.** A memo travels on the context. The HTTP gateway installs a fresh one per `/api/` request, the
  MCP adapter one per JSON-RPC request (a tool call or a resource read), and the live hub's audience holds one across
  frames. Background work carries none and reads every time; a batch question (`CanDocs`, `ProjectsWith`) starts its
  own when its context has none.
- **When it is dropped.** Every answer is thrown away at the next commit through the storage serializer, which every
  write goes through, so a check never answers past a write: a grant, role, membership, or Project access change takes
  effect on the very next check, within a request or across live frames, not on the next request. A read that fails
  for any reason but not found is never kept.
- **Batched reads.** A person's overwrites on many resources come back in one statement, and where many docs live in
  one more, so `CanDocs` answers a list of docs across projects in two reads plus each project's layers once.
- **A new question.** `ProjectsWith` lists the projects of a workspace in which a person holds an action, from the
  layers and one read of the workspace's projects, so a list can filter in SQL instead of row by row.

The callers' interface is unchanged: a per-row check costs one memo lookup after the first row of each project.

The trade-offs: a memo grows with the people and resources one request or the live hub touches, cleared at every
commit; a busy instance clears the live memo often and pays the reads again, which is the cost before this decision.
A memo across HTTP requests was not taken: it would have needed the same fence for no measured gain, and a request's
memo dies with its context.

Rejected: an explicit invalidation per write path (grant, role, membership, Project access, project deletion), which
misses the next write path someone adds, where the commit fence cannot; and memoising at the HTTP layer only, which
the MCP tools and the live hub would not inherit (ADR 0019).

Amends ADR 0087, whose live frames no longer cost a full check per socket, and ADR 0097, whose restricted member's
lists are no longer checked row by row. Decided 2026-10-09.
