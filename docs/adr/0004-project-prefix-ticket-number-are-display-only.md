# Project prefix + per-project ticket number are a display-only identifier, not a new primary key

Board's ticket cards need a human-readable id (`REF-102`) instead of a raw
UUID fragment. Rather than replacing `Ticket.ID` as the primary key —
which would touch every UUID-keyed consumer across the codebase (dnd-kit
identity, ticket URLs, notifications, `@mention` resolution, code review,
automations, every MCP tool) for no functional benefit — `Ticket.ID` stays
the UUID and stays authoritative everywhere. `Project` gains an immutable
uppercase `Prefix` (2-5 letters, unique per workspace, set once at creation)
and `Ticket` gains a per-project sequential `Number`; the pair renders as
`PREFIX-NUMBER` purely for display on the card. `Prefix` is immutable
because `internal/gitprovider/ticketid.go` already parses this exact
`PREFIX-NUMBER` shape out of branch names and commit messages — changing a
project's prefix later would silently break any reference already written
into git history.

Amended: a prefix is 2-5 characters, a letter followed by letters or digits (`P1`, `V2API`), not letters only.

Amended by ADR 0089: a prefix is unique within its workspace, in the database as well as the use-case, so two workspaces may share one; a key names a ticket only together with its workspace, and every key lookup takes one.
