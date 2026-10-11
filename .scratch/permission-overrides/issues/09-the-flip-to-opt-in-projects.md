# 09 — The flip to opt-in projects

**Status:** ready-for-agent

**Blocked by:** 01, 02, 03, 04, 05 (merged on master; the `permission-switch` branch is cut from master after them)

**Branch:** `permission-switch`. The first slice of the switch; it changes access. May take two sessions on one
branch.

Read first: `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2, 3, 8), `practices/mcp.md`
(sections 4, 7, 11), `practices/testing.md` (sections 3, 8, 10), ADRs 0057, 0087, 0088, 0097, 0098, 0102, 0135,
0140, 0147, 0148, the spec (Roles, New permissions, Projects are opt-in, Channels and plays stay workspace-wide,
Evaluation order, Pills), the research (sections 1, 4.5, 7.5, 9 and 10).

## What to build

- A non-Owner is in a project only when its rules allow In this project (`projects:open`) for a role they hold or for
  them by name; otherwise the project and everything in it is not found. Zero roles is allowed.
- Project-area actions count only inside a project the person is in. The paths that ask one with no project named
  move to `projects:create`: project create, the wizard's repository list and scan, detaching a GitHub account, and
  installation frames (`server/cmd/live_audience.go:265-275`).
- New permissions: `projects:create` (workspace area, with the table's per-action area), `projects:access` (project
  area), and `projects:open` (rule only, outside the role grid, like the computer permissions). A migration grants
  `projects:create` to roles holding `projects:write` and `projects:access` to roles holding `members:write`, the way
  migration 0046 did.
- Restricted branches leave `internal/access`, `internal/chat` (`refuseRestricted`, `IsRestricted`) and
  `internal/memories` (`interview.go:161`). `workspacesOf` reads every membership.
- Project create lists, in the same write, the creator by name (Allow `projects:open` and `projects:access`) and each
  of the creator's roles (Allow `projects:open`), over HTTP and MCP.
- Writers accept role targets on docs, plays and projects, with hold-what-you-give. Writing project rules takes
  `projects:access` on the project.
- Kept out: a person rule denying `projects:open` on a project.
- Unreadable is not found: the doc list's no-query path filters by `DocsWith(DocsRead)`, a doc read without
  `docs:read` answers not found, and `can_open` leaves list items.
- Pills: `resolveDoc` and `resolveTicket` (`internal/mentions/usecase.go:275-326`) keep showing full content, locked,
  across the project line.
- A DM whose other participants are all Owners of the workspace takes membership alone (`CreateDM`,
  `internal/chat/conversation_usecase.go:28`), so a person with no role can message an Owner.
- `CONTEXT.md` matches what this slice built.

## Acceptance criteria

- [ ] `TestIntegration_PermissionSuitesAnswerTheSameWithAMemo` and `…LiveAudienceMemo_FollowsEveryChange` pass with
      the fixture's `uClient` as a client listed in one project.
- [ ] `TestListTools_ShowWhatTheRowByRowCheckShowed`: `restrictedClient` becomes an unlisted member and a member with
      zero roles; `doc_list` parity becomes "listed equals openable".
- [ ] The "docs list" and "a play's queued runs" rows of `TestStatements_ListsCostTheSameAtAnyLength` keep their
      counts; `TestCanDocs_ReadsEachLayerOnceForTheLot` moves out of `restricted_test.go`; the slice 01 parity test
      runs over the new fixture.
- [ ] The access and chat tests in research section 10 are deleted or rewritten.
- [ ] A zero-role member reads People, public channels, DMs and the inbox, starts a DM with an Owner, and reaches
      nothing in a project.
- [ ] Pills from an unlisted project show their full content, locked.
- [ ] A creator's new project opens for them, over HTTP and MCP.
- [ ] `doc_list`, `ticket_list` and one live frame: statements and median time on the heavy copy, before and after,
      in the commit message.
