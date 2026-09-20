# 03 — Web: Instance version section with the Upgrade action

**Status:** resolved
**Type:** task
**Blocked by:** None — build against the HTTP contract in the spec; the
server lands in ticket 02.

## Scope

`web/src` only: hooks, the settings section, sidebar badge link, live
event topic, tests.

## Build

- `InstanceUpgradeHooks.tsx`: `useInstanceUpgrade()` (query on
  `GET /api/instance/upgrade`, refetch every 5 s while the record is
  `pending`/`started`) and `useRequestInstanceUpgrade()` (mutation on
  `POST`, 409 body's `reason` surfaced as the error).
- `InstanceVersionSection.tsx` in `components/settings/`, rendered first in
  `InstanceSettingsPanel.tsx`, built on `SettingsCard`: facts row
  (version, channel, newest release link), the `Upgrade to vX` button with
  a confirmation dialog, disabled state with `reason` as helper text, the
  in-progress / completed / failed states from the spec. Mono Console: no
  new colors; reuse the existing status badge language.
- `useLiveEvents` `pushTopics`: `instance.upgrade_changed` invalidates the
  upgrade and version queries.
- `SidebarFooter.tsx`: the update badge links to the instance settings
  page (the section's anchor) instead of the docs URL.
- Tests for the hook states and every section state; run
  `bun run --cwd web lint`, `typecheck`, `test`.

## Acceptance

- Section renders all five states from mocked responses.
- Lint, typecheck, and tests green; no new dependency.
