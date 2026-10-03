# 10 — Regenerate after a change

**What to build:** When any answer was saved after the memory's last
generation, the memory column's header shows the warning dot and "Regenerate
the memory", which starts the follow-up run again; the run reads every
answer, may ask a new round, and rewrites the memory. A project whose memory
predates its answers (the existing-memory state in 05) gets the same action
once it has answers. Nothing regenerates on its own.

**Blocked by:** 08, 09

**Status:** ready-for-agent

- [ ] Changing one answer under a memory shows the signal; regenerating clears it
- [ ] The existing-memory project path works end to end
