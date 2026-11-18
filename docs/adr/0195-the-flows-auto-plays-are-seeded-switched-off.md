# The flow's auto plays are seeded, switched off

Amends ADR 0132 and ADR 0193.

ADR 0193 made the built-in plays one flow from interview to merge and test, but one step stayed manual: nothing started
the next ticket when its blocker landed, and Fix with AI and Test with AI chained by themselves only where someone had
composed the auto plays by hand, each workspace slightly differently.

Decision: every workspace comes with the flow's auto plays already there, **switched off**, as the decisions check's and
Investigate a crash's are, so running hands-off is switching three things on.

- **Fix with AI, a ticket enters progress**, on the developer, no conditions. It also builds again after a failed test,
  because a failed test report moves the ticket back to progress.
- **Fix with AI, a ticket becomes unblocked**, on the developer, if its stage is backlog, one at a time. The backlog
  condition leaves a ticket someone already picked up alone; one at a time keeps a blocker that frees three tickets
  from starting three runs that collide in one checkout. A workspace whose computers give each run its own worktree
  switches it off.
- **Test with AI, a ticket enters testing**, on whoever moved it, if it has no tester. For a hands-off build the mover
  is the builder's own run, so the developer's computer tests it; a ticket with a named tester waits for that person.
- **Fix with AI moves a backlog ticket to progress before it starts**, so a ticket started by "becomes unblocked" shows
  on the board as being built. **To tickets assigns what it cuts to whoever ran it**, because both Fix with AI auto plays
  run on the developer and a ticket nobody is assigned to cannot start by itself; the person reassigns where someone
  else builds.
- **No double run, and no new machinery.** The unblocked run's move to progress matches the other Fix with AI auto play,
  which waits behind it as "ticket busy" and is checked again at the front of the queue, where it is skipped once the
  ticket has moved on to review, testing or done. A run that stops with the ticket still in progress leaves the waiting
  one to build again, as a person moving it there would.
- **Existing workspaces get the same three**, off, on their copies of Fix with AI and Test with AI, except where the play
  already has an auto play on the same moment and stage: a workspace that built its own keeps it and gets no
  near-duplicate beside it. The two texts move only where they still equal the previous default (ADR 0103).

Rejected: seeding them switched on, since an upgrade must not start agent runs nobody asked for; an agent play whose
only job is to move a ticket, which spends a turn on one tool call; and running the unblocked one on whoever caused the
moment, since a ticket unblocked by a merge an automation recorded has no person to run on.

Decided 2026-11-18.
