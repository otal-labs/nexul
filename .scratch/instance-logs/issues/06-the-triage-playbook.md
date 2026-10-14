# 06: What the triage playbook does and what a filed issue carries

Type: grilling
Status: open
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
