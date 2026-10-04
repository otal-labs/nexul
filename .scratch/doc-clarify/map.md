# Wayfinder map: clarifying a doc

Charted 2026-10-04 with the owner. A client writes what their project
needs in a doc, and it has gaps. A developer presses "Clarify via AI": an
agent reads the doc and asks a round of question cards on the doc page.
The client answers them, in the browser or on the phone. The developer
starts the next round, adding notes if they have any, and the loop goes on
until a run finds no gaps left and writes the whole doc from the client's
text and every answer.

## Destination

"Clarify via AI" built and merged: question cards on the doc page and in
the phone app, answers stored per doc and round, the round-per-run play,
the closing write into the doc, the notifications, and the developer's
close and reopen. This map carries the build: once the decision tickets
are resolved, the build tickets graduate from the fog and are worked here
too.

## Notes

- Decided while charting:
  - A **Clarification** is a doc's rounds of questions and answers. The
    play that runs it is "Clarify via AI", a doc play seeded in every
    workspace beside "To tickets via AI".
  - Anyone who can edit the doc answers; there is no list of named people.
    Anyone who can read the doc sees the questions and answers.
  - One run is one round. The run reads the doc, the earlier rounds, and
    the developer's "Instructions for this run", posts its batch of
    questions, and ends; nothing waits on the client. The developer starts
    each next round.
  - Answers live in the cards, like the interview, and stay editable and
    skippable; a changed answer is picked up by the next run. The doc body
    keeps the client's words until a run finds no gaps left; that run
    writes the whole doc and reports it. The write keeps the client's
    headings and wording, weaves each answer into the section it belongs
    to, adds a section only where nothing fits, and never mentions
    questions or rounds.
  - Each round ends with an optional "Anything else?" box for the client's
    own question. The next run answers it in one plain line under it or
    turns it into questions in its round; either way it reaches the doc.
  - "No gaps left" is a signal; the developer closes the clarification.
    Pressing "Clarify via AI" again reopens it with a new round on top.
    "To tickets via AI" is offered once it is closed.
  - Notifications: a round's questions tell the doc's watchers; the last
    answer of a round tells the person who started that round.
  - Nothing agent-related reaches the client. The panel says "Questions"
    with "Round 1", "Round 2" headings, no author and no AI or Agent
    wording; "no gaps left" shows only to people who can run plays; a doc
    locked by a running round shows the plain locked state. The trail and
    the doc thread stay behind "See doc threads" and the plays permissions,
    as today.
- Technical tickets arrive as a decided answer; look and content tickets
  go to the owner. Plain language with the owner, no ticket numbers in
  questions.
- Web screens are judged at 768, 1024, and 1440px; the phone app on the
  emulator through the device panel.
- Prototype tickets: `/prototype` inside `design-mode`. Grilling tickets:
  `/grilling` + `/domain-modeling`. Production data is live: storage
  changes are a new numbered forward-only migration.
- Grounding: `CONTEXT.md` (Clarification, Interview, Play, Trail, Doc
  thread, Locked doc, Watcher, Restricted member), ADR 0111 (agent prompt
  by reference). The interview machinery this reuses:
  `internal/platform/storage/migrations/0061_interview_answers.sql`,
  `internal/plays/run.go` (`Runner.Answer`), `internal/plays/usecase.go`
  (the seeded plays and their instructions),
  `web/src/components/memory/ProjectInterview.tsx`,
  `web/src/components/play/QuestionCard.tsx`; the doc page is
  `web/src/components/doc/DocDetail.tsx`, the phone's docs under
  `native/src/app/`.

## Decisions so far

- [How the interview's machinery carries over to a doc](issues/01-how-the-interview-carries-over.md): own docs-domain tables beside the interview's; a run posts its round through `doc_update` and ends, `doc_get` returns the rounds; the runner locks at start, unlocks at every end, and only the running round's starter may write through the lock; the checklist rows carry over, `QuestionCard` does not; two new notification kinds; the phone starts from a read-only doc screen.
- [Where clarifications live](issues/03-where-clarifications-live.md): two docs-domain tables (rounds, questions), closed kept on the newest round; see with `docs:read`, answer with `docs:write`, close with `plays:run`; events never carry answers; `doc_get`/`doc_update` carry the rounds.
- [How a round runs](issues/04-the-run-lifecycle.md): built-in `clarify` doc play; the runner opens and closes each round, locking the doc only while it runs (ADR 0121); the agent posts the round or a no-gaps rewrite through `doc_update` and ends; empty failed rounds vanish; two new notification kinds.

## Build

Build tickets 07–12 in `issues/`, worked here per the destination: storage
and agent tools, the play and its rounds, the notifications, the web panel,
the phone view, and the walkthrough.

## Not yet specified

- How a clarification shows in the docs list and the doc's header (an
  "awaiting answers" marker for the client, "answered" for the
  developer), if the prototype shows it is needed.

## Out of scope
