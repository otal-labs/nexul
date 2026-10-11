# 14 — People in this project, the project panel, and "Who's in it"

**Status:** ready-for-agent

**Blocked by:** 06, 07, 11

**Branch:** `permission-switch`.

Read first: `practices/react-guide.md` (F1 to F7, Live events, the self-review), `practices/design-language.md`,
`practices/go.md` (section 17), `practices/testing.md`, ADRs 0079, 0148, the spec (Projects are opt-in, Entities, UI),
`.scratch/permission-rework/issues/03-the-permissions-panel.md` (On a project, and the owner's changes), the research
(sections 7.1 and 7.5).

## What to build

- The "People in this project" card in the project's settings, with its "Kept out" heading below the people who are
  in, replacing `ProjectPeopleAccessSection`. Adding a role or person writes Allow of In this project; each row opens
  the panel on that target.
- The project's panel (the Project group led by "See this project", then every project-area domain), and
  "Permissions…" on project rows.
- Role targets in the doc and play panels.
- The doc panel refuses anyone outside the project: the add popover lists them disabled with "Add <name> to <project>
  first", linking to the project's list.
- The danger zone's "N people lose this project" replaces `RestrictedMembersLoseAccess`.
- The wizard's "Who's in it" step, pre-ticked with the creator's roles, in `SetupSteps`
  (`internal/workspace/model.go:48`) and the web (`web/src/models/ProjectWizard.tsx`). The creator is always in.
- The guide pages `projects-and-repositories.md` and `people-and-access.md`.

## Acceptance criteria

- [ ] A role added on the card opens the project for its holders live.
- [ ] A person kept out disappears from their own sidebar live.
- [ ] Projects made before the step count it as done.
- [ ] Verified at 768, 1024 and 1440px, and the self-review passes.
