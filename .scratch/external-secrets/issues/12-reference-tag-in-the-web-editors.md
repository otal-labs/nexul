# 12 — Tag a secret reference in the web editors and show its save error inline

**Status:** ready-for-agent

**Blocked by:** 11

Read first: `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`
(section 11), the spec (Secret references, Web UI).

## What to build

- `client-core/secretReference.ts`: the TypeScript twin of ticket 11's parser, with its tests beside it.
- In `StackEnvironmentSection.tsx`, `BranchDeployRuleOverridesForm.tsx`, `AutomationSecretForm.tsx`,
  `ManualConnectorDialog.tsx` and `ConnectorAppConfigForm.tsx`: a value that parses as a reference shows
  a tag with the manager's name taken from its scheme ("1Password"), using existing tokens and the shared
  badge component, and the save error shows inline under the field.

## Acceptance criteria

- [ ] `client-core` parser tests mirror ticket 11's table.
- [ ] Each editor shows the tag for a reference and not for a literal URL, and shows a malformed
      reference's save error inline (component tests).
- [ ] Screenshots at 768, 1024 and 1440px in the PR.

Verify: `bun run lint`, `bun run typecheck`, `bun run test` in `web/`.
