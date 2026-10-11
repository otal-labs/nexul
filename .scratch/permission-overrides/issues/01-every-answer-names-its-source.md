# 01 — Every answer names its source

**Status:** ready-for-agent

**Blocked by:** None — can start immediately

**Branch:** master, a normal PR, before the switch. It changes nobody's access.

Read first: `practices/go.md` (sections 2, 9, 10, 17), `practices/architecture.md` (Principles, section 8),
`practices/mcp.md` (sections 3, 4, 6, 7, 8, 11), `practices/testing.md` (sections 3, 5, 10), ADRs 0042, 0087, 0097,
0135, 0140, 0148, the spec (Evaluation order, The source sentence, Source and explain), the research (sections 1 and
9).

No schema change. The answer carries its reason, and the explain path reads it out.

## What to build

- `decide` (`internal/access/usecase.go:95-110`) returns whether the action is held plus the source as ids (kind:
  owner, role, rule, restricted base; entity type and id; target type and id; state). `has`, `applyOverwrite`,
  `check`, `require`, `CanDocs`, `ProjectsWith` and `DocsWith` use it, and every caller still reads only whether it is
  held. No name lookups on this path.
- `Service.Explain(ctx, actorID, resourceType, resourceID, userID)` for `doc`, `play` and `project`, as today's rules
  mean them: per action the entity's rules may set, the answer and its source resolved to names and the sentence, plus
  the answer this entity would give the chosen person if its own rule were Fallback (it feeds the panel's line under
  each Fallback in slice 06).
- `GET /api/permissions/explain?resource_type=&resource_id=&user_id=`, and an optional `user_id` on
  `permission_overwrite_list` that returns that person's `effective` answers with `allowed` and `source`. No new tool.
- Explaining your own access needs only being able to open the entity; another person's takes the entity's manage
  permission (`permissions:write` on a doc, `plays:write` on a play, `members:write` for a project until slice 09).
- A forbidden answer from `Require`, `RequireProject` and `Can` carries the source sentence, looked up only on that
  path. Not found carries none.
- `access.Matrix(ctx, db)`: every member × every catalog action × the workspace, every project, doc, play and channel
  of a database, as sorted JSON. A test helper runs it on the copy named by `NEXUL_ACCESS_MATRIX_DB` and skips when it
  is unset. Every later PR before the switch diffs it on a production copy.
- `CONTEXT.md`: the source of a permission, under Permission overwrite.

## Acceptance criteria

- [ ] `TestExplain_AnswersWhatTheCheckAnswers`: over the `list_paging_test.go` fixture's viewers, every catalog action
      and every doc, play and project, explain's answer equals the check's.
- [ ] Table-driven `TestDecide_Source`: Owner, role, role missing, workspace allow and deny, Restricted with and
      without Project access, doc allow and deny, play deny, each with its source.
- [ ] Explaining another person's access without the manage permission is refused over HTTP and MCP, and an
      unreadable doc is not found.
- [ ] A forbidden doc edit over HTTP and MCP names the rule that denied it; a not-found answer says nothing more.
- [ ] Every `TestStatements_*` guard in `server/cmd/access_memo_test.go` keeps its count.
- [ ] `access.Matrix` on the fixture matches a golden file committed here.
- [ ] The tool count is unchanged.
