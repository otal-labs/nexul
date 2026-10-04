# 06: What the audit of a predecessor produces

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

A project supersedes another (phase 2 in a new repo after phase 1 in the
old one). Once its interview memory is written, the predecessor's code is
audited against it. Decide:

- What the audit produces when the new project is a rewrite: a doc in the
  new project saying what to carry over, what to rebuild, and what to
  avoid; tickets in the new project; findings on the old one; or a mix.
- Whether it is a built-in Audit play seeded in every workspace, or an
  example a workspace adds itself, and where it is started from.
- How it uses the interview: the memory as the yardstick, and the
  question-stance answers ("phase 1 did X, phase 2 does Y") as the list of
  known breaks.
- Whether an audit makes sense for a project's own code too (no
  predecessor), and whether that is in this map.

## Answer

Decided on the recommendations put to the owner, who said to continue;
any of it can be revisited at the walkthrough.

- **A doc, then the existing ticket play.** The audit writes one doc in
  the project's Main folder, "Audit of <what was audited>, <date>",
  grouped under the interview memory's headings. Each finding is one line
  with a verdict, where it is (`path:line`), and the rule it meets or
  breaks. Auditing a predecessor, the verdicts are Carry over, Rebuild,
  and Avoid; auditing the project's own code, they are Keeps and Fix.
  Turning findings into work is "To tickets via AI" on that doc, which
  already exists; the audit files no tickets itself. Each run writes a
  new doc, so earlier audits stay as they were.
- **A built-in play**, "Audit via AI" (key `audit`, type interview),
  seeded beside the others and added to existing workspaces by migration.
  It shows on the Interview page in the memory column's header once the
  interview memory exists; its run state uses the trail icons there, and
  a finished run links the doc.
- **What it measures against**: the interview memory is the yardstick, and
  the stored answers about question sources ("phase 1 did X, phase 2 does
  Y") are the list of known breaks to check first.
- **What it reads**: every `project` source under question, through its
  checkout path as in the drafting run; with none, the project's own
  checkout. So the same button audits a project's own code against its
  rules, which covers a team that has just written an interview for an
  existing codebase.
- **It never asks.** It reads, writes the doc with `doc_create`, and ends
  with a one-line summary, like the drafting run.
