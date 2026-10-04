# 01: How the interview's machinery carries over to a doc

Type: research
Status: open
Blocked by: None — can start immediately

## Question

The interview already asks rounds of question cards, stores the answers
per round, and runs an agent that asks follow-ups. A clarification is the
same loop on a doc, except one run is one round and nobody answers while
the run is alive. Find out, with evidence from the code:

- Whether the interview's answer storage (`interview_answers`, its
  use-cases, events, live push, MCP surface) can be generalised to a doc
  target, or whether a doc-domain table beside it is cleaner, and what
  either costs.
- How a run can post a round of questions and end, instead of asking with
  the harness's question tool and sitting in `waiting`: an MCP tool the
  agent calls with the round, what the trail records, and how the card
  shows questions that came from no live run.
- What a doc play does to its doc today (the lock at start that stays
  after the run, the post into the doc thread), and what it takes to lock
  only while a run is working.
- Which parts of `QuestionCard` and the Interview page's checklist the doc
  page can reuse as they are, including the "something else" free text.
- What the phone app's doc view shows today, and what it has for
  question cards, if anything.
- How the inbox notifies watchers of a doc and one named person, and
  whether the two notifications need new kinds.
