# The decisions check is a play nobody presses

ADR 0055 says nothing fires a play but a person. The decisions check is the
one exception: when a ticket enters a done-stage column, a server-side
consumer of `ticket.status_changed` starts it once, as a play run on the
paired computer of the person who moved the card, or on the ticket's
developer's computer when an automation moved it (a merged PR). It keeps
everything else ADR 0055 asks for: it runs on a person's own harness with
that person's permissions, posts into the ticket's thread, and leaves a trail.

The alternative was a button on done tickets. Rejected: the owner's point is
that people forget, and the check only earns its place if every done ticket
gets the agent's judgement on whether it changed how the project works.

The cost is that an automatic run can fail with nobody watching. So a check
that cannot start (no computer, the setup gate refusing the provider, no
developer to run it for, another run holding the ticket) is kept as a failed
trail on the ticket, the ticket shows "Decisions check didn't run" with a
button to run it on the viewer's own computer, and agents have
`decisions_check_run`. A redelivered event or a move between two done
columns never fires a second check, because done tickets are never reopened
(ADR 0064).
