# 05 — Invitations and the edges of a Restricted member

**Type:** grilling
**Status:** resolved
**Blocked by:** 03

## Question

How does a Restricted member arrive, and what happens at the edges?

- An invitation: can it carry "Only these projects" plus Project access, so a
  client lands already restricted rather than seeing everything until
  someone edits their row?
- Switching a person from "All projects" to "Only these projects": what is
  preselected, and what they keep.
- A hidden project's ticket that assigns or @mentions a Restricted member,
  and a link to it pasted in a DM they read.
- A project archived or deleted while people hold access to it.
- Their personal access tokens and paired agents: confirm they act as the
  person and inherit the limit with nothing extra.

## Answer

Settled with the owner, 2026-10-01.

- **No "All projects / Only these" switch in the UI.** A person's access in
  a workspace shows as rows, the way the role editor shows domains: an
  **Every project** row on top, then one row per project. Every project is
  either *From role* (your team: the role's project areas on every project,
  new ones included) or *None* (a Restricted member). Under None, each
  project row opens into the area levels (tickets Write, docs Read, and so
  on). This is the `restricted` switch of ticket 03, shown as a row.
- **Switching Every project from From role to None** starts every project
  at None: hidden by default. Switching back keeps the per-project levels
  stored and unused, so a round trip loses nothing.
- **Invitations carry it.** The invitation dialog has the same rows per
  workspace, so a client lands restricted with no window of seeing
  everything; redeeming re-checks the giver still holds every level given.
- **Pickers.** A project's assignee and developer pickers leave out
  Restricted members without access to it. An @mention of one still goes in
  the text but creates no notice, since a notice only goes to someone who
  may read its subject. The people list is unchanged.
- **Links.** A link to a hidden ticket or doc in a message the member reads
  shows the generic section chip ("Ticket") from their own cache and opens
  to not found; no title or project name leaks. Already true today.
- **Deleting a project** removes everyone's access to it (there is no
  archive), and the existing delete confirmation names the Restricted
  members who lose access: a signal, not a gate.
- **Tokens and paired agents** act as the person and inherit the limit
  (ticket 03); nothing to add.
