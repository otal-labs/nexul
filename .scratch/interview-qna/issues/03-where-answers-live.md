# 03: Where the answers live

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

Answers are stored per project and stay on the page. Decide:

- Their shape: one set per project, the follow-up questions and answers
  kept beside the template's, and who answered what.
- Whether answers are saved one at a time as the person steps through, so
  leaving halfway keeps progress, and whether a question can be skipped.
- The permission: `memories:write` like the memory itself, or something
  else; who sees them.
- Events and live push, so two people on the page see each other's answers.
- The MCP surface: the tool an agent uses to read and write answers,
  extending an existing tool before adding one.
- What happens to the existing interview threads and their messages in
  production once the Conversation section is gone.

## Answer

- One stored answer per question per project, in a new table in the
  memories domain: the question text, the picked values and free text,
  whether it came from the template or a follow-up, who answered, and when.
  A follow-up and its answer are written by the server when the answer
  reaches the run, not by the agent. A new numbered migration; nothing to
  backfill.
- Each "Next" saves its answer, so leaving halfway keeps progress; any
  answer can be changed later.
- The card has a Skip. Finishing starts the follow-up run even with skips,
  and the run treats a skipped question as a gap and asks it again with a
  recommendation from the code. Clearing an answer is the same as a skip.
- Reading answers needs `memories:read`, answering needs `memories:write`,
  both through project access; starting the run also needs the play
  permission it needs today.
- Saving an answer writes an outbox event with a catalog row and pushes it
  live to the project's audience. The event carries the question and the
  author, not the answer, like the template's event.
- MCP: no new tool. Reading the interview memory also returns the parsed
  template questions and the stored answers, which is how the follow-up run
  finds them; updating the interview memory also accepts answers.
- Existing interview threads in production stay untouched as hidden
  plumbing, readable through `message_list`, and go with the thread when it
  is removed.
- Deleting the interview memory keeps the answers; generating again
  rebuilds it from them. Deleting the project removes both.
