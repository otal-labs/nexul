# A play run names its memories instead of inlining them

Supersedes ADR 0065 for plays; an `@Agent` mention still inlines the interview memory and the other always-included
memories in full.

Amended by ADR 0111: an `@Agent` mention names its always-included memories the same way, no turn carries a memories
index, and a memory's line is its name and id, without its when-to-use.

A play run used to carry the full markdown of every memory it took: the always-included ones, the interview memory,
and the ones picked in the run dialog. The prompt grew with the memories, so a selection had to fit character
ceilings, a run over them was refused before it started, and images in a memory had to be resolved into the turn.
The owner called that a hazard when making a prompt: a picked memory could stop a run, and nobody can see from the
dialog how big a memory is.

Decision: a play run's prompt carries no memory body. It names each memory, the interview memory first as in
ADR 0065, then the other always-included ones, then the picked ones, with its id, name, and when-to-use line, and
tells the agent to read each with `memory_get` before doing anything else and to follow them as standing rules for
the run. A memory the agent cannot read is named in its first message and skipped, a signal rather than a stop. An
answer that resumes a run on a session the harness has lost names them again from the trail's recorded selection. The selection, its check
against the project, and the trail's record of it are unchanged; the size ceilings and the refusal are gone for
plays.

ADR 0065's worry was a cheap model that does not know it needs the rules. That holds for a memory index the agent
may consult; it does not hold for an explicit instruction naming each memory by id as the first step of the run.
The cost: the rules arrive one tool call per memory later instead of in the prompt, a run on a harness without
Nexul's MCP tools gets no memories at all, and an image in a memory reaches the agent only as the
`/api/attachments/<id>` link inside the markdown `memory_get` returns. Rejected: keeping the inlining with higher
ceilings, which moves the hazard instead of removing it.

Decided 2026-10-02.
