# GitHub issues sync to tickets through a label

A project's repository can carry its GitHub issues into tickets and keep each ticket in step with its issue. This is the
inbound half; Nexul writing back to GitHub builds on the same links.

Decision: sync is per repository, off until its Issues switch is turned on, and in label mode, the default, only an issue
carrying the sync's label (`nexul`) becomes a ticket. All mode syncs every issue. The links live in `internal/issuesync`,
which consumes `git.issue` and `git.issue_comment` and reaches tickets and chat only through seams it declares.

- **Mapping.** A new ticket lands in the project's first backlog column, reported by kind `github` with the issue's
  author. Its title, body, labels (all but the trigger label) and open or closed state follow the issue; the developer is
  the first assignee a project member signs in with, and never cleared by an issue with none. Comments become messages
  of a new author kind, `external`, keeping the commenter's name and picture with `via github`; they mention nobody, so a
  GitHub `@handle` never pings a member or starts the Agent.
- **Conflicts.** GitHub wins for every field it sends. An edit made in Nexul stays until the issue next changes.
- **Echo.** An issue is applied only when SHA-256 over its title, body, state, sorted labels and first assignee differs
  from what was last applied and from what Nexul last wrote to GitHub, and a delivery older than the last one applied is
  skipped. An event sent by the App's own account, `<app_slug>[bot]`, is skipped outright.
- **Pausing, deletes and transfers.** Removing the label pauses the link and adding it back resumes it. A deleted or
  transferred issue loses its link and the ticket stays, with a note saying why; deleting the ticket drops the link, so a
  still-labelled issue files a new one on its next change.
- **ADR 0064.** Closing an issue moves its ticket to the first done column with a note giving GitHub's reason. A reopened
  issue whose ticket is done only gets a note, since done tickets stay done; one that is not done goes back to backlog.
- **Missed deliveries.** GitHub never redelivers a webhook it failed to send, so the server lists each enabled sync's
  issues changed since its last sweep at start and every 15 minutes, posting any comments missed with them, and stops
  while fewer than 200 requests are left until the limit resets. Turning sync on backfills its open matching issues,
  woken by the commit that stores the switch.

The cost: a timer polls GitHub, the one loop in the server that does not wait only on a commit, and an edit missed with
its webhook waits for the issue's next change. A label rename on GitHub stops syncing until the sync's label follows it.

Rejected: syncing every issue by default, which floods a busy repository's board; matching tickets by title, which
breaks on the first rename; and reopening a done ticket, which ADR 0064 forbids.

Decided 2026-11-18.
