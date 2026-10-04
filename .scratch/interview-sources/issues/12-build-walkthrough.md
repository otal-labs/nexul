# 12 — Walkthrough on Clutch Hub

**What to build:** An end-to-end run on nexul-box with a paired harness
after 07 to 11 land, on the Clutch Hub repositories. Create a phase 1
project on `Onik97/clutch-hub-backend` and a phase 2 project in a MgClutch
workspace. In phase 2: add phase 1 as a question source, a pasted
standards text and a doc as follow sources; press Draft answers and watch
drafts arrive; confirm some, leave one undrafted question to answer by
hand; finish and answer the follow-up round, which asks about phase 1's
code; check the memory. Add a "Phase 2 requirements" doc and see the
warning dot, redraft, accept one suggestion and dismiss another,
regenerate. Run the audit and turn the doc into tickets with "To tickets
via AI". A second browser sees sources and drafts arrive live; an agent
reads and writes sources over MCP; a restricted member sees "Not visible
to you". Findings become fix-up tickets here. Then move the effort to
shipped on the roadmap and delete this directory.

**Blocked by:** 07, 08, 09, 10, 11

**Status:** resolved

- [x] Every step above passes or has a fix-up ticket
- [x] Decide from the run whether GitHub issues need to be a source kind

## Answer

Walked 2026-10-04 on a local stack with a throwaway T3 Code server and
Codex, `clutch-hub-backend` as phase 1 and a fresh repo as phase 2.

- Sources (phase 1 under question, pasted .NET standards and the
  "Engineering standards" doc to follow, default stances right): passed.
- Draft answers: passed, 10 of 12 drafted, confirmed with Next, one
  answered by hand, one skipped. Fixed here: the empty interview memory a
  drafting run creates showed as "Memory ready" with Audit via AI, and a
  draft landing on the open card left it blank.
- Follow-up round: passed; it asked about phase 1 by file
  (`src/ClutchHub.Api/Endpoints/DiscordCallback`, `src/ClutchHub.Api/Domain`)
  and wrote the memory. Fixed here: during that run "Draft answers"
  showed as a running Draft interview with Stop, which stopped the
  interview.
- Phase 2 requirements: passed; warning dot, redraft with 3 suggested
  changes, one accepted, two dismissed, Regenerate asked about the
  dismissed one and updated the memory. Fixed here: the suggestions sat
  in a folded section; the first now opens. Ticket 13 for the doubled
  source name in "From" and the drafting line's count after a redraft.
- Audit via AI and To tickets via AI: passed; the audit doc linked from
  the page, 10 tickets in phase 2's backlog.
- Second browser: passed; a source and drafts arrived without a reload.
- MCP (the server's `/mcp` with a token for the dev user): passed;
  `memory_get` returns sources with text bodies and drafts,
  `memory_update` added and removed a source and refused `..`.
- Restricted member: passed, phase 1 reads "Project · Not visible to
  you". Fixed here: the questions never loaded for a restricted member,
  since the template read needed `memories:read` on the workspace; the
  page now reads it through the project.
- GitHub issues stay pasted text. The three open issues pasted in one
  text source in seconds, and a drafting run reads them like any text.
  A source kind would need the App's Issues permission on every
  installation for what a paste covers; revisit if a team keeps its
  requirements in issues and needs the change marker on them.
