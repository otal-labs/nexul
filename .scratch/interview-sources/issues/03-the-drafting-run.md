# 03: The drafting run

Type: grilling
Status: resolved
Blocked by: 01, 02

## Question

An agent reads the follow sources and writes a draft answer per template
question. Decide:

- Whether it is its own built-in play or a first step of the Interview
  play, and what starts it: a button on the page once sources exist, and
  whether adding the first source starts it on its own.
- What a draft holds (picked option, free text, the source and the
  passage it came from) and where it is stored: fields on the stored
  answers, or a table beside them. A draft never counts as an answer, and
  the follow-up run only starts once every initial question is answered
  or skipped, as today.
- What happens to questions no follow source speaks to: left blank, and
  the person answers them as now.
- How the run tells the page it is done, through the trail like the
  follow-up run.
- What it does with question sources: nothing at drafting time; they are
  read by the follow-up run.

## Answer

- **Its own built-in play**, "Draft interview" (key `interview-draft`,
  type interview), beside the Interview play, not a first step of it. A
  workspace that edited its Interview play keeps text that knows nothing
  about drafting, so folding drafting into that play would break it
  silently. The new play's instructions are an instance template like
  every built-in play's. `SeedDefaults` gives fresh workspaces five plays,
  and a new numbered migration adds it to every existing workspace.
- **Started by a person**: the page shows "Draft answers" once the project
  has at least one follow source. Adding a source never starts a run on
  its own, because a run takes time on someone's computer and they choose
  when. A project whose only sources are under question gets no button:
  there is nothing to draft from, and the follow-up run reads them.
- **It never asks.** The run reads the interview memory (questions,
  answers, and sources, per 01), reads every follow source, writes drafts,
  and ends. It does not use the question tool, so it never sits waiting.
  It runs on the project's hidden interview thread like the follow-up run,
  so the follow-up run can reuse its session.
- **What it drafts**: every template question (round 0) with no stored
  answer and no skip. Questions already answered are left alone here;
  redrafting them is 05.
- **A draft** holds the picked options and free text, shaped like an
  answer, plus the source ids it came from and one line saying where
  (`practices/testing.md, "Test error paths first"`), at most 500
  characters. A question no follow source speaks to gets no draft, and
  the person answers it on a plain card as now.
- **Storage**: a separate `interview_drafts` table in the memories domain
  (project, question, picks, text, sources, where-line, the trail that
  wrote it, when), unique on project and question, written in the same
  migration as the sources table. Drafts never go in `interview_answers`:
  a row there means answered, and the page's counts, the follow-up run,
  and the memory all read it that way. Confirming a draft is an ordinary
  save of the answer through the card. The draft row stays, so the card
  can keep showing where it came from, and a later drafting run replaces
  it. Deleting the project removes drafts; deleting the interview memory
  keeps them.
- **Write path**: the run writes drafts with `memory_update` on the
  interview memory, which gains a `drafts` field beside `answers`.
  `memory_get` returns them. Same permission as answers
  (`memories:write`), the same outbox event and live push pattern, and no
  new tool.
- **Done**: the page reads the run's state from its trail, the same as the
  follow-up run; drafts arrive one by one through live push while it
  works.
- **Question sources** are not read at drafting time.
