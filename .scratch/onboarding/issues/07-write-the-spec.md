# 07: Write the spec

Type: task
Status: resolved
Blocked by: 02, 03, 04, 05, 06

## Question

Assemble `.scratch/onboarding/spec.md` from the map's decisions and the resolved
tickets. Walk AGENTS.md's "Hit every surface" list, and answer what is left in
Not yet specified or rule it out: the phone app step, live updates of the dot,
reading progress over MCP, the guide page and the roadmap. Then slice it into
build tickets.

## Answer

`spec.md` is written, and build tickets 08 to 15 are in `issues/`. The spec settled what was still open on the
map:
- The phone app shows no dot and no wizard.
- The dot and the Team line update live through `welcome.progress_changed {user_id}`.
- `account_get` and `account_list` return welcome progress, so an agent can see it.
- The guide gets a new welcome wizard page and updates to three others.
- The roadmap lists the effort as Planned.
