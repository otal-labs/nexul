# 17 — The upgrade test, and the switch PR

**Status:** done

**Blocked by:** 12, 13, 14, 15, 16

**Branch:** `permission-switch`, then one squash-merged PR to master.

Read first: AGENTS.md (Branches and pull requests, Hit every surface), `practices/testing.md`, ADR 0148, the spec
(The upgrade, Build and release, Risks), `.scratch/permission-rework/issues/06-build-order-and-release.md`
(section 5, Upgrade testing for the wipe).

## What to build

- Rebase onto master, and number the branch's migrations as the next free numbers.
- Run the local CI mirror, then the box test of rework ticket 06 section 5 on `nexul-box`: start from the newest beta,
  load a production copy and the synthetic Acme workspace with every row shape, take the before Matrix, upgrade with
  the branch's binaries, check rows, Owners, members, things acting as a Developer, old invitation links, pills and
  the phone, Finish setup, and drill the rollback.
- "Nothing widens": every answer a non-Owner holds after the upgrade was held before, except the two the spec names
  (a former Restricted member reads public channels; an instance bit their roles carry now counts).
- ADR 0148 moves to accepted. Check `CONTEXT.md` and the guide pages (`people-and-access`,
  `projects-and-repositories`, `plays`, `api-and-tokens`) against what was built.
- Open the PR with the title "Rebuild permissions on roles: projects are opt-in, people hold several roles, and an
  Owner sets who's in each project again after upgrading", its screenshots, and every box-test assertion in its body.
  Squash-merge it.
- After the merge, once the daily beta carrying it is cut, repeat the box test on the production copy through
  `nexul upgrade --version <that beta>`, then take a `.backup` of production right before nexul.otal.dev upgrades.

## Acceptance criteria

- [ ] Every box-test assertion holds and is recorded in the PR body. All are recorded and all hold but one: "nothing
      widens" fails where the wipe lifts a deny (a play exclusion, a Team deny), which the spec's two exceptions don't
      cover. The PR body lists the answers; the owner decides whether the spec names a third exception.
- [x] The rollback drill boots the previous release on its old data.
- [x] The production copy is deleted afterwards and never enters the repo. Replaced by the owner's decision of
      2026-10-11: no production copy is made; the box test runs on a production-shaped synthetic dataset only.
