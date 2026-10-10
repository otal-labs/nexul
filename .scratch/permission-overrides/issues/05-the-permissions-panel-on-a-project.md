# 05 — The permissions panel on a project, and Team on the same rules

**Status:** ready-for-agent

**Blocked by:** 03, 04

Read first: `practices/react-guide.md` (F1 to F7, Live events), `practices/design-language.md`,
`practices/testing.md`, ADRs 0085, 0097, 0148, the spec (UI structure).

## What to build

- The project's settings page: the read-only "People with access" card
  (`components/settings/ProjectPeopleAccessSection.tsx`) becomes the permissions panel for the project's
  project-area actions, with Check access.
- Team's Project access (`components/team/TeamProjectAccess.tsx`) edits the same person rules, for any member;
  the Every project row stays.
- The guide pages `people-and-access.md` and `projects-and-repositories.md` describe project rules and the
  Restricted member rule in plain words.

## Acceptance criteria

- [ ] A rule set in the project's panel shows in Team for that person, and the other way round, live.
- [ ] A Restricted member's row shows that role and Everyone rules can only take away.
- [ ] Verified at 768, 1024 and 1440px with before and after screenshots.
- [ ] The self-review checklist passes.
