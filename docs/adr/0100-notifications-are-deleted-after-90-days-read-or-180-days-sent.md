# Notifications are deleted 90 days after they are read, or 180 days after they are sent

Nothing ever deleted a notification. Every doc save notified the doc's workspace, so a busy instance grew its
`notifications` table without bound, and an inbox nobody empties kept every row it was ever sent.

Decision: the server deletes notifications on a schedule, and the deletion is permanent.

- A read notification is deleted 90 days after it was read. Each notification records when it was read (`read_at`,
  beside the `read` flag, which stays, so the HTTP inbox only gains a field and MCP is unchanged); marking one read again keeps the first
  read time. Migration 0049 gives every row read before it its creation time as its read time: the real one is
  unknown, and creation is the earliest it could have been.
- Any notification, read or not, is deleted 180 days after it was sent.
- Both periods are constants in the workspace domain, not settings. A daily loop started by the composition root runs
  the deletion, once a minute after the server starts and every 24 hours after that, the same shape as the automation
  run history's retention, and logs how many rows each rule deleted.

The trade-offs: an old notification is gone for good, so the inbox is not a history of everything that happened; the
doc's versions, the ticket's history, and the play's trail stay where that history lives. A row read before the
upgrade may be deleted earlier than its real read time would have allowed. An instance that
is down for days catches up on the next start rather than on schedule.

Rejected: a general scheduled-jobs domain, which one daily delete does not need; making the periods configurable,
which no one has asked for and which would make the inbox behave differently per instance; and the read-time rule
alone, which would keep an unread notification forever.

Decided 2026-10-01.
