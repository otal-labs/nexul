# 06: What the triage playbook does and what a filed issue carries

Type: grilling
Status: resolved
Blocked by: 03

## Question

A person hits a bug, perhaps in the phone app, the desktop launcher, or on an
install nobody can shell into. Their own agent should turn it into a
diagnosed issue on `otal-labs/nexul`, the way T3 Code's `npx t3 triage` hands
an agent a playbook and the machine's facts. What does Nexul's playbook do?

- Where it lives: a versioned skill served like the memory skill
  (`skill_get`), so it always matches the instance's version.
- The facts it gathers, and through which tools: the instance version,
  runner and app versions, OS and arch, recent errors from the Logs area if
  the person holds it, the failing request's `trace_id`.
- The diagnosis step: clone the source at the installed version's tag, and
  how far the agent goes before filing (a cause, a suspected file, a failing
  test).
- What a public issue may carry and what never leaves the instance: instance
  URL, workspace and person names, log lines, tokens. Who reviews the draft
  before it is filed.
- Filing: through the person's own GitHub (their `gh` or GitHub MCP), with an
  issue template and a label, and a duplicate search first.
- What a person without GitHub, or without the Logs area, gets: a draft to
  hand on rather than nothing.

## Answer

- **Two kinds of triage, two places.** A problem in a team's own app goes to their own board
  (ticket 07: the crash ticket and "Investigate a crash"). This playbook is only for a bug in
  Nexul itself, and it files on `otal-labs/nexul`, which anyone can read. That is why it has the
  rules below.
- **Where it lives: a second shipped skill, `nexul-triage`,** beside `nexul-memory` in
  `internal/platform/skills`, versioned by its content and installed by computer setup the same
  way. An agent fetches it with `skill_get` when the installed copy is older. A person runs it by
  hand ("triage this with Nexul"); no play or event starts it.
- **The facts, through tools that already exist:** the server version from `instance_get`; runner
  versions, OS and arch from `computer_list` and `machine_list`; the install type; and the
  failing request's `trace_id` if the person has it. With Logs (`logs:read`), it also reads recent
  errors through `instance_get` with `logs`. Without it, the agent says so and asks the person
  for what they saw.
- **The diagnosis.** It clones `otal-labs/nexul` at the installed version's tag, searches the
  open and closed issues for a duplicate, and checks whether a newer release already fixes it. It
  goes as far as a cause and a suspected file, and a failing test if one is cheap. It never
  patches the installed Nexul. It opens a fix pull request only if the person asks, from a fresh
  clone of `master`.
- **What a report may carry:** the versions, OS, arch and install type; the error and stack
  trace with tokens redacted (the server already redacts what `instance_get` returns); and the
  diagnosis. A few quoted log lines only if the person holds Logs. **Never:** the instance's
  address or any host name, or the names of workspaces, projects, stacks or people. The agent
  replaces them with neutral placeholders (`example.com`, `acme`, `alice`) before drafting.
- **Nothing is posted without a yes.** The agent shows the full draft and posts only after the
  person says yes to that draft. A changed draft needs a new yes.
- **Filing as the person, with Nexul's credit.** It posts through the person's own GitHub (`gh`
  or the GitHub MCP), so the issue's author is them. A new form, `.github/ISSUE_TEMPLATE/
  triage_report.yml` ("Triage report", label `via-triage`), has these fields: What happened,
  Diagnosis, Steps to reproduce, Version, Environment, Evidence, Related issues, Workaround, and
  Filed by. The last reads like "alice via Nexul triage · <agent> (`<model>`) · Nexul
  0.3.40". The person's name in it is the one they give, and they can edit it in the draft.
  The label is created with the form.
- **Without GitHub:** the agent builds the prefilled `issues/new?template=triage_report.yml` link
  and also gives the draft as text, so the person can hand it to someone who has an account.
- **A test keeps the skill and the form in step.** It checks that every field the skill fills
  exists in the form, so renaming one breaks the build instead of producing a half-filled issue.
