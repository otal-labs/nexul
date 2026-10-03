# A play concludes with footer memories, not a chosen column

Supersedes the move-on-success consequence of ADR 0055.

A ticket play's run dialog used to ask which column to move the ticket to when the run ended done, and the runner
applied it afterwards, never backwards. The owner found the picker getting in the way on every run: the right column
depends on how the work went, which nobody knows at the press, and the never-backwards rule then skipped the move
with a note whenever the default automations had already moved the card.

Decision: the run dialog has no column picker and the runner never moves a ticket. A memory can be marked a footer.
The run dialog lists footer memories in their own section at the bottom, and a run names the picked ones, and any
always-included footer, last in the request, after the run's own instructions, telling the agent to read them once
the work is done and follow them to conclude the run. A footer memory says where the ticket goes and anything else
the run should finish with, and the agent moves the ticket with `ticket_update` as the starter. Memories carry the
context and processes; footers carry the conclusion. Only an ordinary memory can be a footer: the interview memory
and the decisions log never are. Old trails lose the column they recorded.

The cost: a ticket move from a play now carries the starter's MCP provenance instead of the play's, and nothing
stops an agent moving a ticket backwards except what the footer says. Rejected: a per-play footer field next to the
instructions, which would duplicate the memory machinery (picking, versions, cloning) for one more block of text.

Decided 2026-10-03.
