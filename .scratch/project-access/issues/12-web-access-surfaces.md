# 12 — Web: the access surfaces

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 08
**Decided in:** tickets 01, 05, 07; spec section "Team, roles, project settings (web)"

## What to build

Every web surface for Restricted members and Project access, over ticket
08's routes. The visual treatment of each level control follows ticket 07,
which is being redone (level controls are moving to compact trailing
dropdowns with described levels); build placement and behaviour from this
ticket and the look from ticket 07 as it stands when you start.

1. **Role editor** (`web/src/components/settings/RoleSettingsSection.tsx`,
   `CreateRoleForm.tsx`, `RoleRow.tsx`): split the level list under two
   microheaders, Workspace (with the instance areas) and Every project, the
   second with "Applies to members whose Every project is From role." and
   its own Every area row. Group by the catalog's `area`; no hardcoded
   domain lists.
2. **Team dialog** (`web/src/components/team/TeamPersonDialog.tsx`,
   `TeamMembershipItem.tsx`, `TeamRoleSelect.tsx`, `models/Team.tsx`,
   `utils/TeamUtility.tsx`): under each workspace's role select, the Every
   project row (From role / None, with "Every project, through their role."
   or "Sees only the projects below. New projects stay hidden."). Under None,
   a count, a search field, and one row per project with a summary of what
   is granted, opening one at a time into the project areas' levels with an
   Every area row. Not shown for the Owner. Read-only with the reason where
   the viewer lacks `members:write`, as today.
3. **Invitation dialog** (`hooks/InvitationHooks.tsx`, `models/Invitation.tsx`
   and its dialog): the same block under each workspace's role.
4. **Project settings, People with access**
   (`ProjectSettingsContent.tsx` and a new section beside
   `ProjectGeneralSection.tsx`): a settings card listing the Restricted
   members who can open the project, each with a summary, "Change access
   from Team." in the footer, and "No restricted member can open
   <project>." when empty; shown to holders of `members:write`.
5. **Delete confirmation** (`ProjectDangerZoneSection.tsx`): a line naming
   the Restricted members who lose access, from the delete impact.
6. **Revoked page**: a project-scoped page whose project read turns into
   not found while open shows the standard empty state "You no longer have
   access to this project", "Someone changed your access. Projects you can
   still open are in the sidebar.", and "Go to Home".
7. **Pickers** (`components/ticket/PersonPickerList.tsx`): the developer and
   tester pickers list `GET /api/projects/{id}/people`, not the whole
   workspace's People.
8. **Live** (`hooks/useLiveEvents.tsx`): on `access.grant.changed` with
   `resource_type: "project"` and on `workspace.member.updated`, invalidate
   workspace-me, projects, board, tickets, docs, memories, conversations,
   Team, and People-with-access queries. The permission-changed path already
   in the hook is the place.
9. **Workspace me** (`models/` and the hook behind `hasPermission`): read
   `restricted` and per-project actions, so project pages hide what the
   server refuses.
10. Every change confirms with a toast naming the person and the project.
11. **Docs**: the roles text in
    `website/src/content/docs/docs/guide/api-and-tokens.md` and the project
    guide (`projects-and-repositories.md`) describe Every project and
    Project access.

## Acceptance criteria

- [ ] Owner gives a client None with tickets Write on one project; the
      client's open browser drops every other project live, and an open
      page of a project taken away shows the revoked state
- [ ] Switching None back to From role and back again shows the levels kept
- [ ] An invitation built with None lands the person restricted
- [ ] Pickers leave out Restricted members without access
- [ ] Component tests for each new state, error paths first
- [ ] Self-review checklist of `practices/react-guide.md` run; F1 to F7 hold
- [ ] Checked at 768, 1024, and 1440px with worst-case names
- [ ] Hit every surface: UI yes; HTTP in ticket 08; MCP in ticket 11; live
      push yes; permissions: the server decides, the web only hides; reverse
      states yes; docs yes
- [ ] `bun run lint && bun run typecheck && bun run test` in `web/`

## Read first

`practices/react-guide.md` (F1 to F7 and the self-review checklist),
`practices/design-language.md`, `practices/typescript.md`,
`practices/testing.md`, `practices/borrowed-practices.md`; ADRs 0078, 0080,
0085, 0097; this effort's `spec.md` and tickets 05 and 07.

## Files likely touched

- `web/src/components/settings/` (roles, project settings, danger zone)
- `web/src/components/team/`
- `web/src/components/ticket/PersonPickerList.tsx`
- `web/src/hooks/useLiveEvents.tsx`, `InvitationHooks.tsx`, project and
  team hooks
- `web/src/models/Team.tsx`, `Invitation.tsx`, `Project.tsx`
- `website/src/content/docs/docs/guide/api-and-tokens.md`,
  `projects-and-repositories.md`
