# 07 — Team: workspace rules as three states, and Check access for every area

**Status:** ready-for-agent

**Blocked by:** 03

Read first: `practices/react-guide.md`, `practices/design-language.md`, `practices/mcp.md`,
`practices/testing.md`, ADRs 0085, 0088, 0097, 0148, the spec.

## What to build

- Team's person dialog: the workspace rule's separate Allow levels and Deny grid
  (`components/team/TeamOverridesForm.tsx`) become one three-state list, the panel's permission list for a
  single person.
- Check access in Team covers workspace and instance areas: an instance-area answer names the workspace and
  role (or Owner) that holds it, since it is held in any unrestricted workspace (ADR 0088).
- Explain accepts `resource_type: workspace`; `permission_overwrite_list` too.
- ADR 0148 moves from proposed to accepted; `CONTEXT.md` and the guide match what shipped.

## Acceptance criteria

- [ ] A workspace rule saved as three states round-trips through `account_update` unchanged.
- [ ] `TestExplain_InstanceAreaNamesTheWorkspaceThatHoldsIt`, including a Restricted membership that holds
      nothing at instance level.
- [ ] Verified at 768, 1024 and 1440px.
