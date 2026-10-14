# 01: Hosts follow their machine, placing an automation takes Deploy to, and automations become workspace actions

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Read first: `AGENTS.md`, `CONTEXT.md` (Machine rules, Instance area), ADR 0151 and 0154, `practices/go.md` (section 17),
`practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, the spec.

## What to build

Decisions 1 to 4 of the spec on the server: the host checks in `HostsService` and `SetHost`, the list filtered by the
caller's machines, the enrollment code re-checked on use, the catalog change, a migration deleting the rules on the
`automations` instance area, the route permission table, MCP descriptions, and the guide (`runners.md`,
`people-and-access.md`, and the automations page where it mentions hosts). `CONTEXT.md`'s Instance area entry follows.

## Acceptance criteria

- [ ] Every test in the spec's Testing section.
- [ ] `TestShippedTemplates_HoldNoInstanceArea` passes without the automations carve-out.
- [ ] Hit every surface: HTTP, MCP, live (any host topic's audience follows See on the machine), guide.
