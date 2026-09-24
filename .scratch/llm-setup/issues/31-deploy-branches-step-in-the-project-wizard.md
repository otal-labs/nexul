# 31 — Deploy branches step in the project wizard

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 30
**Decided in:** ticket 10

## What to build

New project-wizard step: branch rows starting with the default branch and the service hostname, wildcard rows with a live URL example (dots, slashes, and capitals become dashes and lowercase, trimmed to hostname limits), a network picker per row listing what runs on each network, Advanced overrides per row, and a warning on rows sharing production's services. Skippable.

Branch deploy rules only accept overrides when they deploy their own copy (a
name suffix or a wildcard); a rule that redeploys the base in place cannot
override the base's own values. Every row except the default branch's
therefore derives its own copy, so its network and overrides always apply.

## Acceptance criteria

- [ ] The rows create the matching branch deploy rules
- [ ] `feature/dot.test` previews as `dot-test.<domain>`
- [ ] The production-sharing warning appears exactly when the row shares the default branch's network without overrides

## Surfaces

- Mirrors into the stack page
- Docs: project wizard guide

## Read first

`practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, `practices/testing.md`, `practices/go.md`, `practices/architecture.md`, and the decision tickets above.

## Verification

`bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px; `go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `web/src/components/wizard/ (new step)`
- `web/src/pages/ProjectWizardPage.tsx`
- `internal/deploy/`

**Size:** M
