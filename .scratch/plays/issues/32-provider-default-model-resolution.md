# 32 — Provider default model resolution

**What to build:** A turn started with an empty model resolves the provider's default model before the harness sees it: the harness client lists the provider's models and picks the one flagged default, and the pairing resolution falls back the same way, so "Provider default" in settings never sends an empty model. T3 rejects an empty `modelSelection.model` with a defect error today.

**Blocked by:** None — can start immediately

**Status:** done

- [ ] A project link or default with no model starts a turn with the provider's default model; a provider with no default is refused with a clear reason
- [ ] The T3 client test covers the empty-model path
