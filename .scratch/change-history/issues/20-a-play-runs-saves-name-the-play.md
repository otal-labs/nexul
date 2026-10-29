# 20 — A play run's saves name the play

**Status:** needs-triage
**Blocked by:** None — can start immediately

Spec: `../spec.md`, Storage (`via`). Decision 2 wants "Alice, via Implement
with AI"; today a run's agent reaches MCP with the person's per-computer
token, and nothing in a request names the run, so its saves read "via MCP".

Find the smallest way for a run's MCP calls to carry its trail: a per-run
token, a header the harness can set per thread, or matching the person's
running trail on the same target. Then `identity.Actor` gains the trail and
play label, and every history writer records the label as `via`.

Done when: a save made inside a play run records the play's label, with a
test through the MCP adapter.
