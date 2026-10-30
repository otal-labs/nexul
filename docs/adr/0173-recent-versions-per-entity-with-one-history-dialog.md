# Recent versions per entity, with one History dialog

Docs kept versions nobody could see, memories versioned every save (even an unchanged one) with no diff, and
tickets and plays kept no history, so "who changed the acceptance criteria" had no answer.

Decision: docs, ticket descriptions, play instructions and memories keep body versions, and tickets and plays also
keep field changes. One History dialog shows each version against the one before it, word by word, and restores
an older one.

- **A table per entity.** `doc_versions`, `ticket_description_versions`, `play_instruction_versions` and
  `memory_versions` hold bodies; `ticket_changes` and `play_changes` hold one row per changed field. Each table
  cascades on its entity's delete. `automation_versions` stays as it is. Every entity answers the same
  `/history`, `/versions/{n}` and `/versions/{n}/restore` shape.
- **The last ten.** Ten unnamed versions are kept per body, and a hundred field changes per ticket or play. Named
  doc versions and every automation version are never pruned. A version stores its words added and removed when it
  is written, recomputed when a prune removes the version before it, so the list never diffs on open.
- **A version per sitting.** The live editor writes one version once the text has had no edits for five minutes,
  or when its last editor leaves, credited to everyone who edited. A save that changes nothing writes no version.
- **Restore adds a version.** Restoring writes the old text as a new version on top, after saving the open
  sitting. Field changes are never undone from history.
- **Via.** A version records how it was saved: the automation's name, `mcp` for an agent, or nothing for the
  browser. An MCP save of a ticket or doc made while the same person has a play run starting or running on it
  records the play's label instead, found by one indexed lookup of their own trails. A run's MCP calls carry the
  person's computer token, which names no run.

The trade-off: history is short by design. Anything older than ten versions is gone, and history starts on the day
it shipped.

Rejected: one shared versions table, which would need a polymorphic owner and lose the cascade; keeping every
version, which grows without bound for a doc edited daily; a per-run MCP token or header, which every provider's
MCP configuration would need per thread.

Decided 2026-10-29.
