# The MCP tool ceiling rises to 102 for roles

Roles had no MCP path, so an agent could not read, create, clone, edit, or
remove a role, which breaks the web app parity ADR 0019 asks for. The server
stood at 99 tools under the ceiling of 100 that ADR 0068 set. Folding came
first: reading roles joined `workspace_list`, which returns a workspace's
roles and the permission catalog when asked for one workspace by id, and
creating, cloning, and editing a role share one patch-style `role_update`.
Delete could not fold in, because ADR 0068 keeps delete a separate tool so a
client can ask for confirmation of exactly that call. The only way to stay
at 100 was merging tools that have nothing to do with each other, which the
owner declined.

Decision: the ceiling becomes 102, exactly enough for the 101 tools the
server now has, so the next new tool fails the surface test again. The
ceiling lives in `internal/mcp/surface_test.go` and is raised only by an ADR
naming why no existing tool could carry the capability; extending an
existing tool before adding one still holds.

The trade-off: a client that caps an agent at 100 tools across all its
servers now drops some of Nexul's, and every added definition costs context
in every session. Accepted over hiding roles from agents or blurring
unrelated tools together.

Decided 2026-09-28, amending ADR 0068.
