# 04 — Clone automatically

**Status:** ready-for-agent
**Blocked by:** 02

Spec: `../spec.md`, the project checkout, step 3. ADR 0162's folder rule.

With the person's switch on, a computer with no checkout of a project that has a GitHub repository clones it before the
run, remembers it and uses it; a computer that may not clone, or a clone that fails, is passed over with the reason.

Done when: a test with the harness fake shows the clone, the remembered checkout, and the pass-over on a failed clone.
