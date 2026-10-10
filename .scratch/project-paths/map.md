# Wayfinder map: starting a project without deploying it

Charted 2026-10-05 with the owner. Not every project starts as code to
deploy. An owner makes a project for a client to write docs in before any
code exists, or attaches a repository that is not ready to deploy yet. The
project wizard handles both badly: the only way out after naming the project
is a muted "Skip for now" that lands on the board without saying what was
made, and picking a repository always carries on into deploying it, so
"repository but no deploy" is only reachable through Project settings.

## Destination

Built and merged: the wizard's Repository step offers "No repository yet"
and "Attach without deploying" as visible choices, both end on a Done
screen that says what was made and keeps the interview offer, and a
project with an undeployed repository shows the way back to deploying it.
This map carries the build: the build tickets are worked here once the
decision tickets they wait on are resolved.

## Notes

- Decided while charting:
  - The choice lives on the Repository step, where the question comes up;
    no new up-front "what is this project for?" step. The project already
    exists by then (ADR 0079).
  - Both early exits end on a Done screen, not the board: it names what
    was made, keeps the interview offer, and links onward (docs, board,
    inviting people).
  - A project stores no intent ("docs only", "not deploying"). What it has
    decides what it shows: no repository, no deploy prompts; a repository
    no stack builds from, a "Deploy this repository" prompt.
  - "Add service" on a project with exactly one undeployed repository goes
    straight to that repository instead of asking for it again, and the
    project shows a "Deploy this repository" prompt while one is waiting.
- Look and copy tickets go through `design-mode` and are reacted to by the
  owner; technical questions are answered in the ticket and not grilled.
- Web only, built at 768px and verified at 768, 1024, and 1440px. Read
  `practices/react-guide.md` and `practices/design-language.md` before any
  web edit.

- Since ADR 0143, Skip for now records the step skipped and moves to the next
  step instead of leaving for the board, every step opens from the row, and
  the Done step lists what was skipped and ends with Finish. The early exits
  built here land on that Done step.

## Decisions so far

- [How attaching without deploying and the Add-service shortcut work](issues/03-attach-and-shortcut-mechanics.md): no server change; attach through the existing repos endpoint after the scan, save the tests answer on every exit, derive "undeployed" in the web, Done renders on the project.

## Not yet specified

- Whether the website guide on projects and repositories
  (`website/src/content/docs/docs/guide/projects-and-repositories.md`) and
  the setup wizard guide describe the skip in a way the new choices make
  wrong; settled in the walkthrough.

## Out of scope

- MCP: `project_create` already makes a project with no repository, and
  `project_update` with `add_repos` attaches one without a stack, so agents
  can already do both.
- The phone app: it has no project wizard (web 768px and up only).
- Remembering a project's intent: ruled out while charting, see Notes.
