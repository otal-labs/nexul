# 0059. Memories may also be workspace-scoped

A memory may now belong to the workspace instead of one project:
`project_id` is nullable, and empty means workspace scope. Create, list, and
clone all take an optional project id; when it is empty a workspace id is
required instead, and the memory reaches every turn in that workspace
rather than one project's threads. Listing a project's memories returns its
workspace's workspace-scoped memories first, then its own.

Two gaps drove this. The Agent in a plain chat with no ticket or doc
resolves an empty project id, so the memory protocol in its instructions was
dead text there — nothing was ever indexed or inlined. And team-wide
standing rules (tone, what never to touch, how to name things) had no home;
the only way to approximate one was cloning the same memory into every
project and letting the copies drift.

The trade-off ADR 0056 named for scoping memories to a project — no memory
is a shared dependency across projects — is accepted here only for
standing rules that are meant to be shared: a workspace memory is a
deliberate shared dependency, not an accident of scope. Clone still exists
for the case that isn't shared: copying a memory from one project to
another (or to the workspace) as an independent, divergeable start.

Decided 2026-09-18.
