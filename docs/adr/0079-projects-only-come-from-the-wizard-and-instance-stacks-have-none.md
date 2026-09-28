# Projects only come from the project wizard, and the instance's own stacks belong to no project

Every install used to seed a project named General in the default workspace, and the owner wizard renamed it
and gave it a prefix. First run also filed the tunnel or Traefik stack it deploys under that project, because a
stack required one. So the owner's first project was a placeholder nobody chose, the proxy that serves the whole
instance sat inside it as if it were that project's service, and deleting or reshaping the project dragged the
instance's front door along.

Decision: a project only ever comes from the project wizard, whose backend call is the one use-case that makes a
project with its status columns, ticket types, and starter memory; `project_create` over MCP runs the same
use-case. Nothing seeds a project and creating a workspace creates only the workspace. The owner wizard names the
workspace and then opens the project wizard for the first project; a workspace with no project shows a New
project action on the sidebar, the board, and the docs list instead of failing, and nothing sends anyone into the
wizard on their own. The wizard creates the project the moment it is named, so every rung after that can be
skipped and the project stays; the naming rung can be skipped too, leaving the workspace with none.

The stacks the instance deploys for itself, every gateway's backing stack (cloudflared, Traefik), have no project:
`stacks.project_id` is nullable and stays a foreign key when set. A stack that builds from a repository still
needs a project, because the repository belongs to one (ADR 0039).

Access does not change: a stack's project never gated who could see or manage it. Every signed-in member reaches
every stack over HTTP and MCP, and a scoped token needs the `stacks` permissions, so an instance stack is visible
and manageable exactly like any other stack. Topology and Settings → DNS list it; its stack page links back to
Topology instead of a project.

Migration 0029 rebuilds `stacks` with the column nullable, moves existing gateway stacks out of their project, and
deletes the seeded General project only on an instance nobody has signed in to yet; anywhere the owner already
has it, it is their project now. Migrations run with foreign keys off, as SQLite requires for a table rebuild, and
`PRAGMA foreign_key_check` refuses a commit that leaves a dangling row.

Rejected: keeping a seeded project and hiding it until used, which still files the proxy under a project the
owner never made. Also rejected: an "Infrastructure" project to hold the instance's stacks, which is a project
nobody can use for work and one more thing to explain.

Supersedes the default-project half of ADR 0052. Decided 2026-09-28.
