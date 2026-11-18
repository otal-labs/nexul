# A project starts with default memories, and setup installs no third-party skills

Memories are the project's skills. A memory already has a skill's shape, a name, a one-line when-to-use that decides
when it is read, and a markdown body, and unlike a skill file it lives in Nexul, where the team reads and edits it, and
reaches every agent on every computer. So:

- **Five ship**, one per step of the work: Writing a doc, Writing tickets, Building a ticket, Reviewing a change, and
  Finishing a ticket, the last always included and in Footer (ADR 0112). Each is a `SKILL.md` embedded in the binary:
  frontmatter with `name` (the title), `description` (the when-to-use line) and the optional `always_included` and
  `footer`, then the body.
- **They are instance templates** of a new kind, `memory`, living only at the instance (ADR 0103), addable like the
  template roles (ADR 0149): the instance edits the five, adds its own, and deletes those; the five reset. A template is
  checked before it is stored.
- **A new project copies them once**, in the transaction that creates it, as ordinary memories at version 1, and never
  rewrites them, the way its ticket types' body templates are copied. A template that no longer parses is skipped and
  logged; it never stops a project from being created.
- **An existing project gets them only if it had no memories of its own**: none beside the interview memory and the
  decisions log, or only the untouched placeholder an older release seeded, "Working in this project", which goes. A
  project with any other memory is left exactly as it is.
- **Computer setup installs only Nexul's own skills** (`nexul-memory`, `nexul-triage`), and checks only those. It no
  longer runs `npx skills add mattpocock/skills`, which needed npm and the network at setup time, collided with skills a
  person already had under the same names, was written for a local repository and `gh` issues rather than tickets, docs
  and memories, and drifted per computer where nobody in Nexul could see it. Nothing removes a set already installed.

The cost: a team's edit to a default reaches only projects made afterwards, and an existing project that kept the
placeholder beside its own memories never gets the defaults; a team that wants them clones them in.

Rejected: keeping the third-party install beside the memories, which leaves two sources of process that disagree;
writing the memories out as skill files on each computer, two copies that drift; projects following the instance's
memories live, which would undo a team's edits to its own; and seeding every existing project, where a second
always-included footer would contradict the one several teams already wrote.

Amends ADR 0063 (setup installs Nexul's skills, not a default set) and ADR 0079 (a project starts with the default
memories, not a starter memory). Decided 2026-11-18.
