# Every read is checked where the entity lives

ADR 0023 left tickets, chat, and projects deliberately coarse until one sweep could apply the grid of ADR 0010
everywhere. Until then a signed-in session read every ticket, stack, deploy, runner, the topology, and every
conversation on the instance, in any workspace, and the live socket pushed every entity's payload to every open
browser. Only scoped tokens were held to the grid, by URL path. Project and board-setting changes needed the
instance-admin bit, and creating a doc, a channel, or a DM needed nothing.

Decision: the use-case layer checks every read, write, and delete against the permission table, in the workspace
the entity belongs to, so the HTTP gateway, the MCP tools, and the live socket all get the same answer (ADR 0019).
The access domain's resolver answers every check, so the Owner bypass and workspace-wide overwrites apply unchanged
(ADR 0042).

- A ticket, doc, stack, deploy, project, or board setting is checked in its project's workspace. Project and
  board-setting changes take `projects:write` and deletes `projects:delete` there; the instance-admin bit no longer
  opens them. Creating a doc takes `docs:write`, a channel, voice channel, or DM `chat:write`.
- What every member reads needs membership only, never a bit an owner can take away: the project list and each
  project's columns, types, and categories, channels, their own DMs, their inbox, and People (ADR 0086). A DM is
  read by its participants alone; a ticket thread takes `tickets:read`, an interview thread `memories:read`, a doc
  thread `docs:thread`. Posting in a conversation takes only being able to read it.
- Runners, the topology canvas, machines, and the instance's own stacks belong to no workspace (ADR 0079). They are
  checked against every workspace the caller belongs to: holding the action in any one of them is enough, which is
  what the sidebar already shows. A deleted stack's kept deploy history is checked the same way.
- A caller outside the workspace gets not found, so fetching an id never confirms what another workspace holds; a
  member without the action gets forbidden. A list or search that spans projects leaves out what the caller cannot
  read instead of failing, and a link to a ticket the caller cannot read is left out of the one they can.
- A notice is only created for someone who may read its subject, and an inbox shows only the workspaces its owner
  still belongs to.
- The live socket checks every frame against the same read, per open socket, and a topic that names no rule reaches
  nobody.
- A call with no actor is the server's own (event consumers, the runner protocol, workers) and passes; every
  adapter attaches one. A token acts as its creator and is held to both its scopes and the creator's grant; a
  shipped default automation has no creator, so its scopes, checked at the gateway, are its whole grant.

The trade-offs: a role now has to carry the reads its people need, so a member whose role grants nothing sees the
workspace's projects, channels, and people but no tickets, stacks, or docs, which is what the web already shows.
Instance-level areas answer to the most generous of a caller's workspaces rather than the one selected, because the
server has no selected workspace to ask. Each live frame costs a permission check per open socket, fine for a
single team and the first thing to batch per person if an instance grows.

Rejected: checking reads in the HTTP handlers, which the MCP tools would not inherit; sending id-only frames, which
would have changed every live consumer in the web and phone clients; and gating chat and the inbox on `chat:read`
and `notifications:read`, which would let an owner cut a member off from their own conversations.

Supersedes ADR 0023, and the "every signed-in member reaches every stack" paragraph of ADR 0079. Decided 2026-09-29.
