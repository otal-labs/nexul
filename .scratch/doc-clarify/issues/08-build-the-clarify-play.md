# 08 — The Clarify via AI play and its rounds

**What to build:** The run lifecycle decided in 04, on the storage from 07.
The built-in `clarify` doc play seeded by `SeedDefaults` and by a forward
migration into existing workspaces, with the instructions from 05 as the
code default and in the instance templates. The runner opens a round when a
Clarify run launches (recording starter, trail, and whether it took the
lock) and closes it at every end of the run, unlocking the doc if this run
locked it, and removing a round that ended with no questions and no
verdict. A run followed again after a restart ends its round the same way.
The run dialog shows the signal when the last round still has unanswered
questions.

**Blocked by:** 05, 07

**Status:** ready-for-agent

- [ ] Runner tests for every end: done, failed, stopped, silence, restart
      before start, restart mid-run; a doc locked before the run stays locked
- [ ] Seed migration tested by upgrading; new workspaces get the play
- [ ] ADR 0121 and the `play_run` / `doc_update` descriptions agree with the code
