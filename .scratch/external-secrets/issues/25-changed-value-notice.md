# 25 — Show the changed-value notice on the stack page, on the web and the phone

**Status:** ready-for-agent

**Blocked by:** 24

Read first: `practices/react-guide.md`, `practices/native.md` (section 4), `practices/design-language.md`,
the spec (Runners, Noticing a changed value).

## What to build

- Web: the stack page shows "<NAMES> changed since the last deploy" with the existing Redeploy for those
  who can deploy, followed live through the stack hooks' follower for `service.secrets_changed`.
- Phone: the same notice on `native/src/app/(tabs)/deploys/stack/`, opening the existing Redeploy sheet,
  with a `refreshes` line on the stack query for the new topic.

## Acceptance criteria

- [ ] Component tests on both apps: the notice shows for any viewer, the button only with deploy
      permission, and it clears after a deploy.
- [ ] The live request-count guards (`liveRequests.test.tsx`, `requestCounts.test.tsx`) still pass.
- [ ] Web screenshots at 768, 1024 and 1440px; a phone screenshot.

Verify: `bun run lint`, `typecheck` and `test` in `web/` and `native/`.
