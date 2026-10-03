# 18 — The picked model runs even on a reused T3 thread

**What to build:** On protocol 2, `message.dispatch` carries `modelSelection` when the target asks for
something the reused thread's selection (from the snapshot) is not: a different provider instance, a
non-empty model that differs, or non-empty options that differ. An empty `Target.Model` keeps the
thread's model, unless the provider differs, in which case the provider's default is resolved; empty
`ModelOptions` keep the thread's options; a sent `modelSelection` always has a non-empty model. A
different provider instance also sends the Full prompt, because T3 hands the new provider only a
summary. Update the `harness.Target.ModelOptions` doc comment and `CONTEXT.md` model options entry
(ADR 0058's per-run choice now holds on reused threads).

**Blocked by:** 05

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md` and `research/runs-nexul-did-not-start.md`.

- [ ] Table: model changed → sent; provider changed → sent with Full prompt and a resolved model; unchanged → no `modelSelection` key; empty model, same provider → not sent; empty options → thread options kept
