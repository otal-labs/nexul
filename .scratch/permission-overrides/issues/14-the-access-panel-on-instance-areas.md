# 14 — The access panel on instance areas

**Status:** ready-for-agent

**Blocked by:** 03, 13

Read first: `practices/react-guide.md` (all of F1 to F7, Live events), `practices/design-language.md`,
`practices/testing.md`, ADR 0148, the spec (UI structure, Instance areas).

## What to build

- Each instance-settings entry that is an instance area (Instance, Sign-in providers, each connector, DNS,
  Templates, Team, and the rest of the entity table, computer activity included) gains an access section for the
  instance Owner: the permissions panel from ticket 03 with a role or person picker and no Everyone,
  limited to that entity's actions, with Check access. Other viewers do not see the section.
- The role picker lists roles with their workspace named; a person picker lists accounts.
- The panel warns before a Deny that removes a person's last route to an area.
- Live: another open page updates when a rule changes.
- The guide page for instance settings shows the panel and the DNS-but-not-connectors example in plain words.

## Acceptance criteria

- [ ] Giving a person DNS and no connector works end to end, and Check access shows the source for each.
- [ ] A viewer who is not the instance Owner sees no access section.
- [ ] Verified at 768, 1024 and 1440px, with before and after screenshots in the PR.
- [ ] The self-review checklist in `practices/react-guide.md` passes.
