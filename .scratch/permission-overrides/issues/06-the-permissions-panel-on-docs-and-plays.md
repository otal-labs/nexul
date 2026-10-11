# 06 — The permissions panel on docs and plays

**Status:** done

**Blocked by:** 01

**Branch:** master, a normal PR, before the switch. People targets only; role targets arrive in slice 14.

Read first: `practices/react-guide.md` (F1 to F7, Data fetching, Live events, Testing, the self-review),
`practices/design-language.md` (the shared core and the web app section), `practices/go.md` (section 17),
`practices/mcp.md` (sections 4, 6, 11), `practices/testing.md`, ADRs 0088, 0148, the spec (States, Entities, UI),
`.scratch/permission-rework/issues/03-the-permissions-panel.md` (all of it, and the owner's changes at its end win),
the research (sections 2.4 and 7.1).

## What to build

- The panel as locked in rework ticket 03, under `components/access/`: the "Permissions" dialog with Rules | Check
  access, the target column (a scrolling tab row at 768px), the rules pane grouped by domain, and the owner's
  **three icon buttons** per permission (check, slash, cross as a radio group, each with its label as tooltip and
  `aria-label`), a locked Allow with its sentence, the source line under each Fallback, and Check access.
- It replaces `PermissionsForm` on docs, and "Exclude users" on plays (`PlayRow.tsx:31-43`), where the Run row shows
  today's exclusions as Deny. A doc carries Docs and Access ("Manage this doc's access"); a play carries Run.
- `SetGrants` and `SetPlayGrants` take a `state` (Allow, Fallback, Deny). HTTP `PUT /api/permissions` and
  `permission_overwrite_update` gain `state` beside `grant`, which keeps its meaning when `state` is absent.
- Hold-what-you-give on doc and play rules (`internal/access/usecase.go:392-416`): nobody allows, or lifts a Deny on,
  what they don't hold on that entity; adding a Deny is never limited.
- The guide pages `people-and-access.md` and `plays.md` show the panel.

Rules saved here before the switch are person rules, and the wipe deletes them like any other sharing.

## Acceptance criteria

- [x] A person set to Deny and another to Allow on one doc works end to end, and Check access shows both sources.
- [x] Another open page updates without a refresh when a rule changes.
- [x] `TestGrants_NobodyAllowsWhatTheyLack` for docs and plays; a Deny is always accepted.
- [x] `PermissionsForm.test.tsx` is replaced by the panel's tests.
- [x] Verified at 768, 1024 and 1440px with before and after screenshots, and the react-guide self-review passes.
