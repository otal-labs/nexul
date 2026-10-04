# An interview source is pointed at, with a follow or question stance

Extends ADR 0115 and ADR 0122: an interview can start from what a project already has.

Most projects arrive with material: a practices folder, ADRs, a standards doc, or a predecessor whose phase 1 is done
but whose code is not good enough for phase 2. Drafting answers from all of it alike would write a predecessor's
weaknesses into the new project's rules, and copying it in would leave two versions to drift.

Decision: a project's interview keeps a list of sources it points at, each with a stance.

- **Pointed at, never copied.** A source is a path in the project's checkout, a doc, a memory, another project as a
  whole (its checkout, memories, and interview answers), or pasted text; an uploaded text file becomes pasted text.
  Refs are resolved when read, so a deleted doc, memory, or project shows as gone, and drafts made from it stay drafts.
  Importing a repository's markdown into Nexul is its own concern, not this one.
- **Follow or question.** Follow is material the team stands behind, and the drafting run drafts answers from it.
  Question is how something was done, not how it should be, such as a predecessor's code: it is never drafted from, and
  the follow-up run asks about it with a recommended answer. The page suggests follow for written material and question
  for code and other projects; the person can change either.
- **No reading around access.** Adding a doc, memory, or project source takes read on it, so nobody gets something
  drafted into answers they could not read, and a reader without access sees the kind and "Not visible to you", never
  the name. A run reads as its starter, so it reaches only what the starter can.
- **A superseding project is a new project.** Phase 2 gets its own project and repository with phase 1 as a question
  source; there is no project-to-project link beyond that.
- **The audit measures against the interview.** "Audit via AI" reads the question sources' code, or the project's own
  checkout when there are none, and writes one dated doc of findings under the interview memory's headings: carry
  over, rebuild, or avoid for a predecessor; keeps or fix for its own code. "To tickets via AI" on that doc turns it
  into work, so the audit files nothing itself.

The cost: refs resolved on every read instead of cached labels, a run that cannot reach another project's code unless
its starter has that project checked out on the same computer, and GitHub issues reaching the interview only as pasted
text.

Rejected: copying sources into Nexul, which drifts from the repository; one stance for everything, which drafts from
code nobody stands behind; editing the template's hints per project, which changes them for every project sharing the
template; a GitHub issues source kind, which would ask every installation for the Issues permission to save a paste;
and an audit that files tickets directly, which skips a person reading the findings first.

Decided 2026-10-04.
