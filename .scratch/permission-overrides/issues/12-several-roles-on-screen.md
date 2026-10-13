# 12 — Several roles on screen

**Status:** done

**Blocked by:** 11

**Branch:** `permission-switch`.

Read first: `practices/react-guide.md` (F1 to F7, Forms, Live events, the self-review), `practices/design-language.md`,
`practices/testing.md`, ADRs 0085, 0088, 0148, the spec (Roles, UI, including Copy for a member with no role),
`.scratch/permission-rework/issues/04-several-roles-on-screen.md`, the research (section 2.6).

## What to build

The look of rework ticket 04, with the spec's zero-roles copy:

- Team rows with role chips (`TeamPersonRow`).
- The person dialog's Roles checklist with "Can't give" and its reasons, and the zero-roles warning, replacing
  `TeamRoleSelect`, `TeamOverridesForm` and `TeamProjectAccess`.
- The invitation dialog's checklist (at least one role per workspace), with `InvitationProjectAccess` and the
  overrides disclosure gone, and the invitation page's chips.
- The zero-roles empty state, whose Owner rows open a DM, and its sidebar, which keeps Channels and Direct messages
  when they have rows.
- Your roles, and the account row listing the roles.
- The guide page `people-and-access.md`.

## Acceptance criteria

- [x] A role ticked or unticked updates the other person's sidebar live.
- [x] The removed components and their tests are gone.
- [x] A person with no role sees the empty state, reads public channels and opens a DM with an Owner from it.
- [x] Verified at 768, 1024 and 1440px, and the self-review passes.

## Notes for later slices

- `GET /api/workspaces/{id}/people` marks each Owner with `owner`, so a member with no role sees whom to ask; the
  invitation preview and acceptance name every role of a grant as `roles`.
- The dialog judges "Can't give" in the browser from the workspace's roles and the viewer's own roles on `/me`, the
  union the server checks; the server still refuses anything the browser let through.
- The form dialog gained `setBlocked`, which disables its OK button without a spinner (Create link).
- `ProjectAccessBlock`, `EveryProjectRow`, `ProjectAccessRow` and `PermissionGrid` went with their last users;
  `usePermissionLock` now has no provider and always reads false.
- The guide's "Hold someone to chosen projects" section still describes Every project and Restricted; slices 13 and 14
  own it.
