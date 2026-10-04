# Research: how the interview's question machinery carries over to a doc

Answers ticket `01-how-the-interview-carries-over.md`. Read from source on 2026-10-04, at `875dd4ff`. Every path is
relative to the repo root. The decisions in `map.md` are taken as given; where this file recommends something, it is
for the grilling tickets on storage and the run lifecycle to adopt or reject.

## 1. The interview's answer storage: generalise it, or a doc table beside it

**What exists**
- One table, `interview_answers`: workspace, project, round, question, the follow-up's options, `multi_select` and
  `why`, `selected`, `free_text`, `skipped`, `answered_by`, `answered_at`
  (`internal/platform/storage/migrations/0061_interview_answers.sql:2-17`). It is unique on `(project_id, round,
  question)` (`:17`), and `project_id` is `NOT NULL` with a cascade from `projects` (`:5`).
- Six queries: list, upsert (round 0), update (a stored question only, keeping its options and why), delete, last
  round, insert (`internal/platform/storage/queries/interview_answers.sql`).
- Use-cases in the memories domain (`internal/memories/answers.go`):
  - `ListAnswers` needs `memories:read` on the project (`:24-37`).
  - `SaveAnswer` needs `memories:write`. Round 0 upserts. A later round only updates a question that is already
    stored (`:41-67`).
  - `ClearAnswer` deletes a round-0 row but leaves a later round's question bare, with no picks, no text and not
    skipped (`:71-97`).
  - `RecordRound` is the server writing a whole round at once, with no permission check (`:101-139`).
- Events are `interview_answer.saved` and `interview_answer.cleared`. Their payload carries the question text and the
  author, never the answer (`internal/memories/events.go:13-19`, `:66-74`). Both have catalog rows
  (`internal/integrations/catalog.go:914-938`), a workspace scope for automations
  (`server/cmd/automation_scope.go:55`), and a live frame for anyone with `memories:read` on the project
  (`server/cmd/live_audience.go:113-114`, `:297-302`).
- MCP has no tool of its own. `memory_create` kind `interview` and `memory_get` return the questions and the stored
  answers. `memory_update` takes round-0 `answers` (`internal/memories/mcp.go:35`, `:48`, `:145-147`, `:270-305`).

**A bare stored question is already an "unanswered" row.** `ClearAnswer` leaves a follow-up with its question,
options and why, but no answer (`answers.go:69-91`). The web counts a row with no picks, no text and no skip as
pending (`web/src/models/InterviewAnswer.tsx:54-59`). So a round the agent posted, and nobody has answered yet, fits
the row shape as it is.

**What a clarification needs that the table lacks**
- A doc to hang off.
- Per round: the "Anything else?" text and the line that answers it, who started the round (for the last-answer
  notification), and whether its run is still going.
- Per doc: open or closed, and the "no gaps left" signal.

**Option A: generalise `interview_answers`.**
- **Schema.** Add a nullable `doc_id`. SQLite treats NULLs in a unique index as distinct, so the interview's
  uniqueness needs an expression index such as `COALESCE(doc_id, '')`. The upsert's `ON CONFLICT (project_id, round,
  question)` has to be rewritten to match it (`interview_answers.sql:5-10`). The per-round and per-doc fields above
  still need a second table.
- **Permissions.** Every use-case branches on the target: project-level `memories:*` for the interview, and doc-level
  `docs:read` and `docs:write` (doc sharing, Restricted members) for a doc.
- **Events.** `interview_answer.*` is a published contract with `project_id` required (ADR 0044). Its payload carries
  the question text, and the live frame goes to `memories:read` holders, so a doc's questions would reach people who
  cannot open the doc unless the audience branches too.
- **Ownership.** The rows belong to the memories domain, so the docs domain would reach into memories for its own
  data, against ADR 0017.
- **Naming.** `CONTEXT.md` lists "interview" under Clarification's _Avoid_.

**Option B: a docs-domain table beside it (recommended).**
- **Storage.** A new numbered migration: the question rows with the same columns plus `doc_id`, cascading from `docs`.
  Then either a small rounds table (round, starter, trail, "Anything else?" text and reply, whether it locked the doc)
  or round fields on the first row. Ticket 03 picks.
- **Queries.** The six queries are copied with `doc_id` in place of `project_id`.
- **Use-cases.** About 185 lines, mirroring `answers.go`, gated by `docs:read` and `docs:write` on the doc through the
  docs service's own `require`.
- **Events.** `doc_question.*` or similar, with catalog rows and a live frame that reuses `readsDoc`
  (`live_audience.go:282-285`), the same check `doc.updated` uses (`:103`, `:263-270`).
- **Cost.** One migration and an upgrade test like `migration_0061_test.go`, a repo, use-cases, events, catalog and
  automation-scope rows, and HTTP routes. Most of it is a near-copy. `normalizeAnswer` (`answers.go:142-179`) is small
  enough to copy rather than share across domains.

**Recommendation: B.** The tables have different access rules, a different event audience, and different owners. A
shared table would need a branch in every one of those places to keep a doc's questions from leaking to people who
can read memories but not the doc.

## 2. Posting a round and ending, instead of waiting on the question tool

**How the interview asks today**
- The run asks with the harness question tool.
- `trailObserver.OnQuestion` sets the trail to `waiting`, stops the silence timer, stores `trail.Question`, and writes
  `play.run_waiting` (`internal/plays/run.go:757-768`).
- If the harness closes the turn under the question, the trail stays `waiting` (`run.go:791-801`).
- `Runner.Answer` resumes the run. Only the starter or a `plays:write` holder may answer (`run.go:378-428`), and the
  answered round is copied into the stored answers afterwards (`recordFollowUps`, `run.go:430-451`).
- None of that fits a client who answers days later, without `plays:write`, from a doc page. A round has to be data
  the run writes and then leaves behind.

**The write path: extend the doc tools, add no tool**
- The surface has 107 tools against a budget of 108. The budget test asserts the count stays under it
  (`internal/mcp/surface_test.go:21`, `:38-40`; counted 107 with a throwaway test at `875dd4ff`).
- A new tool needs an ADR raising the ceiling, and `practices/mcp.md` §4 asks to extend an existing tool first.
- The interview's precedent is fields on the existing memory tools (§1).
- Fit for docs:
  - `doc_update` is a patch where every field is optional (`internal/docs/mcp.go:63-71`, `:214-251`). It takes a
    `questions` field: the round's questions, each with its options, multi-select and why. It also takes a field for
    the reply to the previous round's "Anything else?" and a `no_gaps` flag.
  - `doc_get` returns the clarification: its rounds, their answers, the "Anything else?" texts and replies, and
    whether it is open.

**Which round and who posted it**
- The agent's MCP calls are made with the paired computer's own access token, "Nexul MCP on <computer>", minted for
  the computer's owner (`internal/pairing/mcp.go:319-323`). A play runs on the starter's own computer
  (`run.go:300`). So the actor of the `doc_update` call is the round's starter, and the existing actor exclusion in
  notifications already keeps them out of the watchers' notice (§6).
- `refuseIfActive` allows one live run per target (`run.go:552-563`), so a doc has at most one round running at a
  time.
- **Recommended rule:** the runner opens the round through a seam when the run starts, and closes it when the run
  finishes (§3). `doc_update` accepts `questions` only while a round is running on the doc, and only from its
  starter.
- That rule keeps a chat agent or a script from posting rounds outside a run. The same record gives the closing
  write its permission through the lock (§3).

**How the run ends**
- After the `doc_update` call the agent replies and its turn ends.
- `OnFinished` takes the ordinary path to `finish` (`run.go:791-812`, `:906-916`).
- The trail ends `done` and the starter gets the existing "finished" notification (`internal/workspace/usecase.go:1354-1360`).
  Only the starter receives it, so its "Clarify via AI" wording never reaches the client.
- Nothing waits, so the silence timer, restarts and an offline computer only matter while the round is being
  written.

**What the trail records**
- The `doc_update` call is a tool step like any other. `CONTEXT.md` Trail says an MCP call's row names its server and
  tool.
- The run's reply is the prose at the end. The starter's "Started" message (`run.go:336-341`) and the run's notes
  (`run.go:949-957`) land in the doc thread as they do for any doc play.
- Nothing new is needed on the trail. The trail's `question` field stays empty for these runs.

**If the agent asks with the question tool anyway**
- The trail parks in `waiting`. With today's lock it keeps the doc locked until someone answers it or presses Stop
  behind "See doc threads" (`DocDetail.tsx:116-124`).
- The instructions must say: never use your question tool, post the round with `doc_update` and end.
- A guard in the runner is possible: for this play, `OnQuestion` could stop the run with a note. It is not needed,
  because Stop already frees the doc (§3), so ticket 04 can leave it out.

**How the page shows questions that came from no live run**
- The interview page already has the path. `buildSections` turns stored rounds into rows
  (`web/src/models/InterviewAnswer.tsx:83-98`).
- `InterviewQuestionForm` with no `onAnswer` saves each answer on its own through the save route
  (`web/src/components/memory/InterviewQuestionForm.tsx:313-326`), which updates the stored question
  (`answers.go:62-66`).
- The live branch (`useInterviewLiveRound`, `row.live`, answering through the trail) is the part a doc does not need.

## 3. What a doc play does to its doc today, and locking only while a round works

**Today**
- `launch` opens the doc thread (`run.go:331`, `:667-672`) and posts "Started <play>" there (`:336-341`).
- It then calls `lockDoc` (`:345-347`, `:358-372`). `LockForPlay` locks with no permission check and reports whether
  this call did the locking (`internal/docs/usecase.go:369-383`).
- The note "Locked the doc because the run started; it stays locked after the run ends." goes onto the trail and into
  the thread (`run.go:60-61`).
- Nothing unlocks the doc. ADR 0107 records this, and lists unlocking at run end as rejected by the owner for doc
  plays in general.
- A locked doc refuses `Update` (`usecase.go:289-291`). The collab relay drops every edit to it
  (`internal/collab/session.go:100-105`). The web page drops its edit session, so the title and body render
  read-only, and shows the plain "Locked" signal with Unlock for `docs:lock` holders
  (`web/src/components/doc/DocDetail.tsx:58-62`, `:137`; `DocLockedSignal.tsx`).
- **Someone typing when the lock lands:** their edits from that moment are dropped on the server, without an error
  shown in the editor. The page then flips to read-only when `doc.updated` with `locked` arrives.

**The clash**
- The round that finds no gaps writes the whole doc during its own run.
- Under today's lock that write is refused, even though "Locked doc" in `CONTEXT.md` says "from people and agents
  alike".
- The agent can unlock first (`doc_update` with `locked: false` and the body, `docs/mcp.go:219-224`), but only when the
  starter holds `docs:lock` (`usecase.go:360`). Starting a doc play deliberately does not require it (ADR 0107).

**What "locked only while a round works" takes**
1. **Lock as today at start.** Record on the round whether this run did the locking: `LockForPlay` already returns
   that. The record must be persisted, because the observer is gone after a restart, and
   `ResumeRunsAfterRestart` re-follows the run (`run.go:879-904`).
2. **Unlock at every end, if this run locked the doc.** The ends are `finish` (`run.go:906-916`), which covers
   done, failed, interrupted, silence and Stop. It also covers Stop without a live turn (`:869-876`) and a restart
   before the run started (`:888-892`). Each needs a no-check docs use-case beside `LockForPlay`. A doc someone had
   already locked stays locked.
3. **Let the closing write through.** Options:
   - **(a)** `doc_update` with `no_gaps` and a body is accepted through the lock while that doc's round is running
     and the caller is its starter. The round record from §2 is the provenance, so docs still gets no `locked_by`
     column, which ADR 0107 rejected. People stay refused, because the collab relay and the plain update path never
     carry the flag. **Recommended.**
   - **(b)** The agent leaves the final body on the round, and the runner writes it after unlocking at `finish` when
     the run is `done`. This is atomic, but it stores a pending body and moves a doc write into the plays seam.
   - **(c)** The agent unlocks itself. This depends on the starter holding `docs:lock` and on the agent obeying, so
     it is not recommended.
   - **(d)** No real lock, and the page renders read-only while a round runs. Nothing on the server stops an edit
     made mid-run from being overwritten by the closing write's full-body replace. Not recommended.
4. **Which plays behave this way.** The built-in key (`internal/plays/model.go:60-61`, `usecase.go:238-256`) survives
   renames and clones. A new play type would let custom clarify plays exist too, which no one has asked for, so the
   built-in key is enough.
5. **Records to update:** an ADR amending ADR 0107 for this play, plus the `play_run` and `doc_update` descriptions,
   which say the doc "stays locked" (`internal/plays/run_mcp.go:121-122`, `internal/docs/mcp.go:182-183`), and
   `CONTEXT.md` Locked doc.

**Answers are not doc edits.** They live in their own table, so a lock need not refuse them, neither the run's lock
nor a long-standing one. Whether answering is paused while a round's run is reading is a lifecycle question for
ticket 04. Nothing in the code forces it.

## 4. What the doc page can reuse from `QuestionCard` and the Interview checklist

| Piece | Reuse as is? | Why |
|---|---|---|
| `QuestionStep` (`web/src/components/play/QuestionStep.tsx`) | Yes | Option rows (radio or checkbox), the hint, and the "Something else…" free text (`:162-169`). No wording to change. |
| `QuestionOptionRow` | Yes | Used through `QuestionStep`. |
| `InterviewSection` (`web/src/components/memory/InterviewSection.tsx`) | Yes, once moved to a shared folder | A foldable header with a label and a count. Pure display. Fits "Round 1", "Round 2". |
| `InterviewQuestionRow` | Yes, once moved | The numbered marker, title and one-line answer. Pure display over a row. |
| `countLine`, `rowStatus`, `firstPendingKey`, `nextPendingKey`, `answerLine` (`models/InterviewAnswer.tsx`) | Yes, once typed against a shared row shape | Pure functions over rows. |
| `InterviewQuestionForm` | With changes | It is bound to `useSaveInterviewAnswer` and `projectId` (`:308`, `:318-326`). It needs a save callback instead. Its "Why I'm asking:" (`:340`) is first person from the asker, which is a wording question for the owner. |
| `InterviewChecklistFeed` | Pattern only | Interview-specific: the live round, the "Follow-up i of n" label (`:120`), the done row, and memory wording (`:128-131`). |
| `buildSections` | Pattern only | It labels rounds "Follow-ups from the agent N" (`models/InterviewAnswer.tsx:87`), which is Agent wording. |
| `recommendedDraft` | Yes, if the owner keeps "(Recommended)" | It pre-picks options labelled "(Recommended)" (`:135-140`). That label shows to the client. |
| `QuestionCard` (`web/src/components/play/QuestionCard.tsx`) | No | It answers a live harness request in one submit, with no per-question save or Skip. Its `aria-label` is "Question from the Agent" (`:64`). |

- The "Anything else?" box and its reply line have no counterpart in the interview, so they are new.
- The F3 shared-display rule (`practices/react-guide.md`) argues for moving the reusable rows and sections out of
  `components/memory/` rather than importing interview components into the doc page.

## 5. The phone app's doc view

- `native/src/app/(tabs)/more/docs/[id].tsx` renders `DocScreen`. That is the title, "v<version> · updated <time>",
  and the body as markdown, read-only (`native/src/components/docs/DocScreen.tsx:31-37`,
  `native/src/components/docs/DocBody.tsx`).
- There is no thread, no plays, no lock signal and no editing.
- **No question card exists on the phone.** The only question-shaped thing is a handed-off agent's step kind
  `question` in chat (`native/src/models/Chat.tsx:59`). The app's UI primitives are `button`, `input` and `text`
  (`native/src/components/ui/`), so a radio or checkbox row comes from React Native Reusables through the shadcn CLI.
- Live updates already invalidate the doc on `doc.*` (`native/src/hooks/useLiveEvents.tsx:47-49`). The new
  clarification topics need entries there.
- The server is always newer than the app. An app from before this change shows the doc with no questions and keeps
  working.

## 6. Notifying a doc's watchers and one named person

- **Watchers.** `onDocActivity` lists the doc's watchers and notifies them (`internal/workspace/usecase.go:1269-1291`).
- **One named person.** `fanOutByUserID` notifies one person, which is how a play's starter hears about their run
  (`usecase.go:1368-1379`).
- **Shared filtering.** Both drop the actor and anyone who cannot open the doc (`create`, `usecase.go:1508-1511`;
  `canOpen`, `:1186-1200`). A Restricted member without the doc is skipped, and so is the round's own starter.
- **Push.** It says only "New activity in <workspace>" (`internal/push/push.go:158`), so it needs nothing per kind.

**Do the two notifications need new kinds? Yes.**
- **The kind list.** `Kind` is a wire contract, additive only (`internal/workspace/model.go:218-231`).
- **Why not `doc.updated`.** The inbox would read "doc updated" (`web/src/utils/InboxUtility.tsx:8`, `:42-54`).
- **Why not `play.run_waiting`.** It means a trail is waiting. Its title is built as "<play label> needs your answer
  on <doc>" (`usecase.go:1377`), which would put "Clarify via AI" in front of the client.
- **What to add.** Two kinds, for example `doc.questions_asked` for the watchers and `doc.questions_answered` for the
  round's starter. Both use subject type `doc` and the doc's title as `subject_title`, so they group under the doc in
  the inbox (`InboxUtility.tsx:64-79`).
- **Cost.**
  - Two constants and two handlers subscribed in `server/cmd/subscriptions.go`.
  - On the web: `NotificationKind`, `kindLabels`, and a `docSummary` part.
  - On the phone: `native/src/models/Notification.tsx` and `NotificationRow.tsx:8-18`. Its `Record` type forces both
    entries.
  - An older app shows an empty label for an unknown kind (`NotificationRow.tsx:41`), which is harmless.
- **Sources.** "Questions asked" comes from the round-posted event. "Last answer" comes from the save that leaves the
  round with no pending row; the use-case knows that when it saves, and it emits an event that names the round's
  starter.

## Open for the owner

- Whether the client sees "(Recommended)" on an option and has it pre-picked, as the interview does
  (`models/InterviewAnswer.tsx:135-140`), or the recommendation stays for the developer only.
- Whether a question's why-line reads as "Why I'm asking:" (`InterviewQuestionForm.tsx:340`). It is first person but
  names no agent.
- The inbox label wording for the two new kinds, for example "questions for you" and "answers in".
