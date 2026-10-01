# 01 — Scope and ground rules

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Can a workspace member be held to some of its projects, and if so, how is it
set, what does it carry, and what else in the workspace does it touch?

## Answer

Settled with the owner while charting, 2026-10-01.

- **Why not a workspace per client.** Projects live where the team works. A
  client giving feedback on one OTAL project belongs in OTAL, and must not
  see the others.
- **Per person, hidden by default.** (Shown as an Every project row rather
  than a switch, per ticket 05.) Each membership is either "All
  projects" (today's behaviour, every existing member after the upgrade) or
  "Only these projects" (a Restricted member). A Restricted member sees a
  project only once given access; projects created later stay hidden.
- **Invisible, not locked.** A hidden project is gone from the project list,
  search, the @ picker, the board, notifications, and the live socket, name
  included. A direct link reads as not found. This reverses ADR 0087's rule
  that every member reads the project list.
- **Project access carries levels.** The same None, Read, Write, Delete
  editor roles use (ADR 0078), per project area, project settings included.
  For a Restricted member it is the whole answer inside that project; their
  role does not add to it.
- **The role editor splits in two.** A Workspace section (chat, channels,
  plays, people, roles and the like) and an Every project section (tickets,
  docs, stacks, deploys, memories, project settings and the like). An "All
  projects" person gets the Every project levels on every project; a
  Restricted member ignores that section. Creating a project stays a
  workspace permission. Which areas fall on which side is ticket 02.
- **Channels.** A Restricted member sees no ordinary channel, only DMs, the
  threads of tickets and docs they can read, and Private channels they have
  been added to. Private channels are a feature for everyone, in scope here.
- **Instance-level areas.** A restricted membership counts for nothing at
  instance level: runners, machines, the topology, DNS, connectors never
  open through it, whatever the role carries.
- **People stay visible.** ADR 0086 stands; a Restricted member still sees
  the workspace's names and pictures.
- ~~**Workspace memories** reach a Restricted member's agent turns only if
  their role reads memories.~~ Superseded by ticket 02: workspace memories
  are removed.
- **Who sets it.** Anyone with `members:write` in the workspace, in the Team
  dialog on the person's workspace row. Project settings shows who has
  access, read-only for now. Nobody grants a level they don't hold.
- **Destination.** A spec with its ADRs, then implementation tickets.
