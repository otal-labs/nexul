# Status columns are fully configurable; the five stages they belong to are not

A project's board columns are owner-defined — create, rename, reorder,
delete — but every status must declare one of five fixed, ordered stages:
`backlog`, `progress`, `review`, `testing`, `done`. The board orders columns
by stage first, then position, and only `done` is terminal.

The alternative (free-form columns with no stage) was rejected because
everything that reasons about a ticket's state — the finished/blocked rules,
automations, the agent — would have to match on display names, and a rename
would silently break it. With stages, a status keeps a stable identity and a
stable machine-readable meaning across any rename. The cost is that a
workflow that doesn't fit the five stages cannot be expressed; a project
that doesn't need a stage drops it by having no column in it (a repo whose
PRs deploy straight off green CI has no `testing` column), and a fresh
project seeds one column per stage.

Decided 2026-08-28, replacing the earlier active/completed kinds, which were
remapped onto the stages in place.
