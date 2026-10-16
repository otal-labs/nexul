# 19: Build the triage skill and the Triage report form

Type: task
Status: resolved
Blocked by: 11

## What to build

Ticket 06, as decided: the `nexul-triage` skill beside `nexul-memory`, installed by computer setup
and served by `skill_get`; `.github/ISSUE_TEMPLATE/triage_report.yml` with the `via-triage` label;
a test that every field the skill fills exists in the form.

## Surfaces

- MCP: `skill_get` lists it.
- GitHub: the form and the label.
- Docs: a triage guide page.

## Built

- Setup now compares one skills-set version (`skills.SetVersion`, a hash of every shipped skill),
  so installs set up before this show "skills out of date" once and receive the triage skill.
- First-time setup installs every shipped skill, not memory alone.
- Filing uses `gh issue create --label via-triage` with `### <label>` sections in the form's
  order, since `gh` can't submit a form. The `via-triage` label exists on the repo.
