# Wayfinder map: an interview drafted from what a project already has

Charted 2026-10-04 with the owner. The interview as questions assumes a
project starts from nothing: blank answers, and an agent that only asks.
Most projects arrive with something already: a practices folder, a set of
ADRs, a standards doc, a predecessor project whose phase 1 is done but
whose code is not good enough for phase 2. The person points the Interview
page at that material, an agent drafts an answer to each of the template's
questions from it, and the person steps through the cards again confirming
or changing each draft. Material that arrives later redrafts only the
answers it touches.

## Destination

Interview sources built and merged: adding sources to a project's
interview with a follow or question stance, the drafting run, drafts on
the question cards that count only once confirmed, redrafting after a
source is added or changes, and an audit of a predecessor project measured
against the new project's interview memory. Walked end to end on Clutch
Hub in the MgClutch workspace. This map carries the build: once the
decision tickets are resolved, the build tickets graduate from the fog.

## Notes

- Decided while charting:
  - A source is pointed at, never copied in: paths in the project's
    checkout, the project's docs and memories, another project in the
    workspace (its checkout, its interview answers, its memories), or text
    the person pastes or a file they upload. There may be a lot or very
    little.
  - Each source has a stance. **Follow** is written material the team
    stands behind (a practices folder, ADRs, a standards doc): the agent
    drafts answers from it. **Question** is evidence of how something was
    done, not how it should be (a predecessor's code): the agent never
    drafts from it and instead asks about it in the follow-up rounds, with
    a recommended answer. Code defaults to question, written material to
    follow; the person can change either.
  - The template is never edited per project. What the agent produces is a
    draft answer per question: the option it would pick, free text, and
    the source it came from. A draft is not an answer: it shows as the
    recommended pick on the card, and confirming it is one Next. Nothing a
    person has not confirmed is recorded, the rule the interview already
    has.
  - A new or changed source redrafts only the answers it touches, and only
    those come back to the person. This is also how phase 2 requirements
    arriving later as a doc feed the interview.
  - A project that supersedes another is a new project with its own repo;
    the old project is one of its sources, under question. There is no
    project-to-project link beyond that.
  - The audit needs a yardstick, so it comes after the interview: the new
    project's interview memory is what the predecessor's code is measured
    against.
  - The interview's other rules stand: the follow-up run asks about skips,
    gaps, and contradictions a round at a time, and the memory is written
    from the full set of confirmed answers.
- Nexul's own `practices/` stays in the repo: it is the public standard for
  outside contributors and for agents that never touch Nexul, and CI lints
  against it. The feature does not care where a source lives.
- Technical tickets arrive as a decided answer; look and content tickets
  go to the owner. Plain language with the owner, no ticket numbers in
  questions.
- Web screens are judged at 768, 1024, and 1440px.
- Prototype tickets: `/prototype` inside `design-mode`. Grilling tickets:
  `/grilling` + `/domain-modeling`. Production data is live: storage
  changes are a new numbered forward-only migration.
- Grounding: `CONTEXT.md` (Interview, Interview memory, Interview source,
  Memory, Doc, Play, Trail), ADR 0115 (the interview is questions plus a
  follow-up run), ADR 0111 (agent prompt by reference). Code:
  `internal/memories/interview.go`,
  `internal/platform/storage/migrations/0061_interview_answers.sql`,
  `internal/plays/run.go` (`Runner.Answer`), `internal/plays/usecase.go`
  (the Interview play's instructions),
  `web/src/components/memory/ProjectInterview.tsx`,
  `web/src/components/play/QuestionCard.tsx`. `.scratch/doc-clarify/`
  reuses the same machinery for docs; check it before changing anything
  both depend on.

## Decisions so far

- [How a run reads another project and the sources it is given](issues/02-reading-another-project.md): one checkout per run today, found through the starter's project link for the other project; its docs, memories, and answers are already readable over MCP; a prompt-only source list is lost on resume; GitHub issues are not reachable.
- [What a source is and where it is kept](issues/01-what-a-source-is.md): path, doc, memory, whole project, or pasted text (uploads become text) with a required follow or question stance, in a new memories-domain table; adding a ref takes read on it, gone refs show as gone, read by the run through the interview memory tools.
- [The drafting run](issues/03-the-drafting-run.md): its own built-in play, "Draft interview", started by a person once a follow source exists; it never asks, drafts every unanswered template question into a separate drafts table through `memory_update`, and ends; a draft counts only once saved as an answer.

## Not yet specified

- The build tickets: source storage, the drafting run, drafts on the cards,
  redrafting, the audit, MCP, and the walkthrough on Clutch Hub (create the
  project in MgClutch, point its interview at its repo, follow the run
  through its trail). They graduate once the decision tickets resolve.
- What the follow-up run does differently with question sources than with
  the project's own checkout, which it already reads for contradictions:
  possibly nothing beyond reading more than one checkout.
- How the drafting copes with a very large source (a whole predecessor
  repo, a long practices folder) against one session's budget.
- Whether GitHub issues become a source kind, which needs the App's Issues
  permission on every installation; pasted text covers them until the
  walkthrough shows otherwise.

## Out of scope

- Bringing a repo's markdown into Nexul as docs or memories, so a team can
  retire the files: an import, its own effort. Here sources are pointed at,
  never copied.
- Moving Nexul's own `.scratch/` tracker and ADRs into Nexul tickets and
  docs: dogfooding, unrelated to the interview.
