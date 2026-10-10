# 03 — The permissions panel on a doc

**Status:** ready-for-agent

**Blocked by:** 02

Read first: `practices/react-guide.md` (all of F1 to F7, Data fetching, Live events, Testing),
`practices/design-language.md` (the shared core and the web app section), `practices/testing.md`, ADR 0148,
the spec (UI structure).

This is the first sight of the panel every entity reuses, so it starts in design mode: the structure below is
fixed, the look is not.

## What to build

- A reusable permissions panel under `components/access/`: targets (Everyone, roles with a rule, people with a
  rule, and an add control for a role or a person) and the selected target's permissions grouped by domain,
  each with Allow, Fallback, Deny. Under each Fallback, the answer it falls back to. An Allow the giver
  cannot give is disabled with the reason.
- "Check access" from ticket 01 moves into the panel; a source naming another entity links to it.
- The doc's Permissions dialog uses the panel, replacing `PermissionsForm` for docs.
- Built from the settings kit and the existing access components where they fit; registry components installed
  through the shadcn CLI.
- The guide page `people-and-access.md` shows the panel.

## Acceptance criteria

- [ ] Setting a role to Deny and one of its people to Allow on a doc works end to end, and Check access shows
      both sources.
- [ ] Another open page updates without a refresh when a rule changes.
- [ ] Verified at 768, 1024 and 1440px, with before and after screenshots in the PR.
- [ ] The self-review checklist in `practices/react-guide.md` passes.
