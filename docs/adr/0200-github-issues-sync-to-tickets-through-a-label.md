# GitHub issues sync to tickets through a label

Builds on ADR 0199, keeps ADR 0064, and raises the `tools/list` budget set by ADR 0191.

A team that files bugs on GitHub kept a second copy of each one by hand as a ticket. ADR 0199 gave the App the Issues
permission and the webhook `git.issue` and `git.issue_comment`; this is what Nexul does with them.

Decision: a project switches the sync on per attached repository. In label mode, the default, an issue carrying the
label (`nexul` unless changed) becomes one ticket in the project's first backlog column, reported by its GitHub login;
in all mode every issue does. The link is kept in its own table, `github_issue_links`, one issue per ticket, owned by the
`issuesync` domain, which reads and writes tickets and chat only through interfaces of its own (ADR 0017).

- Mapping. GitHub wins for the fields it sends: the title and Markdown body through the ticket's update path, which
  resets an open editor (ADR 0124); the labels but the trigger label; the developer as the first assignee who signed in to
  Nexul with that GitHub login and can open the project, the others ignored, and nobody mapped leaves the developer as
  it is. Comments become ticket-thread messages by an `external` author, a snapshot of the login and avatar with `via`
  `github`, whose mentions are never parsed, so an outsider's `@Agent` starts nothing; edits and deletes follow through
  `github_comment_links`.
- Status. A closed issue moves its ticket to the first done column with a note giving the reason. A reopened issue never
  takes a done ticket out of done (ADR 0064): it only gets a note saying to report a bug; a ticket not yet done moves back
  to the first backlog column.
- Pausing. Removing the label pauses the link and keeps the ticket; adding it back resumes it and applies what changed.
- Conflicts and echo. Each applied issue stores a SHA-256 over its title, body, state, sorted labels and first assignee,
  in that order. A delivery whose hash matches the last one applied, or the last one Nexul wrote (left for the outbound
  half to fill), writes nothing, and one older than the last applied is dropped. A delivery sent by the App's own
  `<app_slug>[bot]` is skipped.
- Deletes and transfers drop the link, leave a note and keep the ticket.
- Missed deliveries. GitHub never redelivers, so turning a sync on lists its open matching issues, and at server start
  and every 15 minutes each enabled sync lists the issues updated since its last sweep. The sweep stops while GitHub's
  `X-RateLimit-Remaining` is under 200 until the reset, or for a rate-limited answer's wait, and an unfinished sweep runs
  again. One lock covers every issue, so a webhook and a sweep never file two tickets.
- Turning it on is owners only: `project_update` takes `github_issue_sync` and `project_get` shows it, and the web
  project settings put an Issues switch on each repository row, which says when the installation lacks the permission.

The cost: `project_update`'s field adds about 600 characters to `tools/list`. The colour descriptions it repeats are held
by a test that every field lists its values, so nothing repeated could be cut; `advertisedBudget` rises from 193,000 to
193,600. A ticket filed, then a crash before its link is saved, leaves one ticket the next delivery would file again.

Rejected: matching issues to tickets by title or a key in the body, which breaks on a rename; a Nexul-side label list
that must match GitHub's; polling without webhooks.

Decided 2026-11-18.
