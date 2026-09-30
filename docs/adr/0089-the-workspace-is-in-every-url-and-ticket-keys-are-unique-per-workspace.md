# The workspace is in every URL, and ticket keys are unique per workspace

Project prefixes had an instance-wide unique index, so two workspaces could not both have a `WEB` project, and web
URLs such as `/docs/ONLY/<id>` or `/tickets/WEB-12` named no workspace: the selected one was whatever the browser
last remembered, so a shared link opened in someone else's current workspace.

Decision: a workspace has a slug, and every page reached from its sidebar lives under it.

- A slug is lowercase letters and digits joined by single dashes, at most 48 characters, unique on the instance. It
  is derived from the name when the workspace is created (`My Workspace` becomes `my-workspace`; a taken slug gets
  `-2`, `-3`, and so on), and paths the server or web app own at the top level (`settings`, `wizard`, `api`,
  `login`, and the rest) are refused. A rename keeps the slug unless the rename names a new one, so links keep
  working through a rename; the owner wizard names both at once because nothing links to the workspace yet.
  Migration 0039 backfills existing workspaces by the same rule.
- Board, tickets, docs, memories, chat, inbox, runners, topology, automations, stacks and deploys, Configuration,
  project settings, the interview, and the project wizard are `/<slug>/…`. Runners and topology stay instance-level
  on the server (ADR 0087) but live in the workspace sidebar, so they carry a slug too. Personal and instance pages
  (`/settings/*`, `/login`, `/invite`, `/setup`, the onboarding wizards) stay unprefixed.
- The URL decides the selected workspace, and the client's stored selection follows it. An unknown slug, or one the
  viewer is not a member of, is not found. The switcher keeps the section and drops the item, since a project,
  doc, ticket, or conversation belongs to the workspace it was opened in.
- A project prefix is unique within its workspace, so a ticket key such as `WEB-12` names a ticket only together
  with its workspace. Every key lookup takes a workspace, as an id or a slug: the web app has it from the URL, and
  `GET /api/tickets/{key}` and the ticket MCP tools take an optional `workspace`. Without one, a key resolves among
  the tickets the caller can read; a key found in several of the caller's workspaces is refused with their slugs
  ("WEB-12 exists in otal and rixwave; pass workspace") rather than guessed. Keys named inside `ticket_update` and
  `ticket_create` resolve in the ticket's own workspace. The @ picker lists a key's match in every workspace.

Rejected: redirecting old unprefixed paths to the current workspace. It would keep a second route table alive for
links that name no workspace, and a guessed workspace is exactly the ambiguity this removes; an old path is not
found. The HTTP API stays id-based and only grows (ADR 0082): workspace payloads gain `slug`, and nothing moves.

The trade-off: a key is no longer a complete reference on its own, so a key pasted outside a URL, such as in a
commit message, needs the reader's context to pick the workspace. Mention links stored in document bodies are
`/tickets/<id>` and `/docs/<id>`, ids that need no workspace, and are unchanged.

Amends ADR 0004. Decided 2026-09-30.
