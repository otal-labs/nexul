# 24 — The access panel on instance areas

**Status:** ready-for-agent

**Blocked by:** 06, 23

**Branch:** master, a normal PR, after the switch.

Read first: `practices/react-guide.md` (F1 to F7, Live events, the self-review), `practices/design-language.md`,
`practices/testing.md`, ADR 0148, the spec (Instance areas, UI), `.scratch/permission-rework/issues/03-the-permissions-panel.md`.

## What to build

- Each instance-settings entry that is an instance area (Instance, Sign-in providers, each connector, DNS, Templates,
  Team, and the rest of the entity table) gains an access section for the instance Owner only: the panel from slice 06
  with role and person targets, limited to that entity's actions, with Check access.
- The role picker names each role's workspace; the person picker lists accounts.
- The panel warns before a Deny that removes a person's last route to an area.
- Another open page updates when a rule changes.
- The guide page for instance settings shows the panel and the DNS-but-not-connectors example.

## Acceptance criteria

- [ ] Giving a person DNS and no connector works end to end, and Check access shows each source.
- [ ] A viewer who is not the instance Owner sees no access section.
- [ ] Verified at 768, 1024 and 1440px with before and after screenshots, and the self-review passes.
