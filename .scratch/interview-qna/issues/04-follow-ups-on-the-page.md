# 04: How the follow-up run asks on the Interview page

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

The follow-up run is a play run on a paired computer. Its questions must
show in the question card on the Interview page, not in a thread. Find out
how a play run's questions reach the web today and what it takes to move
them:

- The path from the harness's question tool to `QuestionCard` today
  (the trail, the interview thread, `TrailQuestionCard`), and how an answer
  travels back to the waiting run.
- Whether a play run needs a conversation at all, or can run against the
  interview target with no thread.
- How the run is handed the stored answers and the current memory: in the
  prompt, or by name through an MCP tool, per ADR 0111.
- What the page can show while the run works, when it stops, fails, or the
  computer goes offline, and how a run is resumed.
- Differences between the T3 protocol 1 and protocol 2 clients that matter
  here (`.scratch/t3-orchestrator-v2/`).

## Answer

Findings with evidence: `research/follow-ups-on-the-page.md`.

Show the running interview's current question on the Interview page with
`TrailQuestionBody`, which already answers through the run route, and drop
the Conversation section; the question and answer path and the pipeline
stay as they are, so it works on both T3 protocols and stays clear of the
protocol 2 work in `internal/agent/pipeline.go`. A run keeps only its
latest question, so `Runner.Answer` also writes each follow-up and its
answer into the stored answers when the run is an interview, which keeps
earlier follow-ups and the "Follow-up N" count. Per ADR 0111 the run is
pointed at the stored answers through an existing MCP tool, not given them
in the prompt. The interview thread stays as hidden plumbing (it holds the
T3 session id) until the protocol 2 changes merge; removing it touches
about ten places in the pipeline.
