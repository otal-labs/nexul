# 02 — What the rework touches in the code

**Type:** research
**Status:** resolved
**Blocked by:** None

## Question

What does each decision in ticket 01 change in today's code?
- every check path and list filter
- every writer of `permission_overwrites`
- every use of Restricted, `every_project` and project access
- the live audience rules
- the MCP fields
- invitations
- the web editors
- the phone app

What must the wipe migration delete and keep, row by row? Which statement-count guards will move, and why?
Write the findings to `research/what-the-rework-touches.md`, with file:line references, grouped so the rewrite
can slice tickets from them.

## Answer

Researched on 2026-10-11 in [research/what-the-rework-touches.md](../research/what-the-rework-touches.md), grouped
by area with file:line references.

- **Evaluator.** Opt-in projects turn today's Restricted-only project read into a read for every member, in
  `layers()`, `ProjectsWith` and the memo. Two evaluators outside `internal/access` also assume one role: the role
  editor's `requireAction` and the invitation SQL.
- **One role.** It is assumed in the schema, in 13 sqlc queries, in the tenancy use-cases, in invitations,
  in `/me`, in client-core and in five MCP tools.
- **Restricted also hides public channels.** Dropping it without channel rules widens what a client reads.
- **The wipe on production** deletes 2 project rows and one Restricted flag. It keeps all 32 doc rows, which are
  every creator's own `CreatorGrant`. The one open invitation carries nothing to strip.
- **MCP** sits at 110 tools against an effective ceiling of 110.
- **Performance guards** compare few rows with many, so they hold if every new read is batched. Their fixtures must
  list people in projects, or they pass for the wrong reason.
- **Open questions for the owner** close the research file.

