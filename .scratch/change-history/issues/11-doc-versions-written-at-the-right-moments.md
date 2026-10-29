# 11 — Doc versions: no empty saves, a cap of 10, and a history route

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, decisions 4, 5 and 7, The route shape, Storage.

A doc save whose title and body match the last version writes none. Saving
past 10 unnamed versions prunes the oldest; named ones stay. `doc_versions`
gains `editor_ids` and `via` (migration asked for in the PR). `GET
/api/docs/{id}/history` returns the shared shape.

Done when: integration tests on real SQLite show the skipped save, the prune
keeping named versions, `via` recorded from a play run's actor, and the
history route returning no bodies; a reader without docs read gets 404.
