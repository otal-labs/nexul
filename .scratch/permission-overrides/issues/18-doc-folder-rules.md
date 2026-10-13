# 18 — Doc folder rules

**Status:** done

**Blocked by:** 17

**Branch:** master, a normal PR, after the switch. A new ability; nobody's access changes on upgrade.

Read first: `practices/go.md` (sections 14, 17), `practices/react-guide.md`, `practices/mcp.md` (sections 4, 11),
`practices/testing.md` (sections 3, 10), ADRs 0096, 0135, 0140, 0148, the spec (Entities, Evaluation order).

## What to build

- A doc folder (ADR 0096) takes role and person rules for `docs:*` and `permissions:write`, on the chain between the
  project and the doc: project, folder, doc, and on each the role rules then the person's. A doc's folder is read in
  the same statement as the doc's rules.
- Rules are written with `permissions:write` on the folder, under hold-what-you-give. Moving a doc to another folder
  changes which folder rules reach it at once.
- `DocsWith` and `doc_list` include folder rules, filtered in SQL, so a folder Deny of `docs:read` removes its docs
  before paging. A folder shows while it holds a doc the reader can read.
- `permission_overwrite_list` and `_update` accept `resource_type: folder`; explain covers folders.
- "Permissions…" on folder rows opens the panel limited to doc actions, which refuses anyone outside the project. A
  deleted folder's rules go with it, and its docs move to Main as before.
- The guide page `people-and-access.md`.

## Acceptance criteria

- [x] `TestDecide_FolderBetweenProjectAndDoc`: the spec's worked example extended with a folder, with sources.
- [x] `TestFolderRules_MovingADocChangesWhatReachesIt`.
- [x] `list_paging_test.go` gains folder rules and stays green; new `TestStatements_FolderRulesCostTheSameAtAnyLength`.
- [x] Verified at 768, 1024 and 1440px.
