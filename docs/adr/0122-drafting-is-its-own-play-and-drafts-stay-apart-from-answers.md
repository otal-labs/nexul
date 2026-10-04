# Drafting is its own play, and drafts stay apart from answers

Extends ADR 0115: an interview can start from its sources instead of blank questions.

A project's interview can be pointed at follow sources, the material the team stands behind, so an agent can draft the
Interview template's answers from them for a person to confirm. That drafting had to fit beside a follow-up run that
asks questions and a store where a row means answered.

Decision: drafting is a built-in play of its own, "Draft interview" (key `interview-draft`, type interview), and its
drafts live in their own table, never in the answers.

- **Its own play, not a step of the Interview play.** A workspace that edited its Interview play keeps text that knows
  nothing about drafting, so folding drafting into it would break those workspaces silently. The new play's
  instructions are an instance template like every built-in play's; fresh workspaces are seeded with it and a
  migration adds it to every existing one. It runs on the project's interview like the Interview play, in the same
  hidden thread, one run at a time on the project, and it never asks.
- **Drafts are not answers.** A row in `interview_answers` means answered, and the page's counts, the follow-up run,
  and the memory all read it that way. Drafts sit in `interview_drafts`, one per question, written by the run through
  `memory_update` with its trail id, which the run's prompt names. A draft equal to the stored answer is dropped, and a
  draft on an answered question is a suggested change the person accepts or dismisses; the answer is never
  overwritten.
- **Stale suggestions clear when a run starts.** As a drafting run starts, the server deletes the project's drafts on
  questions that already have an answer, each as a dismissed draft, so only the suggestions the new run drafts again
  come back. Drafts on unanswered or skipped questions stay for the run to replace.
- **Another project is named, not copied.** For each project source the run's prompt names the project and its
  checkout path when the starter's project link for it is on the run's computer and that computer still lists the
  T3 project; otherwise it says there is no checkout there and the run reads only that project's memories and
  answers. Nexul never clones a checkout onto a paired computer.

The cost: a second interview play in every workspace that has to be kept apart from the Interview play wherever the
page picks one, and a drafting run that reads every follow source each time, which costs reading time but cannot
miss a question a change quietly affected. Clearing suggestions at the start means a dismissed suggestion can come
back from a later run while a source still disagrees.

Rejected: drafting as the Interview play's first step, which breaks edited plays; drafts as flagged rows among the
answers, which every reader of the answers would have to filter; and keeping old suggestions until the run replaces
them, which leaves a suggestion the sources no longer support on the page.

Decided 2026-10-04.
