# 14 — The History dialog, on docs

**Status:** ready-for-agent
**Blocked by:** 10, 13

Spec: `../spec.md`, decision 9 and The diff. Look: ticket 05's answer.

The shared `HistoryDialog` (list, rendered body with word marks and margin
bars, Show changes switch, Restore this version) and its first home, the
doc page's menu. Adds History, Version and Sitting to `CONTEXT.md`, and a
History page to the user guide.

Done when: the dialog works on the demo data and the worst-case seed at 768,
1024 and 1440; a component test covers picking a version, the switch, the
disabled restore on the current one and a restore; screenshots in the PR.
