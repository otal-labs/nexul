# A document belongs to exactly one project, and documents relate only through intentional `@` mentions

Docs were originally grouped into **Collections**, a folder-like taxonomy
kept deliberately workspace-level so one document could span several
projects. It was retired on 2026-08-19: a concrete example — a "Backend"
Collection sitting next to, but structurally unlinked from, a "Backend"
project — showed two sidebar entries with the same name and no relationship
between them, which read as confusing rather than useful. `Doc.ProjectID` is
now mandatory and single, exactly the rule `Ticket.ProjectID` already used,
so both content types hang off a Project instead of two taxonomies a user
has to reconcile. A project's docs are a flat list; no sub-grouping replaced
Collections. The accepted cost is that a genuinely cross-cutting document
must pick a primary project or exist twice.

Relationships between documents are **only** the `@` references somebody
deliberately typed. There are no backlinks and no automatic "referenced by"
index. A backlink graph would make every doc's relationships a side effect
of other people's edits, and the cheap version of it (scan every body on
every change) is exactly the per-keystroke work the collaboration and search
paths are designed to avoid. A doc that wants to point at context in another
project does it with an explicit mention.

References are stored as canonical internal Markdown links carrying the
entity type and stable id (`/tickets/<id>`); titles are labels and never
identifiers, so renaming a target can never break a reference.
