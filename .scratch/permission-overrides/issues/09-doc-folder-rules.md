# 09 — Doc folder rules that reach their docs

**Status:** ready-for-agent

**Blocked by:** 04

Read first: `practices/go.md` (sections 14, 17), `practices/react-guide.md`, `practices/testing.md`
(sections 3, 10), ADRs 0096, 0135, 0140, 0148, the spec (Entities, Evaluation order).

## What to build

- A doc folder (ADR 0096) takes Everyone, role and person rules for `docs:*`, on the chain between the project
  and the doc: the evaluator walks project, then folder, then doc, each with Everyone, role, person. The nearer
  entity wins as everywhere else. A doc's folder is read in the same statement as the doc's rules.
- Rules are written with `permissions:write` on the folder, under the hold-what-you-give rule judged on the
  folder. Moving a doc to another folder changes which folder rules reach it at once; the move itself keeps
  taking `docs:write` and notifies nobody.
- `DocsWith` and `doc_list` include the folder's rules, filtered in SQL, so a folder deny removes its docs from
  a list before paging.
- `permission_overwrite_list` and `_update` accept `resource_type: folder`; explain covers folders. The
  folder's menu in the docs list opens the permissions panel limited to doc actions. A deleted folder's rules
  go with it, and its docs move to Main as before.
- The guide page `people-and-access.md`.

## Acceptance criteria

- [ ] `TestDecide_FolderBetweenProjectAndDoc`: the spec's worked example extended with a folder, with sources.
- [ ] `TestFolderRules_MovingADocChangesWhatReachesIt`.
- [ ] `list_paging_test.go` gains folder rules and stays green; a new
      `TestStatements_FolderRulesCostTheSameAtAnyLength`.
- [ ] A Restricted member gains nothing from a folder rule for a role or Everyone.
- [ ] Verified at 768, 1024 and 1440px.
