# 01: How the interview's machinery carries over to a doc

Type: research
Status: resolved
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

## Answer

Findings in [research/how-the-interview-carries-over.md](../research/how-the-interview-carries-over.md).

- **Storage.** A doc needs its own tables in the docs domain beside the interview's. Sharing `interview_answers`
  would need a branch on the target in every permission check, event, and live frame to keep a doc's questions away
  from people who can read memories but not the doc.
- **Posting a round.** A run posts its round through `doc_update` and then ends normally. `doc_get` returns the
  rounds. There are 107 of 108 tools, so no new tool. The trail shows the call as an ordinary step and ends done. The
  page shows stored questions the way the Interview page already shows stored follow-ups, saving one answer at a time.
- **Locking.** Today a doc play locks its doc at start for good, and that lock would refuse the closing write. The
  runner should lock at start, record that on the round, and unlock at every end. The closing write should be
  accepted only from the running round's starter.
- **Web reuse.** The option rows with "Something else…", the round sections, and the question rows carry over once
  they move to a shared folder. `QuestionCard` and the interview's round labels carry Agent wording, so they don't.
- **Phone.** The doc screen is read-only and has nothing for questions yet.
- **Notifications.** Both need new kinds. The current ones would read "doc updated" or show the play's name to the
  client.
