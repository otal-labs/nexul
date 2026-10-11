# 13 — The Access page and the one-time setup

**Status:** ready-for-agent

**Blocked by:** 11

**Branch:** `permission-switch`.

Read first: `practices/react-guide.md` (F1 to F7, Routing, Live events, the self-review),
`practices/design-language.md`, `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2, 3),
`practices/testing.md` (section 10), ADRs 0098, 0148, the spec (Projects are opt-in, The upgrade: a wipe and a
one-time setup), `.scratch/permission-rework/issues/05-the-one-time-project-setup.md`.

## What to build

The look of rework ticket 05:

- `/:workspace/access` and its sidebar row (ShieldCheck, between Automations and Configuration), for anyone holding
  Manage this project's access. A non-Owner sees only the Projects grid with the projects they are in and may manage.
- The Owner's one-time redirect, the "Set up access" sidebar row until Finish, and the "Projects being set up"
  waiting state for everyone else.
- Both grids: projects × roles with the People column, and public channels × roles (Owner only), with the lead rows'
  tri-state, filters and sticky heads.
- One save request that writes the project rules and the channel deny rules in one transaction, and sets the setup
  marker on Finish.
- The live refresh for members when the Owner finishes.
- The Access page's tabs after setup, each card saving from its footer.
- The guide page `people-and-access.md`.

## Acceptance criteria

- [ ] New `TestStatements_AccessGridCostsTheSameAtAnyLength`: the grid's read and the Finish write at 3 projects and
      at 30.
- [ ] Finish with every project left empty still works.
- [ ] A member's projects appear without a refresh.
- [ ] Leaving with unsaved ticks asks first.
- [ ] Verified at 768, 1024 and 1440px, including the 30 × 10 case, and the self-review passes.
