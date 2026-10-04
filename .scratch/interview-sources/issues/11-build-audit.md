# 11 — The audit

**What to build:** The "Audit via AI" built-in play from 06. Builtin key
`audit`, type interview, its instructions as code default and instance
template; `SeedDefaults` seeds it and a new numbered migration adds it to
every existing workspace. The runner names `project` sources under
question with their checkout paths as in 08, and says to audit the
project's own checkout when there are none. The instructions: read the
interview memory and its answers; read the audited code; write one doc
in the project's Main folder with `doc_create`, titled "Audit of <what>,
<date>", grouped under the memory's headings, one line per finding with
its verdict (Carry over / Rebuild / Avoid for a predecessor, Keeps / Fix
for own code), `path:line`, and the rule; check the question-source
answers first; never ask; end with a one-line summary naming the doc.
The page: "Audit via AI" in the memory column's header once the memory
exists, the run state with the trail icons, and a link to the doc when
the run finishes.

**Blocked by:** 07

**Status:** ready-for-agent

- [ ] A run on a fake harness writes the doc, for a predecessor and for
      own code
- [ ] Seed and migration add the play once; an edited workspace keeps its
      plays
- [ ] The button, run state, and doc link verified at 768, 1024, 1440px
- [ ] `CONTEXT.md` gains Audit if it is a term the product now uses
