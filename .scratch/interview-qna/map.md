# Wayfinder map: the interview as questions

Charted 2026-10-03 with the owner. The interview stops being a
conversation in a thread. The Interview page steps a person through the
workspace's Interview template as a list of question cards, with no agent
involved; once the last one is answered, a play run reads the answers and
the codebase, asks follow-ups about the gaps a batch at a time, and writes
the interview memory.

## Destination

The new interview built and merged: questions on the Interview page,
stored answers, the follow-up run, the generated memory, and the old
conversation, its thread section, and "Start from the template" gone.
This map carries the build: once the decision tickets are resolved, the
build tickets graduate from the fog and are worked here too.

## Notes

- Decided while charting:
  - The question cards replace the conversation; there is one way in.
  - The template is the list of questions, each with a hint and options
    "to give an idea". It is never copied into the memory any more, and it
    no longer decides the memory's headings. Workspace and instance layers,
    reset, and clone keep working as they do.
  - Answering the template's questions involves no agent. The follow-up run
    starts once all are answered, reads the answers and the checkout, and
    asks about gaps and about anything the code contradicts, a batch at a
    time with a recommended answer each, until nothing is left. It never
    records something from the code the person has not confirmed. There is
    no "should I scan the codebase?" question.
  - The follow-ups show in the same card on the Interview page ("Follow-up
    1 of 3"), not in a thread. The Conversation section goes; the trail
    stays as the record of a run.
  - Answers are stored per project and stay on the page. A re-run means
    changing the answers that changed and generating again; the memory is
    rewritten from the full set of answers.
  - The run is given the current interview memory every time and keeps any
    rule no answer contradicts, so hand edits and the memories projects
    already have survive. Existing projects keep their memory as it is and
    start with blank questions; nothing is converted.
  - "Start from the template" is dropped.
- Technical tickets arrive as a decided answer for a yes or no; look and
  content tickets go to the owner. Plain language with the owner, no
  ticket numbers in questions.
- Web screens are judged at 768, 1024, and 1440px.
- Prototype tickets: `/prototype` inside `design-mode`. Grilling tickets:
  `/grilling` + `/domain-modeling`. Production data is live: storage
  changes are a new numbered forward-only migration.
- Grounding: `CONTEXT.md` (Interview, Interview memory, Instance template,
  Play, Trail), ADR 0065 (the interview memory in every turn), ADR 0103
  (instance templates). Code: `internal/memories/interview.go` (template
  default, `CreateInterview`), `internal/plays/usecase.go`
  (`interviewInstructions`), `web/src/components/memory/ProjectInterview.tsx`
  (the page), `web/src/components/play/QuestionCard.tsx` (the card, which
  already takes a list of questions with options and free text).

## Decisions so far

- [How a question is written in the template](issues/01-how-a-question-is-written.md): `##` heading per question, hint below, `- ` single and `- [ ]` multi options, no recommended option, matched to answers by text, parsed on the server, 8,000 cap moves off the template.
- [How the follow-up run asks on the Interview page](issues/04-follow-ups-on-the-page.md): the page shows the run's live question with the existing trail question body, answers go through the run route and are also written to the stored answers, and the thread stays as hidden plumbing for now.

## Not yet specified

- The build tickets: storage and migration, the template parser, the
  page, the follow-up run's instructions, removing the conversation and
  "Start from the template", the MCP surface, and the `CONTEXT.md` and ADR
  updates. They graduate once the decision tickets are resolved.
- Removing the hidden interview thread once the T3 protocol 2 changes
  (`.scratch/t3-orchestrator-v2/`) have merged: it still carries the
  session id, and its messages pile up unseen until then.
- Whether the Interview template's settings editor needs anything beyond
  the markdown field it has, such as a preview of the cards.

## Out of scope

- Keeping the conversation-style interview alongside the questions.
- An agent drafting answers from the code before the person answers.
