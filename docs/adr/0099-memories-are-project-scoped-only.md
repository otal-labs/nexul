# Memories are project-scoped only

ADR 0059 let a memory belong to the workspace, reaching every Agent turn there, including a plain chat. With
Restricted members (ADR 0097) the workspace splits into project areas and workspace areas, and a memory that reaches
every project fits neither: as a project area a workspace memory has no project to check against, and as a workspace
area it would hand one project's agent rules to a client who sees another.

Decision: a memory always belongs to one project. The use-case refuses one without a project, every memory check goes
through the project (`RequireProject`), the Clone dialog and `memory_create` lose the workspace destination, a
project's list is its own memories only, and an Agent turn's index is the project's memories only. A plain chat with
no ticket or doc carries no memories again. A forward-only migration deletes every workspace-scoped memory with its
versions and attachments.

The trade-off: team-wide standing rules lose their home again; a rule meant for every project is cloned into each and
the copies may drift, the cost ADR 0056 accepted. Workspace memories on an upgraded instance are deleted, not moved:
there is no project to move them to that would not leak them to that project's readers.

Rejected: keeping workspace memories as a workspace area, which reaches a Restricted member's agent through their role
whatever projects they hold.

Supersedes ADR 0059, restoring ADR 0056's project-only rule. Decided 2026-10-01.
