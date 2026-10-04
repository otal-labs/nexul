# 02: How a run reads another project and the sources it is given

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

The drafting run, and the follow-up run when a source is under question,
must read material outside the project's own checkout. Find out, with
evidence from the code:

- Where a play run works on the paired computer today (the project's
  checkout, a worktree per thread), and what it would take for the run to
  read a second project's checkout: whether the computer has it, who
  clones or updates it, and what happens when it does not exist there.
- Whether the run's MCP connection, acting as the person who started it,
  can already read another project's docs, memories, and interview
  answers, and what project access does when that person is a restricted
  member.
- How a run is handed things today per ADR 0111 (named by id, read
  through MCP), and whether a source list fits that: named in the prompt,
  or read through the interview memory tools.
- Whether the run can read GitHub issues of the project's repository
  through the connector, as a source kind worth naming.

## Answer

Findings in [research/reading-another-project.md](../research/reading-another-project.md).

- A run works in one T3 project only: the starter's project link for the
  run's own project. Nexul never clones or updates a checkout on a paired
  computer. Reading a second project's checkout means finding its path
  through the starter's project link for that project and the computer's
  T3 project list; the run already has full file access once it has the
  path.
- The run's MCP tools act as the starter, and `memory_get` (which returns
  the interview answers) and `doc_get` already take any project. Another
  project's docs, memories, and answers are readable today. One gap: with
  that project's interview memory deleted, its answers need write access.
- A restricted member without access to the source project gets not found
  on every read and has no project link for it, so no path either.
- A source list carried only in the prompt is lost when an answer resumes
  a run whose harness session is gone; it has to be stored or returned by
  the interview memory tool.
- GitHub issues are not reachable: no issue methods, no tool, and the
  GitHub App lacks Issues: Read, which existing installations would have
  to accept.
