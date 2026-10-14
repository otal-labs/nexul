# 15: An item's own page locks by its rules

Type: task
Status: resolved
Blocked by: 09

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, `practices/testing.md` (section 10), the spec.

## What to build

- `useEntityActions(kind, id)` calls explain once per detail page, for docs, plays, projects, and stacks under the shipped
  rules (ADR 0148). It refetches on the rules' live frame for that item.
- A list action refused on click toasts the source sentence.
- A statement guard covers explain.

## Acceptance criteria

- [x] A rule on one doc locks that doc page's action with the rule's sentence.
- [x] A list keeps one request per page.

## Answer

- **explain already answers every action of one item in one request** (its `effective` list), so nothing was widened.
  `useEntityActions(kind, id)` reads it through the shared `useFetchExplanation` key: every component on a detail page
  shares one request, and `access.grant.changed` already names the item (`resource_type`, `resource_id`), so no event
  field was added. A folder's or project's rule frame also rereads the open docs' and stacks' answers, since it names none.
- **A refusal carries its sentence.** The forbidden envelope gains `details: {action, sentence}`, additive, and
  `errorMessage` shows the sentence, so a refused click on a list action toasts the rule's source.
- **Detail pages moved onto the action components with their answers:** the doc page (Create ticket, Permissions,
  Lock/Unlock, Archive/Restore, Thread, whether the body edits), the play edit dialog (Save waits, with the sentence),
  the project settings (rename, Add a role or person, Remove project) and the stack page (Build & deploy, Redeploy,
  Rollback, Deploy now, environment editing, branch deploy rules, Delete stack). Lists, the rest of the project
  settings sections and the board stay with tickets 11 to 13.
- **The guard** is `TestStatements_ExplainCostsTheSameForAnyNumberOfActions`: a doc, play, project and stack explain
  costs 8, 6, 7 and 8 statements with a rule on one action or on every action, so a read per action fails it.
