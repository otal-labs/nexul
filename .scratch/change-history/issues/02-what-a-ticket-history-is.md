# What a ticket's history is

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

Does a ticket's history snapshot the whole ticket on each save (title,
description, status, assignee, labels, fields), the way doc versions do, or
record a timeline of field changes ("status Todo → In progress by Alice")
with a text diff only for the description? And how does that fit with the
activity timeline already specced in `.scratch/ticket-workflow-depth/`:
one feature, or two?

## Answer

- A ticket's history is a list of changes, not whole copies of the ticket.
  Each change names who made it and when. A change made through a play run
  or an automation also names that, for example "Alice, via Implement with
  AI".
- Recorded: title, description, status, category, type, developer, tester,
  due date, labels, links (blocked-by, found-in), the doc it was filed from,
  and the finished date. Not recorded: the card's position in its column or
  internal fields such as `updated_at`.
- Only a description change carries a text diff. Every other change reads
  as a one-liner ("status Todo → In review").
- Description edits from the live editor follow the same rule as docs
  ([When editing a doc writes a version](06-when-editing-a-doc-writes-a-version.md)).
- Tickets that exist when this ships start their history on ship day, with
  a "Created by" entry rebuilt from the creation date and reporter they
  already store. Nothing is rebuilt from old events.
- Retention follows [How many versions a history keeps](07-how-many-versions-a-history-keeps.md):
  the last 100 field changes and 10 description versions.
- This list is the activity timeline from `.scratch/ticket-workflow-depth/`,
  which now keeps only custom fields. Whether changes show inside the
  ticket's comment thread or in a History view of their own is decided by
  [The "what changed" dialog](05-the-what-changed-dialog.md).
