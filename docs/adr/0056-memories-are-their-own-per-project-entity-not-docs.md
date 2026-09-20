# 0056. Memories are their own per-project entity, not designated docs

A memory used to be an ordinary doc with a switch turned on. That reused
the editor and search for free, and it put agent instructions in the same
list clients read requirements from. The owner's rule is that docs are for
clients and requirements and memories are process and context for agents,
and the two never share a list, a page, or a search result. So a memory is
its own entity with its own table, page, tools, and permission, keeping
only the rich-text editor component from docs.

Memories belong to one project each; there is no workspace-wide set. Agents
write memories too, and what they learn is mostly per project. A memory
that should apply elsewhere is cloned, and the clone may drift; the owner
chose that over an optional project scope so that no memory is a shared
dependency across projects. Every memory is versioned with revert, and an
agent may update one with no approval step, since a gate would make it
stop maintaining them; people are notified, not asked.

The cost is a migration of the existing ticked docs and a second editor
surface. The owner has not deployed publicly and restarts the schema, so the
migration is moot.

Decided 2026-09-16.

Superseded in part by ADR 0059: memories may also be workspace-scoped.
