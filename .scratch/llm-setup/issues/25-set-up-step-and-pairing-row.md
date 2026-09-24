# 25 — Set up step and pairing row

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 19, 24
**Decided in:** ticket 06, ticket 13

## What to build

The dialog's third step in the locked design: the coarse pre-selection (both skill locations ticked), then one live row per provider — running, confirmed, or failed with Retry — with the agent's commentary folded under the active row. Each computer row shows the setup badge, one line per provider with its confirmed-at time, and Set up / Re-run setup opening the dialog. Nothing on the row changes a confirmation. Remove the memory-skill copy box now that the wizard installs it.

## Acceptance criteria

- [ ] The Set up step shows each provider's turn live and Retry re-runs only that provider
- [ ] The row reflects setup live as the wizard runs
- [ ] The copy box is gone and nothing links to it

## Surfaces

- Live push
- Docs: one short page pointing at the wizard; the MCP server guide drops the manual skill step

## Read first

`practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `web/src/components/settings/ComputerRow.tsx, ComputersSection.tsx, MemorySkillSection.tsx`
- `website/src/content/docs/docs/guide/mcp-server.md`

**Size:** M
