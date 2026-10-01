# Templates have an instance layer

Four texts start every workspace or project off: the Interview template, the mention chip template, the built-in
plays' instructions, and each default ticket type's body template. Each had a default in code and nothing between that
default and the workspace or project holding a copy, so an owner who wanted every workspace to ask the same interview
or file bugs the same way had to edit each one by hand, and every new workspace started from the code again.

Decision: every template resolves code default, then instance template, then workspace or project. The instance layer
is optional: a template nobody has edited at the instance is the code default.

- **Two ways down.** The Interview and mention chip templates are stored in a workspace only once it edits its own, so
  an unedited workspace follows the instance live. Built-in play instructions and body templates are copied into each
  workspace or project when it is created and never rewritten afterwards, the way they always were; only workspaces
  and projects made after an instance change start from the new text.
- **Reset goes one layer up.** Resetting a workspace or project gives it the instance's current text: the Interview
  and chip templates drop their own and follow again, a play or ticket type gets a fresh copy. Resetting the instance
  gives the code default.
- **Clone is one operation.** It copies one template's text from the instance, a workspace, or a project over another
  of those places, overwriting it. Plays match across workspaces by the built-in play's stable key, ticket types by
  name, ignoring case; no match is an error, never a guess. Reading the source takes read where it lives, writing the
  target that place's own write, so cloning grants nothing the caller could not do by hand.
- **The `templates` permission.** Writing an instance template takes `templates:write` in any workspace (ADR 0088).
  Reading the instance templates takes only a signed-in member, since every workspace shows them as its default;
  `templates:read` exists to scope a token's reads. The Owner holds both through the bypass, so no role is backfilled:
  nobody could edit instance templates before.
- **Stable play keys.** A seeded play carries a `builtin_key` (`fix-with-ai`, `to-tickets-via-ai`, `interview`,
  `test-with-ai`) kept through renames. Migration 0055 gives each workspace's first play with a seeded label and type
  its key; a play renamed before the upgrade, or deleted, has none, and a clone into that workspace finds no match.
- **The chip's column.** The chip template was moved from the instance to the workspace (migration 0031) so each
  workspace could lay out its own chips, and that stays: the instance layer sits under it rather than replacing it.
  The column is `NOT NULL` with the old default on every row, so "never edited" is an empty string rather than a
  nullable column, which SQLite could only add by rebuilding the table. Migration 0055 empties every row still equal to
  the old default, so those workspaces follow the instance; a workspace that deliberately saved the default text is
  indistinguishable from one that never chose and follows too. Saving an empty chip template resets it. The workspace
  reports `mention_chip_template_edited`, and its `mention_chip_template` is always the resolved text, so clients that
  predate this change keep rendering correctly.
- **One event.** An instance write or reset publishes `instance_template.updated`, instance-wide, and reaches every
  signed-in browser, so a page showing an unedited workspace's template refetches. Writes below the instance publish
  their own domain's event; the chip, which published none, now publishes `workspace.updated`.
- **Over MCP**, `template_get` and `template_update` read and change any template at any layer, with reset and clone
  on `template_update`. They replace `interview_template_get` and `interview_template_update`, so the tool count is
  unchanged.

The trade-offs: a copy made at creation does not see a later instance change, so an owner who wants it everywhere
clones it into each existing workspace. A following template changes under a workspace without anyone there touching
it, which is the point but means the Interview a new project starts from can change between two project creations.
The decisions check play has no stored instructions per workspace, so it has no template.

Rejected: rewriting existing copies when the instance changes, which would silently undo every workspace's own edits;
making every kind follow live, which would need the plays and ticket types to stop storing their text and change what
a play run or a new ticket reads; a separate clone tool or a `template_list` tool, which the tool ceiling has no room
for when `template_update` and `template_get` already name the place.

Decided 2026-10-01. Amends ADR 0088's table with `templates:write`.
