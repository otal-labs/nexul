# A repeating meeting is one row, expanded in its own timezone

Each workspace has a calendar of meetings and ticket due dates, with reminders before them.

Decision: a meeting, repeating or not, is one `meetings` row, and its occurrences are computed when the calendar is
read, never stored one per date.

- **Repeats step in the meeting's timezone.** The row keeps the IANA zone it was made in, and each occurrence is the
  start moved on by days or months in that zone, so a weekly 10:00 stays 10:00 across a clock change. A monthly
  meeting skips a month without its day rather than sliding to the next one. The server embeds the timezone database,
  since a Windows or container host may have none to load.
- **One occurrence changed on its own is a row of its own.** "This one" writes a single meeting with `series_id` and
  `replaces_at` naming the slot it stands in for, and the series leaves that slot out. "This one and the ones after"
  ends the series the second before the slot and starts a new series there, moving the changed occurrences after it
  along. A reply covers the whole row, so declining a repeating meeting declines every occurrence.
- **Only the people in it see it.** A meeting shows to its organiser and its invitees alone, like a private channel,
  so the calendar has no read permission: `calendar:write` creates meetings and `calendar:delete` changes or cancels
  anyone's. Both start off for every role but the Owner. No automation receives a meeting's events, for the same
  reason.
- **Reminders wait on the next one due.** Each row stores its next reminder and the occurrence it is for. One loop
  sleeps until the earliest, or until a commit that may plan an earlier one, sends what is due through the outbox and
  plans the following occurrence. A ticket's developer hears the day before its due date at 09:00 in the timezone
  their app last reported; a person who never opened the app is reminded in UTC.
- **Three tools, not one more read.** `meeting_list`, `meeting_create` and `meeting_update` raise the tool budget by
  three. No existing tool holds meetings, and folding them into a ticket or conversation tool would make that tool's
  arguments mean two things. Replying is a field on `meeting_update`, and a meeting's detail comes on `meeting_list`,
  so there is no `meeting_get`.

The trade-off: a range read expands every repeating meeting the person sees, which is fine for a calendar's tens of
series but would want stored occurrences at thousands. Moving a whole series shifts its changed occurrences by the
same amount, which a new repeat rule can leave off its slots.

Rejected: one row per occurrence, which needs an end date for every series and a rewrite of every future row on each
change; RFC 5545 recurrence rules, far more than none, daily, weekly and monthly need.

Decided 2026-10-24.
