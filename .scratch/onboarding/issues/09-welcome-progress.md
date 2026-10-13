# 09: Each person's welcome progress, skip and live updates

Type: task
Status: open
Blocked by: 08

Read first: `AGENTS.md`, `CONTEXT.md` (Welcome wizard, Template role), `practices/go.md` (section 17 for the list read), `practices/architecture.md` (sections 2 to 6),
`practices/mcp.md`, `practices/testing.md` (section 10), `practices/typescript.md` (section 11), the spec
(Architecture).

## What to build

- **The domain:** `internal/welcome/` with the read seams in the spec, and `Progress` returning each step as
  `done`, `skipped` or `left`, with the dot rule.
- **The person's own routes:** `GET /api/me/welcome`, Skip and Unskip, and the first-run route that sets
  `first_login_done`.
- **Giving a step:**
  - A consumer clears skips and sets `given_at` on `welcome_steps_added` and on a role being given.
  - A phone's first sign-in sets `done_at` for `phone`.
  - Enrollment copies `enrolled_by` from the code to the runner.
- **Live updates:** `welcome.progress_changed {user_id}`. That needs its consumer, its audience rule (the person
  and Team viewers), `livePushTopics`, `make live-topics` and `make event-schemas`.
- **MCP:** `account_get` and `account_list` return `welcome`.
- **`client-core/`:** the step keys, their names and the dot rule, with tests.

## Acceptance criteria

- [ ] Table tests for every step's done rule, including a tunnel-paired computer, a revoked GitHub link, and a
      phone that signed out after connecting.
- [ ] Skip, then giving the step through a role tick and through a newly given role, brings it back as `left`.
- [ ] `projects` lights the dot only after it is given.
- [ ] Integration test on real SQLite: pairing and confirming a computer publishes the topic for that person.
- [ ] A guard test counts the statements `account_list` spends on progress, and it stays the same for 1 or 50
      people.
