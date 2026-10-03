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

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md` and `research/runs-nexul-did-not-start.md`.

- [x] Table: model changed → sent; provider changed → sent with Full prompt and a resolved model; unchanged → no `modelSelection` key; empty model, same provider → not sent; empty options → thread options kept

## Comments

- **Where it lives.** `turn.selection` in `internal/t3clientv2/turn.go` decides, from the snapshot thread's
  `modelSelection` (now read by `appThread`), what `message.dispatch` carries; `turn.model` is the empty-model
  resolution `create` used to do inline, now shared. A thread this turn just created is never compared, since it was
  made on the target's pick.
- **Rule as built.** Another provider instance, another model, or options that differ as a set (order ignored, a
  missing option counts as different) send `modelSelection`; otherwise the key is absent. Empty `Model` keeps the
  thread's, unless the provider differs, and then the provider's default is resolved, so a sent model is never empty.
  A target whose provider is not on the computer fails the turn with "provider X not found" before any upload or
  command.
- **Judgment call: empty options.** The ticket says empty `ModelOptions` keep the thread's options. Built so while the
  provider and model stay the same (nothing is sent). When the model or provider changes, a selection with no options
  is sent, so the new model starts on its own defaults as CONTEXT.md's Model options entry says. The old options were
  chosen for another model, and T3 does not check options against a model.
- **Consequences later tickets should know.**
  - Nexul's pick wins over a model someone changed in T3 Code's own composer: the next Nexul turn with a different
    model puts the thread back on the pick.
  - Setting only options on a thread cannot be undone by sending none, since none means "leave them".
  - A switch of provider instance sends the Full prompt with its images, so a conversation that moves provider
    re-sends its whole context once.
- **Docs.** ADR 0114 gained "The model" bullet and the provider-switch Full prompt; ADRs 0058 and 0106 carry an
  amendment line; CONTEXT.md's Model options entry and the `harness.Target.ModelOptions` comment now say the pick holds
  on a reused thread (protocol 1 keeps its session's own, since it stays frozen).
- **Not done.** Protocol 1 is untouched, so on `t3code` a reused session still keeps its own model.
