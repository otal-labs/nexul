# 10 — Write the spec

**Type:** task
**Status:** resolved
**Blocked by:** 01, 02, 03, 04, 05, 06, 07, 08, 09

## Question

Assemble `.scratch/plays/spec.md` from the map's Decisions so far, in the
repo's spec shape (problem statement, solution, user stories, implementation
decisions), and add the settled terms to `CONTEXT.md`: Play, Play run, Doc
thread, and the sharpened Memory entry. Record as ADRs only what is hard to
reverse, surprising, and a real trade-off: the candidates are
"a play runs on the clicking user's harness" (an extension of ADR 0029),
"memories are their own per-project entity, never docs", and the amendment
to ADR 0010 allowing domain-declared permission verbs (`plays:run`,
`memories:clone`, `docs:thread`). Then hand off to
`/to-tickets`.

## Answer

Written 2026-09-17: `.scratch/plays/spec.md`, three ADRs (0055 play on the
user's harness, 0056 memories as their own entity, 0057 domain-declared
permission verbs, amending 0010), and the glossary entries Play, Trail, Doc
thread, Memory, Permission in `CONTEXT.md`. Next step: `/to-tickets` on the
spec.
