# 04: How a round runs

Type: grilling
Status: resolved
Blocked by: 01

## Question

A technical ticket, answered as a decision. Given the research:

- How a run posts its round and ends, what the trail shows for it, and
  how a run that finds no gaps writes the doc and signals it.
- Locking: the doc locked only while a round's run works, unlike other doc
  plays, and what the client sees if they are editing when it starts.
- Starting a round while the last one has unanswered questions: allowed
  with a signal, or not offered.
- The two notifications: a round's questions to the doc's watchers, the
  round's last answer to its starter.
- Seeding "Clarify via AI" into every workspace, new and existing, with
  its built-in key, and its place in instance templates.
- Raised by the research: unlocking at the end of a round's run reverses
  what ADR 0107 records the owner declining for doc plays, so the answer
  here comes with an ADR amending it.

## Answer

Decided from the research; the lock change is ADR 0121, amending ADR 0107.

- **The play**: "Clarify via AI", a doc play with the built-in key
  `clarify`, seeded into every workspace by `SeedDefaults` and into
  existing ones by a forward migration, with its instructions in the
  instance templates like the other built-ins.
- **A round from start to end**:
  1. Pressing the play launches a doc play run as today (doc thread,
     "Started" message, trail). The runner opens the next round through a
     docs seam, recording the starter, the trail, and whether locking the
     doc took this run's lock.
  2. The prompt names the doc by id (ADR 0111). The agent reads it with
     `doc_get`, which carries the body and every earlier round, plus the
     developer's "Instructions for this run".
  3. The agent posts the round with `doc_update` (`questions`, and
     `anything_else_reply` when the last round had one) and ends its turn.
     A run that finds no gaps instead sends `no_gaps` with the whole new
     body; that write goes through the lock because it comes from the
     running round's starter.
  4. Every end of the run (done, failed, stopped, silent, cut off by a
     restart) closes the round through the same seam: the round stops
     running and, if this run took the lock, the doc unlocks. A doc that
     was locked before the run stays locked. A round that ended with no
     questions and no gaps verdict is removed, so a failed run leaves no
     empty "Round 3" behind; the trail still shows what went wrong to
     people who can see doc threads.
- **The question tool**: the instructions forbid it and say to post the
  round and end. No runner guard; if an agent asks anyway, the trail waits
  as today and Stop frees the doc.
- **While a round runs**: the client sees "More questions are on the way"
  and the doc's plain locked state. Earlier rounds stay answerable; the
  run read them at its start, so a change made now is picked up by the
  next round. An edit typed into the body in the moment the lock lands is
  dropped, as with any lock today.
- **Starting a round with questions still unanswered** is allowed, with a
  signal in the panel and the run dialog ("2 questions in Round 1 are
  still unanswered"). Pending questions stay open, and the agent does not
  ask them again.
- **After the doc is written**: if an answer changes after a round wrote the
  doc, people who can run plays see a signal that answers changed since the
  doc was written; another round picks them up.
- **Notifications**, two new kinds, both on the doc:
  - questions asked: to the doc's watchers when a round posts at least one
    question, minus its starter.
  - questions answered: to the round's starter when its last pending
    question is answered or skipped.
  Their inbox wording is decided with the play's instructions.
- **Restart**: the round's running flag and lock record are stored, so a
  run followed again after a restart (ADR 0119) ends its round the same way.
