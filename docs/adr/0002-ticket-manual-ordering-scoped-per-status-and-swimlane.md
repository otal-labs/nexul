# Ticket manual ordering is scoped per (status, swimlane), not globally per status

Board gains manual ticket ordering within a column (T1). Position resets when
a ticket's swimlane (category) changes, not only when its status changes —
chosen deliberately over a single Trello-style position-per-status, so a
ticket's place is always relative to the lane it's currently grouped under.
Costs a composite key (`ticket_id, status_id, category_id`) instead of a
single position field on the ticket.
