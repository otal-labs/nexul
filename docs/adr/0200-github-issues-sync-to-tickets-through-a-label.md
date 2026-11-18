# GitHub issues sync to tickets through a label

A project's repository can carry its GitHub issues into tickets and keep each ticket in step with its issue, both ways.

Decision: sync is per repository, off until its Issues switch is turned on, and in label mode, the default, only an issue
carrying the sync's label (`nexul`) becomes a ticket. All mode syncs every issue. The links live in `internal/issuesync`,
which consumes `git.issue` and `git.issue_comment` and reaches tickets and chat only through seams it declares.

- **Mapping.** A new ticket lands in the project's first backlog column, reported by kind `github` with the issue's
  author. Its title, body, labels (all but the trigger label) and open or closed state follow the issue; the developer is
  the first assignee a project member signs in with, and never cleared by an issue with none. Comments become messages
  of a new author kind, `external`, keeping the commenter's name and picture with `via github`; they mention nobody, so a
  GitHub `@handle` never pings a member or starts the Agent.
- **Conflicts.** GitHub wins for every field it sends: an issue change applied while a Nexul edit waits to be written
  replaces that edit, which then has nothing left to write.
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

- **Writing back.** A change to a linked ticket's title, body, labels, developer or stage queues a push, and every
  further change moves it to ten quiet seconds after the last; a loop woken by the commit that queues it, and by the
  next due push, writes it as one issue update, as the App's installation so the inbound bot check knows its echo.
  The body is markdown whose instance links are absolute and whose person mentions are `@<github login>`, or links to
  the person's page for someone without one; a Nexul login never reaches GitHub, since it can be an email. Labels are
  set in full plus the trigger label, a done stage closes the issue as completed, and the developer becomes the one
  assignee only with a GitHub login. Tester, due date and category stay in Nexul.
- **Comments back.** A person's message on a linked ticket's thread is posted as a comment headed with their name and
  GitHub login, and its edits and deletes follow through the comment links; agent replies, notes and mirrored GitHub
  comments never go back.
- **No echo, either way.** Besides the inbound and outbound hashes, the link keeps the hash of the ticket as its issue
  takes it whenever the two were last in step, after applying an issue as after writing one; a push whose ticket
  hashes to any of the three writes nothing, so what GitHub changed never goes back out.
- **Creating issues.** Adding the trigger label to an unlinked ticket of a synced project creates its issue. The body
  carries a hidden `<!-- nexul:ticket:<id> -->`, and a comment `<!-- nexul:message:<id> -->`: a retry after a lost
  answer first looks for the marker among what changed since its first try, so a timed-out request GitHub accepted
  is never made twice, and an issue whose link was lost is claimed by its ticket instead of filing another.
- **Deletes and failures.** Deleting a ticket drops its link and leaves the issue. A 404 or 410 from GitHub drops the
  link with a thread note. A rate limit waits out `Retry-After` and rests the owner for the sweep too, other retryable
  failures back off from 30 seconds to half an hour and are dropped after ten tries, and each owner's installation
  gets one write a second and 400 creations an hour, under GitHub's limits on content creation.

The cost: a timer polls GitHub, the one loop in the server that does not wait only on a commit, and an edit missed with
its webhook waits for the issue's next change. A label rename on GitHub stops syncing until the sync's label follows it.
An edit made in Nexul reaches GitHub ten seconds late, and one made while the issue changes on GitHub is overwritten
by GitHub's. Writing as the App means GitHub shows the App, not the person, as the author.

Rejected: syncing every issue by default, which floods a busy repository's board; matching tickets by title, which
breaks on the first rename; reopening a done ticket, which ADR 0064 forbids; writing each change as it commits, which
turns live editing into a write per keystroke pause; and writing as the person, which needs every member's GitHub
token and loses the bot check.

Decided 2026-11-18.
