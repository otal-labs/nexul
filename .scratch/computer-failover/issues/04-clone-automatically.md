# 04 — Clone automatically

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, the project checkout, step 3. ADR 0162's folder rule.

With the person's switch on, a computer with no checkout of a project that has a GitHub repository clones it before the
run, remembers it and uses it; a computer that may not clone, or a clone that fails, is passed over with the reason.

Done when: a test with the harness fake shows the clone, the remembered checkout, and the pass-over on a failed clone.

## Comments

The setting is stored (`pairing_user_defaults.clone_automatically`, `PUT /api/pairing/defaults`) but has no switch on
the Defaults card yet, and a missing checkout reads "Not on this computer". This ticket adds the switch back with the
clone, and the Projects tab's "Cloned before its first run here" for a missing checkout while it is on.
