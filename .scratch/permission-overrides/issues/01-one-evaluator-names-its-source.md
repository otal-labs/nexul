# 01 — One evaluator that names its source, explained on a doc

**Status:** ready-for-agent

**Blocked by:** None — can start immediately

Read first: `practices/go.md` (sections 2, 9, 10, 17), `practices/architecture.md` (Principles, section 8),
`practices/mcp.md` (sections 3, 4, 6, 7, 8), `practices/testing.md` (sections 3, 5, 10),
`practices/react-guide.md` (F1 to F7, the self-review), ADRs 0042, 0087, 0097, 0135, 0140, 0148, the spec.

No schema change and no change in anyone's access: this slice makes the answer carry its reason, and shows it
for one entity type, the doc.

## What to build

- `decide` (`internal/access/usecase.go:95-110`) returns a decision: held or not, plus the source as ids
  (kind: owner, role, workspace rule, entity rule, restricted base; entity type and id; target type and id;
  state). `has`, `applyOverwrite`, `check`, `require`, `CanDocs`, `ProjectsWith` and `DocsWith` use it, and
  every existing caller keeps reading only whether it is held. No name lookups on this path.
- `Service.Explain(ctx, actorID, resourceType, resourceID, userID)` for `doc`: every action a doc rule may set
  (`docs:*`, `permissions:write`), each with held and the source resolved to names and one sentence. Anyone
  explains their own access on a doc they can open; another person's takes `permissions:write` on the doc.
  An unreadable doc is not found.
- `GET /api/permissions/explain?resource_type=doc&resource_id=&user_id=`.
- `permission_overwrite_list` gains an optional `user_id`: with it, the result carries that person's
  `effective` answers with their sources. No new tool.
- A forbidden answer from `Require`, `RequireProject` and `Can` carries the source sentence (looked up only
  on that path); not found never does.
- `access.Matrix`: every member × every catalog action × the workspace, every project, every doc and every
  play of a database, as sorted JSON, with a test helper that runs it on the copy named by
  `NEXUL_ACCESS_MATRIX_DB` and is skipped when unset. Later tickets diff it before and after a migration.
- Web: the doc's Permissions dialog gains "Check access": pick a person, read each permission's answer and
  its sentence. A typed hook in `hooks/PermissionHooks.tsx`; no visual decisions beyond a list of rows in the
  dialog (the panel's look is ticket 03).
- `CONTEXT.md`: **Source** (the rule that decided a permission) and the three states under Permission
  overwrite. The guide page `people-and-access.md` explains Check access.

## Acceptance criteria

- [ ] `TestExplain_AnswersWhatTheCheckAnswers`: for every viewer in the `list_paging_test.go` fixture, every
      doc and every doc action, `Explain`'s held equals `Can`.
- [ ] Table-driven `TestDecide_Source`: Owner, role, role missing, workspace allow, workspace deny, Restricted
      with and without Project access, doc allow, doc deny, play deny, each with the expected source.
- [ ] `TestExplain_AnotherPersonNeedsPermissionsWrite` and `TestExplain_UnreadableDocIsNotFound`, over HTTP
      and MCP.
- [ ] A forbidden doc edit over HTTP and MCP names the rule that denied it; a not-found answer says nothing more.
- [ ] Every `TestStatements_*` guard in `server/cmd/access_memo_test.go` passes with unchanged counts.
- [ ] `access.Matrix` on the fixture matches a golden file committed in this ticket.
- [ ] The tool count is unchanged.
- [ ] The dialog verified at 768, 1024 and 1440px.
