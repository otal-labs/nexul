# 10 — The wipe migration, and the tenancy and invitation removals

**Status:** ready-for-agent

**Blocked by:** 09

**Branch:** `permission-switch`.

Read first: `practices/go.md` (sections 14, 10), `practices/testing.md` (sections 3, 8), ADRs 0042, 0061, 0085,
0088, 0097, 0148, the spec (The upgrade: a wipe and a one-time setup), the research (sections 2, 3 and 4).

## What to build

- The migration (numbered at the final rebase; its tests refer to it by file):
  - deletes `workspace` rows, `project` rows, `play` rows, `doc` rows other than the creator's
    (`user_id IS NOT docs.created_by`), and doc rows whose doc is gone;
  - keeps the creator's doc row exactly as stored;
  - drops `workspace_members.restricted`;
  - moves `invitation_grants` to a role set with its one role copied across, and strips and drops `allow_json`,
    `deny_json`, `restricted` and `project_access_json`; open links keep their role and lose the rest;
  - writes the per-workspace setup marker: unset where the workspace has projects, set where it has none;
  - writes no channel rule: a former Restricted member reads public channels until an Owner hides them;
  - leaves `workspace_invites` with one role, a set of one on joining.
- The code: `SetMemberOverrides`, `project_access.go`, Every project and the invitation per-person validation
  (`validateInvitationProjects`, `requireCreatorHoldsProjects`) go. A held role may be deleted, and the cascade can
  leave someone with zero roles. Removing a role you couldn't have given is refused.
- `CONTEXT.md` matches what this slice removed.

## Acceptance criteria

- [ ] `TestMigration_TheWipe`: a fixture at the previous schema with every row kind (Owner, Team allow and deny,
      Restricted with Project access, dormant project rows, creator and non-creator doc rows, a doc row of a deleted
      doc, play exclusions, and the four shapes of open invitation), upgraded, asserting exactly which rows survive.
- [ ] Running the migrations twice changes nothing the second time.
- [ ] The tenancy and invitation tests in research section 10 are deleted or rewritten.
- [ ] Removing a role you couldn't give is refused; deleting a held role leaves its holders without it.
