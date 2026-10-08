# 01: How "ticket becomes unblocked" and "doc changed" become events

Type: research
Status: open
Blocked by: None — can start immediately

## Question

Two of the six moments have no event today.

- **A ticket becomes unblocked**: its last open Blocked-by ticket reaches a
  done stage, or the last link to an open blocker is deleted. Blocked-by is
  a link, and today only `ticket.link_created` and `ticket.link_deleted`
  exist. Where is "blocked" computed now (the icon, the warning before a
  play runs), and where should a new `ticket.unblocked` topic be published
  so it fires exactly once per unblock, in the same transaction as the
  change that caused it (catalog row and outbox write, ADR 0044)? What
  about a blocker moved out of done again and back?
- **A doc is created or changed**: "changed" fires once edits have stopped
  for 10 minutes, never for an edit an agent's run made. Which doc topics
  exist, how an edit is attributed (person, agent via MCP, a play run's
  agent), and how a settle timer survives a server restart. Is there an
  existing debounce or scheduled-job seam to reuse?

Findings go in `research/01-the-new-moments.md`.
