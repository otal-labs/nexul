# 16 — client-core and the phone

**Status:** done

**Blocked by:** 11

**Branch:** `permission-switch`.

Read first: `practices/typescript.md` (section 11), `practices/native.md` (sections 3, 4, 5), `practices/testing.md`,
ADRs 0082, 0139, 0148, the spec (MCP, the HTTP paragraph), the research (sections 2.8 and 5).

## What to build

- `MyWorkspaceInfo` (`client-core/permissions.ts:32-58`) carries `roles`, and `projectPermissions` reads `projects`
  for every member.
- The phone's locked docs leave lists (`native/src/components/docs/DocRow.tsx`, `DocsListScreen.tsx`).
- `useLiveEvents` (`native/src/hooks/useLiveEvents.tsx:13-26`) handles grant frames that name no person.
- The Restricted cases in `AreaGate.test.tsx` and `BoardScreen.test.tsx` are rewritten.
- The guide page `phone-app.md`, if it mentions what a member sees.

## Acceptance criteria

- [x] `native/src/hooks/requestCounts.test.tsx` and `useLiveEvents.test.tsx` cover a frame that names no person.
- [ ] Checked through the T3 Device panel: an unlisted member sees no projects, and chat works. Not yet: the server
  side was checked on a migrated device-pass copy (`/me` sends `projects: []` and chat permissions, the project list is
  empty) and `AreaGate.test.tsx` holds the tabs, but the Device panel was out of reach from the slice's session.
