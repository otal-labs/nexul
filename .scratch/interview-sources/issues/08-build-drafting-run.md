# 08 — The drafting run

**What to build:** The "Draft interview" built-in play from 03. Builtin key
`interview-draft`, type interview, label "Draft interview", its
instructions as code default and instance template. `SeedDefaults` seeds
five plays; a new numbered migration adds it to every existing workspace.
The runner gives a drafting run the same interview block as the follow-up
run, plus, for each `project` source, that project's id and the checkout
path from the starter's project link for it and the computer's T3 project
list (02); a project with no link on that computer is named with "no
checkout on this computer", and the run uses only its memories and
answers. The instructions: read the interview memory (questions, answers,
sources, drafts); read every follow source; read large sources
selectively (tables of contents, headings, then the passages a question
needs); for every template question with no answer or a skip, write a
draft when a follow source speaks to it; for an answered question, write a
draft only where the sources now disagree with the stored answer (05);
each draft with its source ids and one where-line under 500 characters;
never ask; never draft from a question source; write drafts with
`memory_update` as they are found, then end with a one-line summary.

**Blocked by:** 07

**Status:** ready-for-agent

- [ ] A run on a fake harness: drafts land, an equal draft is dropped, a
      question source is not read for drafts
- [ ] Seed and migration: fresh workspaces get five plays, existing ones
      gain the new play once, an edited Interview play is untouched
- [ ] `CONTEXT.md` (Interview, Interview source) and an ADR recording
      drafting as its own play and drafts kept apart from answers
