# Doc folders nest, and a folder's rules reach everything inside it

ADR 0096 kept a project's doc folders one level deep and called a series inside a series a naming convention. People
grouping docs per client, and inside that per stage, ran out of room: one level meant a flat row of "Acme drafts",
"Acme signed", "Globex drafts" folders.

Decision: a folder may sit inside another folder of its project, to any depth.

- A folder carries `parent_id`, empty at the top level. Migration 0166 adds it and leaves every existing folder at the
  top, so an upgraded instance looks as it did. A name is unique among its siblings ignoring case, so two folders may
  share a name under different parents.
- The default folder, Main, stays at the top level: it can be renamed and hold folders, but never moved or deleted.
- A folder moves into another folder of its project, or to the top, with everything inside it, through
  `POST /api/docs/folders/{id}/move` and `project_update`'s `doc_folders.update[].parent_id`. It never goes inside
  itself or a folder within it; the storage write checks that inside its transaction, so two crossing moves can't make
  a loop. A move publishes `doc.folder.updated` with `previous_parent_id`.
- Deleting a folder deletes every folder inside it, and their rules, and moves all their docs to the deleted folder's
  parent, or to Main for a top-level folder; no doc is ever deleted. It publishes one `doc.folder.deleted` per folder,
  the deleted one first, so every consumer of the topic keeps working unchanged. Moving the folders inside up a level
  instead was rejected: their names could collide with the parent's folders, and a delete that half-fails is worse than
  one that takes the subtree.
- A folder's rules reach every doc and every folder inside it, at any depth. The chain from ADR 0148 becomes the
  project, each folder from the top down to the doc's, then the doc, the nearest rule winning. The access reads stay one
  statement each: a recursive query walks a doc's folders for its rules, and the doc list's turns walk the project's
  folders top down, so a sub-folder inherits its parent's answer unless its own rules turn it. A rule that stopped at
  the folder it names was rejected: a Deny on "Clients" that left "Clients / Acme" open would be a leak nobody expects.
- A reader without `docs:write` sees the folders holding a doc they can open and every folder those sit inside, so the
  tree never shows a folder without the way to it.
- The web list nests a folder's folders above its docs, stepped in one level each, and a folder's count takes in every
  doc inside it. A folder's path ("Clients / Acme") names it in pickers and the doc's breadcrumb.

Amends ADR 0096, whose one level and rejection of nested folders no longer hold, and ADR 0148, whose chain now holds
every folder around a doc. Decided 2026-11-19.
