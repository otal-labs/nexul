# 13 — The Access page and the one-time setup

**Status:** done

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

- [x] New `TestStatements_AccessGridCostsTheSameAtAnyLength`: the grid's read and the Finish write at 3 projects and
      at 30.
- [x] Finish with every project left empty still works.
- [x] A member's projects appear without a refresh.
- [x] Leaving with unsaved ticks asks first.
- [x] Verified at 768, 1024 and 1440px, including the 30 × 10 case, and the self-review passes.

## Notes for later slices

- The grid reads in 10 statements and Finish writes in 17, at 3 projects and at 30
  (`TestStatements_AccessGridCostsTheSameAtAnyLength`). The save writes every moved rule with one upsert and one delete
  through `json_each`, and sends one `access.grant.changed` with `resource_type` `workspace`, which reaches every
  member.
- Channel rules now go through `permission_overwrite_*` and `PUT /api/permissions` with `resource_type` `channel`. They
  set `channels:read` only, and only the Owner writes them; slice 20 moves that to `channels:write` and adds
  `channels:post`. `GET /api/permissions/explain` doesn't cover a channel yet.
- `workspace_update` takes `finish_access_setup`, and `workspace_list` and `/me` carry `access_setup_finished`.
  Migration 0103 (provisional) adds who finished the setup and when.
- The dead `tenancy.ProjectAccessEvent` payload is gone from the grant topic.
- The Invite steps and the person dialog in `people-and-access.md` still describe per-person overrides; slice 12 owns
  them.
