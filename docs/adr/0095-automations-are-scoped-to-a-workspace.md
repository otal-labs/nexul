# Automations are scoped to a workspace

Every automation was one instance-wide row, listed on every workspace's Automations page and checked against the
default workspace's permissions. Switching a default off in one workspace switched it off in all of them, and a
default heard every workspace's events.

Decision: an automation belongs to exactly one workspace, and everything about it follows from that.

- The list, its switch, config, token, host placement, versions, runs, and deletion are checked in the automation's
  own workspace (ADR 0087): outside it the automation is not found, without the action it is forbidden. The HTTP
  list takes an optional `workspace_id`; without one it returns every workspace's automations the caller can read.
  Creating one and every secrets route require `workspace_id`. `automation_list` and `automation_create` take the same
  field; the tool count is unchanged.
- Each workspace has its own copy of each default automation. The default workspace keeps the shipped ids
  (`default-ticket-finished`), so its history, cursors, and runs carry over; another workspace's copy is
  `<id>-<workspace id>`. Creating a workspace seeds them through the same tenancy hook that seeds its plays, and
  every start seeds every workspace, so a new default or new shipped code reaches all of them.
- The secrets pool is per workspace (ADR 0047): a name is unique within its workspace, and an automation receives
  only its own workspace's secrets.
- An event reaches an automation only when it happened in the automation's workspace. The server resolves each
  topic's workspace from its payload: a ticket, doc, or board event through its project, a pull request through
  every workspace among its linked tickets (its repository's when it links none), a deploy through its stack. An
  instance-level event (runners, DNS, accounts, an instance stack) reaches every workspace's automations. A topic
  with no rule reaches none, and a test holds every published topic to having one. The delete payloads of tickets,
  docs, plays, and stacks gained the project or workspace they lived in, since nothing is left to look up.
- A shipped default automation has no creator, so its token's scopes are its whole grant (ADR 0087). It is now also
  held to its own workspace: a check anywhere else answers not found.

The upgrade (migration 0047): every existing automation, custom ones included, and every existing secret belong to
the default workspace, the only one their permissions were ever checked in. Every other workspace gets a copy of
each default with today's switch and empty config, because the config names status columns of the default
workspace's projects.

The trade-offs: a custom automation that reacted to another workspace's events stops hearing them after the upgrade;
it has to be created again in that workspace, since there is no move between workspaces. An instance-level event
now runs once per workspace's subscribed automation. A pull request linking tickets in two workspaces runs both
workspaces' automations.

Rejected: copying custom automations into every workspace, which would run each one once per workspace with a token
nobody holds; and guessing a custom automation's workspace from its creator's memberships or its run history, which
is ambiguous for anyone in more than one workspace.

Amends ADRs 0046, 0047, and 0087. Decided 2026-10-01.
