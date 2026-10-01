# Docs live in one folder per project, with a default Main

ADR 0025 left a project's docs a flat list once Collections were retired. A project that keeps a series together (a
client's eighteen "GetSource EP01" to "EP18" scripts) had no way to group it short of a title prefix, and the list
buried everything else under it.

Decision: a project's docs are grouped into folders, one level deep, and every doc lives in exactly one folder of its
project.

- Every project has a default folder, Main, made in the same transaction as the project. Migration 0048 gives every
  existing project its Main and moves every existing doc into it, so an upgraded instance shows what it showed before,
  under one folder.
- A new doc lands in Main unless it was started in another folder. A doc copied into another project lands in that
  project's Main; a copy within its own project sits beside the original.
- Main can be renamed but never deleted. Deleting any other folder moves its docs to Main in the same transaction; no
  folder delete ever deletes a doc.
- A folder name is unique in its project ignoring case, as a channel name is in its workspace.
- Creating, renaming, and deleting a folder, and moving a doc between folders, take `docs:write`, checked in the
  project's workspace (ADR 0087). Deleting a folder takes `docs:write` rather than `docs:delete`, because no doc is
  destroyed. A reader sees the folders holding a doc they can open; a writer sees every folder, so an empty one stays
  visible to the people who can fill it.
- A move is not an edit: it keeps the doc's version and last-edited time and publishes `doc.moved`, not `doc.updated`,
  so it notifies nobody. Folders publish `doc.folder.created`, `doc.folder.updated` (a rename, with the previous name),
  and `doc.folder.deleted` (with the folder its docs moved to), and `doc.created` and `doc.updated` carry the doc's
  `folder_id`.
- Over MCP, folders are a child collection of the project (ADR 0068): `project_get` lists them as `doc_folders` and
  `project_update` creates, renames, and deletes them, while `doc_list`, `doc_create`, and `doc_update` take
  `folder_id`. The tool count is unchanged.

The trade-offs: one level only, so a series inside a series is a naming convention, not a subfolder. A folder belongs
to one project, so the cross-project grouping Collections offered stays gone; this is the grouping ADR 0025 declined to
add inside a project, and it does not bring back two taxonomies to reconcile. Every doc must have a folder, so there is
no "unfiled" state to design around, at the cost of a folder row even in a project that never wanted one.

Rejected: calling them categories, which already names the board's ticket grouping; sections, which reads as a heading
inside a doc and as the frontend's Section component; nested folders; a doc with no folder; and new `doc_folder_*`
tools, which the tool ceiling has no room for when the project's tools already carry its other child collections.

Amends ADR 0025, whose "a project's docs are a flat list" no longer holds. Decided 2026-10-01.
