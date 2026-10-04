# 03: Where clarifications live

Type: grilling
Status: resolved
Blocked by: 01

## Question

A technical ticket, answered as a decision. Given the research:

- The storage shape: per doc, per round, per question, with the agent's
  options and why-line, picked values, free text, skipped, who answered
  and when, the "Anything else?" text and its reply, and whether the
  clarification is open or closed. Shared with the interview's table or
  its own.
- Permissions: reading the doc to see, editing the doc to answer, running
  plays to see "no gaps left", close, and reopen. Who may close.
- Events and their catalog rows, and live push to everyone on the doc.
- MCP: extending the doc tools so an agent reads and writes rounds,
  before adding a tool.
- What happens to the clarification when the doc is archived, cloned, or
  deleted.

## Answer

Decided from the research (`research/how-the-interview-carries-over.md`),
which found that sharing the interview's table would leak a doc's questions
to people who can read memories but not the doc.

- **Own tables in the docs domain**, apart from `interview_answers`, in one
  new numbered migration:
  - `doc_clarification_rounds`: doc (cascading from docs), round number
    from 1, who started it, its trail, whether its run is still going,
    whether this run took the doc's lock, the "Anything else?" text with
    who wrote it and when, the next run's one-line reply to it, whether the
    round found no gaps and wrote the doc, and when the clarification was
    closed and by whom. Closed lives on the newest round: starting another
    round is what reopens it, so there is no separate open/closed record.
  - `doc_clarification_questions`: an id, doc, round, position, the
    question, its why-line, options, single or multi-select, picked
    values, free text, skipped, who answered and when. Unique on doc,
    round, and question. A posted question with no picks, no text and no
    skip is pending, as in the interview.
- **Permissions**, checked on the doc so doc sharing and Restricted members
  behave as they do for the body:
  - See the questions and answers: `docs:read`.
  - Answer, skip, clear an answer, write "Anything else?": `docs:write`.
    Answers are not doc edits, so a locked doc does not refuse them.
  - Close, and see the "no gaps left" signal: `docs:write` plus
    `plays:run` on the Clarify via AI play. Reopening is starting the play.
- **Events** in the docs domain, each with a catalog row and an outbox
  write, carrying the question and the author but never the answer, with
  live push to everyone who can read the doc (the audience `doc.updated`
  uses): a round posted (with its question count and whether it found no
  gaps), an answer saved, an answer cleared, a round fully answered (naming
  the round's starter, for the notification), the "Anything else?" saved,
  and the clarification closed.
- **HTTP**: read a doc's clarification; save, skip, or clear one question's
  answer by question id; save a round's "Anything else?"; close.
- **MCP**, extending the doc tools rather than adding one (the surface is at
  107 of 108):
  - `doc_get` returns the clarification: every round with its questions,
    answers, "Anything else?" and reply, whether a round is running, and
    whether it is closed.
  - `doc_update` takes `answers` (any agent acting for someone who may edit
    the doc, a peer of the browser), and, only from the running round's
    starter while that round runs, `questions`, `anything_else_reply`, and
    `no_gaps` with the new body. It also takes `clarification_closed`.
- **Archive, clone, delete**: archiving changes nothing (an archived doc is
  still edited and read as before). A clone starts with no clarification,
  as it starts unlocked. Deleting the doc deletes it.
