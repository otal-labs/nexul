# 19 — Memories and automations in the same dialog

**Status:** ready-for-agent
**Blocked by:** 14

Spec: `../spec.md`, decisions 1, 4, 5, The route shape.

Memories skip unchanged saves, keep 10 versions, gain `editor_ids`, answer
`/history`, and open the History dialog in place of their version feed;
`memory_get` lists the 10 kept. Automations answer `/history` from
`automation_versions` and open the same dialog for code, keeping merge and
rollback, with active and pending never pruned. `CONTEXT.md`'s memory entry
changes to "every save that changes something".

Done when: tests cover the memory skip and prune, the automation history
shape, and the existing automation diff and rollback still passing;
screenshots at 768, 1024 and 1440.
