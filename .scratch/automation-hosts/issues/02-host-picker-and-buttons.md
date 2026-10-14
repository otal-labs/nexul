# 02: The host picker lists hosts you may deploy to, and host buttons follow the machine

**Status:** ready-for-agent
**Blocked by:** 01

Read first: `AGENTS.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`,
`practices/testing.md`, the spec, and `.scratch/role-aware-ui/spec.md` for how a locked action looks.

## What to build

The spec's UI section: `AutomationHostPicker` lists the instance host and hosts on machines the viewer may Deploy to;
Add host and Remove host follow the machine's `actions`; the Access card drops "automation hosts". Verified at 768,
1024 and 1440px.

## Acceptance criteria

- [ ] Component tests for the picker's filtering and the buttons' states.
