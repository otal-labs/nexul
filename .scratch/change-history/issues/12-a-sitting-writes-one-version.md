# 12 — The live editor writes one version per sitting

**Status:** ready-for-agent
**Blocked by:** 11

Spec: `../spec.md`, The sitting.

The collab room collects editors since its last version, resets a 5-minute
timer on each update, and calls `CommitSitting` when it fires or the last
writer leaves. A reset from an API save writes the sitting first. Tickets
implement `CommitSitting` as a no-op until ticket 15.

Done when: `testing/synctest` tests show one version after an idle sitting
crediting both editors, one when the last writer leaves, none for a sitting
that changed nothing, and the sitting saved before an API save resets the
room; goleak stays clean.
