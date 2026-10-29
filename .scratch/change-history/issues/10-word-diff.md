# 10 — Word diff that stays fast on long text

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, The diff.

`diffWords` beside `diffLines` in `web/src/utils/DiffUtility.tsx`, both
skipping the matching run at each end before the LCS table. The automation
diff keeps working unchanged.

Done when: table tests cover an insert, a removal, a replacement inside a
line, empty sides and identical text; a guard counts the LCS cells for a
one-word edit in a 5,000-paragraph text and fails if the trim is removed.
