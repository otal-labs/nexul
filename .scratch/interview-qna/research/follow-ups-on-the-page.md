# Research: follow-up questions on the Interview page

Answers ticket `04-follow-ups-on-the-page.md`. Read from source on 2026-10-03, at `fe79e3bf`. Every path is
relative to the repo root.

## 1. How a question gets from the harness to `QuestionCard`, and how the answer gets back

**Harness to server (protocol 1, `internal/t3client`)**
- T3 reports a question as an info-tone `user-input.requested` activity (`internal/t3client/thread.go:515`,
  `:785-786`).
- `questionUpdate` maps it to `harness.Question{RequestID, Questions[]}`. A question with no id is keyed by
  its text (`thread.go:803-821`), and a payload with no request id is dropped (`:805-808`).
- Separately, the `AskUserQuestion` tool call becomes an `ActivityQuestion` step on the trail
  (`thread.go:523`, `:540-542`).
- `Harness.forward` passes the question on as `harness.Update{Question}` (`internal/t3client/harness.go:298-299`).
  The seam types are in `internal/harness/harness.go:204-235`.

**Agent pipeline (`internal/agent/pipeline.go`)**
- `drainTurn` posts the question into the conversation as the Agent's own message: a fenced
  ```` ```nexul-question ```` JSON block (`:621-626`, `:814-824`). It then calls `obs.OnQuestion`.
- The turn is not terminal yet. The harness keeps it open (`harness.go:276`, `pipeline.go:88-89`).

**Plays and the trail (`internal/plays/run.go`)**
- `trailObserver.OnQuestion` stops the silence timer and sets the trail to `waiting`.
- It stores the question as `trail.Question`, writes the `play.run_waiting` outbox event, and publishes a
  `play.run` live frame (`:686-696`, `:834-848`).
- The frame carries `state` and `question` (`internal/plays/events.go:81-91`). It reaches anyone with
  `memories:read` on the project (`server/cmd/live_audience.go:345-346`).

**Web**
- `useLiveEvents` applies the frame to `usePlayRunStore` (`web/src/hooks/useLiveEvents.tsx:275-283`).
- `useLiveTrailQuestion` prefers the live frame's question over the fetched trail's
  (`web/src/hooks/TrailHooks.tsx:155-156`).
- **In a thread:** `MessageRow` → `AgentMessageBody` parses the fence (`web/src/models/Question.tsx:45`).
  - If the message has a trail block, it renders `TrailQuestionBody`, which is answered on the trail
    (`web/src/components/chat/MessageRow.tsx:78-86`).
  - Otherwise it renders `ChatQuestionCard`, which is answered on the conversation
    (`web/src/components/chat/ChatQuestionCard.tsx:13-19`).
- **In the trail dialog:** `TrailDetailBody` → `TrailTranscript` → `TrailTranscriptSegment` → `TrailQuestionCard`
  (`web/src/components/play/TrailTranscriptSegment.tsx:17-20`).
- Both paths end in `QuestionCard` (`web/src/components/play/QuestionCard.tsx:23`). It steps through every
  item of one question request.

**The answer going back (plays path)**
1. `TrailQuestionBody` → `useAnswerTrail` sends `POST /api/plays/runs/{trailID}/answer`
   (`web/src/components/play/TrailQuestionCard.tsx:11-21`, `web/src/hooks/TrailHooks.tsx:133-137`,
   `internal/plays/run_handler.go:41`, `:78-84`).
2. `Runner.Answer` checks that the caller is the starter or holds `plays:write`, and that the trail is
   `waiting`. It then posts the answer summary into the thread (`run.go:356-379`).
3. With a live observer: `o.answer` → `agent.Service.Answer(conversationID, requestID)`
   (`run.go:699-718`) → the active turn found by conversation id (`pipeline.go:827-835`) →
   `t3client` `thread.user-input.respond` on a fresh connection (`harness.go:324-337`,
   `thread.go:247-260`). The trail goes back to `running`.
4. With no live turn (the harness closed it, or the server restarted), the answer resumes the run under
   the same trail. It starts a fresh turn on the same session, with the answer as the request and the
   play's memories named again (`run.go:385-402`, `:510-526`).

The chat-only route (`POST /api/agent/conversations/{id}/answer` → `AnswerFromChat`, `pipeline.go:839-864`)
handles plain `@Agent` questions, not plays.

**Limit:** a trail stores only the latest question and its answer (`internal/plays/model.go:175-176`,
`web/src/utils/TrailTranscriptUtility.tsx:38`). Earlier batches are kept only as thread messages. The page
cannot rebuild "Follow-up 1 of 3" or earlier answers from the trail alone; the answers store from ticket
03 has to hold them.

## 2. Does a play run need a conversation?

Today it does. A play always opens one, and the pipeline is keyed on it.
- `launch` always calls `openThread`. For an interview that is `GetOrCreateInterviewThread`
  (`run.go:309-313`, `:588-595`). It also posts the "Started" message there (`:317-322`).
- `RunTurn` begins with `GetConversation` and fails the turn without one (`pipeline.go:315-320`).
- The harness session is the conversation's `ThreadID`, both read (`:357`) and stored (`SetThread`,
  `:564-576`).
- The prompt cursor is the conversation's (`MessagesSince` and `MarkSynced`, `:430`, `:497-504`). A play
  drops the history anyway (`:443-445`).
- The question, the reply, and failure lines are posted as conversation messages (`:623`, `:672-698`).
- Live stream frames are keyed by conversation id (`:564-568`, `:579-584`).
- The active-turn map, `Answer`, and `Interrupt` are keyed by conversation id (`:204`, `:827-877`). The
  trail calls them with `trail.ConversationID` (`run.go:705`, `:764`).

What already works without a thread:
- `Runner.note` skips the thread when `ConversationID == ""` (`run.go:851-859`).
- The trail keeps its own `HarnessSessionID` (`model.go:166`).
- The pairing setup turn drives `harness.Client.StartTurn` with no conversation at all
  (`internal/pairing/setup_turn.go:408-437`). It treats a question as a failure, so it is a precedent for
  the shape only.

What ties an interview run to the interview thread: `openThread` (`run.go:588-595`), the
`ConversationID` everything above is keyed by, and the interview session title (`pipeline.go:382-384`).
ADR 0055 says a play "posts into the target's thread so the run reads as a conversation"
(`docs/adr/0055-…md:11`), and `CONTEXT.md` (Play, Interview) says the same.

## 3. Handing the run the stored answers and the current memory

ADR 0111 sets the rule: a prompt "names each piece of context and the tool that reads it, and carries none
of it". It rejects inlining because a body "goes stale within a long session". ADR 0105 names each memory
by id for `memory_get`.
- **The current memory is already handled.** `memoriesToRead` names the interview memory first
  (`run.go:496-507`, `:551-580`), and the current instructions start with `memory_create` kind `interview`
  (`internal/plays/usecase.go:35`).
- **The answers should be named the same way:** one line in the interview block (`run.go:913-921`) saying
  where to read them, served by an existing MCP tool.
  - The tool budget is 105 (`internal/mcp/surface_test.go:20`), and `practices/mcp.md:138-146` says to
    extend a tool before adding one.
  - Candidates: `memory_create` kind `interview` (already the run's first call) or `project_get`.
  - Ticket 03 picks one.
- **The one inlined fact today** is `tests_location`, a single word placed in the interview block
  (`run.go:913-921`). That is a reasonable exception for one word. A full set of answers is not.
- **Cost:** one more tool call at the start. A harness without Nexul's MCP gets no answers, but it could
  not write the memory anyway.

## 4. Run state the page can show, and how a run resumes

**States and where they come from**
- Trail states are `starting`, `running`, `waiting`, `done`, `failed` and `interrupted`
  (`model.go:93-104`).
- The live `play.run` frame carries the state, the latest step, the question, `ended_at` and `last_error`
  (`events.go:81-91`). The web reads them with `useLiveTrailState`, `useLiveTrailActivity`,
  `useLiveTrailQuestion` and `useActiveTrail` (`TrailHooks.tsx:149-156`, `:190-194`).
- `PlayButton` already shows "Waiting for your answer" and Stop (`web/src/components/play/PlayButton.tsx:27-80`).

**Computer offline**
- **Before starting:** `useHarnessReadiness` returns the state `offline` with the copy "Your harness is
  offline" (`web/src/hooks/PairingHooks.tsx:218`, `web/src/models/Pairing.tsx:384-395`). The button is
  disabled with that reason.
- **At the press:** a harness refusal is saved as a `failed` trail with `failure_reason`
  (`run.go:283-291`, `internal/pairing/usecase.go:537`).
- **Mid-run:** `t3client` emits a muted "Reconnecting to T3 Code…" note step and redials for up to 5 minutes
  before failing the turn (`internal/t3client/harness.go:196-251`).
- **Silence:** 15 minutes with nothing from the harness fails the run (`run.go:31`, `:743-753`).
- **While waiting:** the silence clock is paused, so a run can wait indefinitely (`run.go:677-696`).
  - An answer while the computer is offline returns an error and the trail stays `waiting`
    (`run.go:705-712`).
  - After a server restart, the answer resumes the run, and an offline computer then fails it at
    target resolution (`pipeline.go:327-331`).

**Stopping and resuming**
- **Stop:** `POST …/stop` interrupts the run, from `waiting` too. The trail ends `interrupted`
  (`run.go:756-806`).
- **No resume after a stop.** `interrupted`, `failed` and `done` are terminal. The way back is a new press,
  which makes a new trail.
- **Re-runs reuse the session.** A new press on the same thread reuses the T3 session through
  `conv.ThreadID` and sends the incremental play part (ADR 0106 as amended by 0111). Without a thread,
  every press would start a fresh session.
- **Restart while running:** a trail left `running` by a server restart stays that way until someone
  presses Stop, which closes it (`run.go:797-806`). I found no sweep at startup.
- **Waiting runs survive restarts:** they resume through the answer (`run.go:385-402`).

## 5. Protocol 1 vs protocol 2 for questions

`internal/t3clientv2` does not exist yet; the effort's tickets are `ready-for-agent`. This section is from
`.scratch/t3-orchestrator-v2/`.

| | Protocol 1 (`t3client`) | Protocol 2 (planned) |
|---|---|---|
| Question arrives as | `user-input.requested` activity (`thread.go:515`) | `user_input_request` turn item, status `waiting`; a live question leaves the run `running` (`research/protocol-2-wire.md:444`) |
| Answer command | `thread.user-input.respond` (`thread.go:247-260`) | `runtime-request.respond`, answers keyed by question id (`protocol-2-wire.md:152-164`) |
| Question ids | id, else the question text (`thread.go:815-818`) | Claude: the question text; Codex async: `"0"`, `"1"`… (`protocol-2-wire.md:180-182`) |
| Async ("message") questions | none | Each required question needs a non-empty string; multi-select is joined with ", ". The turn ends with Terminal at `waiting`, and the answer starts an `async-answer:<requestId>` run (`protocol-2-wire.md:168-172`; issue 08 lines 10-12) |
| Answer after the watch is gone | sent as a new turn with the answer as text (`run.go:385-402`) | `TurnPrompts.Answer *PendingAnswer`, sent as `runtime-request.respond`, never as a new message, or it queues behind the live question forever (`spec.md:163-168`; issue 09) |
| Handed-off agent asks | n/a | Shows only as a step plus the note "a handed-off agent is waiting for an answer in T3 Code". It cannot be answered from Nexul (`spec.md:215-216`) |
| Fields the seam drops | n/a | `allowCustomAnswer` and `required` (`protocol-2-wire.md:403-415`). `harness.QuestionItem` has neither (`harness.go:212-218`), so the card always allows typed text |

What this means for the interview:
- Build only on `harness.Question` and `trail.Question`, which both clients map into. Then the page does
  not care which protocol a computer speaks.
- A message-mode question already parks a play correctly. `OnFinished` keeps the trail `waiting` when the
  turn ends under an unanswered question (`run.go:728-731`).
- The follow-up instructions should tell the agent to ask questions itself, never through a handed-off
  agent.
- The current instructions demand "one question at a time … never a batch" (`usecase.go:37`). That has to
  flip to batches. One question request with several items already renders as steps in `QuestionCard`
  (`QuestionCard.tsx:22-44`).

## 6. Options for putting the follow-ups on the Interview page

**A. Render the active interview trail's question on the page, and keep the thread as hidden plumbing.**

Changes:
- `ProjectInterview` drops `InterviewThreadSection` (`web/src/components/memory/ProjectInterview.tsx:37`).
- It renders `TrailQuestionBody` for `useActiveTrail("interview", project.id)`. That component already
  answers through the trail route and is driven by live frames.
- `interviewInstructions` is rewritten: batches, read the stored answers, no "scan the codebase?"
  question.
- `Runner.Answer` writes each follow-up question and answer into the answers store when the target is an
  interview. This goes through a seam on the consumer side (ADR 0017), so the record does not depend on
  the agent.

Trade-offs:
- No change to the pipeline or the transport, and nothing that collides with the protocol-2 tickets.
- Both protocols work as they are.
- Re-runs still reuse the T3 session.
- The interview thread keeps collecting "Started", question JSON, answers and replies where no page shows
  them. They stay reachable through MCP `message_list`/`message_post` (`internal/chat/mcp.go:336-356`).
- `CONTEXT.md` (Interview, Play) and ADR 0055 need a line saying the interview's thread is plumbing,
  not a conversation.

**B. Option A, plus interview runs with no conversation.**

Changes:
- `TurnRequest` takes no conversation.
- The session id lives on the trail.
- Question and reply posts and the cursor are skipped.
- The active map is keyed by turn.

Trade-offs:
- Cleanest result: no hidden thread, and nothing to migrate later.
- It touches around 10 places in `pipeline.go` (§2) that protocol-2 tickets 09-11 are rewriting now,
  including rekeying the active map per turn (`spec.md:189-190`). Expect merge pain unless it lands after
  them.
- Every run starts a fresh T3 session.

**C. Questions as data: the agent writes the follow-ups to the answers store through MCP and ends its turn.
The page asks them with no run live, and submitting starts the next run.**

Trade-offs:
- No waiting state, so it survives offline computers, restarts and protocol quirks.
- Answers are persisted by construction.
- Every batch is a cold run: more latency, one trail per batch, and the harness question tool goes unused.
- It needs an MCP write path for questions, which counts against the tool budget.
- It is the furthest from how plays work today.

**Recommendation: A.** It is the smallest change that puts follow-ups in the same card on the page. It uses
the trail path that already works on both protocols and survives restarts. The server-side write in
`Runner.Answer` keeps "Follow-up N" and earlier answers on the page even though the trail keeps only the
latest question. Revisit B only once the protocol-2 pipeline changes have merged and the per-turn active
map makes a conversation-less turn cheap. Ticket 03 should still decide what happens to the existing
interview threads in production.
