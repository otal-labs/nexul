# The starting templates carry one flow from interview to merge

Amends ADR 0055, ADR 0103, ADR 0112 and ADR 0115.

The texts every workspace and project starts from (the built-in plays' instructions, the Interview template, the
ticket types' body templates and the Footer of every agent prompt) were long and missed what teams needed, so every
project wrote the same rules again by hand in memories: merge when the checks pass and move the ticket; a run is not
finished while its ticket sits in progress; tickets cut from a doc had no way to verify them and no blockers; the
interview asked twelve questions and then overruled the answers in follow-up rounds; agents polled long commands in a
loop.

Decision: the defaults describe one flow. A project is interviewed; a doc's gaps are asked and written in; a doc
becomes tickets with acceptance criteria, how to verify each, and their blockers; a ticket is built, reviewed, merged
and moved.

- **A play carries its own mechanics and done condition.** Each built-in play works in a project that deleted every
  default memory: it names the tools, the order and when the run is finished. How to write a good doc, ticket or
  review is craft, and stays in memories.
- **To tickets asks nothing.** A product decision the doc leaves open, or a choice the project's rules reserve for a
  person, becomes a ticket of its own whose body opens with "Needs a person: ...", and it blocks the tickets that wait
  on it. A build run stops at a ticket that opens that way, or at a blocker neither merged nor done.
- **A merge needs permission and an independent review.** An agent merges its own pull request only when the
  interview says it may and a review by someone who did not write the code found nothing that blocks. An empty or
  skipped answer to what an agent may do without asking means it asks first.
- **Fix with AI finishes without a memory.** It concludes as the memories named for the end of the run say; with none
  it leaves the pull request open and moves the ticket to the first review column. It continues a ticket that failed
  testing from the failed report.
- **The interview asks seven questions** (what the project is, which tests a change needs, how a change reaches the
  main branch, what an agent may do without asking, the rules no linter checks, its user interfaces, its words), and
  its follow-ups ask only for decisions, at most four a round and two rounds a run. An answer stands.
- **Ticket templates** gain "How to verify" and "Out of scope"; the bug template ends with "Screenshots".
- **A migration carries the new text to every copy nobody edited**, as 0072 did for one play: a play, a ticket type,
  an instance template or a workspace's Interview template still equal to the previous default gets the new one or
  follows it again, and a play's history gains the new text as its newest version, the seeded default.

The trade-offs: a workspace that copied the old text into its own words keeps it, since only an exact match moves. A
workspace following the instance's Interview template sees new questions; answers to the old ones stay stored and
still count in the follow-up run. Fix with AI now moves a card where it used to leave it, so a team that relied on the
card staying put writes a finishing memory.

Rejected: putting the flow in default memories alone, which leaves a play broken in a project that deleted them;
letting To tickets ask its questions in the run, which turns a batch job into a conversation nobody is watching; and
rewriting every copy, edited or not, which undoes a team's own edits (ADR 0103).

Decided 2026-11-18.
